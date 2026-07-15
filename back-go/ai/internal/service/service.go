// Package service — бизнес-логика aisvc. Портировано из
// back/app/services/ai_client.py, api/ai_settings.py и
// services/task_embedding_service.py без изменения правил.
package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
)

const (
	// cacheTTL — кэш AI-настроек компании (_CACHE_TTL_SEC во Flask): после
	// сохранения настроек worst-case минута до подхвата — приемлемо;
	// PUT ai-settings инвалидирует сразу.
	cacheTTL = 60 * time.Second

	// requestTimeout — дефолтный таймаут upstream-запроса (_REQUEST_TIMEOUT).
	requestTimeout = 30 * time.Second

	// defaultMaxTokens / defaultTemperature — дефолты chat() во Flask.
	defaultMaxTokens   = 400
	defaultTemperature = 0.7

	// Семантический поиск: чтобы в выдачу не сыпалась «ерунда» (косинусное
	// сходство у эмбеддингов почти всегда > 0), держим два фильтра —
	//   minSemanticScore  — абсолютный порог осмысленной близости к запросу;
	//   semanticScoreBand — относительный «обрыв»: отбрасываем хиты, сильно
	//                       отставшие от лучшего совпадения (длинный хвост
	//                       слабосвязанных задач), даже если они выше порога.
	// semanticLimit — потолок кандидатов из БД (дальше их всё равно режут пороги).
	minSemanticScore  = 0.30
	semanticScoreBand = 0.12
	semanticLimit     = 50

	// embedBatchSize — размер пачки эмбеддингов (OpenAI принимает до 2048).
	embedBatchSize = 64

	// reindexWorkers — потолок одновременных фоновых переиндексаций задач
	// (ReindexTask отвечает сразу, работа в горутинах под семафором).
	reindexWorkers = 4
)

// StatusResult — ответ gRPC Status.
type StatusResult struct {
	Enabled        bool
	ModelChat      string
	ModelEmbedding string
}

// ChatArgs — запрос gRPC Chat (один ход; циклы tool-calling — у вызывающего).
type ChatArgs struct {
	CompanyID    int64
	MessagesJSON string
	ToolsJSON    string
	MaxTokens    int
	Temperature  float64
	TimeoutSec   float64
}

// AiService — все use-case'ы сервиса (REST + gRPC).
type AiService interface {
	// REST /api/companies/<id>/ai-settings*
	GetSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.AiSettings, error)
	UpdateSettings(ctx context.Context, actor *domain.User, companyID int64, upd dto.AiSettingsUpdate) (*dto.AiSettings, error)
	TestSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.AiTestResult, error)
	IndexingStatus(ctx context.Context, actor *domain.User, companyID int64) (*dto.IndexingStatus, error)
	StartReindex(ctx context.Context, actor *domain.User, companyID int64) (*dto.ReindexQueued, error)

	// REST /api/ai/tv-fact — текущий ТВ-факт дня (nil → JSON null).
	GetTVFact(ctx context.Context, companyID int64) (*domain.TVFact, error)

	// gRPC ai.v1 (SemanticSearch/Embed зовёт tasksvc; Chat — и снаружи, и
	// внутрипроцессно самим ассистентом, см. assistant.go).
	Status(ctx context.Context, companyID int64) (*StatusResult, error)
	Chat(ctx context.Context, args ChatArgs) (*domain.ChatResult, error)
	Embed(ctx context.Context, companyID int64, text string) ([]float32, string, error)
	SemanticSearch(ctx context.Context, companyID int64, query string) ([]domain.SearchHit, error)
	// ScheduleReindexTask — асинхронно, ошибки только в лог (fail-open).
	ScheduleReindexTask(taskID int64)

	// REST /api/ai/assistant/* — деловой ИИ-ассистент (статистика/задачи).
	SendAssistantMessage(ctx context.Context, userID, companyID int64, text string) (*AssistantReply, error)
	GetAssistantHistory(ctx context.Context, userID, companyID int64, limit int, before *time.Time) ([]domain.AssistantMessage, error)
	SendAssistantFeedback(ctx context.Context, userID, companyID, messageID int64, verdict string, reason *string) error

	// REST /api/ai/text-tools — ИИ-инструменты текста заметок (texttools.go).
	TransformText(ctx context.Context, companyID int64, action, style, text string) (string, error)

	// gRPC SupportChat — ИИ техподдержки dev-чата (support.go, зовёт msgsvc).
	SupportReply(ctx context.Context, messagesJSON string) (string, error)
}

type Service struct {
	repo   domain.Repository
	llm    domain.LLMClient
	cipher domain.SecretCipher
	facts  domain.FactCache
	log    *slog.Logger

	// ИИ-ассистент (Сущность 3): хранилище диалога + gRPC-клиент tasksvc
	// (инструменты статистики/поиска задач) + база публичных ссылок на задачи.
	assistants domain.AssistantRepository
	tasks      domain.TasksClient
	appBaseURL string

	// Платформенный LLM техподдержки (support.go): у dev-чата нет компании,
	// компанийные ключи не подходят. Пустой ключ — поддержка без ИИ.
	support SupportConfig

	// кэш «готовых клиентов» per-company (как _cache во Flask ai_client).
	mu    sync.Mutex
	cache map[int64]cacheEntry

	// защита от параллельных бэкфиллов одной компании.
	backfills sync.Map // company_id → struct{}

	// семафор фоновых переиндексаций одной задачи.
	reindexSem chan struct{}
}

type cacheEntry struct {
	client  *aiClient
	expires time.Time
}

// aiClient — расшифрованные настройки компании, готовые к вызовам upstream
// (аналог AIClient во Flask).
type aiClient struct {
	companyID      int64
	apiKey         string
	modelChat      string
	modelEmbedding string
}

var _ AiService = (*Service)(nil)

func New(repo domain.Repository, llmClient domain.LLMClient, cipher domain.SecretCipher,
	facts domain.FactCache, assistants domain.AssistantRepository, tasks domain.TasksClient,
	appBaseURL string, support SupportConfig, log *slog.Logger) *Service {
	return &Service{
		repo:       repo,
		llm:        llmClient,
		cipher:     cipher,
		facts:      facts,
		assistants: assistants,
		tasks:      tasks,
		appBaseURL: appBaseURL,
		support:    support,
		log:        log,
		cache:      map[int64]cacheEntry{},
		reindexSem: make(chan struct{}, reindexWorkers),
	}
}

// ── Ошибки ───────────────────────────────────────────────────────

// errNotFound — {"error": "NOT_FOUND"} без message, как jsonify во Flask.
func errNotFound() *domain.Error {
	return domain.NewError("NOT_FOUND", "", 404)
}

func errNoAccess() *domain.Error {
	return domain.NewError("FORBIDDEN", "Нет доступа к настройкам этой компании", 403)
}

// errAiDisabled — REST отдаёт 409 (UI: «сначала введите ключ и включите AI»),
// gRPC Chat/Embed — 403.
func errAiDisabled(status int) *domain.Error {
	return domain.NewError("AI_DISABLED", "AI выключен или ключ не задан", status)
}

// ── Клиент компании с кэшем (get_ai_client) ──────────────────────

// clientFor — nil без ошибки, если AI выключен / ключа нет / ключ
// нерасшифровываемый. Положительный результат кэшируется на cacheTTL,
// отрицательный затирает кэш (чтобы выключение подхватилось сразу).
func (s *Service) clientFor(ctx context.Context, companyID int64) (*aiClient, error) {
	now := time.Now()
	s.mu.Lock()
	if e, ok := s.cache[companyID]; ok && e.expires.After(now) {
		s.mu.Unlock()
		return e.client, nil
	}
	s.mu.Unlock()

	company, err := s.repo.GetCompanyAI(ctx, companyID)
	if err != nil {
		return nil, err
	}
	client := s.buildClient(company)

	s.mu.Lock()
	if client != nil {
		s.cache[companyID] = cacheEntry{client: client, expires: now.Add(cacheTTL)}
	} else {
		delete(s.cache, companyID)
	}
	s.mu.Unlock()
	return client, nil
}

func (s *Service) buildClient(company *domain.CompanyAI) *aiClient {
	if company == nil || !company.Enabled || len(company.APIKeyEnc) == 0 {
		return nil
	}
	apiKey, ok := s.cipher.Decrypt(company.APIKeyEnc)
	if !ok {
		s.log.Warn("ai.decrypt_failed", "company_id", company.ID)
		return nil
	}
	return &aiClient{
		companyID:      company.ID,
		apiKey:         apiKey,
		modelChat:      company.ChatModel(),
		modelEmbedding: company.EmbeddingModel(),
	}
}

// invalidateClient — вызывать сразу после изменения AI-настроек компании.
func (s *Service) invalidateClient(companyID int64) {
	s.mu.Lock()
	delete(s.cache, companyID)
	s.mu.Unlock()
}

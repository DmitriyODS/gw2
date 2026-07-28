package domain

import (
	"context"
	"time"
)

// Repository — персистентность aisvc: AI-поля companies + task_embeddings
// (pgvector) + read-only лукапы tasks/departments/users для текстов
// эмбеддингов и подсчётов индексации.
type Repository interface {
	// GetCompanyAI — AI-срез компании; nil — компании нет.
	GetCompanyAI(ctx context.Context, companyID int64) (*CompanyAI, error)
	// UpdateCompanyAI — сохранить ai_*-поля компании.
	UpdateCompanyAI(ctx context.Context, c *CompanyAI) error

	// MembershipLevel — уровень роли пользователя в КОНКРЕТНОЙ компании
	// (user_companies); 0 — не член. Доступ к AI-настройкам скоупится этой
	// компанией, а не активной компанией сессии.
	MembershipLevel(ctx context.Context, userID, companyID int64) (int, error)

	// GetUserAI — личные ИИ-настройки пользователя; строки может не быть —
	// тогда nil без ошибки (человек ещё не подключал свой ключ).
	GetUserAI(ctx context.Context, userID int64) (*UserAI, error)
	// UpsertUserAI — сохранить личные ИИ-настройки (INSERT … ON CONFLICT).
	UpsertUserAI(ctx context.Context, u *UserAI) error

	// CountTasks — все задачи компании (total_tasks в indexing-статусе).
	CountTasks(ctx context.Context, companyID int64) (int, error)
	// CountEmbeddings — проиндексированные; model "" — без фильтра по модели
	// (как count_embeddings во Flask).
	CountEmbeddings(ctx context.Context, companyID int64, model string) (int, error)
	// FindUnindexedTaskIDs — задачи компании без эмбеддинга или с эмбеддингом
	// другой модели.
	FindUnindexedTaskIDs(ctx context.Context, companyID int64, model string) ([]int64, error)

	// GetTaskText / ListTaskTexts — задача(и) с именем отдела и ФИО
	// ответственного; nil/пропуск — нет такой.
	GetTaskText(ctx context.Context, taskID int64) (*TaskText, error)
	ListTaskTexts(ctx context.Context, ids []int64) ([]*TaskText, error)

	// UpsertEmbedding — INSERT ... ON CONFLICT (task_id) DO UPDATE.
	UpsertEmbedding(ctx context.Context, taskID, companyID int64, vector []float32, model string) error
	// SearchEmbeddings — косинусный поиск (оператор <=>) по компании и модели,
	// упорядочен по релевантности; фильтр score > 0 — на вызывающем.
	SearchEmbeddings(ctx context.Context, companyID int64, vector []float32, model string, limit int) ([]SearchHit, error)

	// GetPlatformAI / UpdatePlatformAI — платформенный ключ и модели
	// (задаёт супер-админ в «Аудите платформы»).
	GetPlatformAI(ctx context.Context) (*PlatformAI, error)
	UpdatePlatformAI(ctx context.Context, p *PlatformAI) error
	// ListModels / UpsertModel — каталог моделей и их цены (по цене считается
	// стоимость обращения в токенах доступа).
	ListModels(ctx context.Context) ([]*AIModel, error)
	UpsertModel(ctx context.Context, m *AIModel) error
	// CompanyOwner — создатель компании: ИИ-возможности компании тратят ЕГО
	// токены (0 — компании нет или создатель удалён).
	CompanyOwner(ctx context.Context, companyID int64) (int64, error)

	// AICompanyIDs — компании с включённым AI (цикл генерации ТВ-фактов).
	AICompanyIDs(ctx context.Context) ([]int64, error)
	// TVWeekContext — метрики компании в окне (для контекстного ТВ-факта).
	TVWeekContext(ctx context.Context, companyID int64, start, end time.Time) (*TVWeekContext, error)
}

// FactCache — ТВ-факты дня в Redis. Ключи gw2:ai:tv_fact:{cid} сохранены
// с Flask-времён (services/tv_facts_service.py).
type FactCache interface {
	// GetFact — nil без ошибки, если факта нет или JSON битый.
	GetFact(ctx context.Context, companyID int64) (*TVFact, error)
	SetFact(ctx context.Context, companyID int64, fact *TVFact, ttl time.Duration) error
	DeleteFact(ctx context.Context, companyID int64)
}

// UserReader — read-only доступ к пользователям платформы (auth-мидлварь
// и проверка доступа к настройкам компании).
type UserReader interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	// CompanyActive — активна ли выбранная (активная) компания сессии из
	// токена. nil (Администратор системы) → true.
	CompanyActive(ctx context.Context, companyID *int64) (bool, error)
}

// ChatParams — параметры одного хода chat completion. MessagesJSON/ToolsJSON —
// сырые JSON-массивы в формате OpenAI API (ToolsJSON "" — без инструментов).
type ChatParams struct {
	APIKey string
	// BaseURL — адрес API; пусто = общий адрес платформы. Непустой приходит
	// от пользователя, подключившего СВОЙ сервер модели.
	BaseURL      string
	Model        string
	MessagesJSON string
	ToolsJSON    string
	MaxTokens    int
	Temperature  float64
	Timeout      time.Duration
}

// ChatResult — ответ модели: либо текст, либо сырой JSON массива tool_calls.
type ChatResult struct {
	Content       string
	ToolCallsJSON string // "" — обычный текстовый ответ
	// Usage — расход токенов у провайдера: по нему считается списание с
	// баланса тарифа (billingsvc).
	Usage TokenUsage
}

// TokenUsage — сколько токенов реально потратил запрос.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
}

// Total — суммарный расход обращения.
func (u TokenUsage) Total() int { return u.PromptTokens + u.CompletionTokens }

// EmbedParams — запрос эмбеддингов (тот же upstream, свой набор полей).
type EmbedParams struct {
	APIKey  string
	BaseURL string
	Model   string
	Texts   []string
	Timeout time.Duration
}

// LLMClient — OpenAI-совместимый upstream (ProxyAPI). Ошибки сети/API —
// *Error AI_UPSTREAM (502, таймаут — 504).
type LLMClient interface {
	ChatOnce(ctx context.Context, p ChatParams) (*ChatResult, error)
	// Embed — векторы в порядке входных текстов + израсходованные токены.
	Embed(ctx context.Context, p EmbedParams) ([][]float32, int, error)
}

// TokenMeter — баланс токенов доступа (billingsvc). Недоступный биллинг не
// должен ронять ИИ: реализация fail-open разрешает обращение и не списывает.
type TokenMeter interface {
	// Check — можно ли обратиться к модели: плательщик (для компанийных
	// функций — создатель компании) и остаток его токенов.
	Check(ctx context.Context, userID, companyID int64) (payerID int64, left int64, ok bool)
	// Consume — списать токены доступа после ответа модели.
	Consume(ctx context.Context, u TokenSpend) (ok bool, left int64)
}

// TokenSpend — что именно списываем.
type TokenSpend struct {
	PayerID   int64
	ActorID   int64
	CompanyID int64
	Feature   string
	Model     string
	Usage     TokenUsage
	Billed    int64
	// OwnKey — запрос ушёл на личный ключ пользователя: токены не тратятся,
	// расход только фиксируется.
	OwnKey bool
}

// SecretCipher — Fernet-шифрование AI-ключей компаний (AI_KEY_ENCRYPTION_KEY).
type SecretCipher interface {
	// Encrypt — ErrSecretMisconfigured, если ключ шифрования не задан.
	Encrypt(plain string) ([]byte, error)
	// Decrypt — ok=false, если токен нерасшифровываем (сменили ключ) или
	// шифрование не сконфигурировано: фичи AI тихо выключаются, как во Flask.
	Decrypt(enc []byte) (plain string, ok bool)
}

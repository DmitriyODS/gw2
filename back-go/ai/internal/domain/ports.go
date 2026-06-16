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
	APIKey       string
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
}

// LLMClient — OpenAI-совместимый upstream (ProxyAPI). Ошибки сети/API —
// *Error AI_UPSTREAM (502, таймаут — 504).
type LLMClient interface {
	ChatOnce(ctx context.Context, p ChatParams) (*ChatResult, error)
	// Embed — векторы в порядке входных текстов.
	Embed(ctx context.Context, apiKey, model string, texts []string, timeout time.Duration) ([][]float32, error)
}

// SecretCipher — Fernet-шифрование AI-ключей компаний (AI_KEY_ENCRYPTION_KEY).
type SecretCipher interface {
	// Encrypt — ErrSecretMisconfigured, если ключ шифрования не задан.
	Encrypt(plain string) ([]byte, error)
	// Decrypt — ok=false, если токен нерасшифровываем (сменили ключ) или
	// шифрование не сконфигурировано: фичи AI тихо выключаются, как во Flask.
	Decrypt(enc []byte) (plain string, ok bool)
}

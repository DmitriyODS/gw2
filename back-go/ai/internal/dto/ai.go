// Package dto — JSON-формы REST-ответов, байт-в-байт совместимые с прежними
// Flask-схемами (back/app/schemas/ai_settings.py и api/ai_settings.py).
package dto

// AiSettings — что отдаём наружу. Сырого ключа НЕТ — только маска key_hint.
// AiSettings — ИИ-возможности КОМПАНИИ. Своего ключа у компании нет: работа
// идёт на платформенном, а токены тратятся у создателя компании (shared).
type AiSettings struct {
	Enabled        bool   `json:"enabled"`
	Shared         bool   `json:"shared"`
	FeatSearch     bool   `json:"feat_search"`
	FeatTVFact     bool   `json:"feat_tv_fact"`
	ModelChat      string `json:"model_chat"`
	ModelEmbedding string `json:"model_embedding"`
	// PlatformReady — настроен ли платформенный ключ: без него тумблеры
	// компании ничего не включат, и интерфейс это объясняет.
	PlatformReady bool `json:"platform_ready"`
	// OwnerTokensLeft — остаток токенов создателя компании (-1 — неизвестно).
	OwnerTokensLeft int64 `json:"owner_tokens_left"`
}

// AiSettingsUpdate — распарсенный PUT-боди (nil — поле не передано).
// api_key пустой/None — «не менять»; удаление ключа — флаг clear_key.
type AiSettingsUpdate struct {
	Enabled    *bool
	Shared     *bool
	FeatSearch *bool
	FeatTVFact *bool
}

// MyAiSettings — GET /api/ai/my-settings: личный ключ ИИ-ассистента. Сырого
// ключа, как и у компанийных настроек, наружу нет — только маска key_hint.
type MyAiSettings struct {
	Enabled   bool    `json:"enabled"`
	KeyHint   *string `json:"key_hint"`
	HasKey    bool    `json:"has_key"`
	ModelChat string  `json:"model_chat"`
	// APIBaseURL — свой сервер модели (значим вместе со своим ключом).
	APIBaseURL string `json:"api_base_url"`
	// Тумблеры возможностей: чат Hola и ИИ в заметках.
	FeatAssistant bool `json:"feat_assistant"`
	FeatNotes     bool `json:"feat_notes"`

	// Состояние тарифа и платформы — их показывает карточка настроек.
	PlatformReady bool       `json:"platform_ready"`
	TokensLimit   int64      `json:"tokens_limit"`
	TokensLeft    int64      `json:"tokens_left"`
	Models        []*AIModel `json:"models"`
	// ModelLocked — выбор модели закрыт, все работают на модели платформы.
	// Каталог при этом всё равно приходит: он нужен, чтобы показать, на чём
	// сейчас отвечает ИИ.
	ModelLocked bool `json:"model_locked"`
}

// AIModel — модель из каталога платформы глазами пользователя.
type AIModel struct {
	Code string `json:"code"`
	// Title — как модель называется в интерфейсе (GPT, GEMINI).
	Title string `json:"title"`
	// Rate — во сколько раз обращение дороже базовой модели каталога.
	Rate float64 `json:"rate"`
}

// MyAiSettingsUpdate — распарсенный PATCH-боди (nil — поле не передано).
// api_key пустой — «не менять»; удаление ключа — флаг clear_key.
type MyAiSettingsUpdate struct {
	Enabled       *bool
	APIKey        *string
	ClearKey      bool
	ModelChat     *string
	APIBaseURL    *string
	FeatAssistant *bool
	FeatNotes     *bool
}

// PlatformAiSettings — GET /api/ai/platform (супер-админ): глобальный ключ
// proxy-api и каталог моделей. Сырого ключа наружу нет — только маска.
type PlatformAiSettings struct {
	Enabled        bool               `json:"enabled"`
	HasKey         bool               `json:"has_key"`
	KeyHint        string             `json:"key_hint"`
	BaseURL        string             `json:"base_url"`
	ModelChat      string             `json:"model_chat"`
	ModelEmbedding string             `json:"model_embedding"`
	ModelSupport   string             `json:"model_support"`
	Models         []*PlatformAIModel `json:"models"`
}

// PlatformAIModel — строка каталога с ценой (её правит супер-админ).
type PlatformAIModel struct {
	Code         string  `json:"code"`
	Title        string  `json:"title"`
	Kind         string  `json:"kind"`
	PricePerMTok int64   `json:"price_per_mtok"`
	Selectable   bool    `json:"selectable"`
	IsActive     bool    `json:"is_active"`
	Sort         int     `json:"sort"`
	Rate         float64 `json:"rate"`
}

// PlatformAiUpdate — PATCH /api/ai/platform.
type PlatformAiUpdate struct {
	Enabled        *bool              `json:"enabled"`
	APIKey         *string            `json:"api_key"`
	ClearKey       bool               `json:"clear_key"`
	BaseURL        *string            `json:"base_url"`
	ModelChat      *string            `json:"model_chat"`
	ModelEmbedding *string            `json:"model_embedding"`
	ModelSupport   *string            `json:"model_support"`
	Models         []*PlatformAIModel `json:"models"`
}

// AiTestResult — POST .../ai-settings/test: реальная проверка связи с моделью.
type AiTestResult struct {
	Chat      bool    `json:"chat"`
	Embedding bool    `json:"embedding"`
	Error     *string `json:"error"`
	LatencyMS int64   `json:"latency_ms"`
}

// IndexingStatus — GET .../ai-settings/indexing.
type IndexingStatus struct {
	TotalTasks int    `json:"total_tasks"`
	Indexed    int    `json:"indexed"`
	Pending    int    `json:"pending"`
	Model      string `json:"model"`
	AiEnabled  bool   `json:"ai_enabled"`
}

// ReindexQueued — POST .../ai-settings/reindex-tasks (202 Accepted).
type ReindexQueued struct {
	Queued  bool `json:"queued"`
	Pending int  `json:"pending"`
}

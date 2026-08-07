// Package domain — модели и порты микросервиса работы с ИИ.
//
// aisvc владеет платформенными настройками ИИ (ai_platform_settings,
// ai_models), ИИ-полями companies (ai_enabled, ai_shared, ai_feat_*), личными
// настройками пользователей (user_ai_settings) и таблицей task_embeddings.
// Остальные таблицы (users, roles, tasks, departments) читаются read-only.
package domain

// Дефолтные модели — на случай, если в БД пусто (миграция этого не
// допускает, но всё же). Совпадают с back/app/services/ai_client.py.
const (
	DefaultModelChat      = "gpt-4o-mini"
	DefaultModelEmbedding = "text-embedding-3-small"
)

// Уровни ролей в компании (user_companies.role): 1/2/3. Системной роли 4 нет —
// платформенный супер-админ это отдельный класс (users.is_super_admin).
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelAdmin    = 3
)

// CompanyAI — AI-срез строки companies. Своего ключа у компании больше НЕТ:
// её ИИ-возможности работают на платформенном ключе и тратят токены
// СОЗДАТЕЛЯ компании — если он это разрешил (Shared).
type CompanyAI struct {
	ID         int64
	Enabled    bool
	Shared     bool // создатель разрешил тратить свои токены на компанию
	FeatSearch bool // умный поиск по задачам
	FeatTVFact bool // интересные факты в ТВ-режиме
	OwnerID    int64
}

// AllowsFeature — включена ли конкретная ИИ-возможность компании.
func (c *CompanyAI) AllowsFeature(feature string) bool {
	if c == nil || !c.Enabled || !c.Shared {
		return false
	}
	switch feature {
	case FeatureSearch:
		return c.FeatSearch
	case FeatureTVFact:
		return c.FeatTVFact
	}
	return true
}

// UserAI — личные ИИ-настройки пользователя (user_ai_settings, миграция
// 00066). На них работает ИИ-ассистент: он подключён к ЧЕЛОВЕКУ, а не к
// компании, поэтому ключ переезжает за владельцем между компаниями и живёт,
// когда активной компании нет вовсе. Компанийный CompanyAI при этом остаётся
// на месте — на нём эмбеддинги задач/заметок и ТВ-факт дня.
type UserAI struct {
	UserID    int64
	Enabled   bool
	APIKeyEnc []byte // nil — свой ключ не задан (работаем на платформенном)
	KeyHint   *string
	ModelChat string
	// APIBaseURL — свой сервер модели; значим только вместе со своим ключом.
	APIBaseURL string
	// Тумблеры возможностей: каждую функцию человек включает отдельно.
	FeatAssistant bool
	FeatNotes     bool
}

// ChatModel — выбранная модель с дефолтом платформы.
func (u *UserAI) ChatModel() string {
	if u == nil || u.ModelChat == "" {
		return PlatformModelChat
	}
	return u.ModelChat
}

// OwnKey — человек подключил СВОЙ ключ: запросы уходят на него и токены
// тарифа не тратятся.
func (u *UserAI) OwnKey() bool {
	return u != nil && len(u.APIKeyEnc) > 0
}

// AllowsFeature — включена ли личная ИИ-возможность.
func (u *UserAI) AllowsFeature(feature string) bool {
	if u == nil || !u.Enabled {
		return false
	}
	switch feature {
	case FeatureAssistant:
		return u.FeatAssistant
	case FeatureNotes:
		return u.FeatNotes
	}
	return true
}

// TaskText — задача в объёме, нужном для текста эмбеддинга
// (_build_text_for_task: название + отдел + ответственный).
type TaskText struct {
	ID             int64
	CompanyID      *int64
	Name           string
	DepartmentName *string
	ResponsibleFIO *string
}

// SearchHit — результат семантического поиска: score = 1 - cosine_distance.
type SearchHit struct {
	TaskID int64
	Score  float64
}

// TVFact — факт дня для ТВ-табло (хранится в Redis gw2:ai:tv_fact:{cid}).
// Порядок полей — алфавитный, как сортировка ключей jsonify во Flask.
type TVFact struct {
	GeneratedAt string `json:"generated_at"`
	Kind        string `json:"kind"` // "general" | "context"
	Text        string `json:"text"`
}

// TVWeekContext — метрики компании за последние 7 дней для контекстного
// ТВ-факта (срез _context_for_company во Flask).
type TVWeekContext struct {
	ClosedWeek    int
	ReceivedWeek  int
	TeamHoursWeek float64
	LeaderFIO     *string
	LeaderHours   *float64
	TopDept       *string
}

// Meaningful — есть ли в контексте что-то, кроме нулей (иначе фолбэк на
// general: стыдно показывать «закрыто 0 задач»).
func (c *TVWeekContext) Meaningful() bool {
	if c == nil {
		return false
	}
	return c.ClosedWeek > 0 || c.ReceivedWeek > 0 || c.TeamHoursWeek > 0
}

// User — пользователь в объёме auth-мидлвари и проверки доступа к настройкам.
// Идентичность развязана с компаниями: RoleLevel/CompanyID берутся из токена
// (активная компания сессии), а не из таблицы users.
type User struct {
	ID            int64
	RoleLevel     int
	CompanyID     *int64
	IsActive      bool
	IsSuperAdmin  bool
	CompanyActive bool
}

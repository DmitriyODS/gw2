// Package domain — модели и порты микросервиса работы с ИИ.
//
// aisvc владеет AI-полями companies (ai_enabled, ai_api_key_enc, ai_key_hint,
// ai_model_chat, ai_model_embedding) и таблицей task_embeddings; схему
// по-прежнему ведёт Alembic на стороне Flask. Остальные таблицы (users,
// roles, tasks, departments) читаются read-only.
package domain

// Дефолтные модели — на случай, если в БД пусто (миграция этого не
// допускает, но всё же). Совпадают с back/app/services/ai_client.py.
const (
	DefaultModelChat      = "gpt-4o-mini"
	DefaultModelEmbedding = "text-embedding-3-small"
)

// Уровни ролей — общие с back/app/utils/permissions.py.
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelDirector = 3
	LevelAdmin    = 4
)

// CompanyAI — AI-срез строки companies.
type CompanyAI struct {
	ID             int64
	Enabled        bool
	APIKeyEnc      []byte // nil — ключ не задан
	KeyHint        *string
	ModelChat      string
	ModelEmbedding string
}

// ChatModel / EmbeddingModel — модель с дефолтом (как `or DEFAULT_*` во Flask).
func (c *CompanyAI) ChatModel() string {
	if c.ModelChat == "" {
		return DefaultModelChat
	}
	return c.ModelChat
}

func (c *CompanyAI) EmbeddingModel() string {
	if c.ModelEmbedding == "" {
		return DefaultModelEmbedding
	}
	return c.ModelEmbedding
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
type User struct {
	ID            int64
	RoleLevel     int
	CompanyID     *int64
	IsHidden      bool
	IsRootAdmin   bool
	CompanyActive bool
}

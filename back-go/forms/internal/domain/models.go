package domain

import "time"

// Уровни ролей в компании (общие с authsvc/tasksvc domain.Level*).
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelAdmin    = 3
)

// Состояние приёма ответов.
const (
	StatusDraft  = "draft"  // черновик: ссылка не работает, ответы не принимаются
	StatusOpen   = "open"   // приём открыт
	StatusClosed = "closed" // приём закрыт вручную
)

// Когда отвечающий видит оценку теста.
const (
	QuizImmediately = "immediately"
	QuizManual      = "manual"
)

// Form — форма (опрос или тест). Принадлежит ЧЕЛОВЕКУ; коллеги и компании
// получают её адресно (см. access.go).
type Form struct {
	ID      int64 `json:"id"`
	OwnerID int64 `json:"owner_id"`
	// CompanyID — компания, в которой форма заведена (nil — личная). Решает,
	// чья квота платит за файлы ответов.
	CompanyID   *int64 `json:"company_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`

	AllowAnonymous   bool   `json:"allow_anonymous"`
	OneResponse      bool   `json:"one_response"`
	AllowEdit        bool   `json:"allow_edit"`
	CollectEmail     bool   `json:"collect_email"`
	// CollectName — спрашивать имя у гостя по ссылке (вошедший подписан
	// аккаунтом, и его не спрашивают никогда).
	CollectName      bool   `json:"collect_name"`
	ShowProgress     bool   `json:"show_progress"`
	ShuffleQuestions bool   `json:"shuffle_questions"`
	Confirmation     string `json:"confirmation"`
	ShowSummary      bool   `json:"show_summary"`

	Quiz            bool   `json:"quiz"`
	QuizRelease     string `json:"quiz_release"`
	QuizShowAnswers bool   `json:"quiz_show_answers"`

	OpensAt      *time.Time `json:"opens_at,omitempty"`
	ClosesAt     *time.Time `json:"closes_at,omitempty"`
	MaxResponses int        `json:"max_responses"`

	Position  int       `json:"position"`
	CreatedBy *int64    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Sections — структура формы; без omitempty: форма без разделов должна
	// отдавать [], иначе на клиенте form.sections === undefined.
	Sections []Section `json:"sections"`
	// MyAccess — эффективный уровень спрашивающего (считает сервер).
	MyAccess string `json:"my_access"`
	// OwnerName — чья это форма (вкладки «Поделились» и «Мне назначены»
	// обязаны называть хозяина).
	OwnerName string `json:"owner_name,omitempty"`
	// Responses — сколько ответов собрано (для карточки в списке).
	Responses int `json:"responses"`
	// MyDueAt / MyResponded — обязанность спрашивающего: срок ответа и
	// ответил ли он уже. Заполняются на списке и при чтении формы.
	MyDueAt     *time.Time `json:"my_due_at,omitempty"`
	MyResponded bool        `json:"my_responded"`
}

// Переход после раздела.
const (
	NextNext    = "next"    // следующий по порядку
	NextSection = "section" // к назначенному разделу
	NextSubmit  = "submit"  // отправить форму
)

// Section — раздел-страница формы.
type Section struct {
	ID          int64  `json:"id"`
	FormID      int64  `json:"form_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	NextAction  string `json:"next_action"`
	// NextSectionID — куда идти при NextSection (nil — как «следующий»).
	NextSectionID *int64 `json:"next_section_id"`
	/* NextIndex — то же самое ПОЗИЦИЕЙ раздела в сохраняемом наборе: у только
	   что добавленного раздела id ещё нет, и ссылаться на него неоткуда.
	   Приходит с клиента, репозиторий переводит его в id той же транзакцией
	   (-1 — переход не задан). Наружу не отдаётся. */
	NextIndex int `json:"-"`
	/* Условие показа раздела: он выводится, только если на вопрос-источник дан
	   один из ожидаемых ответов (пустой список — «любой непустой ответ»). Это
	   не ветвление: ветвление уводит на другую страницу, а условие прячет
	   раздел внутри того же маршрута. nil — раздел виден всегда. */
	VisibleQuestionID *int64     `json:"visible_question_id"`
	VisibleValues     []string   `json:"visible_values"`
	CreatedAt         time.Time  `json:"created_at"`
	Questions         []Question `json:"questions"`
}

// Question — вопрос раздела. Config хранит настройки типа (варианты, границы
// шкалы, строки/столбцы сетки, проверку текста), AnswerKey — правильный ответ
// режима теста (см. questions.go).
type Question struct {
	ID          int64          `json:"id"`
	FormID      int64          `json:"form_id"`
	SectionID   int64          `json:"section_id"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Config      map[string]any `json:"config"`
	Position    int            `json:"position"`
	Points      int            `json:"points"`
	AnswerKey   map[string]any `json:"answer_key,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Booking — занимаемое место вопроса «Запись»: какой вариант какого вопроса и
// сколько мест у него всего. Проверку остатка ведёт репозиторий под локом формы.
type Booking struct {
	QuestionKey string
	Option      string
	Capacity    int
}

// Response — одна отправка формы.
type Response struct {
	ID     int64  `json:"id"`
	FormID int64  `json:"form_id"`
	UserID *int64 `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	// Answers — карта строкового id вопроса в значение (форма зависит от типа).
	Answers   map[string]any `json:"answers"`
	Score     int            `json:"score"`
	MaxScore  int            `json:"max_score"`
	Graded    bool           `json:"graded"`
	ShareID   *int64         `json:"share_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	// UserName/UserAvatar — кто ответил (у вошедших); подмешивается JOIN'ом.
	UserName   string  `json:"user_name,omitempty"`
	UserAvatar *string `json:"user_avatar,omitempty"`
	// IP/UserAgent — откуда пришёл ответ. Наружу не отдаются: автору формы это
	// ничего не объясняет, а сведения личные.
	IP        string `json:"-"`
	UserAgent string `json:"-"`
}

// ResponseListFilter — страница ответов: поиск по значениям, сортировка и
// пагинация.
type ResponseListFilter struct {
	FormID int64
	Search string
	// Sort — "created_at" | "score".
	Sort    string
	Desc    bool
	Page    int
	PerPage int
}

// ResponseScope — ответ вместе с формой, которой он принадлежит: разделу
// «Настройки → Хранилище» нужно показать, где лежит файл.
type ResponseScope struct {
	Response  *Response
	FormID    int64
	FormTitle string
	CompanyID int64
	OwnerID   int64
}

// Share — внешняя ссылка на заполнение. Code в адресе — capability.
type Share struct {
	ID     int64  `json:"id"`
	FormID int64  `json:"form_id"`
	Code   string `json:"code"`
	// Name — своё название ссылки: их у формы бывает много, и отзывать нужную
	// удобнее по слову, а не по коду.
	Name string `json:"name"`
	// RequireAuth — заполнить можно только войдя в аккаунт.
	RequireAuth bool      `json:"require_auth"`
	CreatedBy   *int64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	// Visits/LastVisitAt — сводка журнала переходов (для карточки ссылки).
	Visits      int        `json:"visits"`
	LastVisitAt *time.Time `json:"last_visit_at,omitempty"`
	// Responses — сколько ответов пришло именно через эту ссылку.
	Responses int `json:"responses"`
}

// UserShare — адресный доступ: конкретному человеку либо всей компании.
// Уровень respond и есть «назначение».
type UserShare struct {
	ID        int64      `json:"id"`
	FormID    int64      `json:"form_id"`
	UserID    *int64     `json:"user_id"`
	CompanyID *int64     `json:"company_id"`
	Access    string     `json:"access"`
	DueAt     *time.Time `json:"due_at,omitempty"`
	// CreatedBy — КТО выдал доступ (не тот, кому выдали).
	CreatedBy *int64 `json:"created_by,omitempty"`
	// Name/AvatarPath — кому выдано (имя человека или название компании).
	Name       string    `json:"name"`
	AvatarPath *string   `json:"avatar_path,omitempty"`
	Login      string    `json:"login,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ShareVisit — переход по внешней ссылке. У гостя аккаунта нет — остаётся
// только адрес и время.
type ShareVisit struct {
	ID        int64     `json:"id"`
	ShareID   int64     `json:"share_id"`
	UserID    *int64    `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"`
	UserLogin string    `json:"user_login,omitempty"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	VisitedAt time.Time `json:"visited_at"`
}

// Assignee — назначенный человек в контроле исполнения: ответил или нет.
type Assignee struct {
	UserID     int64      `json:"user_id"`
	Name       string     `json:"name"`
	AvatarPath *string    `json:"avatar_path,omitempty"`
	// Via — откуда обязанность: "user" (лично) или название компании.
	Via        string     `json:"via"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	AnsweredAt *time.Time `json:"answered_at,omitempty"`
}

// DueReminder — наступивший срок ответа: кому напомнить и по какой форме.
type DueReminder struct {
	ShareID   int64
	FormID    int64
	FormTitle string
	OwnerID   int64
	DueAt     time.Time
	// UserIDs — назначенные, которые ещё не ответили (по компанийной шаре их
	// много, по личной — один).
	UserIDs []int64
}

// User — идентичность пользователя для авторизации (компания/роль из токена).
type User struct {
	ID            int64
	FIO           string
	AvatarPath    *string
	IsActive      bool
	IsSuperAdmin  bool
	RoleLevel     int
	CompanyID     *int64
	CompanyActive bool
}

// SearchHit — строка глобального поиска (Hola): форма и то, чем она совпала.
type SearchHit struct {
	FormID  int64  `json:"form_id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Status  string `json:"status"`
}

// UploadedFile — метаданные загруженного файла ответа (хранится в Answers).
type UploadedFile struct {
	Path string `json:"path"` // относительный путь в uploads (раздаёт nginx /uploads/)
	Name string `json:"name"`
	Mime string `json:"mime"`
	Size int64  `json:"size"`
	// Thumb — уменьшенная копия картинки для таблицы ответов; пусто, если
	// оригинал и так мал либо формат не декодируется.
	Thumb string `json:"thumb,omitempty"`
}

package domain

import "time"

// Уровни ролей в компании (общие с authsvc/tasksvc domain.Level*).
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelAdmin    = 3
)

// MaxPinnedPosts — лимит одновременно закреплённых постов на компанию (см.
// ErrTooManyPinned): без потолка закреплённая секция разрастается и теряет
// смысл «важного сверху» (аналог SharePoint news boost).
const MaxPinnedPosts = 10

// Topic — тематический раздел портала компании. Создаёт/правит администратор.
type Topic struct {
	ID        int64     `json:"id"`
	CompanyID int64     `json:"company_id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color"`
	Icon      *string   `json:"icon"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Post — пост портала. TopicID — опциональная привязка к разделу.
// PinnedAt/PinnedBy — закрепление (автор поста или администратор компании);
// PinnedUntil — автоистечение пина (NULL = бессрочно), истёкший пин везде
// трактуется как незакреплённый.
type Post struct {
	ID          int64      `json:"id"`
	CompanyID   int64      `json:"company_id"`
	TopicID     *int64     `json:"topic_id"`
	AuthorID    int64      `json:"author_id"`
	Title       *string    `json:"title"`
	Body        string     `json:"body"`
	PinnedAt    *time.Time `json:"pinned_at"`
	PinnedBy    *int64     `json:"pinned_by"`
	PinnedUntil *time.Time `json:"pinned_until"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Заполняются при чтении списка/карточки (без N+1 — батч-загрузка).
	Attachments   []Attachment   `json:"attachments"`
	CommentCount  int            `json:"comment_count"`
	ReactionCount map[string]int `json:"reaction_counts"`
	MyReactions   []string       `json:"my_reactions"`
	// ViewCount — число уникальных зрителей поста; Viewed — просматривал ли его
	// сам зритель (viewer): фронт по нему знает, засчитывать ли свой просмотр.
	ViewCount int  `json:"view_count"`
	Viewed    bool `json:"viewed"`
}

// Attachment — файл-вложение поста (общий uploads-том/S3, префикс "portal").
type Attachment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	FilePath  string    `json:"file_path"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mime      *string   `json:"mime"`
	CreatedAt time.Time `json:"created_at"`
	// URL — вычисляется при сериализации (не хранится).
	URL string `json:"url"`
}

// Comment — плоский комментарий поста (без reply_to — по итогам исследования
// стандарт для новостной ленты, не вложенное дерево).
type Comment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	AuthorID  int64     `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Reaction — реакция пользователя на пост (уникальна по post_id+user_id+emoji).
type Reaction struct {
	PostID    int64     `json:"post_id"`
	UserID    int64     `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

// PostListFilter — выборка постов компании. Pinned трактует пин с истёкшим
// pinned_until как незакреплённый (актуальность пина, не сырой pinned_at).
// BeforeCreatedAt/BeforeID — keyset-курсор хронологии: строго «старше» пары
// (created_at, id) при ORDER BY created_at DESC, id DESC.
type PostListFilter struct {
	CompanyID       int64
	TopicID         *int64
	Pinned          *bool
	Search          string
	Limit           int
	BeforeCreatedAt *time.Time
	BeforeID        int64
}

// User — идентичность пользователя для авторизации (read-only, владелец —
// authsvc).
type User struct {
	ID            int64
	FIO           string
	AvatarPath    *string
	IsActive      bool
	IsSuperAdmin  bool
	CompanyID     *int64 // из токена (активная компания)
	RoleLevel     int    // из токена
	CompanyActive bool
}

// UploadedFile — метаданные сохранённого вложения (см. FileStore.Save).
type UploadedFile struct {
	Path string `json:"file_path"`
	Name string `json:"name"`
	Mime string `json:"mime"`
	Size int64  `json:"size"`
}

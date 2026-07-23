package domain

import "context"

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// RecipientScope — область выборки размещённых мной чужих заметок.
type RecipientScope string

const (
	RecipientFolder  RecipientScope = "folder"  // в моей папке folderID
	RecipientRoot    RecipientScope = "root"     // в моём корне
	RecipientArchive RecipientScope = "archive"  // в моём личном архиве
)

// NoteRepository — персистентность заметок, папок, тегов и всех видов шаринга.
type NoteRepository interface {
	// ── Заметки ──
	// ListNotes — плитки по фильтру (без doc, с excerpt/folder_id/tag_ids).
	ListNotes(ctx Ctx, f NoteListFilter) ([]*Note, error)
	// GetNote — полная заметка (с doc, folder_id и tag_ids); nil — нет такой.
	GetNote(ctx Ctx, id int64) (*Note, error)
	CreateNote(ctx Ctx, n *Note) error
	UpdateNote(ctx Ctx, n *Note) error
	DeleteNote(ctx Ctx, id int64) error
	// MoveNote — сменить папку заметки (folderID nil — в корень).
	MoveNote(ctx Ctx, id int64, folderID *int64) error
	// SetNoteTags — полная замена связей заметки с тегами.
	SetNoteTags(ctx Ctx, noteID int64, tagIDs []int64) error
	// SharedByMeNoteIDs — из ids оставить те, что расшарены владельцем
	// (значок на плитке): есть публичная ссылка / адресат / компания.
	SharedByMeNoteIDs(ctx Ctx, ids []int64) (map[int64]bool, error)
	// ListSharedWithMe — чужие заметки, доступные мне адресно или через
	// расшаренную папку (плитки с owner и my_access). Исключает те, что я уже
	// разместил у себя/отправил в личный архив (есть строка в note_recipient_state).
	ListSharedWithMe(ctx Ctx, userID int64, companyIDs []int64, search string) ([]*Note, error)

	// ── Личный оверлей адресата шаринга (размещение чужих заметок/папок) ──
	// SetNoteRecipientPlacement — разместить расшаренную мне заметку в моей папке
	// (folderID nil — мой корень); снимает личный архив.
	SetNoteRecipientPlacement(ctx Ctx, userID, noteID int64, folderID *int64) error
	// SetNoteRecipientArchived — личный архив расшаренной мне заметки.
	SetNoteRecipientArchived(ctx Ctx, userID, noteID int64, archived bool) error
	// ListRecipientNotes — расшаренные мне заметки, размещённые в моём scope
	// (folder — в папке folderID / root — в моём корне / archive — в личном
	// архиве); folder_id/archived плиток — из оверлея, с owner и my_access.
	ListRecipientNotes(ctx Ctx, userID int64, companyIDs []int64, scope RecipientScope, folderID *int64) ([]*Note, error)
	// SetFolderRecipientPlacement — подшить расшаренную мне папку под мою
	// (parentID nil — мой корень).
	SetFolderRecipientPlacement(ctx Ctx, userID, folderID int64, parentID *int64) error
	// ListRecipientFolders — все расшаренные мне папки, размещённые в моём дереве
	// (parent_id — из оверлея), с owner и my_access; для инъекции в клиентское дерево.
	ListRecipientFolders(ctx Ctx, userID int64, companyIDs []int64) ([]*Folder, error)

	// ── Папки ──
	ListFolders(ctx Ctx, ownerID int64) ([]*Folder, error)
	ListChildFolders(ctx Ctx, parentID int64) ([]*Folder, error)
	// ListSharedRootFolders — папки, расшаренные мне напрямую (роль «корней»
	// раздела «Поделились со мной»), с owner и my_access.
	ListSharedRootFolders(ctx Ctx, userID int64, companyIDs []int64) ([]*Folder, error)
	GetFolder(ctx Ctx, id int64) (*Folder, error)
	CreateFolder(ctx Ctx, f *Folder) error
	UpdateFolder(ctx Ctx, id int64, name, color string) error
	MoveFolder(ctx Ctx, id int64, parentID *int64) error
	DeleteFolder(ctx Ctx, id int64) error
	NextFolderPosition(ctx Ctx, ownerID int64, parentID *int64) (int, error)
	// IsDescendant — folderID является потомком maybeAncestor (защита от циклов
	// при переносе; равенство считается true).
	IsDescendant(ctx Ctx, folderID, maybeAncestor int64) (bool, error)
	// ReparentChildren — перевесить дочерние папки и заметки folderID на newParent
	// (nil — в корень); используется при удалении папки.
	ReparentChildren(ctx Ctx, folderID int64, newParent *int64) error
	// CopyFolderTree — глубокая копия поддерева папки со всеми заметками владельца;
	// возвращает id корневой копии.
	CopyFolderTree(ctx Ctx, ownerID, folderID int64, newParent *int64) (int64, error)

	// ── Теги ──
	ListTags(ctx Ctx, ownerID int64) ([]*Tag, error)
	GetTag(ctx Ctx, id int64) (*Tag, error)
	CreateTag(ctx Ctx, t *Tag) error
	UpdateTag(ctx Ctx, id int64, name, color string) error
	DeleteTag(ctx Ctx, id int64) error
	NextTagPosition(ctx Ctx, ownerID int64) (int, error)
	// OwnedTagIDs — из ids оставить только теги владельца.
	OwnedTagIDs(ctx Ctx, ownerID int64, ids []int64) ([]int64, error)

	// ── Публичные ссылки ──
	ListShares(ctx Ctx, noteID int64) ([]*Share, error)
	CreateShare(ctx Ctx, s *Share) error
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	DeleteShare(ctx Ctx, id, noteID int64) error

	// ── Адресный шаринг заметок (пользователь и компания) ──
	ListNoteMembers(ctx Ctx, noteID int64) ([]*Member, error)
	UpsertNoteUserShare(ctx Ctx, noteID, userID int64, canEdit bool) error
	DeleteNoteUserShare(ctx Ctx, noteID, userID int64) error
	UpsertNoteCompanyShare(ctx Ctx, noteID, companyID int64, name string, canEdit bool, by int64) error
	DeleteNoteCompanyShare(ctx Ctx, noteID, companyID int64) error

	// ── Адресный шаринг папок (пользователь и компания) ──
	ListFolderMembers(ctx Ctx, folderID int64) ([]*Member, error)
	UpsertFolderUserShare(ctx Ctx, folderID, userID int64, canEdit bool) error
	DeleteFolderUserShare(ctx Ctx, folderID, userID int64) error
	UpsertFolderCompanyShare(ctx Ctx, folderID, companyID int64, name string, canEdit bool, by int64) error
	DeleteFolderCompanyShare(ctx Ctx, folderID, companyID int64) error

	// ── Адресация сокет-событий (все, кто видит объект) ──
	// NoteAudienceUserIDs — user_id всех, кто имеет доступ к заметке: адресаты
	// (пользователь/компания→участники) + аудитория расшаренных папок-предков.
	NoteAudienceUserIDs(ctx Ctx, noteID int64) ([]int64, error)
	// FolderAudienceUserIDs — то же для папки (её шары + шары предков).
	FolderAudienceUserIDs(ctx Ctx, folderID int64) ([]int64, error)

	// ── Эмбеддинги (ИИ-поиск, pgvector) ──
	// UpsertNoteEmbedding — сохранить/обновить вектор заметки.
	UpsertNoteEmbedding(ctx Ctx, noteID, ownerID int64, vector []float32, model string) error
	// SearchNoteEmbeddings — id заметок владельца по близости к вектору запроса
	// (той же модели), по убыванию релевантности.
	SearchNoteEmbeddings(ctx Ctx, ownerID int64, vector []float32, model string, archived bool, limit int) ([]int64, error)
	// ListNotesByIDs — плитки заметок владельца по списку id, в порядке ids.
	ListNotesByIDs(ctx Ctx, ownerID int64, ids []int64, archived bool) ([]*Note, error)

	// ── Разрешение эффективного доступа ──
	// NoteAccess — доступ пользователя к заметке с учётом прямых шар и
	// расшаренных папок-предков: (найден, можно ли править).
	NoteAccess(ctx Ctx, userID int64, companyIDs []int64, noteID int64, folderID *int64) (found, canEdit bool, err error)
	// FolderAccess — то же для папки (доступ к папке или любому её предку).
	FolderAccess(ctx Ctx, userID int64, companyIDs []int64, folderID int64) (found, canEdit bool, err error)
}

// UserReader — read-only идентичность и членство пользователей (владелец таблиц
// в рантайме — authsvc; читаем напрямую из общей БД, как и users).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	// UserCompanies — компании, в которых состоит пользователь (id+имя).
	UserCompanies(ctx Ctx, userID int64) ([]*Company, error)
	// CompanyIDs — только id компаний пользователя (скоуп «расшарено компании»).
	CompanyIDs(ctx Ctx, userID int64) ([]int64, error)
	// IsCompanyMember — состоит ли пользователь в компании (авторизация шаринга).
	IsCompanyMember(ctx Ctx, userID, companyID int64) (bool, string, error)
}

// Embedder — векторизация текста для ИИ-поиска (gRPC-клиент aisvc.Embed).
// Enabled=false — AI_GRPC_ADDR не задан, поиск откатывается на текстовый.
type Embedder interface {
	Enabled() bool
	Embed(ctx Ctx, companyID int64, text string) (vector []float32, model string, err error)
}

// EventBus — сокет-события клиентам через Redis gw2:notes:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}

// FileStore — хранилище картинок редактора (pkg/records.FileStore поверх
// pkg/storage: local-том в dev, S3 в prod).
type FileStore interface {
	Save(fileName string, data []byte) (string, error)
	Remove(paths []string)
	// Open — прочитать байты объекта по ключу (встраивание картинок в docx).
	Open(key string) ([]byte, error)
}

// WriteLimiter — троттлинг анонимных правок по коду публичной ссылки (защита
// от вандализма). Redis-реализация fail-open: при недоступности — разрешаем.
type WriteLimiter interface {
	Allow(ctx Ctx, code string) bool
}

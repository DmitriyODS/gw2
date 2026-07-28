package domain

import "context"

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// RecipientScope — область выборки размещённых мной чужих досок.
type RecipientScope string

const (
	RecipientFolder  RecipientScope = "folder"  // в моей папке folderID
	RecipientRoot    RecipientScope = "root"    // в моём корне
	RecipientArchive RecipientScope = "archive" // в моём личном архиве
)

// BoardRepository — персистентность досок, папок, тегов и всех видов шаринга.
type BoardRepository interface {
	// ── Доски ──
	// ListBoards — плитки по фильтру (без сцены, с excerpt и превью).
	ListBoards(ctx Ctx, f BoardListFilter) ([]*Board, error)
	// GetBoard — полная доска (со сценой и folder_id); nil — нет такой.
	GetBoard(ctx Ctx, id int64) (*Board, error)
	// CountBoards — сколько досок уже есть (лимит тарифа).
	CountBoards(ctx Ctx, owner_id int64) (int, error)
	CreateBoard(ctx Ctx, n *Board) error
	UpdateBoard(ctx Ctx, n *Board) error
	DeleteBoard(ctx Ctx, id int64) error
	// MoveBoard — сменить папку доски (folderID nil — в корень).
	MoveBoard(ctx Ctx, id int64, folderID *int64) error
	// SetBoardPreview — ключ миниатюры холста (плитка списка).
	SetBoardPreview(ctx Ctx, boardID int64, key string) error
	// SharedByMeBoardIDs — из ids оставить те, что расшарены владельцем
	// (значок на плитке): есть публичная ссылка / адресат / компания.
	SharedByMeBoardIDs(ctx Ctx, ids []int64) (map[int64]bool, error)
	// ListSharedWithMe — чужие доски, доступные мне адресно или через
	// расшаренную папку (плитки с owner и my_access). Исключает те, что я уже
	// разместил у себя/отправил в личный архив (есть строка в board_recipient_state).
	ListSharedWithMe(ctx Ctx, userID int64, companyIDs []int64, search string) ([]*Board, error)

	// ── Личный оверлей адресата шаринга (размещение чужих досок/папок) ──
	// SetBoardRecipientPlacement — разместить расшаренную мне доску в моей папке
	// (folderID nil — мой корень); снимает личный архив.
	SetBoardRecipientPlacement(ctx Ctx, userID, boardID int64, folderID *int64) error
	// SetBoardRecipientArchived — личный архив расшаренной мне доски.
	SetBoardRecipientArchived(ctx Ctx, userID, boardID int64, archived bool) error
	// ListRecipientBoards — расшаренные мне доски, размещённые в моём scope
	// (folder — в папке folderID / root — в моём корне / archive — в личном
	// архиве); folder_id/archived плиток — из оверлея, с owner и my_access.
	ListRecipientBoards(ctx Ctx, userID int64, companyIDs []int64, scope RecipientScope, folderID *int64) ([]*Board, error)
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
	// ReparentChildren — перевесить дочерние папки и доски folderID на newParent
	// (nil — в корень); используется при удалении папки.
	ReparentChildren(ctx Ctx, folderID int64, newParent *int64) error
	// CopyFolderTree — глубокая копия поддерева папки со всеми досками владельца;
	// возвращает id корневой копии.
	CopyFolderTree(ctx Ctx, ownerID, folderID int64, newParent *int64) (int64, error)

	// ── Публичные ссылки ──
	ListShares(ctx Ctx, boardID int64) ([]*Share, error)
	CreateShare(ctx Ctx, s *Share) error
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	DeleteShare(ctx Ctx, id, boardID int64) error

	// ── Адресный шаринг досок (пользователь и компания) ──
	ListBoardMembers(ctx Ctx, boardID int64) ([]*Member, error)
	UpsertBoardUserShare(ctx Ctx, boardID, userID int64, canEdit bool) error
	DeleteBoardUserShare(ctx Ctx, boardID, userID int64) error
	UpsertBoardCompanyShare(ctx Ctx, boardID, companyID int64, name string, canEdit bool, by int64) error
	DeleteBoardCompanyShare(ctx Ctx, boardID, companyID int64) error

	// ── Адресный шаринг папок (пользователь и компания) ──
	ListFolderMembers(ctx Ctx, folderID int64) ([]*Member, error)
	UpsertFolderUserShare(ctx Ctx, folderID, userID int64, canEdit bool) error
	DeleteFolderUserShare(ctx Ctx, folderID, userID int64) error
	UpsertFolderCompanyShare(ctx Ctx, folderID, companyID int64, name string, canEdit bool, by int64) error
	DeleteFolderCompanyShare(ctx Ctx, folderID, companyID int64) error

	// ── Адресация сокет-событий (все, кто видит объект) ──
	// BoardAudienceUserIDs — user_id всех, кто имеет доступ к доске: адресаты
	// (пользователь/компания→участники) + аудитория расшаренных папок-предков.
	BoardAudienceUserIDs(ctx Ctx, boardID int64) ([]int64, error)
	// FolderAudienceUserIDs — то же для папки (её шары + шары предков).
	FolderAudienceUserIDs(ctx Ctx, folderID int64) ([]int64, error)

	// ── Разрешение эффективного доступа ──
	// BoardAccess — доступ пользователя к доске с учётом прямых шар и
	// расшаренных папок-предков: (найден, можно ли править).
	BoardAccess(ctx Ctx, userID int64, companyIDs []int64, boardID int64, folderID *int64) (found, canEdit bool, err error)
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

// EventBus — сокет-события клиентам через Redis gw2:board:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}

// FileStore — хранилище картинок холста и превью досок (pkg/records.FileStore
// поверх pkg/storage: local-том в dev, S3 в prod).
type FileStore interface {
	// SaveFor — записать файл в квоту ВЛАДЕЛЬЦА (личный раздел): сверх лимита
	// тарифа файл не сохраняется.
	SaveFor(ctx context.Context, userID, companyID int64, fileName string, data []byte) (string, error)
	// RemoveFor — удаление с возвратом места в квоту владельца.
	RemoveFor(ctx context.Context, userID, companyID int64, paths []string)
	// Open — прочитать байты объекта по ключу (встраивание картинок в SVG).
	Open(key string) ([]byte, error)
}

// WriteLimiter — троттлинг анонимных правок по коду публичной ссылки (защита
// от вандализма). Redis-реализация fail-open: при недоступности — разрешаем.
type WriteLimiter interface {
	Allow(ctx Ctx, code string) bool
}

package domain

import (
	"context"
	"io"

	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// Repository — персистентность диска: папки, файлы и оба вида шаринга.
type Repository interface {
	// ── Папки ──
	ListFolders(ctx Ctx, ownerID int64, parentID *int64, trash bool) ([]*Folder, error)
	// SearchFolders — папки по имени ПО ВСЕМУ диску: поиск глобальный, как и
	// у файлов, — искомое чаще лежит не там, где сейчас открыто.
	SearchFolders(ctx Ctx, ownerID int64, query string) ([]*Folder, error)
	GetFolder(ctx Ctx, id int64) (*Folder, error)
	CreateFolder(ctx Ctx, f *Folder) error
	RenameFolder(ctx Ctx, id int64, name, color string) error
	MoveFolder(ctx Ctx, id int64, parentID *int64) error
	// FolderPath — путь от корня до папки (хлебные крошки) одним запросом.
	FolderPath(ctx Ctx, id int64) ([]*Folder, error)
	// FolderSubtree — id папки и всех её потомков: корзина и удаление работают
	// поддеревом целиком.
	FolderSubtree(ctx Ctx, id int64) ([]int64, error)
	// SetFoldersDeleted — отправить папки в корзину или вернуть (nil-время).
	SetFoldersDeleted(ctx Ctx, ids []int64, deleted bool) error
	DeleteFolders(ctx Ctx, ids []int64) error

	// ── Файлы ──
	ListFiles(ctx Ctx, f ListFilter) ([]*File, error)
	GetFile(ctx Ctx, id int64) (*File, error)
	CreateFile(ctx Ctx, f *File) error
	RenameFile(ctx Ctx, id int64, name string) error
	MoveFile(ctx Ctx, id int64, folderID *int64) error
	SetFileStarred(ctx Ctx, id int64, starred bool) error
	// SetFilesDeleted — в корзину или обратно; при удалении папки уезжает всё
	// её содержимое.
	SetFilesDeleted(ctx Ctx, ids []int64, deleted bool) error
	SetFilesDeletedByFolders(ctx Ctx, folderIDs []int64, deleted bool) error
	FilesOfFolders(ctx Ctx, folderIDs []int64) ([]*File, error)
	DeleteFiles(ctx Ctx, ids []int64) error
	// ExpiredTrash — что пора вычистить из корзины (старше TrashKeepDays).
	ExpiredTrash(ctx Ctx) ([]*File, []int64, error)

	// ── Шаринг ──
	CreateShare(ctx Ctx, s *Share) error
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	ListShares(ctx Ctx, fileID, folderID *int64) ([]*Share, error)
	// DeleteShare/DeleteUserShare — снять доступ. Владелец проверяется в SQL
	// (JOIN на цель): чужую выдачу не отозвать, зная её id.
	DeleteShare(ctx Ctx, id, ownerID int64) error

	UpsertUserShare(ctx Ctx, s *UserShare) error
	ListUserShares(ctx Ctx, fileID, folderID *int64) ([]*UserShare, error)
	DeleteUserShare(ctx Ctx, id, ownerID int64) error
	// SharedWithMe — чужие файлы и папки, доступные мне лично или моей компании.
	SharedWithMe(ctx Ctx, userID int64, companyIDs []int64) ([]*Folder, []*File, error)
	// FileAccess/FolderAccess — эффективный доступ с подъёмом по дереву папок:
	// доступ, выданный на папку, КАСКАДИТ на всё её содержимое.
	FileAccess(ctx Ctx, fileID, userID int64, companyIDs []int64) (string, error)
	FolderAccess(ctx Ctx, folderID, userID int64, companyIDs []int64) (string, error)
	// SharedByMe — из id оставить те, что владелец кому-то открыл (значок на
	// плитке): публичная ссылка или адресный доступ.
	SharedByMe(ctx Ctx, fileIDs, folderIDs []int64) (map[int64]bool, map[int64]bool, error)
}

// UserReader — read-only идентичность (таблицы ведёт authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	// UserCompanies — компании пользователя: по ним работает доступ,
	// выданный на компанию.
	UserCompanies(ctx Ctx, userID int64) ([]int64, error)
	SearchUsers(ctx Ctx, query string, limit int) ([]*User, error)
}

// User — минимальная идентичность для подписей и выбора адресата.
type User struct {
	ID         int64   `json:"id"`
	FIO        string  `json:"fio"`
	Login      string  `json:"login"`
	AvatarPath *string `json:"avatar_path"`
}

// FileStore — объекты пользовательского контента (pkg/storage через
// pkg/records.FileStore).
type FileStore interface {
	// SaveFor — записать файл в квоту владельца: сверх лимита файл не
	// сохраняется вовсе.
	SaveFor(ctx context.Context, userID, companyID int64, fileName string, data []byte) (string, error)
	// SaveStreamFor — то же потоком: файл в сотни мегабайт нельзя подержать в
	// памяти ради проверки квоты и записи.
	SaveStreamFor(ctx context.Context, userID, companyID int64, fileName string, r io.Reader, size int64) (string, error)
	// RemoveFor — удаление с возвратом места в квоту.
	RemoveFor(ctx context.Context, userID, companyID int64, paths []string)
	// Remove — удаление БЕЗ учёта: так чистит раздел «Настройки → Хранилище»,
	// где место пересчитывает сам биллинг.
	Remove(paths []string)
	// Open — байты объекта (скачивание и просмотр).
	Open(key string) ([]byte, error)
}

// EventBus — сокет-события клиентам через Redis gw2:drive:events.
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}

// Owner — контракт владельца файлов для раздела «Настройки → Хранилище».
var _ = storagefiles.Owner(nil)

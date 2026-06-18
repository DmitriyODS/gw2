package domain

import "context"

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// RegistryRepository — персистентность реестров, их полей и записей.
type RegistryRepository interface {
	// ── Реестры ──
	ListRegistries(ctx Ctx, companyID int64) ([]*Registry, error)
	// GetRegistry — реестр без полей (для проверок принадлежности).
	GetRegistry(ctx Ctx, id int64) (*Registry, error)
	CreateRegistry(ctx Ctx, r *Registry) error
	UpdateRegistry(ctx Ctx, id int64, name string, position int) error
	DeleteRegistry(ctx Ctx, id int64) error
	NextRegistryPosition(ctx Ctx, companyID int64) (int, error)

	// ── Поля ──
	ListFields(ctx Ctx, registryID int64) ([]Field, error)
	// FieldsByRegistries — батч-загрузка полей для списка реестров (без N+1).
	FieldsByRegistries(ctx Ctx, registryIDs []int64) (map[int64][]Field, error)
	// ReplaceFields — полная замена набора полей реестра в одной транзакции:
	// удаляет отсутствующие, обновляет существующие, вставляет новые. Возвращает
	// id удалённых полей (их данные нужно вычистить из записей).
	ReplaceFields(ctx Ctx, registryID int64, fields []Field) (removed []int64, err error)

	// ── Записи ──
	ListRecords(ctx Ctx, f RecordListFilter) (items []*Record, total int, err error)
	GetRecord(ctx Ctx, id int64) (*Record, error)
	CreateRecord(ctx Ctx, r *Record, searchText string) error
	UpdateRecord(ctx Ctx, id int64, data map[string]any, searchText string) error
	DeleteRecord(ctx Ctx, id int64) error
	// DeleteRecords — массовое удаление; возвращает число удалённых.
	DeleteRecords(ctx Ctx, registryID int64, ids []int64) (int64, error)
	// AllRecords — все записи реестра (для пересчёта search_text после удаления поля).
	AllRecords(ctx Ctx, registryID int64) ([]*Record, error)
	// RecordsForExport — записи для выгрузки: при непустом ids — только они,
	// иначе все по фильтру search. Без пагинации, порядок по created_at DESC.
	RecordsForExport(ctx Ctx, registryID int64, search string, ids []int64) ([]*Record, error)

	// ── Публичные ссылки ──
	CreateShare(ctx Ctx, s *Share) error
	ListShares(ctx Ctx, registryID int64) ([]*Share, error)
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	DeleteShare(ctx Ctx, id, registryID int64) error
}

// UserReader — read-only идентичность пользователей (владелец таблицы — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	CompanyActive(ctx Ctx, companyID *int64) (bool, error)
}

// FileStore — хранение загруженных файлов/картинок в общем uploads-томе.
type FileStore interface {
	// Save — записать файл, вернуть относительный путь в uploads.
	Save(fileName string, data []byte) (string, error)
}

// EventBus — сокет-события клиентам через Redis gw2:registry:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}

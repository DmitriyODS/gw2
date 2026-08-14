package domain

import (
	"context"
	"io"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// RegistryRepository — персистентность реестров, их полей, записей, доступов и
// учётных выдач.
type RegistryRepository interface {
	// ── Реестры ──
	// ListRegistries — реестры области: свои, расшаренные лично, расшаренные
	// компаниям спрашивающего. Уровень доступа считается тем же запросом.
	ListRegistries(ctx Ctx, userID int64, companyIDs []int64, scope string) ([]*Registry, error)
	// GetRegistry — реестр без полей и без проверки доступа (её делает сервис).
	GetRegistry(ctx Ctx, id int64) (*Registry, error)
	// CountOwned — сколько реестров завёл человек (лимит тарифа).
	CountOwned(ctx Ctx, ownerID int64) (int, error)
	CreateRegistry(ctx Ctx, r *Registry) error
	UpdateRegistry(ctx Ctx, id int64, name string, position int, sectionFieldID *int64, accounting bool) error
	DeleteRegistry(ctx Ctx, id int64) error
	NextRegistryPosition(ctx Ctx, ownerID int64) (int, error)

	// ── Доступ ──
	// AccessOf — эффективный уровень человека к реестру (лучший из личной шары
	// и шар его компаний; владельцу — AccessOwner).
	AccessOf(ctx Ctx, registryID, userID int64, companyIDs []int64) (string, error)
	// Audience — кому адресовать сокет-события реестра: владелец, адресаты
	// личных шар и участники компаний, которым реестр раздан.
	Audience(ctx Ctx, registryID int64) ([]int64, error)
	ListUserShares(ctx Ctx, registryID int64) ([]*UserShare, error)
	// PutUserShare — выдать или обновить адресный доступ (повторная выдача
	// меняет уровень, а не плодит строки).
	PutUserShare(ctx Ctx, s *UserShare) error
	DeleteUserShare(ctx Ctx, registryID int64, userID, companyID *int64) error

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
	// SearchRecords — глобальный поиск по записям ВСЕХ доступных человеку
	// реестров (строка поиска Hola): один запрос, без обхода реестров.
	SearchRecords(ctx Ctx, userID int64, companyIDs []int64, query string, limit int) ([]*SearchHit, error)
	GetRecord(ctx Ctx, id int64) (*Record, error)
	CreateRecord(ctx Ctx, r *Record, searchText string) error
	UpdateRecord(ctx Ctx, id int64, data map[string]any, searchText string) error
	DeleteRecord(ctx Ctx, id int64) error
	// DeleteRecords — массовое удаление по фильтру (перечисленные записи либо
	// всё по фильтру экрана за вычетом Exclude). Возвращает УДАЛЁННЫЕ записи: их
	// id нужны событию, а data — чистке файлов, и оба приходят одним запросом
	// (RETURNING), без предварительной выборки.
	DeleteRecords(ctx Ctx, f ExportFilter) ([]*Record, error)
	// AllRecords — все записи реестра (для пересчёта search_text после удаления поля).
	AllRecords(ctx Ctx, registryID int64) ([]*Record, error)
	// RecordsForExport — записи для выгрузки/печати по тому же набору, что
	// показан на экране. Без пагинации, порядок по created_at DESC.
	RecordsForExport(ctx Ctx, f ExportFilter) ([]*Record, error)
	// RecordsOfCompanies — записи вместе с их реестром: раздел «Настройки →
	// Хранилище» показывает, в каком реестре лежит файл. Скоуп — компании,
	// чью квоту оплачивает спрашивающий (их присылает биллинг).
	RecordsOfCompanies(ctx Ctx, companyIDs []int64) ([]*RecordScope, error)

	// ── Учётный реестр ──
	// OpenIssues — открытые выдачи пачкой записей (плашки в таблице без N+1).
	OpenIssues(ctx Ctx, recordIDs []int64) (map[int64]*Issue, error)
	// IssueHistory — все выдачи записи вместе с их событиями, свежие первыми.
	IssueHistory(ctx Ctx, recordID int64) ([]*Issue, error)
	CreateIssue(ctx Ctx, i *Issue, comment string) error
	// ExtendIssue — сдвинуть срок открытой выдачи.
	ExtendIssue(ctx Ctx, issueID int64, dueAt *time.Time, comment string, actorID *int64) error
	// ReturnIssue — закрыть выдачу; ok=false, если её уже закрыли параллельно.
	ReturnIssue(ctx Ctx, issueID int64, at time.Time, comment string, actorID *int64) (bool, error)

	// ── Публичные ссылки ──
	CreateShare(ctx Ctx, s *Share) error
	ListShares(ctx Ctx, registryID int64) ([]*Share, error)
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	UpdateShare(ctx Ctx, id, registryID int64, name, access string, requireAuth bool) error
	DeleteShare(ctx Ctx, id, registryID int64) error
	// LogVisit — записать переход по ссылке (журнал посещений).
	LogVisit(ctx Ctx, v *ShareVisit) error
	ListVisits(ctx Ctx, shareID int64, limit int) ([]*ShareVisit, error)
}

// UserReader — read-only идентичность пользователей (владелец таблицы — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	CompanyActive(ctx Ctx, companyID *int64) (bool, error)
	// CompaniesOf — компании, где человек состоит: через них приходит доступ к
	// реестрам, раздаными компании целиком.
	CompaniesOf(ctx Ctx, userID int64) ([]int64, error)
	// CompanyMembers — участники компании (аудитория событий компанийной шары).
	CompanyMembers(ctx Ctx, companyID int64) ([]int64, error)
	// SearchDirectory — кандидаты в адресаты шаринга из компаний спрашивающего.
	SearchDirectory(ctx Ctx, companyIDs []int64, query string, limit int) ([]*User, error)
	// CompanyName — название компании для карточки выданного доступа.
	CompanyName(ctx Ctx, companyID int64) (string, error)
}

// FileStore — хранение загруженных файлов/картинок (общий uploads-том или S3).
type FileStore interface {
	// SaveFor — записать файл в квоту владельца: сверх лимита тарифа файл не
	// сохраняется. Возвращает ключ хранилища.
	SaveFor(ctx context.Context, userID, companyID int64, fileName string, data []byte) (string, error)
	// SaveStreamFor — то же потоком: гигабайтный файл нельзя подержать в памяти.
	SaveStreamFor(ctx context.Context, userID, companyID int64, fileName string, r io.Reader, size int64) (string, error)
	// RemoveFor — best-effort удаление файлов по ключам с возвратом места в
	// квоту (чистка при удалении записей/полей); ошибки не возвращаются.
	RemoveFor(ctx context.Context, userID, companyID int64, paths []string)
	// Remove — удаление БЕЗ учёта: так чистит раздел «Хранилище», где место
	// пересчитывает сам биллинг (он же инициатор и знает размеры).
	Remove(paths []string)
}

// EventBus — сокет-события клиентам через Redis gw2:registry:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}

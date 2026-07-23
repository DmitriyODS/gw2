package domain

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// VerificationStore — коды/ссылки подтверждения email (таблица
// email_verifications). Upsert перезаписывает запись пользователя (перевыпуск).
type VerificationStore interface {
	Upsert(ctx context.Context, userID int64, code, token string, expiresAt, sentAt time.Time) error
	GetByToken(ctx context.Context, token string) (*Verification, error)
	GetByUserID(ctx context.Context, userID int64) (*Verification, error)
	IncAttempts(ctx context.Context, userID int64) error
	Delete(ctx context.Context, userID int64) error
}

// MailClient — gRPC-клиент к mailsvc (межсервисное общение — только gRPC).
type MailClient interface {
	SendVerification(ctx context.Context, to, fio, code, link string) error
	SendPasswordReset(ctx context.Context, to, fio, link string) error
	SendCompanyInvite(ctx context.Context, to, companyName, roleName, link string) error
}

// PasswordReset — активный токен сброса пароля (одна заявка на пользователя).
type PasswordReset struct {
	UserID     int64
	Token      string
	ExpiresAt  time.Time
	LastSentAt time.Time
}

// PasswordResetStore — токены сброса пароля (таблица password_resets).
type PasswordResetStore interface {
	Upsert(ctx context.Context, userID int64, token string, expiresAt, sentAt time.Time) error
	GetByToken(ctx context.Context, token string) (*PasswordReset, error)
	GetByUserID(ctx context.Context, userID int64) (*PasswordReset, error)
	Delete(ctx context.Context, userID int64) error
}

// CompanyInvite — email-приглашение в компанию с ролью. GetByToken дозаполняет
// CompanyName/RoleName/RoleLevel джойнами (для письма и превью).
type CompanyInvite struct {
	ID          int64
	CompanyID   int64
	Email       string
	RoleID      int64
	Token       string
	InvitedBy   *int64
	ExpiresAt   time.Time
	CompanyName string
	RoleName    string
	RoleLevel   int
}

// CompanyInviteStore — email-приглашения в компанию (таблица company_invites).
type CompanyInviteStore interface {
	Upsert(ctx context.Context, companyID int64, email string, roleID int64, token string, invitedBy *int64, expiresAt time.Time) error
	GetByToken(ctx context.Context, token string) (*CompanyInvite, error)
	Delete(ctx context.Context, id int64) error
}

// UserRepository — персистентность пользователей. Пароли хешируются и
// проверяются на стороне PostgreSQL (pgcrypto, bcrypt через crypt/gen_salt) —
// так же, как это делал Flask.
type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	// GetByYandexID — аккаунт, привязанный к Яндекс ID; nil — не привязан.
	GetByYandexID(ctx context.Context, yandexID string) (*User, error)
	// YandexLinked — привязан ли к аккаунту Яндекс ID (для карточки профиля).
	YandexLinked(ctx context.Context, userID int64) (bool, error)
	// ListAll — все пользователи платформы (список супер-админа), включая
	// деактивированных: активные первыми, затем по id.
	ListAll(ctx context.Context) ([]*User, error)
	// HardDelete — безвозвратно удалить пользователя и все его данные (каскад +
	// ручная чистка RESTRICT-таблиц). Только для супер-админа.
	HardDelete(ctx context.Context, userID int64) error
	// SearchDirectory — глобальный каталог (контакты): активные, ILIKE по
	// fio/login, опционально без excludeID; сортировка по fio.
	SearchDirectory(ctx context.Context, query string, excludeID int64, loginOnly bool) ([]*User, error)
	Create(ctx context.Context, u *User) error
	// UpdateFields — точечное обновление колонок идентичности пользователя.
	UpdateFields(ctx context.Context, id int64, fields map[string]any) error
	GetRole(ctx context.Context, roleID int64) (*Role, error)
	// RoleByLevel — роль по уровню (роли фиксированы); nil — нет такой.
	RoleByLevel(ctx context.Context, level int) (*Role, error)
	// ListRoles — фиксированные роли по возрастанию уровня (GET /api/roles).
	ListRoles(ctx context.Context) ([]*Role, error)
	HashPassword(ctx context.Context, password string) (string, error)
	VerifyPassword(ctx context.Context, password, hash string) (bool, error)

	// ── Членство в компаниях (user_companies) ──
	// ListMemberships — все компании пользователя с ролью в каждой и активностью
	// компании, по created_at ASC.
	ListMemberships(ctx context.Context, userID int64) ([]Membership, error)
	// GetMembership — связка для конкретной компании; nil — не состоит.
	GetMembership(ctx context.Context, userID, companyID int64) (*Membership, error)
	// SharesCompany — есть ли у двух пользователей хотя бы одна общая компания.
	SharesCompany(ctx context.Context, userA, userB int64) (bool, error)
	// AddMembership — INSERT ... ON CONFLICT DO NOTHING.
	AddMembership(ctx context.Context, userID, companyID, roleID int64) error
	// RemoveMembership — удалить связку.
	RemoveMembership(ctx context.Context, userID, companyID int64) error
	// UpdateMembershipRole — сменить роль в конкретной компании.
	UpdateMembershipRole(ctx context.Context, userID, companyID, roleID int64) error
	// SetMembershipPost — должность в конкретной компании.
	SetMembershipPost(ctx context.Context, userID, companyID int64, post *string) error
	// CountCompanyMembersByLevel — активные члены компании с уровнем роли
	// (защита «последнего администратора компании»).
	CountCompanyMembersByLevel(ctx context.Context, companyID int64, level int) (int, error)
	// SearchDirectoryMembers — каталог членов КОМПАНИИ (по user_companies) с
	// ролью в этой компании; только активные, ILIKE по fio/login.
	SearchDirectoryMembers(ctx context.Context, query string, excludeID, companyID int64) ([]*User, error)
	// SearchNonMembers — активные пользователи (не супер-админ), ЕЩЁ НЕ
	// состоящие в компании (кандидаты на добавление), ILIKE по fio/login.
	SearchNonMembers(ctx context.Context, query string, companyID int64) ([]*User, error)
	// CompanyActive — активна ли компания (auth-гейт по активной компании
	// сессии из токена). nil → true (активной компании нет).
	CompanyActive(ctx context.Context, companyID *int64) (bool, error)
}

// LoginThrottle — защита от подбора пароля (Redis, ключи gw2:bf:*).
type LoginThrottle interface {
	// LockRemaining — сколько секунд осталось до снятия блокировки; 0 — нет.
	LockRemaining(ctx context.Context, login string) int
	// RegisterFailure — учесть неудачную попытку; >0 — выставлена блокировка
	// на столько секунд.
	RegisterFailure(ctx context.Context, login string) int
	RegisterSuccess(ctx context.Context, login string)
}

// AvatarStorage — файлы аватарок в общем uploads-каталоге (отдаёт nginx).
type AvatarStorage interface {
	// Save валидирует JPEG/PNG по содержимому и возвращает относительный
	// путь вида avatars/<имя>.<ext>.
	Save(fileBytes []byte) (string, error)
	Delete(avatarPath string)
	// ListFiles / WriteFile — обход и восстановление каталога avatars/
	// для резервной копии (export/import ZIP).
	ListFiles() ([]AvatarFile, error)
	WriteFile(name string, data []byte) error
}

// FileArchive — корневое файловое хранилище (pkg/storage) для ПОЛНОГО бэкапа:
// все загруженные файлы под их ключами (avatars/, registry/, calendar/, notes/,
// portal/, вложения мессенджера). Реализуется тем же storage.Storage.
type FileArchive interface {
	List(ctx context.Context, prefix string) ([]string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Put(ctx context.Context, key string, data []byte, contentType string) error
}

// CompanyRepository — персистентность компаний (таблица companies; схему
// ведёт migrate-контейнер goose).
type CompanyRepository interface {
	// ListCompanies — все компании с директором, по created_at DESC.
	ListCompanies(ctx context.Context) ([]*Company, error)
	// ListCompaniesWhereAdmin — компании, где пользователь — член с ролью
	// администратора (раздел «Компании» обычного пользователя), по created_at DESC.
	ListCompaniesWhereAdmin(ctx context.Context, userID int64) ([]*Company, error)
	// GetCompany — компания с директором; nil — нет такой.
	GetCompany(ctx context.Context, id int64) (*Company, error)
	GetCompanyByName(ctx context.Context, name string) (*Company, error)
	// GetCompanyByInviteCode — компания по коду-приглашению; nil — нет такой.
	GetCompanyByInviteCode(ctx context.Context, code string) (*Company, error)
	// CreateCompany — INSERT; заполняет ID, CreatedAt, IsActive.
	CreateCompany(ctx context.Context, c *Company) error
	// UpdateCompanyFields — точечное обновление колонок компании.
	UpdateCompanyFields(ctx context.Context, id int64, fields map[string]any) error
	// DeleteCompany — удаление; каскады (задачи, юниты, чаты, звонки) — в БД.
	DeleteCompany(ctx context.Context, id int64) error
	// CompanyStats — батч-счётчики сотрудников/задач без N+1.
	CompanyStats(ctx context.Context, ids []int64) (map[int64]CompanyStats, error)
}

// BackupStore — универсальный схемо-независимый дамп/восстановление таблиц.
// Import — TRUNCATE ... RESTART IDENTITY CASCADE + вставки в FK-порядке + setval,
// всё в одной транзакции (любая ошибка откатывает целиком).
type BackupStore interface {
	// AllTables — все base-таблицы public-схемы минус BackupExcluded.
	AllTables(ctx context.Context) ([]string, error)
	// ExportTables — дамп указанных таблиц: имя → JSON-массив строк (to_jsonb).
	ExportTables(ctx context.Context, tables []string) (map[string]json.RawMessage, error)
	// ImportTables — деструктивно заменить данные указанных таблиц.
	ImportTables(ctx context.Context, tables []string, data map[string]json.RawMessage) error
}

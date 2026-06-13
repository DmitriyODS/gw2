package domain

import "context"

// UserRepository — персистентность пользователей. Пароли хешируются и
// проверяются на стороне PostgreSQL (pgcrypto, bcrypt через crypt/gen_salt) —
// так же, как это делал Flask.
type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	// ListVisible — все видимые пользователи (без фильтра по компании,
	// как user_repo.get_all() в Flask), по id.
	ListVisible(ctx context.Context) ([]*User, error)
	// SearchDirectory — каталог: только видимые, ILIKE по fio/login,
	// опционально без excludeID и по компании; сортировка по fio.
	SearchDirectory(ctx context.Context, query string, excludeID int64, companyID *int64) ([]*User, error)
	Create(ctx context.Context, u *User) error
	// UpdateFields — точечное обновление колонок пользователя.
	UpdateFields(ctx context.Context, id int64, fields map[string]any) error
	GetRole(ctx context.Context, roleID int64) (*Role, error)
	// ListRoles — фиксированные роли по возрастанию уровня (GET /api/roles).
	ListRoles(ctx context.Context) ([]*Role, error)
	// CountVisibleByLevel — видимые пользователи с уровнем роли (защита
	// «последнего Администратора системы»).
	CountVisibleByLevel(ctx context.Context, level int) (int, error)
	// IsCompanyDirector — числится ли пользователь корневым Руководителем
	// какой-либо компании (companies.director_id).
	IsCompanyDirector(ctx context.Context, userID int64) (bool, error)
	HashPassword(ctx context.Context, password string) (string, error)
	VerifyPassword(ctx context.Context, password, hash string) (bool, error)

	// ── Членство в компаниях (user_companies) ──
	// ListMemberships — все компании пользователя с ролью в каждой и активностью
	// компании, по created_at ASC (первая — «первичная»).
	ListMemberships(ctx context.Context, userID int64) ([]Membership, error)
	// GetMembership — связка для конкретной компании; nil — не состоит.
	GetMembership(ctx context.Context, userID, companyID int64) (*Membership, error)
	// AddMembership — INSERT ... ON CONFLICT DO NOTHING.
	AddMembership(ctx context.Context, userID, companyID, roleID int64) error
	// RemoveMembership — удалить связку.
	RemoveMembership(ctx context.Context, userID, companyID int64) error
	// UpdateMembershipRole — сменить роль в конкретной компании.
	UpdateMembershipRole(ctx context.Context, userID, companyID, roleID int64) error
	// CountCompanyMembersByLevel — видимые члены компании с уровнем роли
	// (защита «последнего Руководителя компании»).
	CountCompanyMembersByLevel(ctx context.Context, companyID int64, level int) (int, error)
	// SearchDirectoryMembers — каталог членов КОМПАНИИ (по user_companies) с
	// ролью в этой компании; только видимые, ILIKE по fio/login.
	SearchDirectoryMembers(ctx context.Context, query string, excludeID, companyID int64) ([]*User, error)
	// SearchNonMembers — видимые пользователи, ЕЩЁ НЕ состоящие в компании
	// (кандидаты на добавление), ILIKE по fio/login; их первичная роль/компания.
	SearchNonMembers(ctx context.Context, query string, companyID int64) ([]*User, error)
	// SyncPrimaryCompany — пересчитать users.company_id/role_id из старейшей
	// связки (NULL, если членств не осталось). Держит инвариант NULL ⇔ админ.
	SyncPrimaryCompany(ctx context.Context, userID int64) error
	// CompanyActive — активна ли компания (auth-гейт по активной компании
	// сессии из токена). nil → true (Администратор системы).
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

// CompanyRepository — персистентность компаний (таблица companies; схему
// ведёт migrate-контейнер goose).
type CompanyRepository interface {
	// ListCompanies — все компании с директором, по created_at DESC.
	ListCompanies(ctx context.Context) ([]*Company, error)
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

// BackupStore — выгрузка и восстановление основных таблиц для резервной
// копии. Import — TRUNCATE ... RESTART IDENTITY CASCADE + вставки + setval,
// всё в одной транзакции (как прежний backup_service во Flask).
type BackupStore interface {
	ExportData(ctx context.Context) (*BackupData, error)
	ImportData(ctx context.Context, data *BackupData) error
}

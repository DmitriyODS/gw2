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
	// CountVisibleByLevel — видимые пользователи с уровнем роли (защита
	// «последнего Администратора системы»).
	CountVisibleByLevel(ctx context.Context, level int) (int, error)
	// IsCompanyDirector — числится ли пользователь корневым Руководителем
	// какой-либо компании (companies.director_id).
	IsCompanyDirector(ctx context.Context, userID int64) (bool, error)
	HashPassword(ctx context.Context, password string) (string, error)
	VerifyPassword(ctx context.Context, password, hash string) (bool, error)
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
}

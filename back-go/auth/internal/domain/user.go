// Package domain — модели и порты микросервиса авторизации.
//
// Идентичность (users) развязана с компаниями: пользователь — самостоятельная
// сущность, он не знает про компании. Принадлежность и роль живут в связке
// user_companies. Платформенный супер-админ — отдельный класс (users.is_super_admin).
package domain

import "time"

// Уровни ролей В КОМПАНИИ. Супер-админ — не роль, а отдельный флаг (IsSuperAdmin).
const (
	LevelEmployee = 1 // Сотрудник
	LevelManager  = 2 // Менеджер
	LevelAdmin    = 3 // Администратор компании (верхняя роль в компании)
)

type Role struct {
	ID    int64
	Name  string
	Level int
}

// CompanyRef — компания в объёме клеймов токена (name, settings) и проверки
// доступа (is_active).
type CompanyRef struct {
	ID       int64
	Name     string
	IsActive bool
	Settings map[string]any
}

// User — пользователь платформы. Идентичность (id/fio/login/контакты/аватар)
// не зависит от компаний.
//
// Поля контекста компании (CompanyID/Role/Post/CompanyActive) НЕ хранятся в users:
//   - для актора (authSource) заполняются из АКТИВНОЙ компании токена;
//   - для члена компании (списки/каталог) — из связки user_companies.
//
// Вне такого контекста они нулевые.
type User struct {
	ID            int64
	FIO           string
	Login         string
	HashPassword  string
	AvatarPath    *string
	Phone         *string
	Email         *string
	IsDefaultPass bool
	IsActive      bool // глобально активный аккаунт (бан супер-админом — false)
	IsSuperAdmin  bool
	EmailVerified bool // email подтверждён (самостоятельная регистрация); гейт логина
	CreatedAt     time.Time
	LastSeenAt    *time.Time

	CompanyID     *int64
	Role          Role
	Post          *string
	CompanyActive bool
}

// Membership — связка «пользователь ↔ компания» с ролью в этой компании
// (таблица user_companies). Один аккаунт может состоять в нескольких компаниях
// с разными ролями. Активная компания сессии выбирается при login/switch и
// кладётся в токен.
type Membership struct {
	CompanyID int64
	Company   *CompanyRef
	Role      Role
	Post      *string
	CreatedAt time.Time
}

// Verification — активный код/ссылка подтверждения email (таблица
// email_verifications, одна запись на пользователя).
type Verification struct {
	UserID     int64
	Code       string
	Token      string
	Attempts   int
	ExpiresAt  time.Time
	LastSentAt time.Time
}

// Level — уровень роли в компании контекста (0 — нет контекста/роли).
func (u *User) Level() int {
	if u == nil {
		return 0
	}
	return u.Role.Level
}

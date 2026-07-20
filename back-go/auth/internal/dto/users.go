// Package dto — transfer-объекты HTTP-контракта. Формы JSON байт-в-байт
// совместимы с прежними marshmallow-схемами Flask (schemas/user.py) —
// фронт не меняется.
package dto

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// JSONTime — формат marshmallow (ISO8601 с явным смещением +00:00).
type JSONTime time.Time

func (t JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).UTC().Format("2006-01-02T15:04:05.999999-07:00") + `"`), nil
}

type RoleRef struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type CompanyRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// User — профиль пользователя. Идентичность не зависит от компаний; поля
// контекста (post/role/company_id) заполнены, только когда пользователь
// рассматривается в рамках конкретной компании (член компании).
type User struct {
	ID            int64    `json:"id"`
	FIO           string   `json:"fio"`
	Login         string   `json:"login"`
	Post          *string  `json:"post"`
	Role          *RoleRef `json:"role"`
	CompanyID     *int64   `json:"company_id"`
	Phone         *string  `json:"phone"`
	Email         *string  `json:"email"`
	AvatarPath    *string  `json:"avatar_path"`
	IsDefaultPass bool     `json:"is_default_pass"`
	IsActive      bool     `json:"is_active"`
	IsSuperAdmin  bool     `json:"is_super_admin"`
	CreatedAt     JSONTime `json:"created_at"`
	StatusEmoji   *string  `json:"status_emoji"`
	StatusText    *string  `json:"status_text"`
	OnVacation    bool     `json:"on_vacation"`
}

func roleRef(r domain.Role) *RoleRef {
	if r.Level == 0 {
		return nil
	}
	return &RoleRef{ID: r.ID, Name: r.Name, Level: r.Level}
}

func NewUser(u *domain.User) User {
	return User{
		ID:            u.ID,
		FIO:           u.FIO,
		Login:         u.Login,
		Post:          u.Post,
		Role:          roleRef(u.Role),
		CompanyID:     u.CompanyID,
		Phone:         u.Phone,
		Email:         u.Email,
		AvatarPath:    u.AvatarPath,
		IsDefaultPass: u.IsDefaultPass,
		IsActive:      u.IsActive,
		IsSuperAdmin:  u.IsSuperAdmin,
		CreatedAt:     JSONTime(u.CreatedAt),
		StatusEmoji:   u.StatusEmoji,
		StatusText:    u.StatusText,
		OnVacation:    u.OnVacation,
	}
}

func NewUsers(users []*domain.User) []User {
	out := make([]User, 0, len(users))
	for _, u := range users {
		out = append(out, NewUser(u))
	}
	return out
}

// DirectoryUser — публичный профиль (каталог/контакты). Role/Post/CompanyID
// заполнены только в каталоге членов конкретной компании; в глобальном поиске
// (контакты) — nil.
type DirectoryUser struct {
	ID         int64     `json:"id"`
	FIO        string    `json:"fio"`
	Login      string    `json:"login"`
	Post       *string   `json:"post"`
	Role       *RoleRef  `json:"role"`
	CompanyID  *int64    `json:"company_id"`
	Phone      *string   `json:"phone"`
	Email      *string   `json:"email"`
	AvatarPath  *string   `json:"avatar_path"`
	LastSeenAt  *JSONTime `json:"last_seen_at"`
	StatusEmoji *string   `json:"status_emoji"`
	StatusText  *string   `json:"status_text"`
	OnVacation  bool      `json:"on_vacation"`
}

func NewDirectoryUser(u *domain.User) DirectoryUser {
	out := DirectoryUser{
		ID:         u.ID,
		FIO:        u.FIO,
		Login:      u.Login,
		Post:       u.Post,
		Role:       roleRef(u.Role),
		CompanyID:  u.CompanyID,
		Phone:       u.Phone,
		Email:       u.Email,
		AvatarPath:  u.AvatarPath,
		StatusEmoji: u.StatusEmoji,
		StatusText:  u.StatusText,
		OnVacation:  u.OnVacation,
	}
	if u.LastSeenAt != nil {
		ts := JSONTime(*u.LastSeenAt)
		out.LastSeenAt = &ts
	}
	return out
}

func NewDirectoryUsers(users []*domain.User) []DirectoryUser {
	out := make([]DirectoryUser, 0, len(users))
	for _, u := range users {
		out = append(out, NewDirectoryUser(u))
	}
	return out
}

// MembershipDTO — компания пользователя и его роль в ней (для списка компаний
// в теле login/refresh/me: фронт показывает переключатель и пикер при логине).
type MembershipDTO struct {
	CompanyID   int64  `json:"company_id"`
	CompanyName string `json:"company_name"`
	IsActive    bool   `json:"is_active"`
	RoleLevel   int    `json:"role_level"`
	RoleName    string `json:"role_name"`
}

func NewMemberships(ms []domain.Membership) []MembershipDTO {
	out := make([]MembershipDTO, 0, len(ms))
	for _, m := range ms {
		d := MembershipDTO{
			CompanyID: m.CompanyID,
			RoleLevel: m.Role.Level,
			RoleName:  m.Role.Name,
		}
		if m.Company != nil {
			d.CompanyName = m.Company.Name
			d.IsActive = m.Company.IsActive
		}
		out = append(out, d)
	}
	return out
}

// Session — ответ login/select/switch/refresh/change-default. Клеймы продублированы
// в теле, потому что PASETO-payload фронт больше не декодирует (в отличие от
// JWT); refresh-токен уезжает только в HttpOnly-cookie. NeedsCompanySelection —
// этап выбора компании при логине (>1 компании): access/refresh не выдаются,
// фронт показывает пикер и шлёт выбор с SelectToken на /auth/select-company.
type Session struct {
	AccessToken     string          `json:"access_token"`
	UserID          int64           `json:"user_id"`
	ForceChange     bool            `json:"force_change"`
	CompanyID       *int64          `json:"company_id"`
	CompanyName     *string         `json:"company_name"`
	CompanySettings map[string]any  `json:"company_settings"`
	RoleLevel       int             `json:"role_level"`
	IsSuperAdmin    bool            `json:"is_super_admin"`
	Companies       []MembershipDTO `json:"companies"`

	NeedsCompanySelection bool   `json:"needs_company_selection,omitempty"`
	SelectToken           string `json:"select_token,omitempty"`

	RefreshToken string `json:"-"`
}

// ── Запросы ──────────────────────────────────────────────────────

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// RegisterRequest — публичная регистрация: самостоятельное создание аккаунта
// (без компании). Логин генерируется из ФИО (фронт подставляет через
// suggest-login, пользователь может поправить); пустой — сгенерируем сами.
// Пароль виден/редактируется пользователем на фронте. После регистрации —
// подтверждение email кодом/ссылкой.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	FIO      string `json:"fio"`
	Email    string `json:"email"`
}

// RegisterResult — ответ register: сессия НЕ выдаётся, пока email не
// подтверждён. Фронт переходит на экран ввода кода.
type RegisterResult struct {
	Status string `json:"status"` // "verification_required"
	Email  string `json:"email"`
}

// VerifyEmailRequest — подтверждение по ссылке (token) или вводом кода (email+code).
type VerifyEmailRequest struct {
	Token string `json:"token"`
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ResetPasswordRequest — установка нового пароля по токену из письма.
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// PasswordResetResult — ответ reset-password: логин для префилла на экране входа.
type PasswordResetResult struct {
	Login string `json:"login"`
}

// InvitePreview — превью email-приглашения (что увидит получатель до принятия).
type InvitePreview struct {
	CompanyName string `json:"company_name"`
	RoleName    string `json:"role_name"`
	Email       string `json:"email"`
}

type ChangeDefaultRequest struct {
	UserID          int64  `json:"-"`
	NewLogin        string `json:"new_login"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// CreateUserRequest — компанийный администратор заводит сотрудника в СВОЕЙ
// активной компании (company берётся из токена актора, не из тела). Post —
// должность в этой компании (хранится в связке).
type CreateUserRequest struct {
	FIO      string  `json:"fio"`
	Login    string  `json:"login"`
	Post     *string `json:"post"`
	RoleID   int64   `json:"role_id"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

// UpdateUserRequest — PATCH /users/<id> (член активной компании актора):
// nil-указатель = поле не передано. Post обновляет должность в активной компании.
type UpdateUserRequest struct {
	FIO   *string `json:"fio"`
	Login *string `json:"login"`
	Post  *string `json:"post"`
	Phone *string `json:"phone"`
	Email *string `json:"email"`
	// Режим «в отпуске» сотрудника: создатель компании может явно проставить
	// и снять его (гарды tasksvc/petsvc — те же, что при самостоятельном).
	OnVacation *bool `json:"on_vacation"`
}

// UpdateMeRequest — PATCH /users/me. Должность — атрибут членства в компании,
// её задаёт администратор компании, не сам пользователь.
type UpdateMeRequest struct {
	FIO             *string `json:"fio"`
	Login           *string `json:"login"`
	Phone           *string `json:"phone"`
	Email           *string `json:"email"`
	CurrentPassword *string `json:"current_password"`
	NewPassword     *string `json:"new_password"`
	ConfirmPassword *string `json:"confirm_password"`
	// Пользовательский статус (мессенджер): пустая строка — снять.
	StatusEmoji *string `json:"status_emoji"`
	StatusText  *string `json:"status_text"`
	// Режим «в отпуске»: задачи/юниты закрыты, грувик заморожен.
	OnVacation *bool `json:"on_vacation"`
}

// AddMemberRequest — POST /companies/<id>/members: добавить существующего
// пользователя в компанию с ролью.
type AddMemberRequest struct {
	UserID int64 `json:"user_id"`
	RoleID int64 `json:"role_id"`
}

// DirectoryRequest — GET /users/directory.
type DirectoryRequest struct {
	ActorID   int64
	Query     string
	ExcludeID int64
	CompanyID *int64 // только для Администратора системы (из query)
	// LoginOnly — глобальный поиск строго по логину (для мессенджера: найти
	// нового собеседника можно только по точному логину, не листая всех по ФИО).
	// Пустой запрос при LoginOnly отдаёт пусто — каталог по ФИО не «вываливается».
	LoginOnly bool
}

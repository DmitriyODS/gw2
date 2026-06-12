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

// User — форма UserSchema.
type User struct {
	ID            int64       `json:"id"`
	FIO           string      `json:"fio"`
	Login         string      `json:"login"`
	Post          *string     `json:"post"`
	Role          RoleRef     `json:"role"`
	CompanyID     *int64      `json:"company_id"`
	Company       *CompanyRef `json:"company"`
	Phone         *string     `json:"phone"`
	Email         *string     `json:"email"`
	AvatarPath    *string     `json:"avatar_path"`
	IsDefaultPass bool        `json:"is_default_pass"`
	IsHidden      bool        `json:"is_hidden"`
	IsRootAdmin   bool        `json:"is_root_admin"`
	CreatedAt     JSONTime    `json:"created_at"`
}

func NewUser(u *domain.User) User {
	out := User{
		ID:            u.ID,
		FIO:           u.FIO,
		Login:         u.Login,
		Post:          u.Post,
		Role:          RoleRef{ID: u.Role.ID, Name: u.Role.Name, Level: u.Role.Level},
		CompanyID:     u.CompanyID,
		Phone:         u.Phone,
		Email:         u.Email,
		AvatarPath:    u.AvatarPath,
		IsDefaultPass: u.IsDefaultPass,
		IsHidden:      u.IsHidden,
		IsRootAdmin:   u.IsRootAdmin,
		CreatedAt:     JSONTime(u.CreatedAt),
	}
	if u.Company != nil {
		out.Company = &CompanyRef{ID: u.Company.ID, Name: u.Company.Name}
	}
	return out
}

func NewUsers(users []*domain.User) []User {
	out := make([]User, 0, len(users))
	for _, u := range users {
		out = append(out, NewUser(u))
	}
	return out
}

// DirectoryUser — форма UserDirectorySchema (публичный профиль: без
// is_default_pass/is_hidden и прочих внутренних полей).
type DirectoryUser struct {
	ID         int64     `json:"id"`
	FIO        string    `json:"fio"`
	Login      string    `json:"login"`
	Post       *string   `json:"post"`
	Role       RoleRef   `json:"role"`
	CompanyID  *int64    `json:"company_id"`
	Phone      *string   `json:"phone"`
	Email      *string   `json:"email"`
	AvatarPath *string   `json:"avatar_path"`
	LastSeenAt *JSONTime `json:"last_seen_at"`
}

func NewDirectoryUser(u *domain.User) DirectoryUser {
	out := DirectoryUser{
		ID:         u.ID,
		FIO:        u.FIO,
		Login:      u.Login,
		Post:       u.Post,
		Role:       RoleRef{ID: u.Role.ID, Name: u.Role.Name, Level: u.Role.Level},
		CompanyID:  u.CompanyID,
		Phone:      u.Phone,
		Email:      u.Email,
		AvatarPath: u.AvatarPath,
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

// Session — ответ login/refresh/change-default. Клеймы продублированы в
// теле, потому что PASETO-payload фронт больше не декодирует (в отличие от
// JWT); refresh-токен уезжает только в HttpOnly-cookie.
type Session struct {
	AccessToken     string         `json:"access_token"`
	UserID          int64          `json:"user_id"`
	ForceChange     bool           `json:"force_change"`
	CompanyID       *int64         `json:"company_id"`
	CompanyName     *string        `json:"company_name"`
	CompanySettings map[string]any `json:"company_settings"`
	RoleLevel       int            `json:"role_level"`
	IsRootAdmin     bool           `json:"is_root_admin"`

	RefreshToken string `json:"-"`
}

// ── Запросы ──────────────────────────────────────────────────────

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ChangeDefaultRequest struct {
	UserID          int64  `json:"-"`
	NewLogin        string `json:"new_login"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type CreateUserRequest struct {
	FIO       string  `json:"fio"`
	Login     string  `json:"login"`
	Post      *string `json:"post"`
	RoleID    int64   `json:"role_id"`
	CompanyID *int64  `json:"company_id"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	Password  *string `json:"password"`
}

// UpdateUserRequest — PATCH /users/<id>: nil-указатель = поле не передано.
type UpdateUserRequest struct {
	FIO       *string `json:"fio"`
	Login     *string `json:"login"`
	Post      *string `json:"post"`
	CompanyID *int64  `json:"company_id"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
}

// UpdateMeRequest — PATCH /users/me.
type UpdateMeRequest struct {
	FIO             *string `json:"fio"`
	Login           *string `json:"login"`
	Post            *string `json:"post"`
	Phone           *string `json:"phone"`
	Email           *string `json:"email"`
	CurrentPassword *string `json:"current_password"`
	NewPassword     *string `json:"new_password"`
	ConfirmPassword *string `json:"confirm_password"`
}

// DirectoryRequest — GET /users/directory.
type DirectoryRequest struct {
	ActorID   int64
	Query     string
	ExcludeID int64
	CompanyID *int64 // только для Администратора системы (из query)
}

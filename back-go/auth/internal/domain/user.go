// Package domain — модели и порты микросервиса авторизации.
//
// Таблицами users/roles/companies в рантайме (в части auth/users) владеет
// этот сервис, схему ведёт migrate-контейнер (goose).
package domain

import "time"

// Уровни ролей — общие с back/app/utils/permissions.py.
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelDirector = 3
	LevelAdmin    = 4
)

type Role struct {
	ID    int64
	Name  string
	Level int
}

// CompanyRef — компания пользователя в объёме, нужном auth-сервису:
// клеймы токена (name, settings) и проверка доступа (is_active).
type CompanyRef struct {
	ID       int64
	Name     string
	IsActive bool
	Settings map[string]any
}

type User struct {
	ID            int64
	FIO           string
	Login         string
	HashPassword  string
	Post          *string
	Role          Role
	CompanyID     *int64
	Company       *CompanyRef
	AvatarPath    *string
	Phone         *string
	Email         *string
	IsDefaultPass bool
	IsHidden      bool
	IsRootAdmin   bool
	CreatedAt     time.Time
	LastSeenAt    *time.Time
}

func (u *User) Level() int {
	if u == nil {
		return 0
	}
	return u.Role.Level
}

// CompanyActive — у пользователя либо нет компании (Администратор системы),
// либо она должна быть активна.
func (u *User) CompanyActive() bool {
	if u.CompanyID == nil {
		return true
	}
	return u.Company != nil && u.Company.IsActive
}

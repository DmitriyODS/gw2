package dto

import "github.com/DmitriyODS/gw2/back-go/auth/internal/domain"

// Формы JSON компаний и ролей — байт-в-байт совместимы с прежними
// Flask-схемами (schemas/company.py, schemas/role.py). Порядок полей —
// алфавитный: jsonify во Flask сортировал ключи.

// Role — форма RoleSchema.
type Role struct {
	ID    int64  `json:"id"`
	Level int    `json:"level"`
	Name  string `json:"name"`
}

func NewRoles(roles []*domain.Role) []Role {
	out := make([]Role, 0, len(roles))
	for _, r := range roles {
		out = append(out, Role{ID: r.ID, Level: r.Level, Name: r.Name})
	}
	return out
}

// CompanyDirectorRef — форма CompanyDirectorRefSchema.
type CompanyDirectorRef struct {
	AvatarPath *string `json:"avatar_path"`
	FIO        string  `json:"fio"`
	ID         int64   `json:"id"`
	Login      string  `json:"login"`
}

// Company — форма CompanySchema + виртуальные счётчики (_enrich во Flask).
type Company struct {
	CreatedAt      JSONTime            `json:"created_at"`
	Description    *string             `json:"description"`
	Director       *CompanyDirectorRef `json:"director"`
	DirectorID     *int64              `json:"director_id"`
	EmployeesCount int                 `json:"employees_count"`
	ID             int64               `json:"id"`
	IsActive       bool                `json:"is_active"`
	Name           string              `json:"name"`
	Settings       map[string]any      `json:"settings"`
	TasksCount     int                 `json:"tasks_count"`
}

func NewCompany(c *domain.Company, stats domain.CompanyStats) Company {
	out := Company{
		CreatedAt:      JSONTime(c.CreatedAt),
		Description:    c.Description,
		DirectorID:     c.DirectorID,
		EmployeesCount: stats.Employees,
		ID:             c.ID,
		IsActive:       c.IsActive,
		Name:           c.Name,
		Settings:       c.Settings,
		TasksCount:     stats.Tasks,
	}
	if c.Director != nil {
		out.Director = &CompanyDirectorRef{
			AvatarPath: c.Director.AvatarPath,
			FIO:        c.Director.FIO,
			ID:         c.Director.ID,
			Login:      c.Director.Login,
		}
	}
	return out
}

// CompanyList — ответ GET /api/companies.
type CompanyList struct {
	Items []Company `json:"items"`
	Total int       `json:"total"`
}

// WeekendSettings — ответ GET/PUT /api/companies/<id>/weekend-settings.
type WeekendSettings struct {
	WeekendDays []int `json:"weekend_days"`
}

// GrooveSettings — ответ GET/PUT /api/companies/<id>/groove-settings:
// включён ли режим «Мой Groove» (settings.uses_groove).
type GrooveSettings struct {
	Enabled bool `json:"enabled"`
}

// ── Запросы (после schema-валидации в транспорте) ────────────────

// CompanyCreate — распарсенный POST-боди: settings уже прошли schema-load
// с дефолтами недостающих ключей (CompanySettingsSchema).
type CompanyCreate struct {
	Name        string
	Description *string
	DirectorID  *int64
	IsActive    bool
	Settings    map[string]any
}

// CompanyUpdate — распарсенный PATCH-боди: *Set = поле передано (значение
// может быть null для allow_none-полей).
type CompanyUpdate struct {
	Name           *string
	Description    *string
	DescriptionSet bool
	DirectorID     *int64
	DirectorSet    bool
	IsActive       *bool
	Settings       map[string]any // переданные ключи (partial); nil — не передано
	SettingsSet    bool
}

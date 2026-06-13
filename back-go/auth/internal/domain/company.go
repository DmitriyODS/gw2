package domain

import "time"

// DefaultCompanySettings — дефолтные настройки рабочего процесса компании
// (DEFAULT_SETTINGS из back/app/models/company.py). Свежая map на каждый
// вызов — настройки компаний не должны делить ссылки.
func DefaultCompanySettings() map[string]any {
	return map[string]any{
		"uses_yougile": false,
		"uses_stages":  false,
		"uses_calls":   true,
		// Режим «Мой Groove» (геймификация-питомцы) включён по умолчанию.
		"uses_groove": true,
		// Выходные дни компании: 0=Пн … 6=Вс (Python date.weekday()).
		"weekend_days": []any{5, 6},
	}
}

// DefaultWeekend — суббота и воскресенье (DEFAULT_WEEKEND из utils/workweek.py).
var DefaultWeekend = []int{5, 6}

// CompanyDirector — корневой Руководитель в объёме CompanyDirectorRefSchema.
type CompanyDirector struct {
	ID         int64
	FIO        string
	Login      string
	AvatarPath *string
}

// Company — компания целиком (домен управления компаниями; CompanyRef из
// user.go остаётся узким срезом для клеймов токена).
type Company struct {
	ID          int64
	Name        string
	Description *string
	DirectorID  *int64
	Director    *CompanyDirector
	IsActive    bool
	Settings    map[string]any
	CreatedAt   time.Time
	InviteCode  *string
}

// CompanyStats — счётчики для строки таблицы Компаний.
type CompanyStats struct {
	Employees int
	Tasks     int
}

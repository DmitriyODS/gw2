// Package spaces — пространства инструментов: «Моё» и команды.
//
// Вещь любого инструмента (календарь, реестр, форма, заметка, доска, файл,
// ежедневник, расписание) лежит ровно в одном пространстве. Пространство —
// колонка company_id самой вещи: NULL означает личное пространство её
// owner_id, иначе вещь принадлежит команде, а owner_id — её автор.
//
// Членство читается из общей БД напрямую (user_companies), а НЕ из активной
// компании токена: команда не прячет инструменты, человек видит вещи всех
// своих команд сразу. Отключённая команда (companies.is_active = FALSE) прав
// не даёт — это общий выключатель компании.
//
// Здесь — только SQL-условия, общие для сервисов. Аргументы — номера
// плейсхолдеров и колонки из кода сервиса, а не внешние данные: инвариант
// параметризации сохраняется.
package spaces

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Member — «человек user состоит в активной команде col».
func Member(user, col string) string {
	return `EXISTS (SELECT 1 FROM user_companies uc_m
	                  JOIN companies c_m ON c_m.id = uc_m.company_id
	                 WHERE uc_m.user_id = ` + user + `
	                   AND uc_m.company_id = ` + col + `
	                   AND c_m.is_active)`
}

// Admin — «человек user управляет командой col»: её создатель или участник
// с ролью администратора. Управлять — значит удалять вещи команды, переносить
// их и менять уровень доступа участников.
func Admin(user, col string) string {
	return `EXISTS (SELECT 1 FROM user_companies uc_a
	                  JOIN companies c_a ON c_a.id = uc_a.company_id
	                  JOIN roles r_a ON r_a.id = uc_a.role_id
	                 WHERE uc_a.user_id = ` + user + `
	                   AND uc_a.company_id = ` + col + `
	                   AND c_a.is_active
	                   AND (r_a.level >= 3 OR c_a.created_by = uc_a.user_id))`
}

// MyTeams — подзапрос id активных команд человека user (для `IN (...)`).
func MyTeams(user string) string {
	return `SELECT uc_t.company_id FROM user_companies uc_t
	          JOIN companies c_t ON c_t.id = uc_t.company_id
	         WHERE uc_t.user_id = ` + user + ` AND c_t.is_active`
}

// Querier — то, что умеют и пул, и транзакция pgx.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Role — положение человека в команде.
type Role struct {
	Member bool // состоит в активной команде
	Admin  bool // создатель или администратор
}

// RoleIn — положение человека в команде одним запросом. Нужен там, где
// решение принимает Go: например, можно ли положить вещь в эту команду.
func RoleIn(ctx context.Context, db Querier, userID, companyID int64) (Role, error) {
	var r Role
	err := db.QueryRow(ctx, `
		SELECT TRUE, (r.level >= 3 OR c.created_by = uc.user_id)
		  FROM user_companies uc
		  JOIN companies c ON c.id = uc.company_id
		  JOIN roles r ON r.id = uc.role_id
		 WHERE uc.user_id = $1 AND uc.company_id = $2 AND c.is_active`,
		userID, companyID).Scan(&r.Member, &r.Admin)
	if errors.Is(err, pgx.ErrNoRows) {
		return Role{}, nil
	}
	return r, err
}

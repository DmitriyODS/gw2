package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// UserReader — read-only доступ к идентичности и членству пользователей
// (владелец таблиц в рантайме — authsvc): объём — auth-мидлварь и выбор
// аудитории шаринга (компании пользователя, проверка членства).
type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader { return &UserReader{pool: pool} }

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, fio, avatar_path, is_active, is_super_admin
		  FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.IsActive, &u.IsSuperAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// UserCompanies — активные компании пользователя (id+имя) для выбора аудитории
// шаринга. Отключённые компании (is_active=false) исключаются.
func (r *UserReader) UserCompanies(ctx context.Context, userID int64) ([]*domain.Company, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name
		  FROM user_companies uc
		  JOIN companies c ON c.id = uc.company_id
		 WHERE uc.user_id = $1 AND c.is_active
		 ORDER BY c.name, c.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Company{}
	for rows.Next() {
		var c domain.Company
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// CompanyIDs — id компаний пользователя (скоуп «расшарено моей компании»).
// Включает и отключённые: доступ к уже расшаренному контенту не должен
// пропадать при временной блокировке компании.
func (r *UserReader) CompanyIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT company_id FROM user_companies WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// IsCompanyMember — состоит ли пользователь в компании; возвращает и имя
// компании (для денормализации при создании шаринга).
func (r *UserReader) IsCompanyMember(ctx context.Context, userID, companyID int64) (bool, string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `
		SELECT c.name FROM user_companies uc
		  JOIN companies c ON c.id = uc.company_id
		 WHERE uc.user_id = $1 AND uc.company_id = $2`, userID, companyID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, name, nil
}

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

// UserReader — read-only доступ к идентичности пользователей (владелец таблицы
// в рантайме — authsvc); объём — auth-мидлварь и адресаты доступа. Компания и
// роль развязаны: они приходят из токена.
type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader { return &UserReader{pool: pool} }

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, fio, avatar_path, is_active, is_super_admin
		  FROM users
		 WHERE id = $1`, id).
		Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.IsActive, &u.IsSuperAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// CompaniesOf — компании, где человек состоит: через них приходит доступ к
// формам, назначенным компании целиком. Пустой срез, а не nil: он уезжает в SQL
// как ANY($n), и nil там означал бы «сравнить не с чем».
func (r *UserReader) CompaniesOf(ctx context.Context, userID int64) ([]int64, error) {
	return r.ids(ctx, `SELECT company_id FROM user_companies WHERE user_id = $1`, userID)
}

// CompanyMembers — участники компании: их извещают о назначенной ей форме.
func (r *UserReader) CompanyMembers(ctx context.Context, companyID int64) ([]int64, error) {
	return r.ids(ctx, `SELECT user_id FROM user_companies WHERE company_id = $1`, companyID)
}

func (r *UserReader) ids(ctx context.Context, sql string, arg int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, sql, arg)
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

// SearchDirectory — кандидаты в адресаты. Список ограничен компаниями
// спрашивающего: назначать форму можно коллегам, а не каталогу всей платформы.
// Пустой запрос отдаёт начало списка — модалка показывает его сразу.
func (r *UserReader) SearchDirectory(ctx context.Context, companyIDs []int64, query string, limit int) ([]*domain.User, error) {
	if len(companyIDs) == 0 {
		return []*domain.User{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT u.id, u.fio, u.avatar_path, u.is_active, u.is_super_admin
		  FROM users u
		  JOIN user_companies uc ON uc.user_id = u.id
		 WHERE uc.company_id = ANY($1) AND u.is_active
		   AND ($2 = '' OR u.fio ILIKE '%' || $2 || '%' OR u.login ILIKE '%' || $2 || '%')
		 ORDER BY u.fio
		 LIMIT $3`, companyIDs, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.IsActive, &u.IsSuperAdmin); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (r *UserReader) CompanyName(ctx context.Context, companyID int64) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `SELECT name FROM companies WHERE id = $1`, companyID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

// CompanyActive — активность ИМЕННО выбранной (активной) компании сессии.
func (r *UserReader) CompanyActive(ctx context.Context, companyID *int64) (bool, error) {
	if companyID == nil {
		return true, nil
	}
	var active bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_active FROM companies WHERE id = $1`, *companyID).Scan(&active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return active, nil
}

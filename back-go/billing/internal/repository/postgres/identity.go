package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// Identity — read-only доступ к идентичности: пользователи, создатели компаний
// и членство. Таблицы ведёт authsvc, биллинг их только читает (тот же приём,
// что у remindersvc с users и у notesvc с user_companies).
type Identity struct {
	pool *pgxpool.Pool
}

var _ domain.IdentityReader = (*Identity)(nil)

func NewIdentity(pool *pgxpool.Pool) *Identity { return &Identity{pool: pool} }

func (r *Identity) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, fio, login, is_active, is_super_admin, email FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.FIO, &u.Login, &u.IsActive, &u.IsSuperAdmin, &u.Email)
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *Identity) SearchUsers(ctx context.Context, query string, limit int) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, fio, login, is_active, is_super_admin, email
		  FROM users
		 WHERE $1 = '' OR fio ILIKE '%' || $1 || '%' OR login ILIKE '%' || $1 || '%'
		 ORDER BY fio
		 LIMIT $2`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.FIO, &u.Login, &u.IsActive, &u.IsSuperAdmin, &u.Email); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

// CompanyOwner — создатель компании: его тариф действует на всю компанию.
// Компания без создателя (историческая) даёт 0 — лимиты тогда бесплатные.
func (r *Identity) CompanyOwner(ctx context.Context, companyID int64) (int64, error) {
	var owner *int64
	err := r.pool.QueryRow(ctx, `SELECT created_by FROM companies WHERE id = $1`, companyID).Scan(&owner)
	if err != nil {
		if noRows(err) {
			return 0, nil
		}
		return 0, err
	}
	if owner == nil {
		return 0, nil
	}
	return *owner, nil
}

func (r *Identity) OwnedCompanies(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM companies WHERE created_by = $1 ORDER BY id`, userID)
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

func (r *Identity) IsCompanyMember(ctx context.Context, userID, companyID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_companies WHERE user_id = $1 AND company_id = $2)`,
		userID, companyID).Scan(&ok)
	return ok, err
}

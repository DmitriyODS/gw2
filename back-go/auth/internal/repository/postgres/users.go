// Package postgres — доступ к users/roles/companies через pgx.
// Хеширование/проверка паролей — pgcrypto (crypt + gen_salt('bf')),
// совместимо с хешами, созданными Flask.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository { return &UserRepository{pool: pool} }

var _ domain.UserRepository = (*UserRepository)(nil)

const userColumns = `
	u.id, u.fio, u.login, u.hash_password, u.post,
	r.id, r.name, r.level,
	u.company_id, c.id, c.name, c.is_active, c.settings,
	u.avatar_path, u.phone, u.email,
	u.is_default_pass, u.is_hidden, u.is_root_admin,
	u.created_at, u.last_seen_at`

const userFrom = `
	FROM users u
	JOIN roles r ON r.id = u.role_id
	LEFT JOIN companies c ON c.id = u.company_id`

func scanUser(row pgx.Row) (*domain.User, error) {
	var (
		u         domain.User
		companyID *int64
		cID       *int64
		cName     *string
		cActive   *bool
		cSettings map[string]any
	)
	err := row.Scan(
		&u.ID, &u.FIO, &u.Login, &u.HashPassword, &u.Post,
		&u.Role.ID, &u.Role.Name, &u.Role.Level,
		&companyID, &cID, &cName, &cActive, &cSettings,
		&u.AvatarPath, &u.Phone, &u.Email,
		&u.IsDefaultPass, &u.IsHidden, &u.IsRootAdmin,
		&u.CreatedAt, &u.LastSeenAt,
	)
	if err != nil {
		return nil, err
	}
	u.CompanyID = companyID
	if cID != nil {
		u.Company = &domain.CompanyRef{ID: *cID, IsActive: cActive != nil && *cActive, Settings: cSettings}
		if cName != nil {
			u.Company.Name = *cName
		}
	}
	return &u, nil
}

func (r *UserRepository) getOne(ctx context.Context, where string, arg any) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, "SELECT"+userColumns+userFrom+" WHERE "+where, arg)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return r.getOne(ctx, "u.id = $1", id)
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	return r.getOne(ctx, "u.login = $1", login)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, nil
	}
	return r.getOne(ctx, "lower(u.email) = lower($1)", email)
}

func (r *UserRepository) list(ctx context.Context, tail string, args ...any) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, "SELECT"+userColumns+userFrom+" "+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepository) ListVisible(ctx context.Context) ([]*domain.User, error) {
	return r.list(ctx, "WHERE u.is_hidden = FALSE ORDER BY u.id")
}

func (r *UserRepository) SearchDirectory(ctx context.Context, query string, excludeID int64,
	companyID *int64) ([]*domain.User, error) {

	where := []string{"u.is_hidden = FALSE"}
	var args []any
	if excludeID > 0 {
		args = append(args, excludeID)
		where = append(where, fmt.Sprintf("u.id <> $%d", len(args)))
	}
	if companyID != nil {
		args = append(args, *companyID)
		where = append(where, fmt.Sprintf("u.company_id = $%d", len(args)))
	}
	if q := strings.TrimSpace(query); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf("(lower(u.fio) LIKE $%d OR lower(u.login) LIKE $%d)", len(args), len(args)))
	}
	return r.list(ctx, "WHERE "+strings.Join(where, " AND ")+" ORDER BY u.fio ASC", args...)
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO users (fio, login, hash_password, role_id, company_id, post,
		                   phone, email, is_default_pass, is_hidden, is_root_admin, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE, FALSE, now())
		RETURNING id, created_at`,
		u.FIO, u.Login, u.HashPassword, u.Role.ID, u.CompanyID, u.Post,
		u.Phone, u.Email, u.IsDefaultPass,
	).Scan(&u.ID, &u.CreatedAt)
}

// allowedUserFields — колонки, которые сервис может менять точечно.
var allowedUserFields = map[string]bool{
	"fio": true, "login": true, "post": true, "phone": true, "email": true,
	"company_id": true, "role_id": true, "avatar_path": true,
	"hash_password": true, "is_default_pass": true, "is_hidden": true,
}

func (r *UserRepository) UpdateFields(ctx context.Context, id int64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	// Детерминированный порядок — стабильный SQL для логов и тестов.
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if !allowedUserFields[k] {
			return fmt.Errorf("update users: недопустимое поле %q", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	set := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+1)
	for i, k := range keys {
		set = append(set, fmt.Sprintf("%s = $%d", k, i+1))
		args = append(args, fields[k])
	}
	args = append(args, id)
	_, err := r.pool.Exec(ctx,
		fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(set, ", "), len(args)),
		args...)
	return err
}

func (r *UserRepository) GetRole(ctx context.Context, roleID int64) (*domain.Role, error) {
	var role domain.Role
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, level FROM roles WHERE id = $1`, roleID,
	).Scan(&role.ID, &role.Name, &role.Level)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *UserRepository) ListRoles(ctx context.Context) ([]*domain.Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, level FROM roles ORDER BY level`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Level); err != nil {
			return nil, err
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

func (r *UserRepository) CountVisibleByLevel(ctx context.Context, level int) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.level = $1 AND u.is_hidden = FALSE`, level,
	).Scan(&n)
	return n, err
}

func (r *UserRepository) IsCompanyDirector(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM companies WHERE director_id = $1)`, userID,
	).Scan(&exists)
	return exists, err
}

func (r *UserRepository) HashPassword(ctx context.Context, password string) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx,
		`SELECT crypt($1, gen_salt('bf'))`, password,
	).Scan(&hash)
	return hash, err
}

func (r *UserRepository) VerifyPassword(ctx context.Context, password, hash string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT crypt($1, $2) = $2`, password, hash,
	).Scan(&ok)
	return ok, err
}

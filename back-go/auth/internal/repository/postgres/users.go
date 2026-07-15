// Package postgres — доступ к users/roles/companies через pgx.
// Хеширование/проверка паролей — pgcrypto (crypt + gen_salt('bf')),
// совместимо с историческими хешами.
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

// identityCols — чистая идентичность (без контекста компании): users развязаны
// с компаниями, роль/должность живут в связке user_companies.
const identityCols = `
	u.id, u.fio, u.login, u.hash_password,
	u.avatar_path, u.phone, u.email,
	u.is_default_pass, u.is_active, u.is_super_admin, u.email_verified,
	u.created_at, u.last_seen_at, u.status_emoji, u.status_text`

const identityFrom = ` FROM users u`

func scanIdentity(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID, &u.FIO, &u.Login, &u.HashPassword,
		&u.AvatarPath, &u.Phone, &u.Email,
		&u.IsDefaultPass, &u.IsActive, &u.IsSuperAdmin, &u.EmailVerified,
		&u.CreatedAt, &u.LastSeenAt, &u.StatusEmoji, &u.StatusText,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// memberCols — идентичность + контекст КОМПАНИИ (роль/должность/активность) из
// связки user_companies: для каталога членов конкретной компании.
const memberCols = `
	u.id, u.fio, u.login, u.hash_password,
	r.id, r.name, r.level, uc.post,
	uc.company_id, c.is_active,
	u.avatar_path, u.phone, u.email,
	u.is_default_pass, u.is_active, u.is_super_admin, u.email_verified,
	u.created_at, u.last_seen_at, u.status_emoji, u.status_text`

const memberFrom = `
	FROM user_companies uc
	JOIN users u ON u.id = uc.user_id
	JOIN roles r ON r.id = uc.role_id
	JOIN companies c ON c.id = uc.company_id`

func scanMember(row pgx.Row) (*domain.User, error) {
	var (
		u       domain.User
		cActive *bool
	)
	err := row.Scan(
		&u.ID, &u.FIO, &u.Login, &u.HashPassword,
		&u.Role.ID, &u.Role.Name, &u.Role.Level, &u.Post,
		&u.CompanyID, &cActive,
		&u.AvatarPath, &u.Phone, &u.Email,
		&u.IsDefaultPass, &u.IsActive, &u.IsSuperAdmin, &u.EmailVerified,
		&u.CreatedAt, &u.LastSeenAt, &u.StatusEmoji, &u.StatusText,
	)
	if err != nil {
		return nil, err
	}
	u.CompanyActive = cActive == nil || *cActive
	return &u, nil
}

func (r *UserRepository) getOne(ctx context.Context, where string, arg any) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, "SELECT"+identityCols+identityFrom+" WHERE "+where, arg)
	u, err := scanIdentity(row)
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

func (r *UserRepository) listIdentity(ctx context.Context, tail string, args ...any) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, "SELECT"+identityCols+identityFrom+" "+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.User
	for rows.Next() {
		u, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ListAll — все активные пользователи платформы (список супер-админа), по id.
func (r *UserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	return r.listIdentity(ctx, "WHERE u.is_active ORDER BY u.id")
}

// SearchDirectory — глобальный каталог (контакты): активные пользователи,
// ILIKE по fio/login, без excludeID; сортировка по fio.
func (r *UserRepository) SearchDirectory(ctx context.Context, query string, excludeID int64, loginOnly bool) ([]*domain.User, error) {
	where := []string{"u.is_active"}
	var args []any
	if excludeID > 0 {
		args = append(args, excludeID)
		where = append(where, fmt.Sprintf("u.id <> $%d", len(args)))
	}
	if q := strings.TrimSpace(query); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		if loginOnly {
			where = append(where, fmt.Sprintf("lower(u.login) LIKE $%d", len(args)))
		} else {
			where = append(where, fmt.Sprintf("(lower(u.fio) LIKE $%d OR lower(u.login) LIKE $%d)", len(args), len(args)))
		}
	}
	return r.listIdentity(ctx, "WHERE "+strings.Join(where, " AND ")+" ORDER BY u.fio ASC", args...)
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO users (fio, login, hash_password, phone, email,
		                   is_default_pass, is_active, email_verified, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7, now())
		RETURNING id, created_at`,
		u.FIO, u.Login, u.HashPassword, u.Phone, u.Email, u.IsDefaultPass, u.EmailVerified,
	).Scan(&u.ID, &u.CreatedAt)
}

// allowedUserFields — колонки идентичности, которые сервис меняет точечно.
var allowedUserFields = map[string]bool{
	"fio": true, "login": true, "phone": true, "email": true,
	"avatar_path": true, "hash_password": true,
	"is_default_pass": true, "is_active": true, "email_verified": true,
	"status_emoji": true, "status_text": true,
}

func (r *UserRepository) UpdateFields(ctx context.Context, id int64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
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

func (r *UserRepository) RoleByLevel(ctx context.Context, level int) (*domain.Role, error) {
	var role domain.Role
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, level FROM roles WHERE level = $1`, level,
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

// ── Членство в компаниях (user_companies) ──

func scanMembership(row pgx.Row) (*domain.Membership, error) {
	var (
		m       domain.Membership
		cActive *bool
		cName   *string
	)
	c := domain.CompanyRef{}
	err := row.Scan(&m.CompanyID, &cName, &cActive, &c.Settings,
		&m.Role.ID, &m.Role.Name, &m.Role.Level, &m.Post, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.ID = m.CompanyID
	if cName != nil {
		c.Name = *cName
	}
	c.IsActive = cActive != nil && *cActive
	m.Company = &c
	return &m, nil
}

const membershipSelect = `
	SELECT uc.company_id, c.name, c.is_active, c.settings,
	       r.id, r.name, r.level, uc.post, uc.created_at
	  FROM user_companies uc
	  JOIN companies c ON c.id = uc.company_id
	  JOIN roles r ON r.id = uc.role_id`

func (r *UserRepository) ListMemberships(ctx context.Context, userID int64) ([]domain.Membership, error) {
	rows, err := r.pool.Query(ctx, membershipSelect+
		` WHERE uc.user_id = $1 ORDER BY uc.created_at ASC, uc.company_id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Membership
	for rows.Next() {
		m, err := scanMembership(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *UserRepository) GetMembership(ctx context.Context, userID, companyID int64) (*domain.Membership, error) {
	m, err := scanMembership(r.pool.QueryRow(ctx, membershipSelect+
		` WHERE uc.user_id = $1 AND uc.company_id = $2`, userID, companyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *UserRepository) AddMembership(ctx context.Context, userID, companyID, roleID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_companies (user_id, company_id, role_id)
		VALUES ($1, $2, $3) ON CONFLICT (user_id, company_id) DO NOTHING`,
		userID, companyID, roleID)
	return err
}

func (r *UserRepository) RemoveMembership(ctx context.Context, userID, companyID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_companies WHERE user_id = $1 AND company_id = $2`, userID, companyID)
	return err
}

func (r *UserRepository) UpdateMembershipRole(ctx context.Context, userID, companyID, roleID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE user_companies SET role_id = $3 WHERE user_id = $1 AND company_id = $2`,
		userID, companyID, roleID)
	return err
}

func (r *UserRepository) SetMembershipPost(ctx context.Context, userID, companyID int64, post *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE user_companies SET post = $3 WHERE user_id = $1 AND company_id = $2`,
		userID, companyID, post)
	return err
}

// CountCompanyMembersByLevel — активные члены компании с уровнем роли
// (защита «последнего администратора компании»).
func (r *UserRepository) CountCompanyMembersByLevel(ctx context.Context, companyID int64, level int) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM user_companies uc
		JOIN users u ON u.id = uc.user_id
		JOIN roles r ON r.id = uc.role_id
		WHERE uc.company_id = $1 AND r.level = $2 AND u.is_active`, companyID, level,
	).Scan(&n)
	return n, err
}

func (r *UserRepository) SearchDirectoryMembers(ctx context.Context, query string, excludeID, companyID int64) ([]*domain.User, error) {
	args := []any{companyID}
	where := []string{"uc.company_id = $1", "u.is_active"}
	if excludeID > 0 {
		args = append(args, excludeID)
		where = append(where, fmt.Sprintf("u.id <> $%d", len(args)))
	}
	if q := strings.TrimSpace(query); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf("(lower(u.fio) LIKE $%d OR lower(u.login) LIKE $%d)", len(args), len(args)))
	}
	rows, err := r.pool.Query(ctx,
		"SELECT"+memberCols+memberFrom+" WHERE "+strings.Join(where, " AND ")+" ORDER BY u.fio ASC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.User
	for rows.Next() {
		u, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepository) SearchNonMembers(ctx context.Context, query string, companyID int64) ([]*domain.User, error) {
	args := []any{companyID}
	where := []string{
		"u.is_active",
		"u.is_super_admin = FALSE",
		"NOT EXISTS (SELECT 1 FROM user_companies uc WHERE uc.user_id = u.id AND uc.company_id = $1)",
	}
	if q := strings.TrimSpace(query); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf("(lower(u.fio) LIKE $%d OR lower(u.login) LIKE $%d)", len(args), len(args)))
	}
	return r.listIdentity(ctx, "WHERE "+strings.Join(where, " AND ")+" ORDER BY u.fio ASC LIMIT 20", args...)
}

func (r *UserRepository) CompanyActive(ctx context.Context, companyID *int64) (bool, error) {
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

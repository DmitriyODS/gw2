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

// CompanyRepository — доступ к companies через pgx (порт company_repo.py).
type CompanyRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

var _ domain.CompanyRepository = (*CompanyRepository)(nil)

const companyColumns = `
	c.id, c.name, c.description, c.created_by, c.is_active, c.settings, c.created_at, c.invite_code,
	d.id, d.fio, d.login, d.avatar_path`

const companyFrom = `
	FROM companies c
	LEFT JOIN users d ON d.id = c.created_by`

func scanCompany(row pgx.Row) (*domain.Company, error) {
	var (
		c       domain.Company
		dID     *int64
		dFIO    *string
		dLogin  *string
		dAvatar *string
	)
	err := row.Scan(
		&c.ID, &c.Name, &c.Description, &c.CreatedBy, &c.IsActive, &c.Settings, &c.CreatedAt, &c.InviteCode,
		&dID, &dFIO, &dLogin, &dAvatar,
	)
	if err != nil {
		return nil, err
	}
	if dID != nil {
		c.Creator = &domain.CompanyCreator{ID: *dID, AvatarPath: dAvatar}
		if dFIO != nil {
			c.Creator.FIO = *dFIO
		}
		if dLogin != nil {
			c.Creator.Login = *dLogin
		}
	}
	return &c, nil
}

func (r *CompanyRepository) getOne(ctx context.Context, where string, arg any) (*domain.Company, error) {
	row := r.pool.QueryRow(ctx, "SELECT"+companyColumns+companyFrom+" WHERE "+where, arg)
	c, err := scanCompany(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *CompanyRepository) GetCompany(ctx context.Context, id int64) (*domain.Company, error) {
	return r.getOne(ctx, "c.id = $1", id)
}

func (r *CompanyRepository) GetCompanyByName(ctx context.Context, name string) (*domain.Company, error) {
	return r.getOne(ctx, "c.name = $1", name)
}

func (r *CompanyRepository) GetCompanyByInviteCode(ctx context.Context, code string) (*domain.Company, error) {
	return r.getOne(ctx, "c.invite_code = $1", code)
}

func (r *CompanyRepository) ListCompanies(ctx context.Context) ([]*domain.Company, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT"+companyColumns+companyFrom+" ORDER BY c.created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Company
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListCompaniesWhereAdmin — компании, в которых пользователь состоит членом с
// ролью администратора (раздел «Компании» обычного пользователя).
func (r *CompanyRepository) ListCompaniesWhereAdmin(ctx context.Context, userID int64) ([]*domain.Company, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT"+companyColumns+companyFrom+`
		  JOIN user_companies uc ON uc.company_id = c.id
		  JOIN roles r2 ON r2.id = uc.role_id
		 WHERE uc.user_id = $1 AND r2.level >= $2
		 ORDER BY c.created_at DESC`, userID, domain.LevelAdmin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Company
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CompanyRepository) CreateCompany(ctx context.Context, c *domain.Company) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO companies (name, description, created_by, settings, created_at)
		VALUES ($1, $2, $3, $4, now())
		RETURNING id, is_active, created_at`,
		c.Name, c.Description, c.CreatedBy, c.Settings,
	).Scan(&c.ID, &c.IsActive, &c.CreatedAt)
}

// allowedCompanyFields — колонки, которые сервис может менять точечно.
var allowedCompanyFields = map[string]bool{
	"name": true, "description": true, "created_by": true,
	"is_active": true, "settings": true, "invite_code": true,
}

func (r *CompanyRepository) UpdateCompanyFields(ctx context.Context, id int64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if !allowedCompanyFields[k] {
			return fmt.Errorf("update companies: недопустимое поле %q", k)
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
		fmt.Sprintf("UPDATE companies SET %s WHERE id = $%d", strings.Join(set, ", "), len(args)),
		args...)
	return err
}

func (r *CompanyRepository) DeleteCompany(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM companies WHERE id = $1`, id)
	return err
}

func (r *CompanyRepository) CompanyStats(ctx context.Context, ids []int64) (map[int64]domain.CompanyStats, error) {
	out := make(map[int64]domain.CompanyStats, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	for _, id := range ids {
		out[id] = domain.CompanyStats{}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT uc.company_id, COUNT(*)
		  FROM user_companies uc
		  JOIN users u ON u.id = uc.user_id
		 WHERE uc.company_id = ANY($1) AND u.is_active
		 GROUP BY uc.company_id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		s := out[id]
		s.Employees = n
		out[id] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	taskRows, err := r.pool.Query(ctx, `
		SELECT company_id, COUNT(id)
		  FROM tasks
		 WHERE company_id = ANY($1)
		 GROUP BY company_id`, ids)
	if err != nil {
		return nil, err
	}
	defer taskRows.Close()
	for taskRows.Next() {
		var id int64
		var n int
		if err := taskRows.Scan(&id, &n); err != nil {
			return nil, err
		}
		s := out[id]
		s.Tasks = n
		out[id] = s
	}
	return out, taskRows.Err()
}

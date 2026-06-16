package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// CompanyInviteStore — email-приглашения в компанию (company_invites).
type CompanyInviteStore struct {
	pool *pgxpool.Pool
}

func NewCompanyInviteStore(pool *pgxpool.Pool) *CompanyInviteStore {
	return &CompanyInviteStore{pool: pool}
}

var _ domain.CompanyInviteStore = (*CompanyInviteStore)(nil)

func (r *CompanyInviteStore) Upsert(ctx context.Context, companyID int64, email string, roleID int64, token string, invitedBy *int64, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO company_invites (company_id, email, role_id, token, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (company_id, email) DO UPDATE
		   SET role_id = EXCLUDED.role_id, token = EXCLUDED.token,
		       invited_by = EXCLUDED.invited_by, expires_at = EXCLUDED.expires_at, created_at = now()`,
		companyID, email, roleID, token, invitedBy, expiresAt)
	return err
}

func (r *CompanyInviteStore) GetByToken(ctx context.Context, token string) (*domain.CompanyInvite, error) {
	var ci domain.CompanyInvite
	err := r.pool.QueryRow(ctx, `
		SELECT ci.id, ci.company_id, ci.email, ci.role_id, ci.token, ci.invited_by, ci.expires_at,
		       c.name, r.name, r.level
		  FROM company_invites ci
		  JOIN companies c ON c.id = ci.company_id
		  JOIN roles r ON r.id = ci.role_id
		 WHERE ci.token = $1`, token,
	).Scan(&ci.ID, &ci.CompanyID, &ci.Email, &ci.RoleID, &ci.Token, &ci.InvitedBy, &ci.ExpiresAt,
		&ci.CompanyName, &ci.RoleName, &ci.RoleLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ci, nil
}

func (r *CompanyInviteStore) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM company_invites WHERE id = $1`, id)
	return err
}

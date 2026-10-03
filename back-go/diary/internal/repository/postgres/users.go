package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/spaces"
)

// UserReader — read-only доступ к идентичности пользователей (владелец таблицы
// в рантайме — authsvc): объём — auth-мидлварь и адресный шаринг.
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

// TeamRole — положение человека в команде.
func (r *UserReader) TeamRole(ctx context.Context, userID, companyID int64) (spaces.Role, error) {
	return spaces.RoleIn(ctx, r.pool, userID, companyID)
}

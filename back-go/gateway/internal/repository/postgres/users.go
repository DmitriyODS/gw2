// Package postgres — read-only доступ gateway к пользователям платформы
// (auth-мидлварь REST-роутов) и запись users.last_seen_at для presence.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type UserReader struct {
	pool *pgxpool.Pool
}

func NewUserReader(pool *pgxpool.Pool) *UserReader {
	return &UserReader{pool: pool}
}

// AuthInfo — сверка пользователя для pkg-мидлвари. Уровень роли и активная
// компания — ИЗ ТОКЕНА (active); из БД — глобальная активность аккаунта,
// признак супер-админа и активность выбранной компании (active.CompanyID).
func (r *UserReader) AuthInfo(ctx context.Context, userID int64, active pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
	var (
		info          pasetoauth.AuthInfo
		companyActive *bool
	)
	err := r.pool.QueryRow(ctx, `
		SELECT u.is_active, u.is_super_admin, c.is_active
		  FROM users u
		  LEFT JOIN companies c ON c.id = $2
		 WHERE u.id = $1`, userID, active.CompanyID).
		Scan(&info.IsActive, &info.IsSuperAdmin, &companyActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	info.RoleLevel = active.RoleLevel
	// Нет активной компании — нормальное состояние, считается активным.
	info.CompanyActive = companyActive == nil || *companyActive
	return &info, nil
}

// PresenceAudience — круг, которому виден онлайн-статус пользователя:
// коллеги по общим компаниям, собеседники личных диалогов, участники общих
// групп и супер-админы. Супер-админу самому виден весь онлайн (all).
func (r *UserReader) PresenceAudience(ctx context.Context, userID int64) ([]int64, bool, error) {
	var superAdmin bool
	if err := r.pool.QueryRow(ctx,
		`SELECT is_super_admin FROM users WHERE id = $1`, userID).Scan(&superAdmin); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT uc2.user_id
		  FROM user_companies uc1
		  JOIN user_companies uc2 ON uc2.company_id = uc1.company_id
		 WHERE uc1.user_id = $1
		UNION
		SELECT CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
		  FROM conversations
		 WHERE (user_a_id = $1 OR user_b_id = $1) AND user_b_id IS NOT NULL
		UNION
		SELECT cm2.user_id
		  FROM conversation_members cm1
		  JOIN conversation_members cm2 ON cm2.conversation_id = cm1.conversation_id
		 WHERE cm1.user_id = $1
		UNION
		SELECT id FROM users WHERE is_super_admin`, userID)
	if err != nil {
		return nil, false, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	return ids, superAdmin, err
}

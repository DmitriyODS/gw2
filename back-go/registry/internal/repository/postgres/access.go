package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// ignoreNoRows — «строки нет» здесь не ошибка: несуществующий реестр означает
// отсутствие доступа, а не сбой.
func ignoreNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

/* Эффективный доступ и адресные шары.

   Уровень человека к реестру считает СЕРВЕР одним запросом: клиенту нельзя
   доверять решение, что ему показывать. Выражение общее со списком реестров
   (accessExpr в registries.go) — правило одно на всех. */

// AccessOf — эффективный уровень человека к реестру ("" — доступа нет).
func (r *Repo) AccessOf(ctx context.Context, registryID, userID int64, companyIDs []int64) (string, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	var access string
	err := r.pool.QueryRow(ctx,
		`SELECT `+accessExpr+` FROM registries reg WHERE reg.id = $3`,
		userID, companyIDs, registryID).Scan(&access)
	if err != nil {
		return domain.AccessNone, ignoreNoRows(err)
	}
	return access, nil
}

// Audience — кому адресовать сокет-события реестра: владелец, адресаты личных
// шар и участники компаний, которым реестр раздан. Событие уходит поимённо
// (комнаты user_{id}), а не в общую комнату: реестр больше не принадлежит
// компании, и «всем» его показывать нельзя.
func (r *Repo) Audience(ctx context.Context, registryID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT owner_id FROM registries WHERE id = $1
		UNION
		SELECT sh.user_id FROM registry_user_shares sh
		 WHERE sh.registry_id = $1 AND sh.user_id IS NOT NULL
		UNION
		SELECT uc.user_id FROM registry_user_shares sh
		  JOIN user_companies uc ON uc.company_id = sh.company_id
		 WHERE sh.registry_id = $1 AND sh.company_id IS NOT NULL`, registryID)
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

const userShareCols = `sh.id, sh.registry_id, sh.user_id, sh.company_id, sh.access, sh.created_at`

// ListUserShares — кому выдан адресный доступ: люди и компании одним списком с
// именами (карточка «Поделиться» показывает их сразу, без второго запроса).
func (r *Repo) ListUserShares(ctx context.Context, registryID int64) ([]*domain.UserShare, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+userShareCols+`,
		       COALESCE(u.fio, c.name, ''), u.avatar_path, COALESCE(u.login, '')
		  FROM registry_user_shares sh
		  LEFT JOIN users u ON u.id = sh.user_id
		  LEFT JOIN companies c ON c.id = sh.company_id
		 WHERE sh.registry_id = $1
		 ORDER BY sh.company_id NULLS LAST, sh.created_at`, registryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserShare{}
	for rows.Next() {
		var s domain.UserShare
		if err := rows.Scan(&s.ID, &s.RegistryID, &s.UserID, &s.CompanyID, &s.Access,
			&s.CreatedAt, &s.Name, &s.AvatarPath, &s.Login); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

/* PutUserShare — выдать доступ или сменить уровень уже выданного. Повторная
   выдача не плодит строки: адресат один — строка одна (частичные уникальные
   индексы миграции 00081).

   created_by — тот, КТО выдал; подставлять туда адресата нельзя: у компанийной
   шары это её id, а колонка ссылается на пользователей. */
func (r *Repo) PutUserShare(ctx context.Context, s *domain.UserShare) error {
	if s.UserID != nil {
		return r.pool.QueryRow(ctx, `
			INSERT INTO registry_user_shares (registry_id, user_id, access, created_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (registry_id, user_id) WHERE user_id IS NOT NULL
			DO UPDATE SET access = EXCLUDED.access
			RETURNING id, created_at`,
			s.RegistryID, *s.UserID, s.Access, s.CreatedBy).Scan(&s.ID, &s.CreatedAt)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO registry_user_shares (registry_id, company_id, access, created_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (registry_id, company_id) WHERE company_id IS NOT NULL
		DO UPDATE SET access = EXCLUDED.access
		RETURNING id, created_at`,
		s.RegistryID, *s.CompanyID, s.Access, s.CreatedBy).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) DeleteUserShare(ctx context.Context, registryID int64, userID, companyID *int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM registry_user_shares
		 WHERE registry_id = $1
		   AND ($2::bigint IS NULL OR user_id = $2)
		   AND ($3::bigint IS NULL OR company_id = $3)
		   AND (($2::bigint IS NOT NULL AND user_id IS NOT NULL)
		     OR ($3::bigint IS NOT NULL AND company_id IS NOT NULL))`,
		registryID, userID, companyID)
	return err
}

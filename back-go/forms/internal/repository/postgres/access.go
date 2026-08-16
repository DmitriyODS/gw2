package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

// ignoreNoRows — «строки нет» здесь не ошибка: несуществующая форма означает
// отсутствие доступа, а не сбой.
func ignoreNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

/* Эффективный доступ, адресные шары и контроль исполнения.

   Уровень человека к форме считает СЕРВЕР одним запросом: клиенту нельзя
   доверять решение, что ему показывать. Выражение общее со списком форм
   (accessExpr в forms.go) — правило одно на всех. */

// AccessOf — эффективный уровень человека к форме ("" — доступа нет).
func (r *Repo) AccessOf(ctx context.Context, formID, userID int64, companyIDs []int64) (string, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	var access string
	err := r.pool.QueryRow(ctx,
		`SELECT `+accessExpr+` FROM forms f WHERE f.id = $3`,
		userID, companyIDs, formID).Scan(&access)
	if err != nil {
		return domain.AccessNone, ignoreNoRows(err)
	}
	return access, nil
}

// Audience — кому адресовать сокет-события формы: владелец, адресаты личных шар
// и участники компаний, которым форма роздана. Событие уходит поимённо (комнаты
// user_{id}), а не в общую комнату: форма не принадлежит компании.
func (r *Repo) Audience(ctx context.Context, formID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT owner_id FROM forms WHERE id = $1
		UNION
		SELECT sh.user_id FROM form_user_shares sh
		 WHERE sh.form_id = $1 AND sh.user_id IS NOT NULL
		UNION
		SELECT uc.user_id FROM form_user_shares sh
		  JOIN user_companies uc ON uc.company_id = sh.company_id
		 WHERE sh.form_id = $1 AND sh.company_id IS NOT NULL`, formID)
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

const userShareCols = `sh.id, sh.form_id, sh.user_id, sh.company_id, sh.access,
	sh.due_at, sh.created_by, sh.created_at`

// ListUserShares — кому выдан доступ: люди и компании одним списком с именами
// (карточка «Поделиться» показывает их сразу, без второго запроса).
func (r *Repo) ListUserShares(ctx context.Context, formID int64) ([]*domain.UserShare, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+userShareCols+`,
		       COALESCE(u.fio, c.name, ''), u.avatar_path, COALESCE(u.login, '')
		  FROM form_user_shares sh
		  LEFT JOIN users u ON u.id = sh.user_id
		  LEFT JOIN companies c ON c.id = sh.company_id
		 WHERE sh.form_id = $1
		 ORDER BY sh.company_id NULLS LAST, sh.created_at`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserShare{}
	for rows.Next() {
		var s domain.UserShare
		if err := rows.Scan(&s.ID, &s.FormID, &s.UserID, &s.CompanyID, &s.Access,
			&s.DueAt, &s.CreatedBy, &s.CreatedAt, &s.Name, &s.AvatarPath, &s.Login); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

/* PutUserShare — выдать доступ или сменить уровень уже выданного. Повторная
   выдача не плодит строки: адресат один — строка одна (частичные уникальные
   индексы миграции 00085). Новый срок сбрасывает отметку о напоминании: срок
   передвинули — напомнить нужно заново.

   created_by — тот, КТО выдал; подставлять туда адресата нельзя: у компанийной
   шары это её id, а колонка ссылается на пользователей. */
func (r *Repo) PutUserShare(ctx context.Context, s *domain.UserShare) error {
	if s.UserID != nil {
		return r.pool.QueryRow(ctx, `
			INSERT INTO form_user_shares (form_id, user_id, access, due_at, created_by)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (form_id, user_id) WHERE user_id IS NOT NULL
			DO UPDATE SET access = EXCLUDED.access, due_at = EXCLUDED.due_at,
			              reminded_at = NULL
			RETURNING id, created_at`,
			s.FormID, *s.UserID, s.Access, s.DueAt, s.CreatedBy).Scan(&s.ID, &s.CreatedAt)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO form_user_shares (form_id, company_id, access, due_at, created_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (form_id, company_id) WHERE company_id IS NOT NULL
		DO UPDATE SET access = EXCLUDED.access, due_at = EXCLUDED.due_at,
		              reminded_at = NULL
		RETURNING id, created_at`,
		s.FormID, *s.CompanyID, s.Access, s.DueAt, s.CreatedBy).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) DeleteUserShare(ctx context.Context, formID int64, userID, companyID *int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM form_user_shares
		 WHERE form_id = $1
		   AND ($2::bigint IS NULL OR user_id = $2)
		   AND ($3::bigint IS NULL OR company_id = $3)
		   AND (($2::bigint IS NOT NULL AND user_id IS NOT NULL)
		     OR ($3::bigint IS NOT NULL AND company_id IS NOT NULL))`,
		formID, userID, companyID)
	return err
}

/*
Assignees — контроль исполнения: назначенные поимённо и отметка, кто ответил.

	Назначение приходит и лично, и через компанию, поэтому адресаты сначала
	разворачиваются в один список, а потом схлопываются по человеку: срок берём
	ближайший, а источник называем «Лично», если личное назначение есть — оно
	конкретнее компанийного.
*/
func (r *Repo) Assignees(ctx context.Context, formID int64) ([]*domain.Assignee, error) {
	rows, err := r.pool.Query(ctx, `
		WITH targets AS (
		    SELECT sh.user_id, sh.due_at, TRUE AS personal, '' AS via
		      FROM form_user_shares sh
		     WHERE sh.form_id = $1 AND sh.access = 'respond' AND sh.user_id IS NOT NULL
		    UNION ALL
		    SELECT uc.user_id, sh.due_at, FALSE, c.name
		      FROM form_user_shares sh
		      JOIN user_companies uc ON uc.company_id = sh.company_id
		      JOIN companies c ON c.id = sh.company_id
		     WHERE sh.form_id = $1 AND sh.access = 'respond' AND sh.company_id IS NOT NULL
		)
		SELECT t.user_id, COALESCE(u.fio, ''), u.avatar_path,
		       CASE WHEN bool_or(t.personal) THEN 'Лично' ELSE min(t.via) END,
		       min(t.due_at),
		       (SELECT min(fr.created_at) FROM form_responses fr
		         WHERE fr.form_id = $1 AND fr.user_id = t.user_id)
		  FROM targets t
		  JOIN users u ON u.id = t.user_id
		 GROUP BY t.user_id, u.fio, u.avatar_path
		 ORDER BY u.fio`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Assignee{}
	for rows.Next() {
		var a domain.Assignee
		if err := rows.Scan(&a.UserID, &a.Name, &a.AvatarPath, &a.Via, &a.DueAt, &a.AnsweredAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

/*
ClaimDueReminders — забрать наступившие сроки ответа.

	Строка помечается напомненной ТЕМ ЖЕ запросом (FOR UPDATE SKIP LOCKED в
	подзапросе), поэтому при нескольких инстансах сервиса напоминание уходит
	ровно один раз. Напоминаем за сутки до срока — или сразу, если срок уже
	ближе; тем, кто уже ответил, не напоминаем вовсе.
*/
func (r *Repo) ClaimDueReminders(ctx context.Context, now time.Time, limit int) ([]domain.DueReminder, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE form_user_shares sh
		   SET reminded_at = now()
		 WHERE sh.id IN (
		     SELECT id FROM form_user_shares
		      WHERE access = 'respond' AND due_at IS NOT NULL AND reminded_at IS NULL
		        AND due_at <= $1 + interval '24 hours'
		      ORDER BY due_at
		      FOR UPDATE SKIP LOCKED
		      LIMIT $2)
		RETURNING sh.id, sh.form_id, sh.user_id, sh.company_id, sh.due_at`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type claim struct {
		shareID   int64
		formID    int64
		userID    *int64
		companyID *int64
		dueAt     time.Time
	}
	claims := []claim{}
	for rows.Next() {
		var c claim
		if err := rows.Scan(&c.shareID, &c.formID, &c.userID, &c.companyID, &c.dueAt); err != nil {
			return nil, err
		}
		claims = append(claims, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]domain.DueReminder, 0, len(claims))
	for _, c := range claims {
		item := domain.DueReminder{ShareID: c.shareID, FormID: c.formID, DueAt: c.dueAt}
		if err := r.pool.QueryRow(ctx,
			`SELECT title, owner_id FROM forms WHERE id = $1`, c.formID).
			Scan(&item.FormTitle, &item.OwnerID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		users, err := r.pendingRespondents(ctx, c.formID, c.userID, c.companyID)
		if err != nil {
			return nil, err
		}
		if len(users) == 0 {
			continue
		}
		item.UserIDs = users
		out = append(out, item)
	}
	return out, nil
}

// pendingRespondents — назначенные этой шарой, которые ещё не ответили.
func (r *Repo) pendingRespondents(ctx context.Context, formID int64, userID, companyID *int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id
		  FROM users u
		 WHERE u.is_active
		   AND ($2::bigint IS NULL OR u.id = $2)
		   AND ($3::bigint IS NULL OR EXISTS (
		         SELECT 1 FROM user_companies uc
		          WHERE uc.user_id = u.id AND uc.company_id = $3))
		   AND ($2::bigint IS NOT NULL OR $3::bigint IS NOT NULL)
		   AND NOT EXISTS (
		         SELECT 1 FROM form_responses fr
		          WHERE fr.form_id = $1 AND fr.user_id = u.id)`, formID, userID, companyID)
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

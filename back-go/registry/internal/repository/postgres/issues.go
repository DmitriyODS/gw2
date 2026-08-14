package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// uniqueViolation — код Postgres «нарушен уникальный индекс».
const uniqueViolation = "23505"

/* Учётный реестр: выдачи позиций и история их движения.

   Открытая выдача у записи одна — это держит частичный уникальный индекс
   (registry_issues_open_idx). Поэтому «выдать уже выданное» отбивается базой, а
   не проверкой-в-две-операции, между которыми успевает вклиниться сосед. */

const issueCols = `i.id, i.registry_id, i.record_id, i.issued_to, i.holder_name,
	i.holder_phone, i.holder_user_id, i.issued_by, i.issued_at, i.due_at, i.returned_at`

func scanIssue(row pgx.Row) (*domain.Issue, error) {
	var i domain.Issue
	err := row.Scan(&i.ID, &i.RegistryID, &i.RecordID, &i.IssuedTo, &i.HolderName,
		&i.HolderPhone, &i.HolderUserID, &i.IssuedBy, &i.IssuedAt, &i.DueAt,
		&i.ReturnedAt, &i.IssuedByName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// OpenIssues — открытые выдачи пачкой записей: плашки «выдано до / просрочено»
// нужны всей странице таблицы сразу, поэтому один запрос вместо N.
func (r *Repo) OpenIssues(ctx context.Context, recordIDs []int64) (map[int64]*domain.Issue, error) {
	out := map[int64]*domain.Issue{}
	if len(recordIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+issueCols+`, COALESCE(u.fio, '')
		  FROM registry_issues i
		  LEFT JOIN users u ON u.id = i.issued_by
		 WHERE i.record_id = ANY($1) AND i.returned_at IS NULL`, recordIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		out[issue.RecordID] = issue
	}
	return out, rows.Err()
}

// IssueHistory — все выдачи записи со своими событиями, свежие первыми.
func (r *Repo) IssueHistory(ctx context.Context, recordID int64) ([]*domain.Issue, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+issueCols+`, COALESCE(u.fio, '')
		  FROM registry_issues i
		  LEFT JOIN users u ON u.id = i.issued_by
		 WHERE i.record_id = $1
		 ORDER BY i.issued_at DESC, i.id DESC`, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues := []*domain.Issue{}
	ids := []int64{}
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
		ids = append(ids, issue.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return issues, nil
	}

	events, err := r.issueEvents(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		issue.Events = events[issue.ID]
	}
	return issues, nil
}

func (r *Repo) issueEvents(ctx context.Context, issueIDs []int64) (map[int64][]domain.IssueEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.issue_id, e.kind, e.due_at, e.comment, e.actor_id,
		       COALESCE(u.fio, ''), e.created_at
		  FROM registry_issue_events e
		  LEFT JOIN users u ON u.id = e.actor_id
		 WHERE e.issue_id = ANY($1)
		 ORDER BY e.created_at, e.id`, issueIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]domain.IssueEvent{}
	for rows.Next() {
		var e domain.IssueEvent
		if err := rows.Scan(&e.ID, &e.IssueID, &e.Kind, &e.DueAt, &e.Comment,
			&e.ActorID, &e.ActorName, &e.CreatedAt); err != nil {
			return nil, err
		}
		out[e.IssueID] = append(out[e.IssueID], e)
	}
	return out, rows.Err()
}

// CreateIssue — выдать позицию вместе с первым событием истории: выдача без
// записи о ней в журнале — дыра в отчётности, поэтому обе строки в транзакции.
func (r *Repo) CreateIssue(ctx context.Context, i *domain.Issue, comment string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := tx.QueryRow(ctx, `
		INSERT INTO registry_issues
			(registry_id, record_id, issued_to, holder_name, holder_phone,
			 holder_user_id, issued_by, issued_at, due_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,now(),$8)
		RETURNING id, issued_at`,
		i.RegistryID, i.RecordID, i.IssuedTo, i.HolderName, i.HolderPhone,
		i.HolderUserID, i.IssuedBy, i.DueAt).Scan(&i.ID, &i.IssuedAt); err != nil {
		// Открытая выдача у записи одна (частичный уникальный индекс). Гонку
		// двух вкладок ловит база, а не проверка-перед-вставкой, между которой
		// и вставкой успевает вклиниться сосед.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrAlreadyIssued
		}
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO registry_issue_events (issue_id, kind, due_at, comment, actor_id)
		VALUES ($1, 'issue', $2, $3, $4)`,
		i.ID, i.DueAt, comment, i.IssuedBy); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ExtendIssue — сдвинуть срок возврата открытой выдачи.
func (r *Repo) ExtendIssue(ctx context.Context, issueID int64, dueAt *time.Time, comment string, actorID *int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`UPDATE registry_issues SET due_at = $2 WHERE id = $1 AND returned_at IS NULL`,
		issueID, dueAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO registry_issue_events (issue_id, kind, due_at, comment, actor_id)
		VALUES ($1, 'extend', $2, $3, $4)`, issueID, dueAt, comment, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReturnIssue — закрыть выдачу. Условие returned_at IS NULL в самом UPDATE:
// две вкладки, нажавшие «Вернуть» одновременно, не должны дать два возврата с
// разным временем. ok=false — выдачу уже закрыли.
func (r *Repo) ReturnIssue(ctx context.Context, issueID int64, at time.Time, comment string, actorID *int64) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx,
		`UPDATE registry_issues SET returned_at = $2 WHERE id = $1 AND returned_at IS NULL`,
		issueID, at)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO registry_issue_events (issue_id, kind, comment, actor_id)
		VALUES ($1, 'return', $2, $3)`, issueID, comment, actorID); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

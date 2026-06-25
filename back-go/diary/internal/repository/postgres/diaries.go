package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.DiaryRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const diaryCols = `id, owner_id, name, position, created_at, updated_at`

func scanDiary(row pgx.Row) (*domain.Diary, error) {
	var d domain.Diary
	err := row.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Position, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repo) ListOwned(ctx context.Context, ownerID int64) ([]*domain.Diary, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+diaryCols+` FROM diaries WHERE owner_id = $1 ORDER BY position, id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Diary{}
	for rows.Next() {
		d, err := scanDiary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) ListShared(ctx context.Context, userID int64) ([]*domain.Diary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.owner_id, d.name, d.position, d.created_at, d.updated_at,
		       u.fio, u.avatar_path
		  FROM diary_user_shares s
		  JOIN diaries d ON d.id = s.diary_id
		  JOIN users   u ON u.id = d.owner_id
		 WHERE s.user_id = $1
		 ORDER BY u.fio, d.name, d.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Diary{}
	for rows.Next() {
		var d domain.Diary
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Position, &d.CreatedAt, &d.UpdatedAt,
			&d.OwnerName, &d.OwnerAvatar); err != nil {
			return nil, err
		}
		d.Shared = true
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *Repo) GetDiary(ctx context.Context, id int64) (*domain.Diary, error) {
	return scanDiary(r.pool.QueryRow(ctx, `SELECT `+diaryCols+` FROM diaries WHERE id = $1`, id))
}

func (r *Repo) CreateDiary(ctx context.Context, d *domain.Diary) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO diaries (owner_id, name, position) VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		d.OwnerID, d.Name, d.Position).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repo) UpdateDiary(ctx context.Context, id int64, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE diaries SET name = $2, updated_at = now() WHERE id = $1`, id, name)
	return err
}

func (r *Repo) DeleteDiary(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM diaries WHERE id = $1`, id)
	return err
}

func (r *Repo) NextPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM diaries WHERE owner_id = $1`, ownerID).Scan(&pos)
	return pos, err
}

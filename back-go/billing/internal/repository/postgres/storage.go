package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

func (r *Repo) Usage(ctx context.Context, userID int64) ([]*domain.StorageEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT service, bytes FROM billing_storage_usage
		 WHERE user_id = $1 AND bytes > 0 ORDER BY bytes DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.StorageEntry{}
	for rows.Next() {
		var e domain.StorageEntry
		if err := rows.Scan(&e.Service, &e.Bytes); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *Repo) Total(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(sum(bytes), 0) FROM billing_storage_usage WHERE user_id = $1`, userID).
		Scan(&total)
	return total, err
}

func (r *Repo) TotalsFor(ctx context.Context, userIDs []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, COALESCE(sum(bytes), 0) FROM billing_storage_usage
		 WHERE user_id = ANY($1) GROUP BY user_id`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, bytes int64
		if err := rows.Scan(&id, &bytes); err != nil {
			return nil, err
		}
		out[id] = bytes
	}
	return out, rows.Err()
}

// Track — сдвинуть занятое место на дельту (удаление файла шлёт отрицательную).
// GREATEST(...,0) страхует от рассинхрона: у сервиса не должно получаться
// «минус сто мегабайт» из-за потерянного события заливки.
func (r *Repo) Track(ctx context.Context, userID int64, service string, delta int64) (int64, error) {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO billing_storage_usage (user_id, service, bytes) VALUES ($1, $2, GREATEST($3, 0))
		ON CONFLICT (user_id, service) DO UPDATE
		   SET bytes = GREATEST(billing_storage_usage.bytes + $3, 0), updated_at = now()`,
		userID, service, delta); err != nil {
		return 0, err
	}
	return r.Total(ctx, userID)
}

func (r *Repo) Set(ctx context.Context, userID int64, service string, bytes int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_storage_usage (user_id, service, bytes) VALUES ($1, $2, GREATEST($3, 0))
		ON CONFLICT (user_id, service) DO UPDATE SET bytes = GREATEST($3, 0), updated_at = now()`,
		userID, service, bytes)
	return err
}

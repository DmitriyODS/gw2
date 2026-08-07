package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

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

// ── Журнал файлов ────────────────────────────────────────────────────────────

// AddFiles — upsert по ключу: тем же путём идут и первая заливка, и уточнение
// имени/ссылки при сверке с владельцем. Пустыми значениями сверки уже
// известную ссылку не затираем — файл мог быть залит раньше своей сущности.
func (r *Repo) AddFiles(ctx context.Context, userID int64, files []*domain.StoredFile) error {
	if len(files) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, f := range files {
		var company any
		if f.CompanyID > 0 {
			company = f.CompanyID
		}
		batch.Queue(`
			INSERT INTO billing_storage_files
				(storage_key, user_id, service, company_id, file_name, size_bytes, ref_kind, ref_id, ref_title)
			VALUES ($1, $2, $3, $4, $5, GREATEST($6, 0), $7, $8, $9)
			ON CONFLICT (storage_key) DO UPDATE SET
				user_id    = EXCLUDED.user_id,
				service    = EXCLUDED.service,
				company_id = EXCLUDED.company_id,
				file_name  = COALESCE(NULLIF(EXCLUDED.file_name, ''), billing_storage_files.file_name),
				size_bytes = GREATEST(EXCLUDED.size_bytes, 0),
				ref_kind   = COALESCE(NULLIF(EXCLUDED.ref_kind, ''), billing_storage_files.ref_kind),
				ref_id     = COALESCE(NULLIF(EXCLUDED.ref_id, ''), billing_storage_files.ref_id),
				ref_title  = COALESCE(NULLIF(EXCLUDED.ref_title, ''), billing_storage_files.ref_title)`,
			f.Key, userID, f.Service, company, f.Name, f.Size, f.RefKind, f.RefID, f.RefTitle)
	}
	return r.pool.SendBatch(ctx, batch).Close()
}

// RemoveFiles — снять записи по ключам. Возвращает освобождённое место: его
// знает журнал, поэтому мерить объекты в хранилище перед удалением не нужно.
func (r *Repo) RemoveFiles(ctx context.Context, userID int64, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	rows, err := r.pool.Query(ctx, `
		DELETE FROM billing_storage_files
		 WHERE user_id = $1 AND storage_key = ANY($2)
		RETURNING size_bytes`, userID, keys)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var freed int64
	for rows.Next() {
		var size int64
		if err := rows.Scan(&size); err != nil {
			return 0, err
		}
		freed += size
	}
	return freed, rows.Err()
}

func (r *Repo) TopFiles(ctx context.Context, userID int64, service string, limit int) ([]*domain.StoredFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT storage_key, service, COALESCE(company_id, 0), file_name, size_bytes,
		       ref_kind, ref_id, ref_title, created_at
		  FROM billing_storage_files
		 WHERE user_id = $1 AND ($2 = '' OR service = $2)
		 ORDER BY size_bytes DESC, created_at DESC
		 LIMIT $3`, userID, service, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFiles(rows)
}

func (r *Repo) AllFiles(ctx context.Context, userID int64) ([]*domain.StoredFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT storage_key, service, COALESCE(company_id, 0), file_name, size_bytes,
		       ref_kind, ref_id, ref_title, created_at
		  FROM billing_storage_files WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFiles(rows)
}

func (r *Repo) UsageFromFiles(ctx context.Context, userID int64) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT service, COALESCE(sum(size_bytes), 0) FROM billing_storage_files
		 WHERE user_id = $1 GROUP BY service`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var service string
		var bytes int64
		if err := rows.Scan(&service, &bytes); err != nil {
			return nil, err
		}
		out[service] = bytes
	}
	return out, rows.Err()
}

func scanFiles(rows pgx.Rows) ([]*domain.StoredFile, error) {
	out := []*domain.StoredFile{}
	for rows.Next() {
		var f domain.StoredFile
		if err := rows.Scan(&f.Key, &f.Service, &f.CompanyID, &f.Name, &f.Size,
			&f.RefKind, &f.RefID, &f.RefTitle, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &f)
	}
	return out, rows.Err()
}

package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Сторона портала в контракте владельца файлов (раздел «Хранилище»).

   Вложения поста принадлежат КОМПАНИИ, а место за них платит её создатель —
   поэтому отбор идёт по companyIDs (кто их создал, знает биллинг), а не по
   автору поста. */

func (r *Repo) ListStorageFiles(ctx context.Context, companyIDs []int64) ([]storagefiles.File, error) {
	if len(companyIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT a.file_path, a.name, a.created_at, p.id, p.company_id,
		       COALESCE(NULLIF(p.title, ''), left(p.body, 60))
		  FROM portal_attachments a
		  JOIN portal_posts p ON p.id = a.post_id
		 WHERE p.company_id = ANY($1)`, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []storagefiles.File{}
	for rows.Next() {
		var (
			path, name, postTitle string
			createdAt             time.Time
			postID, companyID     int64
		)
		if err := rows.Scan(&path, &name, &createdAt, &postID, &companyID, &postTitle); err != nil {
			return nil, err
		}
		out = append(out, storagefiles.File{
			Key: path, Name: name, Kind: "post", ID: strconv.FormatInt(postID, 10),
			Title: "Публикация: " + postTitle, CompanyID: companyID, CreatedAt: createdAt,
		})
	}
	return out, rows.Err()
}

// DeleteStorageFiles — снять вложения с публикаций. Сама публикация остаётся:
// человек освобождает место, а не стирает ленту компании.
func (r *Repo) DeleteStorageFiles(ctx context.Context, companyIDs []int64, keys []string) ([]string, error) {
	if len(companyIDs) == 0 || len(keys) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		DELETE FROM portal_attachments a
		 USING portal_posts p
		 WHERE p.id = a.post_id AND p.company_id = ANY($1) AND a.file_path = ANY($2)
		RETURNING a.file_path`, companyIDs, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deleted := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		deleted = append(deleted, path)
	}
	return deleted, rows.Err()
}

package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Сторона мессенджера в контракте владельца файлов (раздел «Хранилище»).

   Вложение принадлежит ЗАГРУЗИВШЕМУ, а не собеседнику и не чату: пересланная
   копия — отдельный файл, и место за неё платит тот, кто переслал. Поэтому
   отбор идёт по uploader_id, а компания тут ни при чём — переписка
   кросс-компанийная. */

// ListStorageFiles — живые вложения пользователя вместе с их превью.
func (r *Repo) ListStorageFiles(ctx context.Context, userID int64) ([]storagefiles.File, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT a.id, a.file_path, a.thumb_path, a.file_name, a.created_at,
		       COALESCE(c.title, u.fio, '')
		  FROM message_attachments a
		  LEFT JOIN messages m ON m.id = a.message_id
		  LEFT JOIN conversations c ON c.id = m.conversation_id
		  LEFT JOIN users u ON u.id = CASE
		           WHEN c.is_group OR c.is_dev_chat THEN NULL
		           WHEN c.user_a_id = $1 THEN c.user_b_id
		           ELSE c.user_a_id END
		 WHERE a.uploader_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []storagefiles.File{}
	for rows.Next() {
		var (
			id        int64
			path      string
			thumb     *string
			name      string
			createdAt time.Time
			chat      string
		)
		if err := rows.Scan(&id, &path, &thumb, &name, &createdAt, &chat); err != nil {
			return nil, err
		}
		file := storagefiles.File{
			Key: path, Name: name, Kind: "attachment",
			ID: strconv.FormatInt(id, 10), CreatedAt: createdAt,
		}
		if chat != "" {
			file.Title = "Чат: " + chat
		}
		out = append(out, file)
		if thumb != nil && *thumb != "" {
			preview := file
			preview.Key, preview.Name = *thumb, "Превью: "+name
			out = append(out, preview)
		}
	}
	return out, rows.Err()
}

// DeleteStorageFiles — убрать вложения по ключам файлов. Сообщение остаётся:
// человек чистит место, а не переписку. Возвращает удалённые ключи — превью
// уходит вместе со своим файлом, даже если его в списке не просили.
func (r *Repo) DeleteStorageFiles(ctx context.Context, userID int64, keys []string) ([]string, error) {
	rows, err := r.q(ctx).Query(ctx, `
		DELETE FROM message_attachments
		 WHERE uploader_id = $1 AND (file_path = ANY($2) OR thumb_path = ANY($2))
		RETURNING file_path, thumb_path`, userID, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deleted := []string{}
	for rows.Next() {
		var path string
		var thumb *string
		if err := rows.Scan(&path, &thumb); err != nil {
			return nil, err
		}
		deleted = append(deleted, path)
		if thumb != nil && *thumb != "" {
			deleted = append(deleted, *thumb)
		}
	}
	return deleted, rows.Err()
}

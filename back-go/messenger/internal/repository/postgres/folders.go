package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// ListFolders — папки владельца по position, с ручными привязками чатов одним
// проходом (LEFT JOIN + array_agg), без N+1.
func (r *Repo) ListFolders(ctx context.Context, ownerID int64) ([]*domain.Folder, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT f.id, f.owner_id, f.title, f.emoji, f.position,
		        f.include_personal, f.include_groups, f.include_unread,
		        COALESCE(array_agg(i.conversation_id) FILTER (WHERE i.conversation_id IS NOT NULL), '{}') AS conv_ids
		   FROM chat_folders f
		   LEFT JOIN chat_folder_items i ON i.folder_id = f.id
		  WHERE f.owner_id = $1
		  GROUP BY f.id
		  ORDER BY f.position, f.id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Folder
	for rows.Next() {
		f := &domain.Folder{}
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.Title, &f.Emoji, &f.Position,
			&f.IncludePersonal, &f.IncludeGroups, &f.IncludeUnread, &f.ConversationIDs); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) CountFolders(ctx context.Context, ownerID int64) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM chat_folders WHERE owner_id = $1`, ownerID).Scan(&n)
	return n, err
}

// CreateFolder — новая папка в конец списка владельца.
func (r *Repo) CreateFolder(ctx context.Context, f *domain.Folder) (int64, error) {
	var id int64
	err := r.q(ctx).QueryRow(ctx,
		`INSERT INTO chat_folders
		     (owner_id, title, emoji, position, include_personal, include_groups, include_unread)
		 VALUES ($1, $2, $3,
		         COALESCE((SELECT max(position) + 1 FROM chat_folders WHERE owner_id = $1), 0),
		         $4, $5, $6)
		 RETURNING id`,
		f.OwnerID, f.Title, f.Emoji, f.IncludePersonal, f.IncludeGroups, f.IncludeUnread).Scan(&id)
	return id, err
}

// UpdateFolder — правка полей папки владельца (состав правится отдельно).
func (r *Repo) UpdateFolder(ctx context.Context, ownerID int64, f *domain.Folder) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE chat_folders
		    SET title = $3, emoji = $4,
		        include_personal = $5, include_groups = $6, include_unread = $7
		  WHERE id = $1 AND owner_id = $2`,
		f.ID, ownerID, f.Title, f.Emoji, f.IncludePersonal, f.IncludeGroups, f.IncludeUnread)
	return err
}

func (r *Repo) DeleteFolder(ctx context.Context, ownerID, folderID int64) error {
	_, err := r.q(ctx).Exec(ctx,
		`DELETE FROM chat_folders WHERE id = $1 AND owner_id = $2`, folderID, ownerID)
	return err
}

// ReorderFolders — position = индекс в orderedIDs (только папки владельца).
func (r *Repo) ReorderFolders(ctx context.Context, ownerID int64, orderedIDs []int64) error {
	for pos, id := range orderedIDs {
		if _, err := r.q(ctx).Exec(ctx,
			`UPDATE chat_folders SET position = $3 WHERE id = $1 AND owner_id = $2`,
			id, ownerID, pos); err != nil {
			return err
		}
	}
	return nil
}

// SetFolderItems — полная замена ручных привязок папки. Ownership папки
// проверяется подзапросом на INSERT (чужой folder_id не пройдёт).
func (r *Repo) SetFolderItems(ctx context.Context, ownerID, folderID int64, convIDs []int64) error {
	if _, err := r.q(ctx).Exec(ctx,
		`DELETE FROM chat_folder_items
		  WHERE folder_id = (SELECT id FROM chat_folders WHERE id = $1 AND owner_id = $2)`,
		folderID, ownerID); err != nil {
		return err
	}
	for _, cid := range convIDs {
		if err := r.AddFolderItem(ctx, ownerID, folderID, cid); err != nil {
			return err
		}
	}
	return nil
}

// AddFolderItem — привязать чат к папке владельца (идемпотентно). Подзапрос
// гейтит владельца: чужой folder_id даёт 0 строк и тихо не вставляет.
func (r *Repo) AddFolderItem(ctx context.Context, ownerID, folderID, convID int64) error {
	_, err := r.q(ctx).Exec(ctx,
		`INSERT INTO chat_folder_items (folder_id, conversation_id)
		 SELECT id, $3 FROM chat_folders WHERE id = $1 AND owner_id = $2
		 ON CONFLICT DO NOTHING`,
		folderID, ownerID, convID)
	return err
}

func (r *Repo) RemoveFolderItem(ctx context.Context, ownerID, folderID, convID int64) error {
	_, err := r.q(ctx).Exec(ctx,
		`DELETE FROM chat_folder_items
		  WHERE conversation_id = $3
		    AND folder_id = (SELECT id FROM chat_folders WHERE id = $1 AND owner_id = $2)`,
		folderID, ownerID, convID)
	return err
}

package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
)

/* Перенос компании: выгрузка и вливание портала.

   Переносится сам контент — разделы, публикации с вложениями, обсуждения.
   Реакции, лайки, просмотры и отметки прочтения остаются: это следы КОНКРЕТНЫХ
   людей, а люди в архив не входят, и восстанавливать их «от имени кого-то»
   было бы подделкой. Хештеги не дублируем — сервис пересчитывает их из тела. */

type companyDump struct {
	Topics []dumpTopic `json:"topics"`
	Posts  []dumpPost  `json:"posts"`
}

type dumpTopic struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	Icon      *string   `json:"icon,omitempty"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type dumpPost struct {
	ID          int64         `json:"id"`
	TopicID     *int64        `json:"topic_id,omitempty"`
	AuthorID    int64         `json:"author_id"`
	Title       *string       `json:"title,omitempty"`
	Body        string        `json:"body"`
	PinnedAt    *time.Time    `json:"pinned_at,omitempty"`
	PinnedBy    *int64        `json:"pinned_by,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   *time.Time    `json:"updated_at,omitempty"`
	Attachments []dumpAttach  `json:"attachments,omitempty"`
	Comments    []dumpComment `json:"comments,omitempty"`
}

type dumpAttach struct {
	FilePath  string    `json:"file_path"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mime      *string   `json:"mime,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type dumpComment struct {
	ID        int64     `json:"id"`
	ReplyToID *int64    `json:"reply_to_id,omitempty"`
	AuthorID  int64     `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// ExportCompany — разделы и публикации портала одним JSON.
func (r *Repo) ExportCompany(ctx context.Context, companyID int64) (companydata.Export, error) {
	var dump companyDump

	rows, err := r.pool.Query(ctx,
		`SELECT id, name, color, icon, created_by, created_at FROM portal_topics WHERE company_id = $1 ORDER BY id`,
		companyID)
	if err != nil {
		return companydata.Export{}, err
	}
	dump.Topics = []dumpTopic{}
	for rows.Next() {
		var t dumpTopic
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Icon, &t.CreatedBy, &t.CreatedAt); err != nil {
			rows.Close()
			return companydata.Export{}, err
		}
		dump.Topics = append(dump.Topics, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return companydata.Export{}, err
	}

	rows, err = r.pool.Query(ctx, `
		SELECT id, topic_id, author_id, title, body, pinned_at, pinned_by, created_at, updated_at
		  FROM portal_posts WHERE company_id = $1 ORDER BY id`, companyID)
	if err != nil {
		return companydata.Export{}, err
	}
	dump.Posts = []dumpPost{}
	byID := map[int64]int{}
	for rows.Next() {
		var p dumpPost
		if err := rows.Scan(&p.ID, &p.TopicID, &p.AuthorID, &p.Title, &p.Body,
			&p.PinnedAt, &p.PinnedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			rows.Close()
			return companydata.Export{}, err
		}
		byID[p.ID] = len(dump.Posts)
		dump.Posts = append(dump.Posts, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return companydata.Export{}, err
	}

	postIDs := make([]int64, 0, len(dump.Posts))
	for _, p := range dump.Posts {
		postIDs = append(postIDs, p.ID)
	}

	// Вложения и обсуждения — двумя запросами на весь портал, не по посту.
	files := []string{}
	rows, err = r.pool.Query(ctx, `
		SELECT post_id, file_path, name, size, mime, created_at
		  FROM portal_attachments WHERE post_id = ANY($1) ORDER BY id`, postIDs)
	if err != nil {
		return companydata.Export{}, err
	}
	for rows.Next() {
		var postID int64
		var a dumpAttach
		if err := rows.Scan(&postID, &a.FilePath, &a.Name, &a.Size, &a.Mime, &a.CreatedAt); err != nil {
			rows.Close()
			return companydata.Export{}, err
		}
		if i, ok := byID[postID]; ok {
			dump.Posts[i].Attachments = append(dump.Posts[i].Attachments, a)
		}
		if a.FilePath != "" {
			files = append(files, a.FilePath)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return companydata.Export{}, err
	}

	rows, err = r.pool.Query(ctx, `
		SELECT id, post_id, reply_to_id, author_id, text, created_at
		  FROM portal_comments WHERE post_id = ANY($1) ORDER BY id`, postIDs)
	if err != nil {
		return companydata.Export{}, err
	}
	for rows.Next() {
		var postID int64
		var c dumpComment
		if err := rows.Scan(&c.ID, &postID, &c.ReplyToID, &c.AuthorID, &c.Text, &c.CreatedAt); err != nil {
			rows.Close()
			return companydata.Export{}, err
		}
		if i, ok := byID[postID]; ok {
			dump.Posts[i].Comments = append(dump.Posts[i].Comments, c)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return companydata.Export{}, err
	}

	payload, err := json.Marshal(dump)
	if err != nil {
		return companydata.Export{}, err
	}
	return companydata.Export{Payload: payload, FileKeys: files, Count: len(dump.Posts)}, nil
}

// ImportCompany — влить портал в компанию, созданную под импорт.
func (r *Repo) ImportCompany(ctx context.Context, in companydata.Import) (int, error) {
	var dump companyDump
	if err := json.Unmarshal(in.Payload, &dump); err != nil {
		return 0, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	topics := map[int64]int64{}
	for _, t := range dump.Topics {
		var id int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO portal_topics (company_id, name, color, icon, created_by, created_at)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			in.CompanyID, t.Name, t.Color, t.Icon, in.UserID(t.CreatedBy), t.CreatedAt).Scan(&id); err != nil {
			return 0, err
		}
		topics[t.ID] = id
	}

	for _, p := range dump.Posts {
		var topicID *int64
		if p.TopicID != nil {
			if id, ok := topics[*p.TopicID]; ok {
				topicID = &id
			}
		}
		var pinnedBy *int64
		if p.PinnedBy != nil {
			id := in.UserID(*p.PinnedBy)
			pinnedBy = &id
		}
		var postID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO portal_posts (company_id, topic_id, author_id, title, body,
			                          pinned_at, pinned_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
			in.CompanyID, topicID, in.UserID(p.AuthorID), p.Title, p.Body,
			p.PinnedAt, pinnedBy, p.CreatedAt, p.UpdatedAt).Scan(&postID); err != nil {
			return 0, err
		}

		for _, a := range p.Attachments {
			if _, err := tx.Exec(ctx, `
				INSERT INTO portal_attachments (post_id, file_path, name, size, mime, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				postID, in.FileKey(a.FilePath), a.Name, a.Size, a.Mime, a.CreatedAt); err != nil {
				return 0, err
			}
		}

		// Ветка ответов: родитель всегда старше ребёнка, поэтому обхода в
		// порядке id достаточно, чтобы ссылка уже была переназначена.
		comments := map[int64]int64{}
		for _, c := range p.Comments {
			var replyTo *int64
			if c.ReplyToID != nil {
				if id, ok := comments[*c.ReplyToID]; ok {
					replyTo = &id
				}
			}
			var id int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO portal_comments (post_id, reply_to_id, author_id, text, created_at)
				VALUES ($1, $2, $3, $4, $5) RETURNING id`,
				postID, replyTo, in.UserID(c.AuthorID), c.Text, c.CreatedAt).Scan(&id); err != nil {
				return 0, err
			}
			comments[c.ID] = id
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(dump.Posts), nil
}

package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

const postCols = `id, company_id, topic_id, author_id, title, body, pinned_at, pinned_by, pinned_until, created_at, updated_at`

// pinActiveCond — актуальный пин: закреплён и срок не истёк (истёкший
// pinned_until везде трактуется как незакреплённый — ленивую чистку колонок
// не делаем, read-путь остаётся без записи).
const pinActiveCond = `(pinned_at IS NOT NULL AND (pinned_until IS NULL OR pinned_until > now()))`

func scanPost(row pgx.Row) (*domain.Post, error) {
	var p domain.Post
	err := row.Scan(&p.ID, &p.CompanyID, &p.TopicID, &p.AuthorID, &p.Title, &p.Body,
		&p.PinnedAt, &p.PinnedBy, &p.PinnedUntil, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) ListPosts(ctx context.Context, f domain.PostListFilter, viewerID int64) ([]*domain.Post, error) {
	where := `WHERE company_id = $1`
	args := []any{f.CompanyID}
	if f.TopicID != nil {
		args = append(args, *f.TopicID)
		where += fmt.Sprintf(" AND topic_id = $%d", len(args))
	}
	if f.Pinned != nil {
		if *f.Pinned {
			where += " AND " + pinActiveCond
		} else {
			where += " AND NOT " + pinActiveCond
		}
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		args = append(args, "%"+s+"%")
		where += fmt.Sprintf(" AND (coalesce(title, '') || ' ' || body) ILIKE $%d", len(args))
	}
	if f.Tag != "" {
		args = append(args, f.Tag)
		where += fmt.Sprintf(` AND EXISTS (SELECT 1 FROM portal_post_tags pt
			WHERE pt.post_id = portal_posts.id AND pt.tag = $%d)`, len(args))
	}
	if f.BeforeCreatedAt != nil {
		// Keyset: строго старше пары (created_at, id) — row comparison
		// согласован с ORDER BY created_at DESC, id DESC.
		args = append(args, *f.BeforeCreatedAt, f.BeforeID)
		where += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args))
	}
	order := "created_at DESC, id DESC"
	if f.Pinned != nil && *f.Pinned {
		order = "pinned_at DESC, created_at DESC, id DESC"
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 500
	}
	args = append(args, limit)
	rows, err := r.pool.Query(ctx, `
		SELECT `+postCols+` FROM portal_posts `+where+`
		ORDER BY `+order+`
		LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Post{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachDerived(ctx, out, viewerID); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repo) GetPost(ctx context.Context, id int64) (*domain.Post, error) {
	return scanPost(r.pool.QueryRow(ctx, `SELECT `+postCols+` FROM portal_posts WHERE id = $1`, id))
}

func (r *Repo) GetPostForViewer(ctx context.Context, id, viewerID int64) (*domain.Post, error) {
	p, err := r.GetPost(ctx, id)
	if err != nil || p == nil {
		return p, err
	}
	if err := r.attachDerived(ctx, []*domain.Post{p}, viewerID); err != nil {
		return nil, err
	}
	return p, nil
}

// attachDerived — батч-подгрузка вложений/счётчика комментариев/реакций для
// списка постов (без N+1: по одному запросу на каждую производную вместо
// запроса на пост).
func (r *Repo) attachDerived(ctx context.Context, posts []*domain.Post, viewerID int64) error {
	if len(posts) == 0 {
		return nil
	}
	byID := make(map[int64]*domain.Post, len(posts))
	ids := make([]int64, len(posts))
	for i, p := range posts {
		p.Attachments = []domain.Attachment{}
		p.ReactionCount = map[string]int{}
		p.MyReactions = []string{}
		p.Tags = []string{}
		byID[p.ID] = p
		ids[i] = p.ID
	}

	tagRows, err := r.pool.Query(ctx, `
		SELECT post_id, tag FROM portal_post_tags
		WHERE post_id = ANY($1) ORDER BY tag`, ids)
	if err != nil {
		return err
	}
	for tagRows.Next() {
		var postID int64
		var tag string
		if err := tagRows.Scan(&postID, &tag); err != nil {
			tagRows.Close()
			return err
		}
		if p := byID[postID]; p != nil {
			p.Tags = append(p.Tags, tag)
		}
	}
	tagRows.Close()
	if err := tagRows.Err(); err != nil {
		return err
	}

	attRows, err := r.pool.Query(ctx, `
		SELECT id, post_id, file_path, name, size, mime, created_at
		FROM portal_attachments WHERE post_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	for attRows.Next() {
		var a domain.Attachment
		if err := attRows.Scan(&a.ID, &a.PostID, &a.FilePath, &a.Name, &a.Size, &a.Mime, &a.CreatedAt); err != nil {
			attRows.Close()
			return err
		}
		a.URL = "/uploads/" + a.FilePath
		if p := byID[a.PostID]; p != nil {
			p.Attachments = append(p.Attachments, a)
		}
	}
	attRows.Close()
	if err := attRows.Err(); err != nil {
		return err
	}

	commentRows, err := r.pool.Query(ctx, `
		SELECT post_id, COUNT(*) FROM portal_comments WHERE post_id = ANY($1) GROUP BY post_id`, ids)
	if err != nil {
		return err
	}
	for commentRows.Next() {
		var postID int64
		var n int
		if err := commentRows.Scan(&postID, &n); err != nil {
			commentRows.Close()
			return err
		}
		if p := byID[postID]; p != nil {
			p.CommentCount = n
		}
	}
	commentRows.Close()
	if err := commentRows.Err(); err != nil {
		return err
	}

	reactionRows, err := r.pool.Query(ctx, `
		SELECT post_id, emoji, COUNT(*) FROM portal_reactions
		WHERE post_id = ANY($1) GROUP BY post_id, emoji`, ids)
	if err != nil {
		return err
	}
	for reactionRows.Next() {
		var postID int64
		var emoji string
		var n int
		if err := reactionRows.Scan(&postID, &emoji, &n); err != nil {
			reactionRows.Close()
			return err
		}
		if p := byID[postID]; p != nil {
			p.ReactionCount[emoji] = n
		}
	}
	reactionRows.Close()
	if err := reactionRows.Err(); err != nil {
		return err
	}

	viewRows, err := r.pool.Query(ctx, `
		SELECT post_id, COUNT(*) FROM portal_post_views
		WHERE post_id = ANY($1) GROUP BY post_id`, ids)
	if err != nil {
		return err
	}
	for viewRows.Next() {
		var postID int64
		var n int
		if err := viewRows.Scan(&postID, &n); err != nil {
			viewRows.Close()
			return err
		}
		if p := byID[postID]; p != nil {
			p.ViewCount = n
		}
	}
	viewRows.Close()
	if err := viewRows.Err(); err != nil {
		return err
	}

	if viewerID != 0 {
		mineRows, err := r.pool.Query(ctx, `
			SELECT post_id, emoji FROM portal_reactions
			WHERE post_id = ANY($1) AND user_id = $2`, ids, viewerID)
		if err != nil {
			return err
		}
		for mineRows.Next() {
			var postID int64
			var emoji string
			if err := mineRows.Scan(&postID, &emoji); err != nil {
				mineRows.Close()
				return err
			}
			if p := byID[postID]; p != nil {
				p.MyReactions = append(p.MyReactions, emoji)
			}
		}
		mineRows.Close()
		if err := mineRows.Err(); err != nil {
			return err
		}

		seenRows, err := r.pool.Query(ctx, `
			SELECT post_id FROM portal_post_views
			WHERE post_id = ANY($1) AND user_id = $2`, ids, viewerID)
		if err != nil {
			return err
		}
		for seenRows.Next() {
			var postID int64
			if err := seenRows.Scan(&postID); err != nil {
				seenRows.Close()
				return err
			}
			if p := byID[postID]; p != nil {
				p.Viewed = true
			}
		}
		seenRows.Close()
		if err := seenRows.Err(); err != nil {
			return err
		}
	}
	return nil
}

// MarkView — upsert строки просмотра (идемпотентно): счётчик уникальных
// зрителей наращивается лишь при первом просмотре поста пользователем.
func (r *Repo) MarkView(ctx context.Context, postID, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO portal_post_views (post_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (post_id, user_id) DO NOTHING`, postID, userID)
	return err
}

func (r *Repo) CreatePost(ctx context.Context, p *domain.Post) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		INSERT INTO portal_posts (company_id, topic_id, author_id, title, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		p.CompanyID, p.TopicID, p.AuthorID, p.Title, p.Body,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return err
	}
	if err := replacePostTags(ctx, tx, p.ID, p.CompanyID, p.Tags); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) UpdatePost(ctx context.Context, p *domain.Post) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		UPDATE portal_posts SET topic_id = $2, title = $3, body = $4, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`,
		p.ID, p.TopicID, p.Title, p.Body,
	).Scan(&p.UpdatedAt); err != nil {
		return err
	}
	if err := replacePostTags(ctx, tx, p.ID, p.CompanyID, p.Tags); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// replacePostTags — полная замена набора хештегов поста (delete + insert)
// внутри транзакции создания/правки поста.
func replacePostTags(ctx context.Context, tx pgx.Tx, postID, companyID int64, tags []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM portal_post_tags WHERE post_id = $1`, postID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := tx.Exec(ctx, `
			INSERT INTO portal_post_tags (post_id, company_id, tag) VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`, postID, companyID, tag); err != nil {
			return err
		}
	}
	return nil
}

// PopularTags — топ хештегов компании по числу постов (популярное сверху,
// затем алфавит для стабильности при равенстве).
func (r *Repo) PopularTags(ctx context.Context, companyID int64, limit int) ([]domain.TagCount, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT tag, COUNT(*) AS n FROM portal_post_tags
		WHERE company_id = $1
		GROUP BY tag ORDER BY n DESC, tag ASC LIMIT $2`, companyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TagCount{}
	for rows.Next() {
		var t domain.TagCount
		if err := rows.Scan(&t.Tag, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) DeletePost(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM portal_posts WHERE id = $1`, id)
	return err
}

// pinLockClass — класс advisory-локов закрепления постов (2-ключевая форма
// pg_advisory_xact_lock: класс + company_id), чтобы не пересекаться с локами
// других сервисов в общей БД.
const pinLockClass = 7401

// PinPost — проверка лимита и UPDATE в одной транзакции под локом компании:
// два параллельных Pin не могут вдвоём пройти проверку и закрепить 11-й пост.
// В лимите считаются только АКТУАЛЬНЫЕ пины (истёкший pinned_until слот
// не занимает).
func (r *Repo) PinPost(ctx context.Context, id, companyID, pinnedBy int64, until *time.Time, limit int) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2::int)`,
		pinLockClass, companyID); err != nil {
		return false, err
	}
	// id <> $1 — повторное закрепление уже закреплённого поста не упирается
	// в собственный слот.
	tag, err := tx.Exec(ctx, `
		UPDATE portal_posts SET pinned_at = now(), pinned_by = $3, pinned_until = $4
		WHERE id = $1 AND (
			SELECT COUNT(*) FROM portal_posts
			WHERE company_id = $2 AND `+pinActiveCond+` AND id <> $1
		) < $5`, id, companyID, pinnedBy, until, limit)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, tx.Commit(ctx)
}

// SetPinned — открепление: pinned_until сбрасывается вместе с pinned_at.
func (r *Repo) SetPinned(ctx context.Context, id int64, pinnedAt *time.Time, pinnedBy *int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE portal_posts SET pinned_at = $2, pinned_by = $3, pinned_until = NULL WHERE id = $1`,
		id, pinnedAt, pinnedBy)
	return err
}

func (r *Repo) AddAttachment(ctx context.Context, a *domain.Attachment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO portal_attachments (post_id, file_path, name, size, mime)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		a.PostID, a.FilePath, a.Name, a.Size, a.Mime,
	).Scan(&a.ID, &a.CreatedAt)
}

func (r *Repo) GetAttachment(ctx context.Context, id int64) (*domain.Attachment, error) {
	var a domain.Attachment
	err := r.pool.QueryRow(ctx, `
		SELECT id, post_id, file_path, name, size, mime, created_at
		FROM portal_attachments WHERE id = $1`, id,
	).Scan(&a.ID, &a.PostID, &a.FilePath, &a.Name, &a.Size, &a.Mime, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.URL = "/uploads/" + a.FilePath
	return &a, nil
}

func (r *Repo) DeleteAttachment(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM portal_attachments WHERE id = $1`, id)
	return err
}

func (r *Repo) ListAttachments(ctx context.Context, postID int64) ([]domain.Attachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, post_id, file_path, name, size, mime, created_at
		FROM portal_attachments WHERE post_id = $1 ORDER BY id`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Attachment{}
	for rows.Next() {
		var a domain.Attachment
		if err := rows.Scan(&a.ID, &a.PostID, &a.FilePath, &a.Name, &a.Size, &a.Mime, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.URL = "/uploads/" + a.FilePath
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) AttachmentPaths(ctx context.Context, postID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT file_path FROM portal_attachments WHERE post_id = $1`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

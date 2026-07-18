package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// tagRe — хештег в теле поста: символ #, которому не предшествует буква/цифра/_
// (не режем середину слова и не цепляем URL-якоря вида .../#anchor или
// html-сущности &#39;), затем 2..50 букв/цифр/подчёркиваний, начиная с буквы
// или цифры. Юникод-классы ⇒ кириллица работает наравне с латиницей.
var tagRe = regexp.MustCompile(`(^|[^\p{L}\p{N}_&#/])#([\p{L}\p{N}][\p{L}\p{N}_]{1,49})`)

// extractTags — набор хештегов из тела поста: нормализованные (lower),
// без дублей, в порядке появления, не более domain.MaxPostTags.
func extractTags(body string) []string {
	matches := tagRe.FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, m := range matches {
		tag := strings.ToLower(m[2])
		if len([]rune(tag)) > domain.MaxTagLen || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
		if len(out) >= domain.MaxPostTags {
			break
		}
	}
	return out
}

// PostListParams — сырые параметры выборки постов (топик/закреплённые/поиск +
// keyset-пагинация: Limit 1..50, Cursor — opaque-курсор из next_cursor
// предыдущей страницы).
type PostListParams struct {
	TopicID *int64
	Pinned  *bool
	Search  string
	Tag     string
	Limit   int
	Cursor  string
}

const (
	defaultPageSize = 20
	maxPageSize     = 50
)

// PostFeed — страница ленты. Pinned (актуально закреплённые, целиком) отдаётся
// ТОЛЬКО на первой странице (без cursor); на последующих — пустой. Posts —
// хронология БЕЗ актуально закреплённых (фронт рендерит секции без дублей).
// NextCursor == nil — постов дальше нет.
type PostFeed struct {
	Pinned     []*domain.Post `json:"pinned"`
	Posts      []*domain.Post `json:"posts"`
	NextCursor *string        `json:"next_cursor"`
}

// encodeCursor/decodeCursor — opaque keyset-курсор: base64url от
// "<created_at UnixNano>|<id>" последнего поста страницы.
func encodeCursor(t time.Time, id int64) string {
	return base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "%d|%d", t.UnixNano(), id))
}

func decodeCursor(s string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, 0, domain.ErrBadCursor
	}
	var nano, id int64
	if _, err := fmt.Sscanf(string(raw), "%d|%d", &nano, &id); err != nil || id <= 0 {
		return time.Time{}, 0, domain.ErrBadCursor
	}
	return time.Unix(0, nano).UTC(), id, nil
}

// ListPosts — лента компании с keyset-пагинацией. Первая страница: секция
// pinned (все актуально закреплённые, pinned_at DESC) + первая страница
// хронологии; далее по cursor — только хронология. Явный фильтр ?pinned=
// сохранён для совместимости: pinned=true отдаёт закреплённые в Posts одной
// страницей (их ≤ MaxPinnedPosts, пагинация не нужна).
func (s *Service) ListPosts(ctx context.Context, companyID, viewerID int64, p PostListParams) (*PostFeed, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	base := domain.PostListFilter{
		CompanyID: companyID, TopicID: p.TopicID,
		Search: strings.TrimSpace(p.Search), Tag: strings.ToLower(strings.TrimSpace(p.Tag)),
	}
	feed := &PostFeed{Pinned: []*domain.Post{}, Posts: []*domain.Post{}}

	pinnedTrue := true
	if p.Pinned != nil && *p.Pinned {
		f := base
		f.Pinned, f.Limit = &pinnedTrue, domain.MaxPinnedPosts
		posts, err := s.repo.ListPosts(ctx, f, viewerID)
		if err != nil {
			return nil, err
		}
		feed.Posts = posts
		return feed, nil
	}

	f := base
	pinnedFalse := false
	f.Pinned, f.Limit = &pinnedFalse, limit+1
	if p.Cursor != "" {
		at, id, err := decodeCursor(p.Cursor)
		if err != nil {
			return nil, err
		}
		f.BeforeCreatedAt, f.BeforeID = &at, id
	} else if p.Pinned == nil {
		// Первая страница без явного фильтра — секция закреплённых целиком.
		pf := base
		pf.Pinned, pf.Limit = &pinnedTrue, domain.MaxPinnedPosts
		pinned, err := s.repo.ListPosts(ctx, pf, viewerID)
		if err != nil {
			return nil, err
		}
		feed.Pinned = pinned
	}

	posts, err := s.repo.ListPosts(ctx, f, viewerID)
	if err != nil {
		return nil, err
	}
	if len(posts) > limit {
		posts = posts[:limit]
		last := posts[len(posts)-1]
		cur := encodeCursor(last.CreatedAt, last.ID)
		feed.NextCursor = &cur
	}
	feed.Posts = posts
	return feed, nil
}

func (s *Service) GetPost(ctx context.Context, companyID, id, viewerID int64) (*domain.Post, error) {
	if _, err := s.requirePost(ctx, companyID, id); err != nil {
		return nil, err
	}
	return s.repo.GetPostForViewer(ctx, id, viewerID)
}

// MarkView — зафиксировать просмотр поста зрителем (заход в поле зрения на
// ленте/по ссылке). Идемпотентно; событий в шину не публикует — счётчик не
// realtime-критичен, другим клиентам он подтягивается при следующей загрузке.
func (s *Service) MarkView(ctx context.Context, companyID, id, viewerID int64) error {
	if _, err := s.requirePost(ctx, companyID, id); err != nil {
		return err
	}
	return s.repo.MarkView(ctx, id, viewerID)
}

// CreatePost — топик (если указан) должен принадлежать той же компании.
func (s *Service) CreatePost(ctx context.Context, companyID, authorID int64, topicID *int64, title *string, body string) (*domain.Post, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.ErrPostBodyReq
	}
	if topicID != nil {
		if _, err := s.requireTopic(ctx, companyID, *topicID); err != nil {
			return nil, err
		}
	}
	p := &domain.Post{
		CompanyID: companyID, TopicID: topicID, AuthorID: authorID,
		Title: normTitle(title), Body: body, Tags: extractTags(body),
	}
	if err := s.repo.CreatePost(ctx, p); err != nil {
		return nil, err
	}
	p.Attachments = []domain.Attachment{}
	p.ReactionCount = map[string]int{}
	p.MyReactions = []string{}
	s.bus.Publish(ctx, "post:new", []string{roomAll}, postPayload(p))
	return p, nil
}

// canManage — автор поста или администратор компании (используется для
// правки/удаления и закрепления).
func canManage(p *domain.Post, userID int64, roleLevel int) bool {
	return p.AuthorID == userID || roleLevel >= domain.LevelAdmin
}

func (s *Service) UpdatePost(ctx context.Context, companyID, id, userID int64, roleLevel int, topicID *int64, title *string, body string) (*domain.Post, error) {
	p, err := s.requirePost(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if !canManage(p, userID, roleLevel) {
		return nil, domain.ErrForbidden
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.ErrPostBodyReq
	}
	if topicID != nil {
		if _, err := s.requireTopic(ctx, companyID, *topicID); err != nil {
			return nil, err
		}
	}
	p.TopicID, p.Title, p.Body, p.Tags = topicID, normTitle(title), body, extractTags(body)
	if err := s.repo.UpdatePost(ctx, p); err != nil {
		return nil, err
	}
	full, err := s.repo.GetPostForViewer(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "post:updated", []string{roomAll}, postPayload(full))
	return full, nil
}

func (s *Service) DeletePost(ctx context.Context, companyID, id, userID int64, roleLevel int) error {
	p, err := s.requirePost(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !canManage(p, userID, roleLevel) {
		return domain.ErrForbidden
	}
	paths, err := s.repo.AttachmentPaths(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeletePost(ctx, id); err != nil {
		return err
	}
	if len(paths) > 0 {
		s.files.Remove(paths)
	}
	s.bus.Publish(ctx, "post:deleted", []string{roomAll}, map[string]any{
		"id": id, "company_id": companyID,
	})
	return nil
}

// MaxPinDays — потолок автоистечения пина в днях (кламп 1..30; 0 — бессрочно).
const MaxPinDays = 30

// pinUntil — срок автоистечения из дней: days<=0 — бессрочно (nil),
// days>MaxPinDays — кламп до MaxPinDays.
func pinUntil(days int) *time.Time {
	if days <= 0 {
		return nil
	}
	if days > MaxPinDays {
		days = MaxPinDays
	}
	until := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	return &until
}

// Pin — закрепить пост (автор или администратор), с лимитом MaxPinnedPosts
// одновременно закреплённых на компанию (истёкшие пины слот не занимают).
// days > 0 — пин автоистекает через N дней (кламп 1..30), 0 — бессрочно.
// Лимит соблюдает репозиторий атомарно (PinPost): проверка и UPDATE вне
// одной транзакции пропускали бы 11-й пост при параллельных закреплениях.
func (s *Service) Pin(ctx context.Context, companyID, id, userID int64, roleLevel, days int) (*domain.Post, error) {
	p, err := s.requirePost(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if !canManage(p, userID, roleLevel) {
		return nil, domain.ErrForbidden
	}
	ok, err := s.repo.PinPost(ctx, id, companyID, userID, pinUntil(days), domain.MaxPinnedPosts)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrTooManyPinned
	}
	full, err := s.repo.GetPostForViewer(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "post:pinned", []string{roomAll}, postPayload(full))
	return full, nil
}

func (s *Service) Unpin(ctx context.Context, companyID, id, userID int64, roleLevel int) (*domain.Post, error) {
	p, err := s.requirePost(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if !canManage(p, userID, roleLevel) {
		return nil, domain.ErrForbidden
	}
	if err := s.repo.SetPinned(ctx, id, nil, nil); err != nil {
		return nil, err
	}
	full, err := s.repo.GetPostForViewer(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "post:unpinned", []string{roomAll}, postPayload(full))
	return full, nil
}

func normTitle(title *string) *string {
	if title == nil {
		return nil
	}
	t := strings.TrimSpace(*title)
	if t == "" {
		return nil
	}
	return &t
}

func postPayload(p *domain.Post) map[string]any {
	return map[string]any{
		"id": p.ID, "company_id": p.CompanyID, "topic_id": p.TopicID, "author_id": p.AuthorID,
		"title": p.Title, "body": p.Body, "pinned_at": p.PinnedAt, "pinned_by": p.PinnedBy,
		"pinned_until": p.PinnedUntil,
		"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		"tags":        p.Tags,
		"attachments": p.Attachments, "comment_count": p.CommentCount,
		"reaction_counts": p.ReactionCount, "my_reactions": p.MyReactions,
		"view_count": p.ViewCount,
	}
}

// PopularTags — топ хештегов активной компании (панель «Популярные теги»
// ленты). Пустой набор — валиден (тегов ещё нет).
func (s *Service) PopularTags(ctx context.Context, companyID int64, limit int) ([]domain.TagCount, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.PopularTags(ctx, companyID, limit)
}

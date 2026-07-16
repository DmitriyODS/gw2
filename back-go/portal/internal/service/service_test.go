package service

import (
	"context"
	"io"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRepo — in-memory реализация domain.Repository для тестов бизнес-логики.
type fakeRepo struct {
	topics   map[int64]*domain.Topic
	posts    map[int64]*domain.Post
	comments map[int64]*domain.Comment
	atts     map[int64][]domain.Attachment
	reacts   []domain.Reaction
	seen     map[[2]int64]time.Time // {userID, companyID} → seen_at
	nextID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		topics: map[int64]*domain.Topic{}, posts: map[int64]*domain.Post{},
		comments: map[int64]*domain.Comment{}, atts: map[int64][]domain.Attachment{},
		seen: map[[2]int64]time.Time{},
	}
}

func (f *fakeRepo) id() int64 { f.nextID++; return f.nextID }

// tick — детерминированные «часы» фейка: каждый вызов строго позже предыдущего
// (created_at постов и seen_at сравниваются строгим >, реальный time.Now()
// в быстрых тестах мог бы совпасть). База — реальное «сейчас», чтобы окна
// вроде дедупа системных постов (now-10мин) видели свежесозданные посты.
var fakeClock = time.Now()

func (f *fakeRepo) tick() time.Time {
	fakeClock = fakeClock.Add(time.Second)
	return fakeClock
}

func (f *fakeRepo) ListTopics(_ domain.Ctx, companyID int64) ([]*domain.Topic, error) {
	out := []*domain.Topic{}
	for _, t := range f.topics {
		if t.CompanyID == companyID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (f *fakeRepo) GetTopic(_ domain.Ctx, id int64) (*domain.Topic, error) { return f.topics[id], nil }
func (f *fakeRepo) CreateTopic(_ domain.Ctx, t *domain.Topic) error {
	t.ID = f.id()
	f.topics[t.ID] = t
	return nil
}
func (f *fakeRepo) UpdateTopic(_ domain.Ctx, id int64, name string, color, icon *string) error {
	if t := f.topics[id]; t != nil {
		t.Name, t.Color, t.Icon = name, color, icon
	}
	return nil
}
func (f *fakeRepo) DeleteTopic(_ domain.Ctx, id int64) error { delete(f.topics, id); return nil }

// pinActive — актуальный пин (истёкший pinned_until трактуется как
// незакреплённый) — зеркалит SQL-условие pinActiveCond продовой репы.
func pinActive(p *domain.Post) bool {
	return p.PinnedAt != nil && (p.PinnedUntil == nil || p.PinnedUntil.After(time.Now()))
}

func (f *fakeRepo) ListPosts(_ domain.Ctx, filter domain.PostListFilter, _ int64) ([]*domain.Post, error) {
	out := []*domain.Post{}
	for _, p := range f.posts {
		if p.CompanyID != filter.CompanyID {
			continue
		}
		if filter.TopicID != nil && (p.TopicID == nil || *p.TopicID != *filter.TopicID) {
			continue
		}
		if filter.Pinned != nil && pinActive(p) != *filter.Pinned {
			continue
		}
		if filter.BeforeCreatedAt != nil {
			// Keyset: строго старше пары (created_at, id).
			if p.CreatedAt.After(*filter.BeforeCreatedAt) ||
				(p.CreatedAt.Equal(*filter.BeforeCreatedAt) && p.ID >= filter.BeforeID) {
				continue
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID > out[j].ID
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}
func (f *fakeRepo) GetPost(_ domain.Ctx, id int64) (*domain.Post, error) { return f.posts[id], nil }
func (f *fakeRepo) GetPostForViewer(_ domain.Ctx, id, _ int64) (*domain.Post, error) {
	p := f.posts[id]
	if p == nil {
		return nil, nil
	}
	p.Attachments = f.atts[id]
	if p.ReactionCount == nil {
		p.ReactionCount = map[string]int{}
	}
	if p.MyReactions == nil {
		p.MyReactions = []string{}
	}
	return p, nil
}
func (f *fakeRepo) CreatePost(_ domain.Ctx, p *domain.Post) error {
	p.ID = f.id()
	p.CreatedAt = f.tick()
	f.posts[p.ID] = p
	return nil
}
func (f *fakeRepo) UpdatePost(_ domain.Ctx, p *domain.Post) error { f.posts[p.ID] = p; return nil }
func (f *fakeRepo) DeletePost(_ domain.Ctx, id int64) error {
	delete(f.posts, id)
	delete(f.atts, id)
	return nil
}
func (f *fakeRepo) PinPost(_ domain.Ctx, id, companyID, pinnedBy int64, until *time.Time, limit int) (bool, error) {
	p := f.posts[id]
	if p == nil {
		return false, nil
	}
	n := 0
	for _, q := range f.posts {
		// Истёкшие пины слот не занимают (как pinActiveCond в SQL).
		if q.CompanyID == companyID && pinActive(q) && q.ID != id {
			n++
		}
	}
	if n >= limit {
		return false, nil
	}
	now := time.Now()
	p.PinnedAt, p.PinnedBy, p.PinnedUntil = &now, &pinnedBy, until
	return true, nil
}
func (f *fakeRepo) SetPinned(_ domain.Ctx, id int64, pinnedAt *time.Time, pinnedBy *int64) error {
	if p := f.posts[id]; p != nil {
		p.PinnedAt, p.PinnedBy, p.PinnedUntil = pinnedAt, pinnedBy, nil
	}
	return nil
}

func (f *fakeRepo) AddAttachment(_ domain.Ctx, a *domain.Attachment) error {
	a.ID = f.id()
	f.atts[a.PostID] = append(f.atts[a.PostID], *a)
	return nil
}
func (f *fakeRepo) GetAttachment(_ domain.Ctx, id int64) (*domain.Attachment, error) {
	for _, list := range f.atts {
		for _, a := range list {
			if a.ID == id {
				return &a, nil
			}
		}
	}
	return nil, nil
}
func (f *fakeRepo) DeleteAttachment(_ domain.Ctx, id int64) error {
	for postID, list := range f.atts {
		out := list[:0]
		for _, a := range list {
			if a.ID != id {
				out = append(out, a)
			}
		}
		f.atts[postID] = out
	}
	return nil
}
func (f *fakeRepo) ListAttachments(_ domain.Ctx, postID int64) ([]domain.Attachment, error) {
	return f.atts[postID], nil
}
func (f *fakeRepo) AttachmentPaths(_ domain.Ctx, postID int64) ([]string, error) {
	out := []string{}
	for _, a := range f.atts[postID] {
		out = append(out, a.FilePath)
	}
	return out, nil
}

func (f *fakeRepo) ListComments(_ domain.Ctx, postID int64) ([]*domain.Comment, error) {
	out := []*domain.Comment{}
	for _, c := range f.comments {
		if c.PostID == postID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeRepo) GetComment(_ domain.Ctx, id int64) (*domain.Comment, error) {
	return f.comments[id], nil
}
func (f *fakeRepo) CreateComment(_ domain.Ctx, c *domain.Comment) error {
	c.ID = f.id()
	f.comments[c.ID] = c
	return nil
}
func (f *fakeRepo) DeleteComment(_ domain.Ctx, id int64) error { delete(f.comments, id); return nil }

func (f *fakeRepo) AddReaction(_ domain.Ctx, r *domain.Reaction) error {
	for _, existing := range f.reacts {
		if existing.PostID == r.PostID && existing.UserID == r.UserID && existing.Emoji == r.Emoji {
			return nil
		}
	}
	f.reacts = append(f.reacts, *r)
	return nil
}
func (f *fakeRepo) RemoveReaction(_ domain.Ctx, postID, userID int64, emoji string) error {
	out := f.reacts[:0]
	for _, r := range f.reacts {
		if r.PostID == postID && r.UserID == userID && r.Emoji == emoji {
			continue
		}
		out = append(out, r)
	}
	f.reacts = out
	return nil
}

func (f *fakeRepo) MarkView(_ domain.Ctx, _, _ int64) error { return nil }

func (f *fakeRepo) SeenAt(_ domain.Ctx, userID, companyID int64) (*time.Time, error) {
	if at, ok := f.seen[[2]int64{userID, companyID}]; ok {
		return &at, nil
	}
	return nil, nil
}
func (f *fakeRepo) MarkSeen(_ domain.Ctx, userID, companyID int64) error {
	f.seen[[2]int64{userID, companyID}] = f.tick()
	return nil
}
func (f *fakeRepo) CountPostsAfter(_ domain.Ctx, companyID, excludeAuthorID int64, after *time.Time) (int, error) {
	n := 0
	for _, p := range f.posts {
		if p.CompanyID != companyID || p.AuthorID == excludeAuthorID {
			continue
		}
		if after == nil || p.CreatedAt.After(*after) {
			n++
		}
	}
	return n, nil
}

var _ domain.Repository = (*fakeRepo)(nil)

type fakeBus struct{ events []string }

func (b *fakeBus) Publish(_ domain.Ctx, event string, _ []string, _ any) {
	b.events = append(b.events, event)
}

type fakeFiles struct{ removed []string }

func (f *fakeFiles) Save(_ string, _ []byte) (string, error) { return "portal/x", nil }
func (f *fakeFiles) Remove(paths []string)                   { f.removed = append(f.removed, paths...) }

type fakeMessenger struct {
	dialogs map[int64]int64 // otherUserID → conversationID
	fail    map[int64]bool  // conversationID → CreatePostMessage должен упасть
	calls   []int64         // conversationID, для которых плашка создана
}

func (m *fakeMessenger) EnsureDialog(_ domain.Ctx, _, userBID int64) (int64, error) {
	if m.dialogs == nil {
		return 0, domain.NewError("NO_DIALOG", "нет диалога", 404)
	}
	if id, ok := m.dialogs[userBID]; ok {
		return id, nil
	}
	return 0, domain.NewError("NO_DIALOG", "нет диалога", 404)
}
func (m *fakeMessenger) CreatePostMessage(_ domain.Ctx, convID, _, _ int64, _ domain.PostPreview) (string, []int64, error) {
	if m.fail[convID] {
		return "", nil, domain.NewError("FAIL", "boom", 500)
	}
	m.calls = append(m.calls, convID)
	return `{"id":1}`, []int64{convID}, nil
}

var _ domain.MessengerClient = (*fakeMessenger)(nil)

func newTestService() (*Service, *fakeRepo, *fakeBus) {
	repo := newFakeRepo()
	bus := &fakeBus{}
	svc := New(Deps{Repo: repo, Files: &fakeFiles{}, Bus: bus, Messenger: &fakeMessenger{}, Log: discardLogger()})
	return svc, repo, bus
}

func mustCreatePost(t *testing.T, svc *Service, companyID, authorID int64) *domain.Post {
	t.Helper()
	p, err := svc.CreatePost(context.Background(), companyID, authorID, nil, nil, "тело поста")
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	return p
}

// ── Топики ───────────────────────────────────────────────────────

func TestCreateTopic_RequiresName(t *testing.T) {
	svc, _, _ := newTestService()
	if _, err := svc.CreateTopic(context.Background(), 1, 10, "  ", nil, nil); err != domain.ErrTopicNameReq {
		t.Fatalf("ожидалась ErrTopicNameReq, получено %v", err)
	}
}

func TestCreateTopic_PublishesEvent(t *testing.T) {
	svc, _, bus := newTestService()
	if _, err := svc.CreateTopic(context.Background(), 1, 10, "Новости", nil, nil); err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if len(bus.events) != 1 || bus.events[0] != "topic:created" {
		t.Fatalf("ожидалось topic:created, получено %v", bus.events)
	}
}

// ── Посты: скоуп по компании ─────────────────────────────────────

func TestGetPost_WrongCompanyNotFound(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)
	if _, err := svc.GetPost(context.Background(), 2, p.ID, 10); err != domain.ErrPostNotFound {
		t.Fatalf("ожидалась ErrPostNotFound для чужой компании, получено %v", err)
	}
}

func TestCreatePost_RequiresBody(t *testing.T) {
	svc, _, _ := newTestService()
	if _, err := svc.CreatePost(context.Background(), 1, 10, nil, nil, "   "); err != domain.ErrPostBodyReq {
		t.Fatalf("ожидалась ErrPostBodyReq, получено %v", err)
	}
}

// ── Закрепление: лимит и права ───────────────────────────────────

func TestPin_AuthorAllowedNonAuthorForbidden(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)

	// Не автор и не администратор — запрещено.
	if _, err := svc.Pin(context.Background(), 1, p.ID, 20, domain.LevelEmployee, 0); err != domain.ErrForbidden {
		t.Fatalf("ожидалась ErrForbidden, получено %v", err)
	}
	// Автор — можно.
	if _, err := svc.Pin(context.Background(), 1, p.ID, 10, domain.LevelEmployee, 0); err != nil {
		t.Fatalf("Pin автором: %v", err)
	}
	// Администратор — можно откреплять чужой пост.
	if _, err := svc.Unpin(context.Background(), 1, p.ID, 99, domain.LevelAdmin); err != nil {
		t.Fatalf("Unpin администратором: %v", err)
	}
}

// TestPin_LimitEnforced — не более MaxPinnedPosts закреплённых на компанию
// одновременно (аналог SharePoint boost-лимита, см. domain.ErrTooManyPinned).
func TestPin_LimitEnforced(t *testing.T) {
	svc, _, _ := newTestService()
	var last *domain.Post
	for i := 0; i < domain.MaxPinnedPosts; i++ {
		p := mustCreatePost(t, svc, 1, 10)
		if _, err := svc.Pin(context.Background(), 1, p.ID, 10, domain.LevelEmployee, 0); err != nil {
			t.Fatalf("Pin #%d: %v", i, err)
		}
		last = p
	}
	extra := mustCreatePost(t, svc, 1, 10)
	if _, err := svc.Pin(context.Background(), 1, extra.ID, 10, domain.LevelEmployee, 0); err != domain.ErrTooManyPinned {
		t.Fatalf("ожидалась ErrTooManyPinned после %d закреплённых, получено %v", domain.MaxPinnedPosts, err)
	}
	// Открепление одного освобождает слот.
	if _, err := svc.Unpin(context.Background(), 1, last.ID, 10, domain.LevelEmployee); err != nil {
		t.Fatalf("Unpin: %v", err)
	}
	if _, err := svc.Pin(context.Background(), 1, extra.ID, 10, domain.LevelEmployee, 0); err != nil {
		t.Fatalf("Pin после освобождения слота: %v", err)
	}
}

// ── Удаление поста: права + чистка вложений ──────────────────────

func TestDeletePost_ForbiddenForStranger(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)
	if err := svc.DeletePost(context.Background(), 1, p.ID, 20, domain.LevelEmployee); err != domain.ErrForbidden {
		t.Fatalf("ожидалась ErrForbidden, получено %v", err)
	}
}

func TestDeletePost_RemovesAttachmentFiles(t *testing.T) {
	svc, _, _ := newTestService()
	files := svc.files.(*fakeFiles)
	p := mustCreatePost(t, svc, 1, 10)
	if _, err := svc.AddAttachment(context.Background(), 1, p.ID, 10, domain.LevelEmployee, "a.png", "image/png", []byte("x")); err != nil {
		t.Fatalf("AddAttachment: %v", err)
	}
	if err := svc.DeletePost(context.Background(), 1, p.ID, 10, domain.LevelEmployee); err != nil {
		t.Fatalf("DeletePost: %v", err)
	}
	if len(files.removed) != 1 {
		t.Fatalf("ожидалось удаление 1 файла, получено %v", files.removed)
	}
}

// ── Удаление вложения ────────────────────────────────────────────

func mustAddAttachment(t *testing.T, svc *Service, companyID int64, p *domain.Post) *domain.Attachment {
	t.Helper()
	a, err := svc.AddAttachment(context.Background(), companyID, p.ID, p.AuthorID, domain.LevelEmployee, "a.png", "image/png", []byte("x"))
	if err != nil {
		t.Fatalf("AddAttachment: %v", err)
	}
	return a
}

func TestRemoveAttachment_ByAuthor(t *testing.T) {
	svc, repo, bus := newTestService()
	files := svc.files.(*fakeFiles)
	p := mustCreatePost(t, svc, 1, 10)
	a := mustAddAttachment(t, svc, 1, p)

	if err := svc.RemoveAttachment(context.Background(), 1, a.ID, 10, domain.LevelEmployee); err != nil {
		t.Fatalf("RemoveAttachment автором: %v", err)
	}
	if got, _ := repo.GetAttachment(context.Background(), a.ID); got != nil {
		t.Fatalf("вложение не удалено из репозитория")
	}
	if len(files.removed) != 1 || files.removed[0] != a.FilePath {
		t.Fatalf("ожидалось удаление файла %q, получено %v", a.FilePath, files.removed)
	}
	if last := bus.events[len(bus.events)-1]; last != "post:updated" {
		t.Fatalf("ожидалось событие post:updated, получено %q", last)
	}
}

func TestRemoveAttachment_ForbiddenForStranger(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)
	a := mustAddAttachment(t, svc, 1, p)

	if err := svc.RemoveAttachment(context.Background(), 1, a.ID, 20, domain.LevelEmployee); err != domain.ErrForbidden {
		t.Fatalf("ожидалась ErrForbidden для постороннего, получено %v", err)
	}
	// Администратор — можно (та же проверка, что AddAttachment).
	if err := svc.RemoveAttachment(context.Background(), 1, a.ID, 99, domain.LevelAdmin); err != nil {
		t.Fatalf("RemoveAttachment администратором: %v", err)
	}
}

func TestRemoveAttachment_WrongCompanyNotFound(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)
	a := mustAddAttachment(t, svc, 1, p)

	if err := svc.RemoveAttachment(context.Background(), 2, a.ID, 10, domain.LevelAdmin); err != domain.ErrPostNotFound {
		t.Fatalf("ожидалась ErrPostNotFound для чужой компании, получено %v", err)
	}
	if err := svc.RemoveAttachment(context.Background(), 1, 9999, 10, domain.LevelAdmin); err != domain.ErrAttachmentNotFound {
		t.Fatalf("ожидалась ErrAttachmentNotFound для несуществующего id, получено %v", err)
	}
}

// ── Комментарии ──────────────────────────────────────────────────

func TestComment_DeleteByAuthorOrAdmin(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)
	c, err := svc.CreateComment(context.Background(), 1, p.ID, 30, "привет")
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if err := svc.DeleteComment(context.Background(), 1, c.ID, 99, domain.LevelEmployee); err != domain.ErrForbidden {
		t.Fatalf("ожидалась ErrForbidden для постороннего, получено %v", err)
	}
	if err := svc.DeleteComment(context.Background(), 1, c.ID, 99, domain.LevelAdmin); err != nil {
		t.Fatalf("удаление администратором: %v", err)
	}
}

func TestComment_RequiresText(t *testing.T) {
	svc, _, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)
	if _, err := svc.CreateComment(context.Background(), 1, p.ID, 30, "   "); err != domain.ErrCommentTextReq {
		t.Fatalf("ожидалась ErrCommentTextReq, получено %v", err)
	}
}

// ── Реакции ──────────────────────────────────────────────────────

func TestReaction_AddIsIdempotent(t *testing.T) {
	svc, repo, _ := newTestService()
	p := mustCreatePost(t, svc, 1, 10)
	if err := svc.AddReaction(context.Background(), 1, p.ID, 30, "👍"); err != nil {
		t.Fatalf("AddReaction: %v", err)
	}
	if err := svc.AddReaction(context.Background(), 1, p.ID, 30, "👍"); err != nil {
		t.Fatalf("повторная реакция: %v", err)
	}
	if len(repo.reacts) != 1 {
		t.Fatalf("ожидалась 1 реакция без дублей, получено %d", len(repo.reacts))
	}
}

// ── Пересылка в мессенджер ───────────────────────────────────────

func TestForwardPost_ConversationAndUserIDs(t *testing.T) {
	svc, _, _ := newTestService()
	msgr := svc.messenger.(*fakeMessenger)
	msgr.dialogs = map[int64]int64{200: 555}

	p := mustCreatePost(t, svc, 1, 10)
	res, err := svc.ForwardPost(context.Background(), 1, p.ID, 10, []int64{100}, []int64{200})
	if err != nil {
		t.Fatalf("ForwardPost: %v", err)
	}
	if res.Forwarded != 2 || res.Failed != 0 {
		t.Fatalf("ожидалось forwarded=2 failed=0, получено %+v", res)
	}
	if len(msgr.calls) != 2 {
		t.Fatalf("ожидалось 2 плашки, получено %d", len(msgr.calls))
	}
}

func TestForwardPost_UnresolvedUserIDCountsFailed(t *testing.T) {
	svc, _, _ := newTestService()
	// Диалога с 999 нет — EnsureDialog падает, ветка честно уходит в failed
	// (не теряется молча).
	p := mustCreatePost(t, svc, 1, 10)
	res, err := svc.ForwardPost(context.Background(), 1, p.ID, 10, nil, []int64{999})
	if err != nil {
		t.Fatalf("ForwardPost: %v", err)
	}
	if res.Forwarded != 0 || res.Failed != 1 {
		t.Fatalf("нерезолвленный user_id должен считаться failed=1: %+v", res)
	}
}

func TestForwardPost_DeduplicatesConversations(t *testing.T) {
	svc, _, _ := newTestService()
	msgr := svc.messenger.(*fakeMessenger)
	// user 200 резолвится в тот же диалог 100, что уже передан явно.
	msgr.dialogs = map[int64]int64{200: 100}

	p := mustCreatePost(t, svc, 1, 10)
	res, err := svc.ForwardPost(context.Background(), 1, p.ID, 10, []int64{100, 100}, []int64{200})
	if err != nil {
		t.Fatalf("ForwardPost: %v", err)
	}
	if res.Forwarded != 1 || res.Failed != 0 {
		t.Fatalf("дубли диалога должны схлопнуться в одну плашку: %+v", res)
	}
	if len(msgr.calls) != 1 {
		t.Fatalf("ожидалась 1 плашка, получено %d", len(msgr.calls))
	}
}

func TestStripMarkdown(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"без разметки", "просто текст", "просто текст"},
		{"заголовок", "## Заголовок недели", "Заголовок недели"},
		{"жирный и курсив", "**важно** и *слегка*", "важно и слегка"},
		{"зачёркнутый и код", "~~старое~~ `new()`", "старое new()"},
		{"ссылка", "смотри [доку](https://example.com) тут", "смотри доку тут"},
		{"картинка", "![схема](https://x/y.png) рядом", "схема рядом"},
		{"маркированный список", "- один\n* два\n+ три", "один два три"},
		{"нумерованный список", "1. первый\n2. второй", "первый второй"},
		{"чекбоксы", "- [ ] сделать\n- [x] готово", "сделать готово"},
		{"цитата", "> мудрая мысль", "мудрая мысль"},
		{"фенс и линейка", "```go\ncode()\n```\n---\nтекст", "code() текст"},
		{"таблица", "| a | b |\n| 1 | 2 |", "a b 1 2"},
		{"схлопывание пробелов", "много\n\n\nстрок   и  пробелов", "много строк и пробелов"},
		{"комбинированный", "# Титул\n\n> цитата **жирная**\n\n- [пункт](http://a)\n", "Титул цитата жирная пункт"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := stripMarkdown(c.in); got != c.want {
				t.Fatalf("stripMarkdown(%q) = %q, ожидалось %q", c.in, got, c.want)
			}
		})
	}
}

// ── Непрочитанные посты (бейдж в навигации) ──────────────────────

func TestUnread_WithoutMarkCountsAllForeignPosts(t *testing.T) {
	svc, _, _ := newTestService()
	mustCreatePost(t, svc, 1, 10)
	mustCreatePost(t, svc, 1, 10)
	// Свой пост непрочитанным не считается.
	mustCreatePost(t, svc, 1, 20)

	n, err := svc.UnreadCount(context.Background(), 20, 1)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if n != 2 {
		t.Fatalf("без отметки ожидались все чужие посты (2), получено %d", n)
	}
}

func TestUnread_MarkSeenResetsThenNewPostCounts(t *testing.T) {
	svc, _, _ := newTestService()
	mustCreatePost(t, svc, 1, 10)
	mustCreatePost(t, svc, 1, 10)

	if err := svc.MarkSeen(context.Background(), 20, 1); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	n, err := svc.UnreadCount(context.Background(), 20, 1)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if n != 0 {
		t.Fatalf("после MarkSeen ожидалось 0, получено %d", n)
	}

	// Новый чужой пост после отметки — снова непрочитанный.
	mustCreatePost(t, svc, 1, 10)
	n, err = svc.UnreadCount(context.Background(), 20, 1)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if n != 1 {
		t.Fatalf("после нового поста ожидалось 1, получено %d", n)
	}
}

func TestForwardPost_CreateFailureCountsFailed(t *testing.T) {
	svc, _, _ := newTestService()
	msgr := svc.messenger.(*fakeMessenger)
	msgr.fail = map[int64]bool{100: true}

	p := mustCreatePost(t, svc, 1, 10)
	res, err := svc.ForwardPost(context.Background(), 1, p.ID, 10, []int64{100, 101}, nil)
	if err != nil {
		t.Fatalf("ForwardPost: %v", err)
	}
	if res.Forwarded != 1 || res.Failed != 1 {
		t.Fatalf("ожидалось forwarded=1 failed=1, получено %+v", res)
	}
}

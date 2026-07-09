package service

import (
	"context"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// ── Закрепление с автоистечением (pinned_until) ──────────────────

func TestPin_DaysSetsPinnedUntilAndClamps(t *testing.T) {
	svc, _, _ := newTestService()

	// days=0 — бессрочно.
	p := mustCreatePost(t, svc, 1, 10)
	got, err := svc.Pin(context.Background(), 1, p.ID, 10, domain.LevelEmployee, 0)
	if err != nil {
		t.Fatalf("Pin(days=0): %v", err)
	}
	if got.PinnedUntil != nil {
		t.Fatalf("days=0 должен давать бессрочный пин, получено %v", got.PinnedUntil)
	}

	// days=7 — примерно now+7 суток.
	p2 := mustCreatePost(t, svc, 1, 10)
	got, err = svc.Pin(context.Background(), 1, p2.ID, 10, domain.LevelEmployee, 7)
	if err != nil {
		t.Fatalf("Pin(days=7): %v", err)
	}
	want := time.Now().Add(7 * 24 * time.Hour)
	if got.PinnedUntil == nil || got.PinnedUntil.Sub(want).Abs() > time.Minute {
		t.Fatalf("ожидался pinned_until ≈ now+7д, получено %v", got.PinnedUntil)
	}

	// days=100 — кламп до MaxPinDays.
	p3 := mustCreatePost(t, svc, 1, 10)
	got, err = svc.Pin(context.Background(), 1, p3.ID, 10, domain.LevelEmployee, 100)
	if err != nil {
		t.Fatalf("Pin(days=100): %v", err)
	}
	want = time.Now().Add(MaxPinDays * 24 * time.Hour)
	if got.PinnedUntil == nil || got.PinnedUntil.Sub(want).Abs() > time.Minute {
		t.Fatalf("ожидался кламп до %d дней, получено %v", MaxPinDays, got.PinnedUntil)
	}
}

// Истёкший пин трактуется как незакреплённый: не попадает в pinned-секцию
// (уходит в хронологию) и не занимает слот лимита.
func TestPin_ExpiredNotInPinnedSectionAndFreesLimit(t *testing.T) {
	svc, repo, _ := newTestService()

	// 10 закреплённых, у одного срок уже истёк.
	var expired *domain.Post
	for i := 0; i < domain.MaxPinnedPosts; i++ {
		p := mustCreatePost(t, svc, 1, 10)
		if _, err := svc.Pin(context.Background(), 1, p.ID, 10, domain.LevelEmployee, 0); err != nil {
			t.Fatalf("Pin #%d: %v", i, err)
		}
		if i == 0 {
			expired = p
		}
	}
	past := time.Now().Add(-time.Hour)
	repo.posts[expired.ID].PinnedUntil = &past

	// Пин с истёкшим сроком — не в секции закреплённых, а в хронологии.
	feed, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{})
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(feed.Pinned) != domain.MaxPinnedPosts-1 {
		t.Fatalf("ожидалось %d актуально закреплённых, получено %d", domain.MaxPinnedPosts-1, len(feed.Pinned))
	}
	foundInChrono := false
	for _, p := range feed.Posts {
		if p.ID == expired.ID {
			foundInChrono = true
		}
	}
	if !foundInChrono {
		t.Fatalf("пост с истёкшим пином должен уйти в хронологию")
	}

	// Слот лимита освободился — новый пин проходит.
	extra := mustCreatePost(t, svc, 1, 10)
	if _, err := svc.Pin(context.Background(), 1, extra.ID, 10, domain.LevelEmployee, 0); err != nil {
		t.Fatalf("Pin при истёкшем слоте должен проходить: %v", err)
	}
}

// ── Keyset-пагинация ленты ───────────────────────────────────────

func TestListPosts_KeysetTwoPages(t *testing.T) {
	svc, _, _ := newTestService()
	var ids []int64
	for i := 0; i < 5; i++ {
		ids = append(ids, mustCreatePost(t, svc, 1, 10).ID)
	}

	page1, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{Limit: 2})
	if err != nil {
		t.Fatalf("страница 1: %v", err)
	}
	if len(page1.Posts) != 2 || page1.Posts[0].ID != ids[4] || page1.Posts[1].ID != ids[3] {
		t.Fatalf("страница 1: ожидались посты %d,%d, получено %+v", ids[4], ids[3], postIDs(page1.Posts))
	}
	if page1.NextCursor == nil {
		t.Fatalf("страница 1: ожидался next_cursor")
	}

	page2, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{Limit: 2, Cursor: *page1.NextCursor})
	if err != nil {
		t.Fatalf("страница 2: %v", err)
	}
	if len(page2.Posts) != 2 || page2.Posts[0].ID != ids[2] || page2.Posts[1].ID != ids[1] {
		t.Fatalf("страница 2: ожидались посты %d,%d, получено %+v", ids[2], ids[1], postIDs(page2.Posts))
	}
	if len(page2.Pinned) != 0 {
		t.Fatalf("pinned-секция только на первой странице")
	}

	page3, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{Limit: 2, Cursor: *page2.NextCursor})
	if err != nil {
		t.Fatalf("страница 3: %v", err)
	}
	if len(page3.Posts) != 1 || page3.Posts[0].ID != ids[0] || page3.NextCursor != nil {
		t.Fatalf("страница 3: ожидался хвост из 1 поста без курсора, получено %+v (cursor %v)",
			postIDs(page3.Posts), page3.NextCursor)
	}
}

// Новый пост, созданный МЕЖДУ выборками страниц, не сдвигает keyset-курсор:
// вторая страница не дублирует и не пропускает посты (в отличие от OFFSET).
func TestListPosts_CursorStableWhenNewPostInserted(t *testing.T) {
	svc, _, _ := newTestService()
	var ids []int64
	for i := 0; i < 4; i++ {
		ids = append(ids, mustCreatePost(t, svc, 1, 10).ID)
	}

	page1, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{Limit: 2})
	if err != nil {
		t.Fatalf("страница 1: %v", err)
	}

	// Свежий пост появился после выборки первой страницы.
	mustCreatePost(t, svc, 1, 10)

	page2, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{Limit: 2, Cursor: *page1.NextCursor})
	if err != nil {
		t.Fatalf("страница 2: %v", err)
	}
	if len(page2.Posts) != 2 || page2.Posts[0].ID != ids[1] || page2.Posts[1].ID != ids[0] {
		t.Fatalf("страница 2 после вставки: ожидались посты %d,%d без дублей/пропусков, получено %+v",
			ids[1], ids[0], postIDs(page2.Posts))
	}
}

func TestListPosts_BadCursorRejected(t *testing.T) {
	svc, _, _ := newTestService()
	if _, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{Cursor: "не курсор"}); err != domain.ErrBadCursor {
		t.Fatalf("ожидалась ErrBadCursor, получено %v", err)
	}
}

// Актуально закреплённые не дублируются в хронологии — только в секции Pinned.
func TestListPosts_PinnedExcludedFromChronology(t *testing.T) {
	svc, _, _ := newTestService()
	a := mustCreatePost(t, svc, 1, 10)
	mustCreatePost(t, svc, 1, 10)
	if _, err := svc.Pin(context.Background(), 1, a.ID, 10, domain.LevelEmployee, 0); err != nil {
		t.Fatalf("Pin: %v", err)
	}

	feed, err := svc.ListPosts(context.Background(), 1, 10, PostListParams{})
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(feed.Pinned) != 1 || feed.Pinned[0].ID != a.ID {
		t.Fatalf("ожидался 1 закреплённый пост %d, получено %+v", a.ID, postIDs(feed.Pinned))
	}
	for _, p := range feed.Posts {
		if p.ID == a.ID {
			t.Fatalf("закреплённый пост не должен дублироваться в хронологии")
		}
	}
}

func postIDs(posts []*domain.Post) []int64 {
	out := make([]int64, 0, len(posts))
	for _, p := range posts {
		out = append(out, p.ID)
	}
	return out
}

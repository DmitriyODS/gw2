package redisx

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type nopPub struct{ events []string }

func (p *nopPub) Publish(_ context.Context, event string, _ []string, _ any) {
	p.events = append(p.events, event)
}

func newChanges(t *testing.T) (*Changes, *miniredis.Miniredis, *time.Time) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := NewChanges(rdb, slog.New(slog.DiscardHandler))
	now := time.Now()
	c.now = func() time.Time { return now }
	return c, mr, &now
}

func TestSinceReturnsChangedAndGone(t *testing.T) {
	c, _, now := newChanges(t)
	ctx := context.Background()
	pub := Publisher{Inner: &nopPub{}, Changes: c}

	// Первый заход: эпохи нет — полная синхронизация, эпоха начинается.
	_, _, cursor, ok := c.Since(ctx, 1, 0)
	if ok {
		t.Fatal("без эпохи дельта невозможна")
	}

	*now = now.Add(time.Second)
	pub.Publish(ctx, "message:new", []string{"user_1", "user_2"}, map[string]any{"conversation_id": 10})
	pub.Publish(ctx, "conversation:deleted", []string{"user_1"}, map[string]any{"conversation_id": 11})
	pub.Publish(ctx, "folders:changed", []string{"user_1"}, map[string]any{"conversation_id": 12}) // не про список

	changed, removed, next, ok := c.Since(ctx, 1, cursor)
	if !ok || len(changed) != 1 || changed[0] != 10 || len(removed) != 1 || removed[0] != 11 {
		t.Fatalf("дельта: changed=%v removed=%v ok=%v", changed, removed, ok)
	}
	// Второму — только его диалог.
	if ch, _, _, _ := c.Since(ctx, 2, cursor); len(ch) != 1 || ch[0] != 10 {
		t.Fatalf("дельта второго: %v", ch)
	}
	// Ушедший и вернувшийся диалог — снова в изменённых, не в удалённых.
	*now = now.Add(time.Second)
	pub.Publish(ctx, "message:new", []string{"user_1"}, map[string]any{"conversation_id": 11})
	changed, removed, _, _ = c.Since(ctx, 1, next+1)
	if len(removed) != 0 || len(changed) != 1 || changed[0] != 11 {
		t.Fatalf("вернувшийся диалог: changed=%v removed=%v", changed, removed)
	}
}

// Журнал не ручается за полноту — только полная синхронизация.
func TestSinceRefusesWhenJournalIncomplete(t *testing.T) {
	c, mr, now := newChanges(t)
	ctx := context.Background()
	_, _, cursor, _ := c.Since(ctx, 1, 0)

	// Курсор старше окна хранения.
	*now = now.Add(Retention + time.Hour)
	if _, _, _, ok := c.Since(ctx, 1, cursor); ok {
		t.Fatal("курсор старше окна принят")
	}
	// Redis потерял данные — эпохи нет.
	_, _, cursor, _ = c.Since(ctx, 1, 0)
	mr.FlushAll()
	if _, _, _, ok := c.Since(ctx, 1, cursor); ok {
		t.Fatal("после сброса Redis дельта недостоверна")
	}
}

// Касание в ту же миллисекунду, что и курсор, не теряется.
func TestSinceCursorInclusive(t *testing.T) {
	c, _, _ := newChanges(t)
	ctx := context.Background()
	_, _, cursor, _ := c.Since(ctx, 1, 0)
	c.Touch(ctx, []int64{1}, 5, false) // время то же, что у курсора
	if ch, _, _, ok := c.Since(ctx, 1, cursor); !ok || len(ch) != 1 {
		t.Fatalf("касание на границе курсора потеряно: %v ok=%v", ch, ok)
	}
}

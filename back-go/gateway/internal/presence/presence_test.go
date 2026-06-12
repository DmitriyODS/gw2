package presence

// Тесты переходов онлайн/офлайн — поверх miniredis, без настоящей БД
// (LastSeenWriter — фейк). Семантика повторяет прежний Flask-presence:
// онлайн = хотя бы одно живое видимое соединение; событие только на переходе.

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type busEvent struct {
	Event   string
	Rooms   []string
	Payload map[string]any
}

type fakeBus struct{ events []busEvent }

func (b *fakeBus) Publish(_ context.Context, event string, rooms []string, payload any) {
	b.events = append(b.events, busEvent{event, rooms, payload.(map[string]any)})
}

func (b *fakeBus) last() *busEvent {
	if len(b.events) == 0 {
		return nil
	}
	return &b.events[len(b.events)-1]
}

type fakeLastSeen struct{ written []int64 }

func (f *fakeLastSeen) SetLastSeen(_ context.Context, userID int64, _ time.Time) error {
	f.written = append(f.written, userID)
	return nil
}

func newTestPresence(t *testing.T) (*Presence, *fakeBus, *fakeLastSeen, *time.Time) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	bus := &fakeBus{}
	seen := &fakeLastSeen{}
	p := New(rdb, seen, bus, slog.New(slog.DiscardHandler))
	now := time.Now()
	p.now = func() time.Time { return now }
	return p, bus, seen, &now
}

func TestConnectGoesOnlineOnce(t *testing.T) {
	p, bus, _, _ := newTestPresence(t)
	ctx := context.Background()

	p.OnConnect(ctx, 7, "c1")
	if len(bus.events) != 1 || bus.events[0].Event != "presence:update" {
		t.Fatalf("events = %+v", bus.events)
	}
	ev := bus.events[0]
	if ev.Payload["online"] != true || ev.Payload["user_id"] != int64(7) ||
		ev.Payload["last_seen_at"] != nil {
		t.Fatalf("payload = %v", ev.Payload)
	}
	if len(ev.Rooms) != 1 || ev.Rooms[0] != "all" {
		t.Fatalf("rooms = %v", ev.Rooms)
	}

	// Вторая вкладка — события нет (не на переходе).
	p.OnConnect(ctx, 7, "c2")
	if len(bus.events) != 1 {
		t.Fatalf("спам presence:update: %+v", bus.events)
	}
}

func TestDisconnectLastConnectionGoesOffline(t *testing.T) {
	p, bus, seen, _ := newTestPresence(t)
	ctx := context.Background()

	p.OnConnect(ctx, 7, "c1")
	p.OnConnect(ctx, 7, "c2")
	p.OnDisconnect(ctx, 7, "c1")
	if bus.last().Payload["online"] != true {
		t.Fatal("офлайн при живой второй вкладке")
	}
	p.OnDisconnect(ctx, 7, "c2")
	last := bus.last()
	if last.Payload["online"] != false || last.Payload["last_seen_at"] == nil {
		t.Fatalf("payload = %v", last.Payload)
	}
	if len(seen.written) != 1 || seen.written[0] != 7 {
		t.Fatalf("last_seen written = %v", seen.written)
	}
}

func TestVisibilityHiddenGoesOffline(t *testing.T) {
	p, bus, _, _ := newTestPresence(t)
	ctx := context.Background()

	p.OnConnect(ctx, 7, "c1")
	p.OnVisibility(ctx, 7, "c1", false)
	if bus.last().Payload["online"] != false {
		t.Fatal("скрытая единственная вкладка должна давать офлайн")
	}
	// Heartbeat возвращает в строй.
	p.OnHeartbeat(ctx, 7, "c1")
	if bus.last().Payload["online"] != true {
		t.Fatal("heartbeat не вернул онлайн")
	}
}

func TestSweepDropsStaleConnections(t *testing.T) {
	p, bus, seen, now := newTestPresence(t)
	ctx := context.Background()

	p.OnConnect(ctx, 7, "c1")
	p.OnConnect(ctx, 8, "c2")

	// 7 молчит дольше StaleAfter, 8 шлёт heartbeat.
	*now = now.Add(StaleAfter / 2)
	p.OnHeartbeat(ctx, 8, "c2")
	*now = now.Add(StaleAfter/2 + 2*time.Second)
	p.SweepOnce(ctx)

	offline := map[int64]bool{}
	for _, ev := range bus.events {
		if ev.Payload["online"] == false {
			offline[ev.Payload["user_id"].(int64)] = true
		}
	}
	if !offline[7] || offline[8] {
		t.Fatalf("offline = %v (ожидался только 7)", offline)
	}
	if len(seen.written) != 1 || seen.written[0] != 7 {
		t.Fatalf("last_seen = %v", seen.written)
	}

	ids, err := p.OnlineUserIDs(ctx)
	if err != nil || len(ids) != 1 || ids[0] != 8 {
		t.Fatalf("online = %v, err = %v", ids, err)
	}
}

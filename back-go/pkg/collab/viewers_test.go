package collab

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestViewersLifecycle(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	v := New(rdb, "gw2:test:viewers:")
	now := time.Now()
	v.now = func() time.Time { return now }
	ctx := context.Background()

	_ = v.Touch(ctx, 1, 10)
	_ = v.Touch(ctx, 1, 20)
	_ = v.Touch(ctx, 2, 30) // другой документ
	if got, _ := v.List(ctx, 1); len(got) != 2 {
		t.Fatalf("зрители = %v", got)
	}

	_ = v.Leave(ctx, 1, 20)
	// 10 замолчал дольше окна — ушёл без leave (закрыли вкладку).
	now = now.Add(TTL + time.Second)
	if got, _ := v.List(ctx, 1); len(got) != 0 {
		t.Fatalf("после leave и таймаута остались %v", got)
	}
	_ = v.Touch(ctx, 1, 10)
	if got, _ := v.List(ctx, 1); len(got) != 1 || got[0] != 10 {
		t.Fatalf("вернувшийся зритель: %v", got)
	}
}

func TestAccessCacheExpires(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	v := New(rdb, "gw2:test:viewers:")
	ctx := context.Background()

	if _, _, ok := v.CachedAccess(ctx, 1, 10); ok {
		t.Fatal("доступ в кэше до проверки")
	}
	v.RememberAccess(ctx, 1, 10, 3, "edit")
	if access, owner, ok := v.CachedAccess(ctx, 1, 10); !ok || access != "edit" || owner != 3 {
		t.Fatalf("кэш: %q %d %v", access, owner, ok)
	}
	mr.FastForward(TTL + time.Second)
	if _, _, ok := v.CachedAccess(ctx, 1, 10); ok {
		t.Fatal("доступ пережил окно — отзыв шары не сработал бы")
	}
}

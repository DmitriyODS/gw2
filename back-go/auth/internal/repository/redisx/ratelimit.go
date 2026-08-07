// Package redisx — RateLimiter: общий лимит "не больше N за фиксированное
// окно" по произвольному ключу (Redis INCR + EXPIRE на первый инкремент).
// В отличие от LoginThrottle (эскалирующая блокировка одного логина) — плоский
// счётчик без нарастающего бэкоффа, годится для IP-лимита регистрации и
// суточного капа на резенды писем. Redis недоступен → fail-open: пропускаем,
// только логируем (как и у LoginThrottle — это щит от автоматического
// злоупотребления при обычной эксплуатации, а не от целевой атаки на Redis).
package redisx

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

type RateLimiter struct {
	rdb *redis.Client
	log *slog.Logger
}

func NewRateLimiter(rdb *redis.Client, log *slog.Logger) *RateLimiter {
	return &RateLimiter{rdb: rdb, log: log}
}

var _ domain.RateLimiter = (*RateLimiter)(nil)

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int) {
	full := "gw2:rl:" + key
	n, err := r.rdb.Incr(ctx, full).Result()
	if err != nil {
		r.log.Warn("ratelimit.redis_failed", "key", key, "error", err)
		return true, 0
	}
	if n == 1 {
		r.rdb.Expire(ctx, full, window)
	}
	if n <= int64(limit) {
		return true, 0
	}
	ttl, err := r.rdb.TTL(ctx, full).Result()
	if err != nil || ttl <= 0 {
		return false, int(window.Seconds())
	}
	return false, int(ttl.Seconds())
}

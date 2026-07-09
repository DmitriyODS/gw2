// Package redis — Redis-зависимости notesvc: троттлинг анонимных правок по
// коду публичной ссылки.
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// WriteLimiter — скользящее окно фиксированной минуты: INCR по ключу кода с
// TTL 60 с; свыше лимита — отказ. Fail-open: недоступный Redis не блокирует
// правки (вандализм по ссылке — меньший риск, чем отказ всего шаринга).
type WriteLimiter struct {
	rdb   *redis.Client
	limit int64
}

var _ domain.WriteLimiter = (*WriteLimiter)(nil)

func NewWriteLimiter(rdb *redis.Client, limit int64) *WriteLimiter {
	return &WriteLimiter{rdb: rdb, limit: limit}
}

func (l *WriteLimiter) Allow(ctx context.Context, code string) bool {
	key := "gw2:notes:shared_rl:" + code
	n, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true
	}
	if n == 1 {
		l.rdb.Expire(ctx, key, time.Minute)
	}
	return n <= l.limit
}

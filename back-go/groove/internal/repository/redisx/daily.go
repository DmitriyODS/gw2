// Package redisx — дневные счётчики и кэши Groove в Redis.
//
// ВСЁ fail-open: Redis лёг — капы не применяются (начисляем без лимита,
// лучше так, чем ронять), кэши пустые. Ключи 1-в-1 совпадают с прежними
// ключами Flask — данные переживают переезд.
package redisx

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

type Daily struct {
	rdb *redis.Client
	log *slog.Logger
}

var _ domain.Daily = (*Daily)(nil)

func New(rdb *redis.Client, log *slog.Logger) *Daily {
	return &Daily{rdb: rdb, log: log}
}

func todayMSK() string {
	return time.Now().In(domain.MSK).Format("2006-01-02")
}

func dailyKey(userID int64) string {
	return "gw2:groove:daily:" + strconv.FormatInt(userID, 10) + ":" + todayMSK()
}

// TakeBudget — сколько из want ещё помещается в дневной кап источника.
// Атомарность как во Flask: read-then-incr (гонка двух запросов одного
// пользователя редка и не критична для геймификации).
func (d *Daily) TakeBudget(ctx context.Context, userID int64, source string, want, cap int) int {
	if want <= 0 {
		return 0
	}
	key := dailyKey(userID)
	used, err := d.rdb.HGet(ctx, key, source).Int()
	if err != nil && err != redis.Nil {
		return want // Redis лёг — не наказываем пользователя
	}
	granted := min(want, cap-used)
	if granted <= 0 {
		return 0
	}
	if err := d.rdb.HIncrBy(ctx, key, source, int64(granted)).Err(); err != nil {
		return want
	}
	d.rdb.Expire(ctx, key, 48*time.Hour)
	return granted
}

func (d *Daily) Left(ctx context.Context, userID int64, source string, cap int) int {
	used, err := d.rdb.HGet(ctx, dailyKey(userID), source).Int()
	if err != nil {
		used = 0
	}
	return max(0, cap-used)
}

// ─────────────────────────── кэши ──────────────────────────────────

func (d *Daily) GetCache(ctx context.Context, key string) string {
	val, err := d.rdb.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return val
}

func (d *Daily) SetCache(ctx context.Context, key, value string, ttl time.Duration) {
	if err := d.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		d.log.Warn("redis.set_failed", "key", key, "error", err)
	}
}

func (d *Daily) Exists(ctx context.Context, key string) bool {
	n, err := d.rdb.Exists(ctx, key).Result()
	return err == nil && n > 0
}

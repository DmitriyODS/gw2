// Package redisx — защита от подбора пароля (порт back/app/services/login_throttle.py).
//
// После каждых 5 подряд неудачных попыток входа логин блокируется на
// экспоненциально растущее время: 10с, 20с, 40с… Удачный вход сбрасывает
// счётчик. Ключи и семантика прежние (gw2:bf:*), TTL счётчика — сутки.
// Redis недоступен → fail-open: вход не блокируем, лишь логируем.
package redisx

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

const (
	initialDelaySec = 10
	lockEveryNFails = 5
	attemptsTTL     = 24 * time.Hour
)

type LoginThrottle struct {
	rdb *redis.Client
	log *slog.Logger
}

func NewLoginThrottle(rdb *redis.Client, log *slog.Logger) *LoginThrottle {
	return &LoginThrottle{rdb: rdb, log: log}
}

var _ domain.LoginThrottle = (*LoginThrottle)(nil)

func normalize(login string) string { return strings.ToLower(strings.TrimSpace(login)) }

func attemptsKey(login string) string { return "gw2:bf:attempts:" + normalize(login) }
func lockKey(login string) string     { return "gw2:bf:locked_until:" + normalize(login) }

func (t *LoginThrottle) LockRemaining(ctx context.Context, login string) int {
	if login == "" {
		return 0
	}
	raw, err := t.rdb.Get(ctx, lockKey(login)).Result()
	if err != nil {
		if err != redis.Nil {
			t.log.Warn("throttle.redis_failed", "op", "lock_remaining", "error", err)
		}
		return 0
	}
	until, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	remaining := int(math.Ceil(until - float64(time.Now().UnixNano())/1e9))
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (t *LoginThrottle) RegisterFailure(ctx context.Context, login string) int {
	if login == "" {
		return 0
	}
	akey := attemptsKey(login)
	attempts, err := t.rdb.Incr(ctx, akey).Result()
	if err != nil {
		t.log.Warn("throttle.redis_failed", "op", "register_failure", "error", err)
		return 0
	}
	t.rdb.Expire(ctx, akey, attemptsTTL)

	if attempts%lockEveryNFails != 0 {
		return 0
	}
	steps := attempts / lockEveryNFails
	delay := initialDelaySec * (1 << (steps - 1)) // 10, 20, 40, 80…
	until := float64(time.Now().UnixNano())/1e9 + float64(delay)
	t.rdb.Set(ctx, lockKey(login), fmt.Sprintf("%f", until), time.Duration(delay+5)*time.Second)
	return int(delay)
}

func (t *LoginThrottle) RegisterSuccess(ctx context.Context, login string) {
	if login == "" {
		return
	}
	if err := t.rdb.Del(ctx, attemptsKey(login), lockKey(login)).Err(); err != nil {
		t.log.Warn("throttle.redis_failed", "op", "register_success", "error", err)
	}
}

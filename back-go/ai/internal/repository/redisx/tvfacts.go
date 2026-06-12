// Package redisx — кэш ТВ-фактов дня (порт back/app/services/tv_facts_service.py).
//
// Ключи gw2:ai:tv_fact:{company_id} сохранены с Flask-времён: значение —
// JSON {generated_at, kind, text}, TTL вдвое больше тика генерации (если
// тик пропустится, на табло остаётся прошлый факт, а не фолбэк).
package redisx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

type FactCache struct {
	rdb *redis.Client
	log *slog.Logger
}

func NewFactCache(rdb *redis.Client, log *slog.Logger) *FactCache {
	return &FactCache{rdb: rdb, log: log}
}

var _ domain.FactCache = (*FactCache)(nil)

func factKey(companyID int64) string {
	return fmt.Sprintf("gw2:ai:tv_fact:%d", companyID)
}

func (c *FactCache) GetFact(ctx context.Context, companyID int64) (*domain.TVFact, error) {
	raw, err := c.rdb.Get(ctx, factKey(companyID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	var fact domain.TVFact
	if err := json.Unmarshal([]byte(raw), &fact); err != nil {
		// Битый JSON — как json.JSONDecodeError во Flask: молча None.
		return nil, nil
	}
	return &fact, nil
}

func (c *FactCache) SetFact(ctx context.Context, companyID int64, fact *domain.TVFact, ttl time.Duration) error {
	raw, err := json.Marshal(fact)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, factKey(companyID), raw, ttl).Err()
}

// DeleteFact — затереть факт (AI у компании выключили). Ошибки глотаем,
// как try/except вокруг delete во Flask.
func (c *FactCache) DeleteFact(ctx context.Context, companyID int64) {
	if err := c.rdb.Del(ctx, factKey(companyID)).Err(); err != nil {
		c.log.Warn("ai.tv_facts.redis_del_failed", "company_id", companyID, "error", err)
	}
}

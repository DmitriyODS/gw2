package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// GeoCache — кэш «IP → город» для карточек сеансов: один и тот же адрес не
// гоняется во внешний гео-сервис при каждом входе. Пустое значение тоже
// кэшируется (адрес, который провайдер не знает), поэтому маркер отсутствия —
// сам факт наличия ключа, а не непустая строка.
type GeoCache struct {
	rdb *redis.Client
	ttl time.Duration
}

const geoCacheTTL = 30 * 24 * time.Hour

func NewGeoCache(rdb *redis.Client) *GeoCache { return &GeoCache{rdb: rdb, ttl: geoCacheTTL} }

func geoKey(ip string) string { return "gw2:geoip:" + ip }

func (c *GeoCache) Get(ctx context.Context, ip string) (string, bool) {
	city, err := c.rdb.Get(ctx, geoKey(ip)).Result()
	if err != nil {
		return "", false
	}
	return city, true
}

func (c *GeoCache) Set(ctx context.Context, ip, city string) {
	_ = c.rdb.Set(ctx, geoKey(ip), city, c.ttl).Err()
}

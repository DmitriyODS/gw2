// Спаривание устройств (QR-вход и ТВ-код): pending-сессии в Redis
// (ключи gw2:devlink:*). Короткоживущие, одноразовые — значение JSON от
// domain.DeviceLink, TTL задаёт сервис.
package redisx

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

type DeviceLinkStore struct {
	rdb *redis.Client
}

func NewDeviceLinkStore(rdb *redis.Client) *DeviceLinkStore { return &DeviceLinkStore{rdb: rdb} }

var _ domain.DeviceLinkStore = (*DeviceLinkStore)(nil)

func devLinkKey(code string) string { return "gw2:devlink:" + code }

func (s *DeviceLinkStore) Save(ctx context.Context, code string, dl domain.DeviceLink, ttl time.Duration) error {
	b, err := json.Marshal(dl)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, devLinkKey(code), b, ttl).Err()
}

func (s *DeviceLinkStore) Get(ctx context.Context, code string) (*domain.DeviceLink, error) {
	raw, err := s.rdb.Get(ctx, devLinkKey(code)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var dl domain.DeviceLink
	if err := json.Unmarshal(raw, &dl); err != nil {
		return nil, err
	}
	return &dl, nil
}

func (s *DeviceLinkStore) Delete(ctx context.Context, code string) error {
	return s.rdb.Del(ctx, devLinkKey(code)).Err()
}

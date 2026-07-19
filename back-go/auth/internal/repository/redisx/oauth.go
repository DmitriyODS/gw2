package redisx

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// OAuthCodeStore — одноразовые коды согласия OAuth (связка аккаунтов Алисы)
// в Redis: TTL истёк или код уже забрали — Pop вернёт nil.
type OAuthCodeStore struct {
	rdb *redis.Client
}

func NewOAuthCodeStore(rdb *redis.Client) *OAuthCodeStore { return &OAuthCodeStore{rdb: rdb} }

func oauthCodeKey(code string) string { return "gw2:oauth:code:" + code }

func (s *OAuthCodeStore) Save(ctx context.Context, code string, oc domain.OAuthCode, ttl time.Duration) error {
	raw, err := json.Marshal(oc)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, oauthCodeKey(code), raw, ttl).Err()
}

// Pop — атомарно забрать и удалить код (GETDEL): каждый код одноразовый.
func (s *OAuthCodeStore) Pop(ctx context.Context, code string) (*domain.OAuthCode, error) {
	raw, err := s.rdb.GetDel(ctx, oauthCodeKey(code)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var oc domain.OAuthCode
	if err := json.Unmarshal([]byte(raw), &oc); err != nil {
		return nil, err
	}
	return &oc, nil
}

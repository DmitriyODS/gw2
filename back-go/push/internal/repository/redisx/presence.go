// Package redisx — чтение presence из Redis (тот же SET, что пишет gatewaysvc).
package redisx

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// onlineKey — SET онлайн-пользователей (gatewaysvc: gw2:presence:online).
const onlineKey = "gw2:presence:online"

type Presence struct {
	rdb *redis.Client
}

func NewPresence(rdb *redis.Client) *Presence { return &Presence{rdb: rdb} }

// Offline — подмножество ids, которых нет в онлайне. При ошибке Redis —
// fail-open: считаем всех офлайн (лучше лишний пуш, чем пропущенный).
func (p *Presence) Offline(ctx context.Context, ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	members := make([]string, len(ids))
	for i, id := range ids {
		members[i] = strconv.FormatInt(id, 10)
	}
	res, err := p.rdb.SMIsMember(ctx, onlineKey, toAny(members)...).Result()
	if err != nil || len(res) != len(ids) {
		return ids, nil
	}
	out := make([]int64, 0, len(ids))
	for i, online := range res {
		if !online {
			out = append(out, ids[i])
		}
	}
	return out, nil
}

func toAny(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

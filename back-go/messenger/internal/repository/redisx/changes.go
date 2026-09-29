// Package redisx — журнал изменений списка диалогов для дельта-синхронизации.
//
// Каждое изменение, которое клиент видит в списке, msgsvc и так публикует
// сокет-событием в user_{id} с conversation_id — на этом держится realtime.
// Журнал пишется в ТОЙ ЖЕ точке (декоратор публикатора), поэтому дельта ровно
// так же полна, как realtime, и не зависит от того, помнит ли очередной путь
// изменения «отметить диалог».
//
// Ключи: gw2:msg:changed:<uid> и gw2:msg:gone:<uid> — ZSET (member — id
// диалога, score — момент в мс), gw2:msg:changes_epoch — с какого момента
// журнал полон. Эпоха пропадает вместе с Redis (перезапуск без диска, чистка)
// и сбрасывается при ошибке записи — тогда клиент получает полный список, а не
// дельту с дырой.
package redisx

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

const (
	epochKey = "gw2:msg:changes_epoch"
	// Retention — сколько журнал помнит изменения. Клиент, проспавший дольше,
	// получает полный список.
	Retention = 7 * 24 * time.Hour
)

// listEvents — события, меняющие строку списка диалогов; deleted — уход
// диалога из списка.
var listEvents = map[string]bool{
	"message:new": true, "message:updated": true, "message:deleted": true,
	"message:read": true, "conversation:pin": true, "group:updated": true,
	"conversation:deleted": true,
}

type Changes struct {
	rdb *redis.Client
	log *slog.Logger
	now func() time.Time
}

func NewChanges(rdb *redis.Client, log *slog.Logger) *Changes {
	return &Changes{rdb: rdb, log: log, now: time.Now}
}

var _ domain.ConversationChanges = (*Changes)(nil)

func changedKey(uid int64) string { return "gw2:msg:changed:" + strconv.FormatInt(uid, 10) }
func goneKey(uid int64) string    { return "gw2:msg:gone:" + strconv.FormatInt(uid, 10) }

// Touch — диалог изменился для пользователей (gone — ушёл из их списка).
func (c *Changes) Touch(ctx context.Context, userIDs []int64, convID int64, gone bool) {
	if len(userIDs) == 0 || convID <= 0 {
		return
	}
	now := c.now()
	ms := float64(now.UnixMilli())
	stale := strconv.FormatInt(now.Add(-Retention).UnixMilli(), 10)
	pipe := c.rdb.Pipeline()
	pipe.SetNX(ctx, epochKey, now.UnixMilli(), 0)
	for _, uid := range userIDs {
		add, drop := changedKey(uid), goneKey(uid)
		if gone {
			add, drop = drop, add
		}
		pipe.ZAdd(ctx, add, redis.Z{Score: ms, Member: convID})
		pipe.ZRem(ctx, drop, convID)
		pipe.ZRemRangeByScore(ctx, add, "-inf", stale)
		pipe.Expire(ctx, add, Retention)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		// Касание потеряно — журнал больше не полон: сбрасываем эпоху, и
		// следующая синхронизация каждого клиента будет полной.
		c.log.Warn("messenger.changes_touch_failed", "conversation_id", convID, "error", err)
		_ = c.rdb.Del(ctx, epochKey).Err()
	}
}

// Since — что изменилось у пользователя после since (мс) и курсор для
// следующего раза. ok=false — журнал не может поручиться за полноту.
func (c *Changes) Since(ctx context.Context, userID, since int64) (changed, removed []int64, cursor int64, ok bool) {
	now := c.now()
	cursor = now.UnixMilli()
	epoch, err := c.rdb.Get(ctx, epochKey).Int64()
	if err != nil {
		// Эпохи нет (свежий Redis, сброс) — начинаем её сейчас: следующая
		// синхронизация уже сможет быть дельтой.
		_ = c.rdb.SetNX(ctx, epochKey, cursor, 0).Err()
		return nil, nil, cursor, false
	}
	if since <= 0 || since < epoch || since < now.Add(-Retention).UnixMilli() {
		return nil, nil, cursor, false
	}
	// Граница включающая: курсор взят ДО сборки списка, и касание в ту же
	// миллисекунду после неё иначе потерялось бы. Повтор строки клиенту
	// безвреден — он применяет её тем же upsert.
	min := strconv.FormatInt(since, 10)
	pipe := c.rdb.Pipeline()
	ch := pipe.ZRangeByScore(ctx, changedKey(userID), &redis.ZRangeBy{Min: min, Max: "+inf"})
	gn := pipe.ZRangeByScore(ctx, goneKey(userID), &redis.ZRangeBy{Min: min, Max: "+inf"})
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, nil, cursor, false
	}
	return parseIDs(ch.Val()), parseIDs(gn.Val()), cursor, true
}

func parseIDs(raw []string) []int64 {
	out := make([]int64, 0, len(raw))
	for _, s := range raw {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// Publisher — декоратор публикатора: событие уходит как раньше, а касание
// списка пишется в журнал тем же вызовом.
type Publisher struct {
	Inner   domain.EventPublisher
	Changes *Changes
}

func (p Publisher) Publish(ctx context.Context, event string, rooms []string, payload any) {
	p.Inner.Publish(ctx, event, rooms, payload)
	if !listEvents[event] {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var body struct {
		ConversationID int64 `json:"conversation_id"`
	}
	if json.Unmarshal(raw, &body) != nil || body.ConversationID <= 0 {
		return
	}
	p.Changes.Touch(ctx, usersOf(rooms), body.ConversationID, event == "conversation:deleted")
}

func usersOf(rooms []string) []int64 {
	out := make([]int64, 0, len(rooms))
	for _, r := range rooms {
		if id, err := strconv.ParseInt(strings.TrimPrefix(r, "user_"), 10, 64); err == nil && strings.HasPrefix(r, "user_") {
			out = append(out, id)
		}
	}
	return out
}

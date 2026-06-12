// Package presence — учёт присутствия (онлайн-статус) пользователей.
//
// Порт прежнего back/app/sockets/presence.py с переносом состояния в Redis —
// это снимает ограничение «один процесс»: всё состояние соединений живёт
// в общих ключах, событие presence:update уходит через Redis-канал шлюза
// и доставляется клиентам любого инстанса.
//
// Пользователь «онлайн», пока у него есть хотя бы одно соединение с видимой
// вкладкой И недавним heartbeat'ом. Почему heartbeat, а не одна видимость:
// на мобильных (особенно iOS Safari) при сворачивании сокет «замораживается»
// — дисконнект приходит с большой задержкой или теряется. Клиент шлёт
// presence:heartbeat каждые ~25с, пока вкладка видима; sweeper раз в
// SweepInterval опускает в офлайн тех, от кого сигналов не было дольше
// StaleAfter. last_seen_at пишется в users на переходе в офлайн.
//
// Ключи Redis:
//   gw2:presence:beats  — ZSET, member "uid:connID", score — unix-время
//                         последнего сигнала живой видимой вкладки;
//   gw2:presence:online — SET онлайн-пользователей (для переходов и REST).
package presence

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	beatsKey  = "gw2:presence:beats"
	onlineKey = "gw2:presence:online"

	// SweepInterval / StaleAfter — те же, что во Flask-presence.
	SweepInterval = 15 * time.Second
	StaleAfter    = 60 * time.Second

	roomAll = "all"
)

// LastSeenWriter — запись users.last_seen_at (pgx-пул).
type LastSeenWriter interface {
	SetLastSeen(ctx context.Context, userID int64, at time.Time) error
}

// PGLastSeen — стандартная реализация поверх общей PostgreSQL.
type PGLastSeen struct{ Pool *pgxpool.Pool }

func (w PGLastSeen) SetLastSeen(ctx context.Context, userID int64, at time.Time) error {
	_, err := w.Pool.Exec(ctx,
		`UPDATE users SET last_seen_at = $2 WHERE id = $1`, userID, at)
	return err
}

// Bus — публикация presence:update (events.Publisher gateway-канала).
type Bus interface {
	Publish(ctx context.Context, event string, rooms []string, payload any)
}

type Presence struct {
	rdb      *redis.Client
	lastSeen LastSeenWriter
	bus      Bus
	log      *slog.Logger
	now      func() time.Time
}

func New(rdb *redis.Client, lastSeen LastSeenWriter, bus Bus, log *slog.Logger) *Presence {
	return &Presence{rdb: rdb, lastSeen: lastSeen, bus: bus, log: log, now: time.Now}
}

func member(userID int64, connID string) string {
	return strconv.FormatInt(userID, 10) + ":" + connID
}

func memberUserID(m string) int64 {
	idx := strings.IndexByte(m, ':')
	if idx <= 0 {
		return 0
	}
	id, _ := strconv.ParseInt(m[:idx], 10, 64)
	return id
}

// isoUTC — datetime.isoformat() Python: микросекунды опускаются, если 0.
func isoUTC(t time.Time) string {
	u := t.UTC()
	s := u.Format("2006-01-02T15:04:05")
	if us := u.Nanosecond() / 1000; us != 0 {
		s += fmt.Sprintf(".%06d", us)
	}
	return s + "+00:00"
}

// beat — пометить соединение живым и видимым.
func (p *Presence) beat(ctx context.Context, userID int64, connID string) {
	if err := p.rdb.ZAdd(ctx, beatsKey, redis.Z{
		Score:  float64(p.now().Unix()),
		Member: member(userID, connID),
	}).Err(); err != nil {
		p.log.Warn("presence.beat_failed", "user_id", userID, "error", err)
		return
	}
	p.setOnline(ctx, userID)
}

// drop — соединение больше не видимо/живо.
func (p *Presence) drop(ctx context.Context, userID int64, connID string) {
	if err := p.rdb.ZRem(ctx, beatsKey, member(userID, connID)).Err(); err != nil {
		p.log.Warn("presence.drop_failed", "user_id", userID, "error", err)
		return
	}
	p.maybeOffline(ctx, userID)
}

func (p *Presence) OnConnect(ctx context.Context, userID int64, connID string) {
	p.beat(ctx, userID, connID)
}

func (p *Presence) OnDisconnect(ctx context.Context, userID int64, connID string) {
	p.drop(ctx, userID, connID)
}

// OnVisibility — клиент сообщил, что его вкладка стала видимой/скрытой.
func (p *Presence) OnVisibility(ctx context.Context, userID int64, connID string, visible bool) {
	if visible {
		p.beat(ctx, userID, connID)
	} else {
		p.drop(ctx, userID, connID)
	}
}

// OnHeartbeat — регулярный пинг живой видимой вкладки (возвращает в строй
// соединение, опущенное sweeper'ом).
func (p *Presence) OnHeartbeat(ctx context.Context, userID int64, connID string) {
	p.beat(ctx, userID, connID)
}

// setOnline — переход в онлайн; событие только на переходе, чтобы не спамить
// presence:update.
func (p *Presence) setOnline(ctx context.Context, userID int64) {
	added, err := p.rdb.SAdd(ctx, onlineKey, userID).Result()
	if err != nil {
		p.log.Warn("presence.online_failed", "user_id", userID, "error", err)
		return
	}
	if added == 0 {
		return
	}
	p.bus.Publish(ctx, "presence:update", []string{roomAll}, map[string]any{
		"user_id": userID, "online": true, "last_seen_at": nil,
	})
}

// maybeOffline — если живых видимых соединений не осталось, выставить офлайн
// с актуальным last_seen.
func (p *Presence) maybeOffline(ctx context.Context, userID int64) {
	alive, err := p.aliveUsers(ctx)
	if err != nil {
		p.log.Warn("presence.alive_failed", "error", err)
		return
	}
	if alive[userID] {
		return
	}
	p.setOffline(ctx, userID)
}

func (p *Presence) setOffline(ctx context.Context, userID int64) {
	removed, err := p.rdb.SRem(ctx, onlineKey, userID).Result()
	if err != nil {
		p.log.Warn("presence.offline_failed", "user_id", userID, "error", err)
		return
	}
	if removed == 0 {
		return
	}
	now := p.now().UTC()
	if err := p.lastSeen.SetLastSeen(ctx, userID, now); err != nil {
		p.log.Warn("presence.last_seen_failed", "user_id", userID, "error", err)
	}
	p.bus.Publish(ctx, "presence:update", []string{roomAll}, map[string]any{
		"user_id": userID, "online": false, "last_seen_at": isoUTC(now),
	})
}

// aliveUsers — пользователи с хотя бы одним свежим видимым соединением.
func (p *Presence) aliveUsers(ctx context.Context) (map[int64]bool, error) {
	minScore := strconv.FormatInt(p.now().Add(-StaleAfter).Unix(), 10)
	members, err := p.rdb.ZRangeByScore(ctx, beatsKey, &redis.ZRangeBy{
		Min: "(" + minScore, Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}
	out := map[int64]bool{}
	for _, m := range members {
		if id := memberUserID(m); id > 0 {
			out[id] = true
		}
	}
	return out, nil
}

// OnlineUserIDs — снимок онлайн-пользователей (REST /api/messenger/presence).
func (p *Presence) OnlineUserIDs(ctx context.Context) ([]int64, error) {
	raw, err := p.rdb.SMembers(ctx, onlineKey).Result()
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(raw))
	for _, s := range raw {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

// SweepOnce — один прогон: вычистить просроченные соединения и опустить в
// офлайн пользователей, у которых не осталось живых видимых вкладок
// (включая «осиротевших» после рестарта шлюза).
func (p *Presence) SweepOnce(ctx context.Context) {
	maxScore := strconv.FormatInt(p.now().Add(-StaleAfter).Unix(), 10)
	if err := p.rdb.ZRemRangeByScore(ctx, beatsKey, "-inf", maxScore).Err(); err != nil {
		p.log.Warn("presence.sweep_trim_failed", "error", err)
		return
	}
	alive, err := p.aliveUsers(ctx)
	if err != nil {
		p.log.Warn("presence.sweep_failed", "error", err)
		return
	}
	online, err := p.OnlineUserIDs(ctx)
	if err != nil {
		p.log.Warn("presence.sweep_failed", "error", err)
		return
	}
	for _, uid := range online {
		if !alive[uid] {
			p.setOffline(ctx, uid)
		}
	}
}

// RunSweeper — фоновый цикл; завершается по ctx.
func (p *Presence) RunSweeper(ctx context.Context) {
	ticker := time.NewTicker(SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.SweepOnce(ctx)
		}
	}
}

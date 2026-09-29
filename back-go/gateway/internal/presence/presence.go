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
// Событие уходит не всей платформе, а АУДИТОРИИ пользователя (Audience):
// коллегам по общим компаниям, собеседникам и супер-админам. Широковещание
// росло как N² и будило телефоны из-за входа и выхода незнакомых людей.
//
// Ключи Redis:
//
//	gw2:presence:beats     — ZSET, member "uid:connID", score — unix-время
//	                         последнего сигнала живой видимой вкладки (sweeper);
//	gw2:presence:conns:uid — HASH connID → то же время: «жив ли человек»
//	                         проверяется по его соединениям, а не по всему ZSET;
//	gw2:presence:online    — SET онлайн-пользователей (для переходов и REST).
package presence

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/pkg/events"
)

const (
	beatsKey    = "gw2:presence:beats"
	onlineKey   = "gw2:presence:online"
	connsPrefix = "gw2:presence:conns:"

	// SweepInterval / StaleAfter — те же, что во Flask-presence.
	SweepInterval = 15 * time.Second
	StaleAfter    = 60 * time.Second

	// audienceTTL — сколько живёт снимок аудитории: переходы онлайн/офлайн
	// частые (каждое сворачивание вкладки), а круг общения меняется редко.
	audienceTTL = time.Minute
)

// Audience — круг людей, которым виден онлайн-статус пользователя: общие
// компании, диалоги и группы, супер-админы. Отношение симметрично, поэтому тот
// же круг — это и те, чей статус виден ему самому. all — видит всех (супер-админ).
type Audience interface {
	PresenceAudience(ctx context.Context, userID int64) (ids []int64, all bool, err error)
}

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

type audienceEntry struct {
	ids []int64
	all bool
	at  time.Time
}

type Presence struct {
	rdb      *redis.Client
	lastSeen LastSeenWriter
	bus      Bus
	audience Audience
	log      *slog.Logger
	now      func() time.Time

	mu    sync.Mutex
	cache map[int64]audienceEntry
}

func New(rdb *redis.Client, lastSeen LastSeenWriter, bus Bus, audience Audience, log *slog.Logger) *Presence {
	return &Presence{
		rdb: rdb, lastSeen: lastSeen, bus: bus, audience: audience, log: log,
		now: time.Now, cache: map[int64]audienceEntry{},
	}
}

// audienceOf — круг пользователя с кэшем на audienceTTL.
func (p *Presence) audienceOf(ctx context.Context, userID int64) (audienceEntry, error) {
	now := p.now()
	p.mu.Lock()
	e, ok := p.cache[userID]
	p.mu.Unlock()
	if ok && now.Sub(e.at) < audienceTTL {
		return e, nil
	}
	ids, all, err := p.audience.PresenceAudience(ctx, userID)
	if err != nil {
		return audienceEntry{}, err
	}
	e = audienceEntry{ids: ids, all: all, at: now}
	p.mu.Lock()
	// Протухшие снимки чистим на записи — отдельный цикл ради этого не нужен.
	if len(p.cache) > 1024 {
		for id, old := range p.cache {
			if now.Sub(old.at) >= audienceTTL {
				delete(p.cache, id)
			}
		}
	}
	p.cache[userID] = e
	p.mu.Unlock()
	return e, nil
}

// publish — presence:update аудитории пользователя (и ему самому: статус видят
// его же вкладки на других устройствах).
func (p *Presence) publish(ctx context.Context, userID int64, payload map[string]any) {
	e, err := p.audienceOf(ctx, userID)
	if err != nil {
		p.log.Warn("presence.audience_failed", "user_id", userID, "error", err)
		return
	}
	rooms := make([]string, 0, len(e.ids)+1)
	rooms = append(rooms, events.UserRoom(userID))
	for _, id := range e.ids {
		if id != userID {
			rooms = append(rooms, events.UserRoom(id))
		}
	}
	p.bus.Publish(ctx, "presence:update", rooms, payload)
}

func connsKey(userID int64) string { return connsPrefix + strconv.FormatInt(userID, 10) }

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
	ts := p.now().Unix()
	pipe := p.rdb.TxPipeline()
	pipe.ZAdd(ctx, beatsKey, redis.Z{Score: float64(ts), Member: member(userID, connID)})
	pipe.HSet(ctx, connsKey(userID), connID, ts)
	// Хеш ушедшего пользователя исчезнет сам: sweeper чистит поля, а TTL — ключ.
	pipe.Expire(ctx, connsKey(userID), 2*StaleAfter)
	if _, err := pipe.Exec(ctx); err != nil {
		p.log.Warn("presence.beat_failed", "user_id", userID, "error", err)
		return
	}
	p.setOnline(ctx, userID)
}

// drop — соединение больше не видимо/живо.
func (p *Presence) drop(ctx context.Context, userID int64, connID string) {
	pipe := p.rdb.TxPipeline()
	pipe.ZRem(ctx, beatsKey, member(userID, connID))
	pipe.HDel(ctx, connsKey(userID), connID)
	if _, err := pipe.Exec(ctx); err != nil {
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
	p.publish(ctx, userID, map[string]any{
		"user_id": userID, "online": true, "last_seen_at": nil,
	})
}

// maybeOffline — если живых видимых соединений не осталось, выставить офлайн.
// Уход чистый (дисконнект/скрытие вкладки) — last_seen = сейчас.
func (p *Presence) maybeOffline(ctx context.Context, userID int64) {
	alive, err := p.isAlive(ctx, userID)
	if err != nil {
		p.log.Warn("presence.alive_failed", "error", err)
		return
	}
	if alive {
		return
	}
	p.setOffline(ctx, userID, p.now())
}

// setOffline — перевод в офлайн с явным временем last_seen. Время передаёт
// вызывающий: чистый уход — момент ухода, sweep «зависшего» соединения —
// время последнего heartbeat'а (а НЕ момента sweep'а, иначе «был в сети»
// показывает время на StaleAfter+SweepInterval позже реального).
func (p *Presence) setOffline(ctx context.Context, userID int64, at time.Time) {
	removed, err := p.rdb.SRem(ctx, onlineKey, userID).Result()
	if err != nil {
		p.log.Warn("presence.offline_failed", "user_id", userID, "error", err)
		return
	}
	if removed == 0 {
		return
	}
	at = at.UTC()
	if err := p.lastSeen.SetLastSeen(ctx, userID, at); err != nil {
		p.log.Warn("presence.last_seen_failed", "user_id", userID, "error", err)
	}
	p.publish(ctx, userID, map[string]any{
		"user_id": userID, "online": false, "last_seen_at": isoUTC(at),
	})
}

// isAlive — есть ли у пользователя свежее видимое соединение.
func (p *Presence) isAlive(ctx context.Context, userID int64) (bool, error) {
	conns, err := p.rdb.HGetAll(ctx, connsKey(userID)).Result()
	if err != nil {
		return false, err
	}
	minScore := p.now().Add(-StaleAfter).Unix()
	for _, v := range conns {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil && ts > minScore {
			return true, nil
		}
	}
	return false, nil
}

// VisibleOnline — онлайн из круга пользователя (REST /api/messenger/presence):
// чужой онлайн платформы ему знать незачем.
func (p *Presence) VisibleOnline(ctx context.Context, userID int64) ([]int64, error) {
	online, err := p.OnlineUserIDs(ctx)
	if err != nil {
		return nil, err
	}
	e, err := p.audienceOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	if e.all {
		return online, nil
	}
	allowed := make(map[int64]struct{}, len(e.ids)+1)
	allowed[userID] = struct{}{}
	for _, id := range e.ids {
		allowed[id] = struct{}{}
	}
	out := online[:0]
	for _, id := range online {
		if _, ok := allowed[id]; ok {
			out = append(out, id)
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
	now := p.now()
	staleBefore := now.Add(-StaleAfter).Unix()

	// Снимок «последний сигнал по пользователю» снимаем ДО вычистки: после
	// ZRemRangeByScore просроченные члены пропадут, а их score — это и есть
	// время, которое нужно записать в last_seen уходящему в офлайн.
	withScores, err := p.rdb.ZRangeWithScores(ctx, beatsKey, 0, -1).Result()
	if err != nil {
		p.log.Warn("presence.sweep_failed", "error", err)
		return
	}
	lastBeat := make(map[int64]int64)
	pipe := p.rdb.Pipeline()
	for _, z := range withScores {
		m, _ := z.Member.(string)
		id := memberUserID(m)
		if id <= 0 {
			continue
		}
		s := int64(z.Score)
		if s > lastBeat[id] {
			lastBeat[id] = s
		}
		if s <= staleBefore {
			pipe.HDel(ctx, connsKey(id), m[strings.IndexByte(m, ':')+1:])
		}
	}
	pipe.ZRemRangeByScore(ctx, beatsKey, "-inf", strconv.FormatInt(staleBefore, 10))
	if _, err := pipe.Exec(ctx); err != nil {
		p.log.Warn("presence.sweep_trim_failed", "error", err)
		return
	}

	online, err := p.OnlineUserIDs(ctx)
	if err != nil {
		p.log.Warn("presence.sweep_failed", "error", err)
		return
	}
	for _, uid := range online {
		if lastBeat[uid] > staleBefore {
			continue // есть свежий видимый сигнал — пользователь жив
		}
		at := now
		if s, ok := lastBeat[uid]; ok {
			at = time.Unix(s, 0)
		}
		p.setOffline(ctx, uid, at)
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

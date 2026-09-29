// Package collab — кто сейчас держит документ открытым (заметка, доска).
//
// События совместной работы (курсоры, живые правки) нужны только открытому
// редактору, а рассылались всей аудитории документа: шаринг на компанию будил
// каждый её телефон на каждое движение чужого курсора. Реестр зрителей сужает
// адресатов до тех, кто действительно в документе.
//
// Зритель отмечается любым кадром редактора (join, курсор, heartbeat, правка)
// и считается ушедшим после leave или окна TTL без кадров — оно совпадает с
// порогом устаревания соавтора на клиенте. Состояние — ZSET в Redis
// (member — user_id, score — время последнего кадра), поэтому реестр общий
// для всех инстансов сервиса.
package collab

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// TTL — сколько зритель живёт без кадров (клиентский heartbeat — 10 с).
const TTL = 30 * time.Second

type Viewers struct {
	rdb    *redis.Client
	prefix string
	now    func() time.Time
}

// New — реестр документов одного вида; prefix — "gw2:<svc>:viewers:".
func New(rdb *redis.Client, prefix string) *Viewers {
	return &Viewers{rdb: rdb, prefix: prefix, now: time.Now}
}

func (v *Viewers) key(docID int64) string { return v.prefix + strconv.FormatInt(docID, 10) }

// Touch — пользователь в документе. Ключ живёт чуть дольше окна: документ,
// который все закрыли, не оставляет мусора.
func (v *Viewers) Touch(ctx context.Context, docID, userID int64) error {
	pipe := v.rdb.TxPipeline()
	pipe.ZAdd(ctx, v.key(docID), redis.Z{Score: float64(v.now().Unix()), Member: userID})
	pipe.Expire(ctx, v.key(docID), 2*TTL)
	_, err := pipe.Exec(ctx)
	return err
}

// Leave — пользователь закрыл документ.
func (v *Viewers) Leave(ctx context.Context, docID, userID int64) error {
	return v.rdb.ZRem(ctx, v.key(docID), userID).Err()
}

// List — кто в документе сейчас; попутно вычищает ушедших по таймауту.
func (v *Viewers) List(ctx context.Context, docID int64) ([]int64, error) {
	stale := strconv.FormatInt(v.now().Add(-TTL).Unix(), 10)
	pipe := v.rdb.TxPipeline()
	pipe.ZRemRangeByScore(ctx, v.key(docID), "-inf", stale)
	members := pipe.ZRange(ctx, v.key(docID), 0, -1)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(members.Val()))
	for _, m := range members.Val() {
		if id, err := strconv.ParseInt(m, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

func (v *Viewers) accessKey(docID, userID int64) string {
	return v.key(docID) + ":access:" + strconv.FormatInt(userID, 10)
}

// RememberAccess — запомнить проверенный доступ зрителя на окно TTL. Кадры
// редактора идут до восьми раз в секунду, а честная проверка доступа — это
// подъём по дереву папок; после отзыва шары соавтор теряет рассылку не позже
// чем через TTL. Ошибка Redis не мешает: проверка просто повторится.
func (v *Viewers) RememberAccess(ctx context.Context, docID, userID, ownerID int64, access string) {
	_ = v.rdb.Set(ctx, v.accessKey(docID, userID),
		access+":"+strconv.FormatInt(ownerID, 10), TTL).Err()
}

// CachedAccess — доступ, проверенный за последние TTL; ok=false — проверять.
func (v *Viewers) CachedAccess(ctx context.Context, docID, userID int64) (access string, ownerID int64, ok bool) {
	raw, err := v.rdb.Get(ctx, v.accessKey(docID, userID)).Result()
	if err != nil {
		return "", 0, false
	}
	i := strings.LastIndexByte(raw, ':')
	if i <= 0 {
		return "", 0, false
	}
	owner, err := strconv.ParseInt(raw[i+1:], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return raw[:i], owner, true
}

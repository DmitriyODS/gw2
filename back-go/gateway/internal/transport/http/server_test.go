package http

// Интеграционный smoke-тест WS-шлюза: настоящий Fiber-сервер + miniredis
// (presence и мост событий) + PASETO-пара, сгенерированная на лету.
// Проверяет handshake (_connected/_error), доставку событий микросервисов
// в комнаты и REST /api/messenger/presence.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/alicebob/miniredis/v2"
	"github.com/fasthttp/websocket"
	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/gateway/internal/bridge"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/hub"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/presence"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type nopLastSeen struct{}

func (nopLastSeen) SetLastSeen(context.Context, int64, time.Time) error { return nil }

type selfAudience struct{}

func (selfAudience) PresenceAudience(context.Context, int64) ([]int64, bool, error) {
	return nil, false, nil
}

func signToken(t *testing.T, secret paseto.V4AsymmetricSecretKey, userID int64) string {
	return signCompanyToken(t, secret, userID, 0)
}

func signCompanyToken(t *testing.T, secret paseto.V4AsymmetricSecretKey, userID, companyID int64) string {
	t.Helper()
	tok := paseto.NewToken()
	if companyID > 0 {
		_ = tok.Set("company_id", companyID)
	}
	tok.SetSubject(fmt.Sprint(userID))
	tok.SetString("type", "access")
	tok.SetIssuedAt(time.Now())
	tok.SetNotBefore(time.Now())
	tok.SetExpiration(time.Now().Add(15 * time.Minute))
	return tok.V4Sign(secret, nil)
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func startServer(t *testing.T) (addr string, rdb *redis.Client, secret paseto.V4AsymmetricSecretKey) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	secret = paseto.NewV4AsymmetricSecretKey()
	verifier, err := pasetoauth.NewVerifier(secret.Public().ExportHex())
	if err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.DiscardHandler)
	h := hub.New()
	bus := events.NewPublisher(rdb, log, "gw2:gateway:events")
	pres := presence.New(rdb, nopLastSeen{}, bus, selfAudience{}, log)

	server := NewServer(Deps{
		Hub:      h,
		Presence: pres,
		Ring:     nil, // ринг-фаза в этом тесте не дёргается
		Verifier: verifier,
		Auth: func(context.Context, int64, pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
			return &pasetoauth.AuthInfo{RoleLevel: 1, IsActive: true, CompanyActive: true}, nil
		},
		Log: log,
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go bridge.New(rdb, h, log).Run(ctx)

	addr = freePort(t)
	go func() { _ = server.Listen(addr) }()
	t.Cleanup(func() { _ = server.Shutdown() })

	// Ждём готовности порта.
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return addr, rdb, secret
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("сервер не поднялся")
	return
}

func dial(t *testing.T, addr string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readFrame(t *testing.T, conn *websocket.Conn) hub.Frame {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var f hub.Frame
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("bad frame %s: %v", raw, err)
	}
	return f
}

func authConnect(t *testing.T, conn *websocket.Conn, token string) {
	t.Helper()
	frame, _ := json.Marshal(map[string]any{"event": "auth", "data": map[string]any{"token": token}})
	if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
		t.Fatal(err)
	}
}

func TestWSAuthAndEventDelivery(t *testing.T) {
	addr, rdb, secret := startServer(t)

	conn := dial(t, addr)
	authConnect(t, conn, signToken(t, secret, 7))
	if f := readFrame(t, conn); f.Event != "_connected" {
		t.Fatalf("ожидался _connected, получен %s", f.Event)
	}

	// Событие микросервиса в комнату all доезжает до клиента.
	env, _ := json.Marshal(map[string]any{
		"event": "task:created", "rooms": []string{"all"},
		"payload": map[string]any{"id": 5},
	})
	if err := rdb.Publish(context.Background(), "gw2:tasks:events", env).Err(); err != nil {
		t.Fatal(err)
	}
	// presence:update (онлайн) и task:created приходят в любом порядке.
	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		got[readFrame(t, conn).Event] = true
	}
	if !got["task:created"] || !got["presence:update"] {
		t.Fatalf("получены кадры: %v", got)
	}

	// Личная комната: событие для user_7 доезжает, для user_8 — нет.
	env, _ = json.Marshal(map[string]any{
		"event": "pet:update", "rooms": []string{"user_7"},
		"payload": map[string]any{"xp": 1},
	})
	_ = rdb.Publish(context.Background(), "gw2:messenger:events", env).Err()
	if f := readFrame(t, conn); f.Event != "pet:update" {
		t.Fatalf("ожидался pet:update, получен %s", f.Event)
	}
}

func TestWSRejectsBadToken(t *testing.T) {
	addr, _, _ := startServer(t)
	conn := dial(t, addr)
	authConnect(t, conn, "garbage")
	if f := readFrame(t, conn); f.Event != "_error" {
		t.Fatalf("ожидался _error, получен %s", f.Event)
	}
}

func publishTo(t *testing.T, rdb *redis.Client, event string, room string) {
	t.Helper()
	env, _ := json.Marshal(map[string]any{"event": event, "rooms": []string{room}, "payload": nil})
	if err := rdb.Publish(context.Background(), "gw2:tasks:events", env).Err(); err != nil {
		t.Fatal(err)
	}
}

// Компанийные события доходят только до соединений с этой активной компанией,
// а кадр переавторизации пересаживает соединение в новую компанию.
func TestWSCompanyRoomAndReauth(t *testing.T) {
	addr, rdb, secret := startServer(t)
	conn := dial(t, addr)
	authConnect(t, conn, signCompanyToken(t, secret, 7, 5))
	if f := readFrame(t, conn); f.Event != "_connected" {
		t.Fatalf("ожидался _connected, получен %s", f.Event)
	}
	if f := readFrame(t, conn); f.Event != "presence:update" {
		t.Fatalf("ожидался presence:update, получен %s", f.Event)
	}

	publishTo(t, rdb, "foreign", "company_6")
	publishTo(t, rdb, "own", "company_5")
	if f := readFrame(t, conn); f.Event != "own" {
		t.Fatalf("дошло событие чужой компании: %s", f.Event)
	}

	authConnect(t, conn, signCompanyToken(t, secret, 7, 6))
	// Кадр переавторизации ответа не имеет — шлём пробы в новую компанию,
	// пока первая из них не дойдёт.
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				// t.Fatal из чужой горутины нельзя — ошибку публикации пережидаем.
				env, _ := json.Marshal(map[string]any{"event": "probe", "rooms": []string{"company_6"}})
				_ = rdb.Publish(context.Background(), "gw2:tasks:events", env).Err()
			}
		}
	}()
	for readFrame(t, conn).Event != "probe" {
	}
	close(stop)
	// Хвост probe-кадров мог остаться в пути — дочитываем до маркера.
	publishTo(t, rdb, "old", "company_5")
	publishTo(t, rdb, "marker", "user_7")
	for {
		f := readFrame(t, conn)
		if f.Event == "old" {
			t.Fatal("после смены компании дошло событие прежней")
		}
		if f.Event == "marker" {
			break
		}
	}
}

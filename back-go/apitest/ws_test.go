package apitest

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
)

// wsFrame — кадр протокола gatewaysvc: {"event": ..., "data": ...}.
type wsFrame struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// Obj — data кадра как JSON-объект (nil, если это не объект).
func (f wsFrame) Obj() map[string]any {
	var m map[string]any
	_ = json.Unmarshal(f.Data, &m)
	return m
}

// wsClient — тестовый WS-клиент шлюза: читает кадры в фоне и раздаёт их
// через waitFrame. Закрывается t.Cleanup.
type wsClient struct {
	conn   *websocket.Conn
	frames chan wsFrame
}

// dialWS — соединение с /ws БЕЗ auth-кадра (для негативных сценариев).
func dialWS(t *testing.T) *wsClient {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(gatewayWSURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	c := &wsClient{conn: conn, frames: make(chan wsFrame, 64)}
	t.Cleanup(c.close)
	go func() {
		defer close(c.frames)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var f wsFrame
			if json.Unmarshal(raw, &f) == nil && f.Event != "" {
				c.frames <- f
			}
		}
	}()
	return c
}

// connectWS — полный handshake: dial + кадр auth + ожидание _connected.
func connectWS(t *testing.T, token string) *wsClient {
	t.Helper()
	c := dialWS(t)
	c.emit(t, "auth", map[string]any{"token": token})
	f := c.waitFrame(t, "_connected", 10*time.Second)
	if f.Obj()["user_id"] == nil {
		t.Fatalf("_connected без user_id: %s", f.Data)
	}
	return c
}

func (c *wsClient) close() { _ = c.conn.Close() }

// emit — отправить кадр {"event", "data"}.
func (c *wsClient) emit(t *testing.T, event string, data any) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"event": event, "data": data})
	if err != nil {
		t.Fatalf("ws marshal %s: %v", event, err)
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("ws write %s: %v", event, err)
	}
}

// waitFrame — ждать кадр события event (кадры других событий пропускаются:
// в комнату all может прилетать что угодно от параллельных тестов).
func (c *wsClient) waitFrame(t *testing.T, event string, timeout time.Duration) wsFrame {
	t.Helper()
	f, err := c.tryWaitFrame(event, nil, timeout)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// waitFrameMatch — ждать кадр события event, удовлетворяющий predicate.
func (c *wsClient) waitFrameMatch(t *testing.T, event string,
	predicate func(wsFrame) bool, timeout time.Duration) wsFrame {
	t.Helper()
	f, err := c.tryWaitFrame(event, predicate, timeout)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (c *wsClient) tryWaitFrame(event string, predicate func(wsFrame) bool,
	timeout time.Duration) (wsFrame, error) {

	deadline := time.After(timeout)
	for {
		select {
		case f, ok := <-c.frames:
			if !ok {
				return wsFrame{}, fmt.Errorf("ws закрыт до кадра %q", event)
			}
			if f.Event == event && (predicate == nil || predicate(f)) {
				return f, nil
			}
		case <-deadline:
			return wsFrame{}, fmt.Errorf("кадр %q не пришёл за %s", event, timeout)
		}
	}
}

// expectNoFrame — убедиться, что кадр события event НЕ приходит в течение
// window (например, чужие адресные события не протекают).
func (c *wsClient) expectNoFrame(t *testing.T, event string, window time.Duration) {
	t.Helper()
	if f, err := c.tryWaitFrame(event, nil, window); err == nil {
		t.Fatalf("неожиданный кадр %q: %s", event, f.Data)
	}
}

// Package http — транспорт gatewaysvc: WS-эндпоинт /ws (realtime-шлюз) и
// REST (/api/messenger/presence, /healthz).
//
// Протокол WS — кадры {"event": ..., "data": ...}:
//   - первый кадр клиента — {"event": "auth", "data": {"token": "<PASETO>"}}
//     (таймаут authTimeout); невалидный токен → {"event": "_error"} и close,
//     успех → {"event": "_connected", "data": {"user_id": N}} и подписка на
//     комнаты all/user_{id} и company_{id} активной компании токена;
//   - повторный кадр auth на живом соединении (обновился токен, сменилась
//     активная компания) пересаживает клиента в комнату новой компании —
//     переподключаться ради этого не нужно;
//   - дальше клиент шлёт presence:visibility / presence:heartbeat / call:*;
//   - сервер пингует каждые pingInterval, нет pong'а дольше pongWait — close.
package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"github.com/DmitriyODS/gw2/back-go/gateway/internal/hub"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/presence"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/ring"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

const (
	authTimeout  = 10 * time.Second
	pingInterval = 25 * time.Second
	pongWait     = 60 * time.Second
	writeWait    = 10 * time.Second
)

type Server struct {
	app *fiber.App
}

type Deps struct {
	Hub      *hub.Hub
	Presence *presence.Presence
	Ring     *ring.Ring
	Bus      ring.Bus
	Verifier *pasetoauth.Verifier
	Auth     pasetoauth.AuthSource
	Log      *slog.Logger
}

func NewServer(d Deps) *Server {
	app := httpserver.New(httpserver.Config{AppName: "gw2-gatewaysvc", Log: d.Log})
	s := &wsHandler{deps: d}

	// Онлайн-пользователи (presence) — exact-роут /api/messenger/presence;
	// отдаётся только круг спрашивающего (см. presence.Audience).
	auth := pasetoauth.NewMiddleware(d.Verifier, d.Auth)
	app.Get("/api/messenger/presence", auth.RequireAuth, func(c *fiber.Ctx) error {
		online, err := d.Presence.VisibleOnline(c.Context(), pasetoauth.UserID(c))
		if err != nil {
			d.Log.Error("presence.list_failed", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера",
			})
		}
		return c.JSON(fiber.Map{"online": online})
	})

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", websocket.New(s.handle))

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type wsHandler struct {
	deps Deps
}

func writeFrame(conn *websocket.Conn, frame []byte) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.TextMessage, frame)
}

// handle — жизненный цикл одного WS-соединения (выполняется в горутине
// fasthttp; блокирующее чтение здесь допустимо).
func (h *wsHandler) handle(conn *websocket.Conn) {
	log := h.deps.Log
	defer conn.Close()

	// ── handshake: первый кадр — auth ────────────────────────────
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var authFrame struct {
		Event string `json:"event"`
		Data  struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &authFrame)
	claims := pasetoauth.Claims{}
	if authFrame.Event == "auth" && authFrame.Data.Token != "" {
		claims = h.deps.Verifier.ParseAccess(authFrame.Data.Token)
	}
	if claims.UserID == 0 {
		log.Warn("ws.connect_rejected", "reason", "invalid token")
		_ = writeFrame(conn, hub.MarshalFrame("_error",
			json.RawMessage(`{"code":"AUTH_FAILED"}`)))
		return
	}
	// Правовые документы не приняты — доступа нет нигде, включая realtime
	// (152-ФЗ): иначе события разделов продолжали бы приходить под плашкой
	// согласия. Фронт в этом состоянии сокет и не поднимает.
	if claims.LegalRequired {
		log.Warn("ws.connect_rejected", "reason", "legal consent required", "user_id", claims.UserID)
		_ = writeFrame(conn, hub.MarshalFrame("_error",
			json.RawMessage(`{"code":"LEGAL_CONSENT_REQUIRED"}`)))
		return
	}
	userID := claims.UserID

	client := hub.NewClient(userID)
	h.deps.Hub.Add(client, "all", events.UserRoom(userID))
	h.deps.Hub.SetCompany(client, companyRoom(claims))

	ctx := context.Background()
	h.deps.Presence.OnConnect(ctx, userID, client.ConnID)
	log.Info("ws.connect", "user_id", userID)

	connected, _ := json.Marshal(map[string]any{"user_id": userID})
	if err := writeFrame(conn, hub.MarshalFrame("_connected", connected)); err != nil {
		h.cleanup(client)
		return
	}

	// ── writer: исходящие кадры + пинги ──────────────────────────
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-client.Done():
				return
			case frame := <-client.Frames():
				if err := writeFrame(conn, frame); err != nil {
					client.Close()
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					client.Close()
					return
				}
			}
		}
	}()

	// ── reader: входящие команды ─────────────────────────────────
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		select {
		case <-client.Done():
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		var frame hub.Frame
		if err := json.Unmarshal(raw, &frame); err != nil {
			continue
		}
		h.dispatch(client, frame)
	}

	client.Close()
	<-writerDone
	h.cleanup(client)
	log.Info("ws.disconnect", "user_id", userID)
}

func (h *wsHandler) cleanup(client *hub.Client) {
	h.deps.Hub.Remove(client)
	h.deps.Presence.OnDisconnect(context.Background(), client.UserID, client.ConnID)
}

// dispatch — маршрутизация входящих кадров клиента.
func (h *wsHandler) dispatch(client *hub.Client, frame hub.Frame) {
	ctx := context.Background()
	switch frame.Event {
	case "auth":
		h.reauth(client, frame.Data)
	case "presence:visibility":
		var data struct {
			Visible *bool `json:"visible"`
		}
		_ = json.Unmarshal(frame.Data, &data)
		visible := data.Visible == nil || *data.Visible
		h.deps.Presence.OnVisibility(ctx, client.UserID, client.ConnID, visible)
	case "presence:heartbeat":
		h.deps.Presence.OnHeartbeat(ctx, client.UserID, client.ConnID)
	case "call:start", "call:invite", "call:accept", "call:decline", "call:leave", "call:end":
		// gRPC до 10с — в горутине, чтобы не блокировать чтение (пинг-понг).
		go h.deps.Ring.Dispatch(client.UserID, frame.Event, frame.Data)
	case "typing":
		// Эфемерный индикатор «печатает…»: релеим собеседнику без БД. Клиент
		// сам сообщает to_user_id (знает other_user диалога).
		h.relayTyping(ctx, client.UserID, frame.Data)
	}
}

// reauth — свежий токен на живом соединении. Чужой токен (другой
// пользователь) соединение закрывает: личная комната уже выдана прежнему.
// Невалидный или протухший — игнорируется, клиент остаётся где был.
func (h *wsHandler) reauth(client *hub.Client, raw json.RawMessage) {
	var data struct {
		Token string `json:"token"`
	}
	if json.Unmarshal(raw, &data) != nil || data.Token == "" {
		return
	}
	claims := h.deps.Verifier.ParseAccess(data.Token)
	if claims.UserID == 0 {
		return
	}
	if claims.UserID != client.UserID || claims.LegalRequired {
		client.Close()
		return
	}
	h.deps.Hub.SetCompany(client, companyRoom(claims))
}

func companyRoom(claims pasetoauth.Claims) string {
	if claims.CompanyID == nil || *claims.CompanyID <= 0 {
		return ""
	}
	return events.CompanyRoom(*claims.CompanyID)
}

func (h *wsHandler) relayTyping(ctx context.Context, fromUserID int64, raw json.RawMessage) {
	if h.deps.Bus == nil {
		return
	}
	var data struct {
		ConversationID int64 `json:"conversation_id"`
		ToUserID       int64 `json:"to_user_id"`
		Typing         *bool `json:"typing"`
	}
	if json.Unmarshal(raw, &data) != nil || data.ToUserID == 0 || data.ConversationID == 0 {
		return
	}
	typing := data.Typing == nil || *data.Typing
	h.deps.Bus.Publish(ctx, "typing", []string{events.UserRoom(data.ToUserID)}, map[string]any{
		"conversation_id": data.ConversationID,
		"user_id":         fromUserID,
		"typing":          typing,
	})
}

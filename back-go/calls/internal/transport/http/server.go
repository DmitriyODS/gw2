// Package http — HTTP-транспорт (Fiber): REST /api/calls/* и вебхуки LiveKit.
//
// Пути и формы JSON байт-в-байт совместимы с прежним Flask-блюпринтом
// api/calls.py — фронт не меняется, nginx просто маршрутизирует
// /api/calls/ на этот сервис вместо Flask.
package http

import (
	"errors"
	"log/slog"
	"strconv"

	"aidanwoods.dev/go-paseto"
	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/livekit"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/service"
)

const historyLimit = 100

type Server struct {
	app *fiber.App
}

func NewServer(eps endpoint.Endpoints, svc service.CallService, lk *livekit.Client,
	users domain.UserReader, pasetoPublic paseto.V4AsymmetricPublicKey, log *slog.Logger) *Server {

	app := fiber.New(fiber.Config{
		AppName:               "gw2-callsvc",
		DisableStartupMessage: true,
		// Тело вебхука читаем как есть; JSON-парсинг — вручную в хендлерах.
	})
	auth := &authParser{public: pasetoPublic, users: users}
	h := &handlers{eps: eps, svc: svc, lk: lk, auth: auth, log: log}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := app.Group("/api/calls")
	api.Get("/history", auth.requireAuth, h.history)
	api.Get("/active", auth.requireAuth, h.activeCall)
	api.Post("/:id<int>/token", auth.requireAuth, h.rejoinToken)
	api.Get("/join/:code", h.joinInfo)
	api.Post("/join/:code", h.joinByCode)
	api.Post("/livekit-webhook", h.livekitWebhook)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	eps  endpoint.Endpoints
	svc  service.CallService
	lk   *livekit.Client
	auth *authParser
	log  *slog.Logger
}

// respondError — бизнес-ошибка в форме {code, message} с её http-статусом,
// прочее — 500 (как Flask-обработчик ошибок).
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	if de := domain.AsDomainError(err); de != nil {
		return c.Status(de.HTTPStatus).JSON(fiber.Map{
			"code": de.Code, "message": de.Message,
		})
	}
	h.log.Error("http.internal_error", "path", c.Path(), "error", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера",
	})
}

func (h *handlers) history(c *fiber.Ctx) error {
	resp, err := h.eps.History(c.Context(), endpoint.HistoryRequest{
		UserID: currentUserID(c), Limit: historyLimit,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	calls := resp.([]*dto.CallDTO)
	if calls == nil {
		calls = []*dto.CallDTO{}
	}
	return c.JSON(calls)
}

func (h *handlers) activeCall(c *fiber.Ctx) error {
	resp, err := h.eps.ActiveCall(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) rejoinToken(c *fiber.Ctx) error {
	callID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code": "NOT_IN_CALL", "message": "Звонок не найден",
		})
	}
	resp, err := h.eps.RejoinToken(c.Context(), endpoint.RejoinTokenRequest{
		CallID: callID, UserID: currentUserID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) joinInfo(c *fiber.Ctx) error {
	resp, err := h.eps.JoinInfo(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) joinByCode(c *fiber.Ctx) error {
	userID := h.auth.optionalUserID(c)

	var body struct {
		Name string `json:"name"`
	}
	_ = c.BodyParser(&body) // пустое/невалидное тело допустимо для участника

	if userID == 0 && body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "NAME_REQUIRED", "message": "Представьтесь, чтобы войти в звонок",
		})
	}
	resp, err := h.eps.JoinByCode(c.Context(), dto.JoinByCodeRequest{
		Code: c.Params("code"), UserID: userID, GuestName: body.Name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) livekitWebhook(c *fiber.Ctx) error {
	event, err := h.lk.VerifyWebhook(c.Body(), c.Get(fiber.HeaderAuthorization))
	if err != nil {
		h.log.Warn("livekit.webhook_rejected", "error", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "BAD_SIGNATURE"})
	}
	name, _ := event["event"].(string)
	room, _ := event["room"].(map[string]any)
	participant, _ := event["participant"].(map[string]any)
	roomName, _ := room["name"].(string)
	identity, _ := participant["identity"].(string)

	if err := h.svc.HandleWebhook(c.Context(), dto.WebhookEvent{
		Event: name, Room: roomName, Identity: identity,
	}); err != nil && !errors.Is(err, c.Context().Err()) {
		// Вебхук не должен ретраиться LiveKit'ом из-за внутренних ошибок.
		h.log.Error("livekit.webhook_apply_failed", "event", name, "error", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

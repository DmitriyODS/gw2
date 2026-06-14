// Package http — REST pushsvc: регистрация/удаление токенов устройств.
package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/push/internal/service"
)

type Server struct{ app *fiber.App }

func NewServer(svc *service.Service, verifier *pasetoauth.Verifier, log *slog.Logger) *Server {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h := &handlers{svc: svc, log: log}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Только валидность токена: токены регистрирует любой авторизованный
	// пользователь (force_change-гейт тут не нужен — pushsvc вне основного API).
	auth := pasetoauth.NewMiddleware(verifier, nil)
	api := app.Group("/api/push")
	api.Post("/register", auth.RequireToken, h.register)
	api.Post("/unregister", auth.RequireToken, h.unregister)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	svc *service.Service
	log *slog.Logger
}

type tokenBody struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func (h *handlers) register(c *fiber.Ctx) error {
	var body tokenBody
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "BAD_REQUEST", "message": "token обязателен",
		})
	}
	if err := h.svc.Register(c.Context(), pasetoauth.UserID(c), body.Token, body.Platform); err != nil {
		h.log.Warn("push.register_failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "INTERNAL_ERROR", "message": "Не удалось сохранить токен",
		})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) unregister(c *fiber.Ctx) error {
	var body tokenBody
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "BAD_REQUEST", "message": "token обязателен",
		})
	}
	if err := h.svc.Unregister(c.Context(), body.Token); err != nil {
		h.log.Warn("push.unregister_failed", "error", err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

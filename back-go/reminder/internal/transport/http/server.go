// Package http — HTTP-транспорт remindersvc (Fiber): REST /api/reminders/*.
// Все ручки требуют только авторизации (RequireAuth) — напоминание личное и от
// компании не зависит; скоуп по владельцу проверяет сервис. Хендлеры зовут
// сервис напрямую (без go-kit endpoint-обёрток: middleware-цепочек здесь нет).
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/reminder/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/reminder/internal/service"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Напоминания не зависят от
// компании, поэтому CompanyActive всегда true; из БД берём лишь глобальную
// активность.
func authSource(users domain.UserReader) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64, _ pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetUser(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		return &pasetoauth.AuthInfo{
			IsActive:      u.IsActive,
			IsSuperAdmin:  u.IsSuperAdmin,
			CompanyActive: true,
			User:          u,
		}, nil
	}
}

func NewServer(svc *service.Service, users domain.UserReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := httpserver.New(httpserver.Config{AppName: "gw2-remindersvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{svc: svc, log: log}

	api := app.Group("/api/reminders", auth.RequireAuth)
	api.Get("", h.list)
	api.Post("", h.create)
	api.Get("/upcoming", h.upcoming)
	api.Get("/linked", h.linked) // ?kind=diary|calendar&record_id=
	api.Get("/:id<int>", h.get)
	api.Patch("/:id<int>", h.update)
	api.Delete("/:id<int>", h.remove)
	api.Post("/:id<int>/snooze", h.snooze)
	api.Post("/:id<int>/done", h.done)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	svc *service.Service
	log *slog.Logger
}

func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func currentUserID(c *fiber.Ctx) int64 { return pasetoauth.UserID(c) }

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

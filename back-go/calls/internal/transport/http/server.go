// Package http — HTTP-транспорт (Fiber): REST /api/calls/* и вебхуки LiveKit.
//
// Пути и формы JSON байт-в-байт совместимы с прежним Flask-блюпринтом
// api/calls.py — фронт не меняется, nginx просто маршрутизирует
// /api/calls/ на этот сервис вместо Flask.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/livekit"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

const historyLimit = 100

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Активная компания и роль —
// ИЗ ТОКЕНА (active); из БД — идентичность, глобальная активность аккаунта и
// активность выбранной компании.
func authSource(users domain.UserReader) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64, active pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetUser(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		companyActive, err := users.CompanyActive(ctx, active.CompanyID)
		if err != nil {
			return nil, err
		}
		u.CompanyActive = companyActive
		return &pasetoauth.AuthInfo{
			RoleLevel:     active.RoleLevel,
			IsActive:      u.IsActive,
			IsSuperAdmin:  u.IsSuperAdmin,
			CompanyActive: companyActive,
			User:          u,
		}, nil
	}
}

func NewServer(eps endpoint.Endpoints, svc service.CallService, lk *livekit.Client,
	users domain.UserReader, verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	// Тело вебхука читаем как есть; JSON-парсинг — вручную в хендлерах.
	app := httpserver.New(httpserver.Config{AppName: "gw2-callsvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, svc: svc, lk: lk, auth: auth, log: log}

	api := app.Group("/api/calls")
	api.Get("/history", auth.RequireAuth, h.history)
	api.Get("/active", auth.RequireAuth, h.activeCall)
	api.Post("/:id<int>/token", auth.RequireAuth, h.rejoinToken)
	api.Get("/join/:code", h.joinInfo)
	api.Post("/join/:code", h.joinByCode)
	api.Post("/livekit-webhook", h.livekitWebhook)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

// currentUserID — id пользователя из Locals (после RequireAuth).
func currentUserID(c *fiber.Ctx) int64 {
	return pasetoauth.UserID(c)
}

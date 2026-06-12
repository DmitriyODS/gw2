// Package http — HTTP-транспорт (Fiber): REST /api/companies/:id/ai-settings*
// и /api/ai/tv-fact.
//
// Пути и формы JSON байт-в-байт совместимы с прежними Flask-блюпринтами
// api/ai_settings.py и api/ai_tv.py — фронт не меняется, nginx маршрутизирует
// regex ^/api/companies/\d+/ai-settings и префикс /api/ai на этот сервис.
//
// Доступ к настройкам: уровень роли DIRECTOR+ (мидлварь), причём
// Руководитель — только своя компания, Администратор системы (is_root_admin) —
// любая (проверка в сервисном слое — как _check_access во Flask).
// ТВ-факт — любой авторизованный в company-scope.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари (is_hidden, активность
// компании, уровень роли) поверх доменного UserReader.
func authSource(users domain.UserReader) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetUser(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		return &pasetoauth.AuthInfo{
			RoleLevel:     u.RoleLevel,
			IsHidden:      u.IsHidden,
			CompanyActive: u.CompanyActive,
			User:          u,
		}, nil
	}
}

func NewServer(eps endpoint.Endpoints, users domain.UserReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := fiber.New(fiber.Config{
		AppName:               "gw2-aisvc",
		DisableStartupMessage: true,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := app.Group("/api/companies/:companyId<int>/ai-settings",
		auth.RequireAuth, auth.RequireRole(domain.LevelDirector))
	api.Get("", h.getSettings)
	api.Put("", h.updateSettings)
	api.Post("/test", h.testSettings)
	api.Get("/indexing", h.indexingStatus)
	api.Post("/reindex-tasks", h.reindexTasks)

	app.Get("/api/ai/tv-fact", auth.RequireAuth, h.tvFact)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

// currentUser — полный доменный пользователь из Locals (после RequireAuth).
func currentUser(c *fiber.Ctx) *domain.User {
	u, _ := pasetoauth.CurrentUser(c).(*domain.User)
	return u
}

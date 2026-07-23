// Package http — HTTP-транспорт (Fiber): REST /api/companies/:id/ai-settings*
// и /api/ai/tv-fact.
//
// Пути и формы JSON байт-в-байт совместимы с прежними Flask-блюпринтами
// api/ai_settings.py и api/ai_tv.py — фронт не меняется, nginx маршрутизирует
// regex ^/api/companies/\d+/ai-settings и префикс /api/ai на этот сервис.
//
// Доступ к настройкам: администратор ИМЕННО этой компании (членство с ролью ≥ 3)
// или супер-админ — проверяет сервис (resolveCompany) по компании из пути, а не
// по активной компании сессии. Транспорт лишь требует авторизацию. ТВ-факт —
// любой авторизованный в company-scope.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Активная компания и роль
// в ней — ИЗ ТОКЕНА (active); из БД — глобальная активность аккаунта и
// активность выбранной компании.
func authSource(users domain.UserReader) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64, active pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetUser(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		u.CompanyID = active.CompanyID
		u.RoleLevel = active.RoleLevel
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

func NewServer(eps endpoint.Endpoints, users domain.UserReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := httpserver.New(httpserver.Config{AppName: "gw2-aisvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	api := app.Group("/api/companies/:companyId<int>/ai-settings", auth.RequireAuth)
	api.Get("", h.getSettings)
	api.Put("", h.updateSettings)
	api.Post("/test", h.testSettings)
	api.Get("/indexing", h.indexingStatus)
	api.Post("/reindex-tasks", h.reindexTasks)

	app.Get("/api/ai/tv-fact", auth.RequireAuth, h.tvFact)

	app.Post("/api/ai/assistant/messages", auth.RequireAuth, h.sendAssistantMessage)
	app.Get("/api/ai/assistant/history", auth.RequireAuth, h.getAssistantHistory)
	app.Post("/api/ai/assistant/feedback", auth.RequireAuth, h.sendAssistantFeedback)

	app.Post("/api/ai/text-tools", auth.RequireAuth, h.transformText)
	app.Post("/api/ai/proofread", auth.RequireAuth, h.proofread)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

// currentUser — полный доменный пользователь из Locals (после RequireAuth).
func currentUser(c *fiber.Ctx) *domain.User {
	u, _ := pasetoauth.CurrentUser(c).(*domain.User)
	return u
}

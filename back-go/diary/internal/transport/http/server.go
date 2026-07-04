// Package http — HTTP-транспорт diarysvc (Fiber): REST /api/diaries/*. Все
// приватные ручки требуют только авторизации (RequireAuth) — ежедневник личный
// и от компании не зависит; доступ к чужому ежедневнику (read-only) проверяется
// в сервисе по владельцу/адресному шарингу. Публичные ссылки /shared/* идут
// мимо авторизации (код-capability).
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/diary/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Ежедневник не зависит от
// компании, поэтому CompanyActive всегда true (отключённая активная компания не
// должна закрывать личный раздел); из БД берём лишь глобальную активность.
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

func NewServer(eps endpoint.Endpoints, users domain.UserReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := httpserver.New(httpserver.Config{AppName: "gw2-diarysvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	// Middleware группы монтируется на весь префикс (Fiber), поэтому публичные
	// ссылки /api/diaries/shared/* пропускаем мимо авторизации — доступ по
	// коду-capability, без сессии.
	api := app.Group("/api/diaries", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/diaries/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичный read-only доступ по коду ссылки (без авторизации).
	api.Get("/shared/:code", h.sharedDiary)
	api.Get("/shared/:code/records", h.sharedEntries)
	api.Get("/shared/:code/export", h.sharedExport)

	// Ежедневники.
	api.Get("", h.listDiaries) // ?tab=mine|shared
	api.Post("", h.createDiary)
	api.Get("/:id<int>", h.getDiary)
	api.Patch("/:id<int>", h.updateDiary)
	api.Delete("/:id<int>", h.deleteDiary)

	// Публичные ссылки (управление владельцем).
	api.Get("/:id<int>/shares", h.listShares)
	api.Post("/:id<int>/shares", h.createShare)
	api.Delete("/:id<int>/shares/:shareId<int>", h.revokeShare)

	// Адресный доступ (поделиться с пользователем).
	api.Get("/:id<int>/members", h.listMembers)
	api.Post("/:id<int>/members", h.addMember)
	api.Delete("/:id<int>/members/:userId<int>", h.removeMember)

	// Записи.
	api.Get("/:id<int>/records", h.listEntries)
	api.Get("/:id<int>/export", h.exportEntries)
	api.Post("/:id<int>/records", h.createEntry)
	api.Post("/:id<int>/records/bulk-delete", h.bulkDeleteEntries)
	api.Post("/:id<int>/records/reorder", h.reorderEntries)
	api.Get("/:id<int>/records/:rid<int>", h.getEntry)
	api.Patch("/:id<int>/records/:rid<int>", h.updateEntry)
	api.Patch("/:id<int>/records/:rid<int>/done", h.setDone)
	api.Patch("/:id<int>/records/:rid<int>/link", h.setLink)
	api.Patch("/:id<int>/records/:rid<int>/move", h.moveEntry)
	api.Delete("/:id<int>/records/:rid<int>", h.deleteEntry)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	eps endpoint.Endpoints
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

func recordID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("rid")
	return int64(id)
}

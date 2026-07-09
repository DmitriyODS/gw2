// Package http — HTTP-транспорт notesvc (Fiber): REST /api/notes/*. Все
// приватные ручки требуют только авторизации (RequireAuth) — заметка личная и
// от компании не зависит; скоуп по владельцу проверяется в сервисе. Публичные
// ссылки /shared/* идут мимо авторизации (код-capability).
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

// uploadMaxBytes — лимит картинки редактора (как у вложений мессенджера).
const uploadMaxBytes = 25 * 1024 * 1024

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Заметки не зависят от
// компании, поэтому CompanyActive всегда true (отключённая активная компания
// не должна закрывать личный раздел); из БД берём лишь глобальную активность.
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

	app := httpserver.New(httpserver.Config{
		AppName: "gw2-notesvc", Log: log, BodyLimit: uploadMaxBytes + 1024*1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	// Middleware группы монтируется на весь префикс (Fiber), поэтому публичные
	// ссылки /api/notes/shared/* пропускаем мимо авторизации — доступ по
	// коду-capability, без сессии.
	api := app.Group("/api/notes", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/notes/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичный доступ по коду ссылки (без авторизации); запись — только по
	// edit-ссылке, с троттлингом в сервисе.
	api.Get("/shared/:code", h.sharedNote)
	api.Put("/shared/:code", h.sharedUpdate)

	// Группы — до "/:id<int>", чтобы литеральный сегмент не съедался параметром.
	api.Get("/groups", h.listGroups)
	api.Post("/groups", h.createGroup)
	api.Patch("/groups/:id<int>", h.updateGroup)
	api.Delete("/groups/:id<int>", h.deleteGroup)

	// Заметки.
	api.Get("", h.listNotes) // ?group_id=&search=&archived=1 | ?shared=1 (поделились со мной)
	api.Post("", h.createNote)
	api.Post("/import", h.importNote)
	api.Get("/:id<int>", h.getNote)
	api.Patch("/:id<int>", h.updateNote)
	api.Delete("/:id<int>", h.deleteNote)
	api.Put("/:id<int>/groups", h.setGroups)

	// Публичные ссылки (управление владельцем).
	api.Get("/:id<int>/shares", h.listShares)
	api.Post("/:id<int>/shares", h.createShare)
	api.Delete("/:id<int>/shares/:shareId<int>", h.revokeShare)

	// Адресный шаринг пользователям платформы (управление владельцем) и
	// collab-броадкаст совместного редактирования (владелец или адресат).
	api.Get("/:id<int>/members", h.listMembers)
	api.Post("/:id<int>/members", h.addMember)
	api.Delete("/:id<int>/members/:userId<int>", h.removeMember)
	api.Post("/:id<int>/collab", h.collab)

	// Картинки редактора и txt-экспорт.
	api.Post("/:id<int>/uploads", h.upload)
	api.Get("/:id<int>/export", h.exportNote)

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

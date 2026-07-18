// Package http — HTTP-транспорт notesvc (Fiber): REST /api/notes/*. Все
// приватные ручки требуют только авторизации (RequireAuth) — заметка/папка
// личная и от компании не зависит; скоуп по владельцу и эффективный доступ
// (шары, расшаренные папки-предки) проверяются в сервисе. Публичные ссылки
// /shared/* идут мимо авторизации (код-capability). Хендлеры зовут сервис
// напрямую (без go-kit endpoint-обёрток: middleware-цепочек здесь нет).
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/service"
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

	app := httpserver.New(httpserver.Config{
		AppName: "gw2-notesvc", Log: log, BodyLimit: uploadMaxBytes + 1024*1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{svc: svc, log: log}

	// Публичные ссылки /api/notes/shared/* — мимо авторизации (код-capability).
	api := app.Group("/api/notes", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/notes/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичный доступ по коду ссылки (без авторизации).
	api.Get("/shared/:code", h.sharedNote)
	api.Put("/shared/:code", h.sharedUpdate)

	// Папки (литеральный префикс — до "/:id<int>").
	api.Get("/folders", h.listFolders)
	api.Post("/folders", h.createFolder)
	api.Get("/folders/:id<int>/children", h.folderChildren)
	api.Patch("/folders/:id<int>", h.updateFolder)
	api.Post("/folders/:id<int>/move", h.moveFolder)
	api.Post("/folders/:id<int>/copy", h.copyFolder)
	api.Delete("/folders/:id<int>", h.deleteFolder)
	api.Get("/folders/:id<int>/export", h.exportFolder)
	api.Get("/folders/:id<int>/members", h.listFolderMembers)
	api.Post("/folders/:id<int>/members", h.shareFolder)
	api.Delete("/folders/:id<int>/members/user/:userId<int>", h.unshareFolderUser)
	api.Delete("/folders/:id<int>/members/company/:companyId<int>", h.unshareFolderCompany)

	// Компании пользователя (аудитория шаринга).
	api.Get("/companies", h.myCompanies)

	// Теги.
	api.Get("/tags", h.listTags)
	api.Post("/tags", h.createTag)
	api.Patch("/tags/:id<int>", h.updateTag)
	api.Delete("/tags/:id<int>", h.deleteTag)

	// Заметки.
	api.Get("", h.listNotes)
	api.Post("", h.createNote)
	api.Get("/export", h.exportAll) // zip группировки (scope=all|archive|shared)
	api.Post("/import", h.importNote)
	api.Get("/:id<int>", h.getNote)
	api.Patch("/:id<int>", h.updateNote)
	api.Delete("/:id<int>", h.deleteNote)
	api.Post("/:id<int>/move", h.moveNote)
	api.Post("/:id<int>/copy", h.copyNote)
	api.Put("/:id<int>/tags", h.setTags)
	api.Get("/:id<int>/export", h.exportNote)
	api.Post("/:id<int>/uploads", h.upload)

	// Публичные ссылки (управление владельцем).
	api.Get("/:id<int>/shares", h.listShares)
	api.Post("/:id<int>/shares", h.createShare)
	api.Delete("/:id<int>/shares/:shareId<int>", h.revokeShare)

	// Адресный шаринг заметки (пользователь/компания) и collab-броадкаст.
	api.Get("/:id<int>/members", h.listNoteMembers)
	api.Post("/:id<int>/members", h.shareNote)
	api.Delete("/:id<int>/members/user/:userId<int>", h.unshareNoteUser)
	api.Delete("/:id<int>/members/company/:companyId<int>", h.unshareNoteCompany)
	api.Post("/:id<int>/collab", h.collab)

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

func currentUserID(c *fiber.Ctx) int64    { return pasetoauth.UserID(c) }
func currentCompanyID(c *fiber.Ctx) int64 { return pasetoauth.CompanyID(c) }

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

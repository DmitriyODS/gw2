// Package http — HTTP-транспорт boardsvc (Fiber): REST /api/boards/*. Все
// приватные ручки требуют только авторизации (RequireAuth) — доска/папка
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

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/board/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

const (
	// uploadMaxBytes — лимит картинки на холст (как у картинок заметок).
	uploadMaxBytes = 25 * 1024 * 1024
	// previewMaxBytes — лимит миниатюры доски (её снимает сам холст).
	previewMaxBytes = 3 * 1024 * 1024
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Доски не зависят от
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
		AppName: "gw2-boardsvc", Log: log, BodyLimit: uploadMaxBytes + 1024*1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{svc: svc, log: log}

	// Публичные ссылки /api/boards/shared/* — мимо авторизации (код-capability).
	api := app.Group("/api/boards", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/boards/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичный доступ по коду ссылки (без авторизации).
	api.Get("/shared/:code", h.sharedBoard)
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

	// Доски.
	api.Get("", h.listBoards)
	api.Post("", h.createBoard)
	api.Get("/export", h.exportAll) // zip группировки (scope=all|archive|shared)
	api.Post("/import", h.importBoard)
	api.Get("/:id<int>", h.getBoard)
	api.Patch("/:id<int>", h.updateBoard)
	api.Delete("/:id<int>", h.deleteBoard)
	api.Post("/:id<int>/move", h.moveBoard)
	api.Post("/:id<int>/copy", h.copyBoard)
	api.Get("/:id<int>/export", h.exportBoard)
	api.Post("/:id<int>/uploads", h.upload)
	api.Put("/:id<int>/preview", h.setPreview)

	// Публичные ссылки (управление владельцем).
	api.Get("/:id<int>/shares", h.listShares)
	api.Post("/:id<int>/shares", h.createShare)
	api.Delete("/:id<int>/shares/:shareId<int>", h.revokeShare)

	// Адресный шаринг доски (пользователь/компания) и collab-броадкаст.
	api.Get("/:id<int>/members", h.listBoardMembers)
	api.Post("/:id<int>/members", h.shareBoard)
	api.Delete("/:id<int>/members/user/:userId<int>", h.unshareBoardUser)
	api.Delete("/:id<int>/members/company/:companyId<int>", h.unshareBoardCompany)
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

func currentUserID(c *fiber.Ctx) int64 { return pasetoauth.UserID(c) }

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

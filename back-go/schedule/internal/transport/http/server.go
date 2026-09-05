// Package http — HTTP-транспорт schedulesvc (Fiber): REST /api/schedules/*.
//
// Расписание принадлежит человеку, поэтому ролей компании на роутах нет: здесь
// проверяется только «вошёл ли», а владение и доступ на чтение решает сервис.
// Публичные ссылки /shared/* идут мимо авторизации — доступ по коду-capability.
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/service"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Расписание не зависит от
// компании, поэтому CompanyActive всегда true: отключённая активная компания не
// должна закрывать личный раздел.
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

	app := httpserver.New(httpserver.Config{AppName: "gw2-schedulesvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{svc: svc, log: log}

	// Мидлварь группы монтируется на весь префикс (Fiber), поэтому публичные
	// ссылки пропускаем мимо авторизации: доступ по коду, без сессии.
	api := app.Group("/api/schedules", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/schedules/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичная ссылка (read-only, без входа).
	api.Get("/shared/:code", h.sharedSchedule)

	api.Get("/search", h.search) // глобальный поиск Hola
	api.Get("/agenda", h.agenda) // живая плитка рабочего стола
	api.Get("/directory", h.directory)
	api.Get("/companies", h.companies)

	api.Get("", h.listSchedules) // ?tab=mine|shared
	api.Post("", h.createSchedule)
	api.Post("/import", h.importNew)
	api.Get("/:id<int>", h.getSchedule)
	api.Patch("/:id<int>", h.updateSchedule)
	api.Delete("/:id<int>", h.deleteSchedule)
	api.Get("/:id<int>/export", h.export) // ?format=xlsx|json
	api.Post("/:id<int>/import", h.importInto)

	// Структура: справочник категорий и набор полей карточки.
	api.Put("/:id<int>/fields", h.replaceFields)
	api.Post("/:id<int>/categories", h.createCategory)
	api.Patch("/:id<int>/categories/:cid<int>", h.updateCategory)
	api.Delete("/:id<int>/categories/:cid<int>", h.deleteCategory)

	// Занятия.
	api.Post("/:id<int>/items", h.createItem)
	api.Patch("/:id<int>/items/:itemId<int>", h.updateItem)
	api.Delete("/:id<int>/items/:itemId<int>", h.deleteItem)

	// Публичные ссылки (управление владельцем).
	api.Get("/:id<int>/shares", h.listShares)
	api.Post("/:id<int>/shares", h.createShare)
	api.Delete("/:id<int>/shares/:shareId<int>", h.revokeShare)

	// Адресный доступ.
	api.Get("/:id<int>/access", h.listUserShares)
	api.Post("/:id<int>/access", h.shareWith)
	api.Delete("/:id<int>/access", h.unshare)

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

func paramID(c *fiber.Ctx, name string) int64 {
	id, _ := c.ParamsInt(name)
	return int64(id)
}

// Package http — HTTP-транспорт registrysvc (Fiber): REST /api/registries/*.
// Структуру реестров (создание/правка полей) меняет администратор компании,
// записи — любой её участник; всё скоупится по активной компании из токена.
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/endpoint"
)

const uploadMaxBytes = 25 * 1024 * 1024

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари: активная компания и роль
// берутся ИЗ ТОКЕНА, из БД — только идентичность и активность выбранной компании.
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

	app := fiber.New(fiber.Config{
		AppName:               "gw2-registrysvc",
		DisableStartupMessage: true,
		BodyLimit:             uploadMaxBytes + 1024*1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	employee := auth.RequireRole(domain.LevelEmployee)
	admin := auth.RequireRole(domain.LevelAdmin)

	// Middleware группы монтируется на весь префикс (Fiber), поэтому публичные
	// ссылки /api/registries/shared/* пропускаем мимо авторизации — доступ по
	// коду-capability, без сессии.
	api := app.Group("/api/registries", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/registries/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичный read-only доступ по коду ссылки (без авторизации).
	api.Get("/shared/:code", h.sharedRegistry)
	api.Get("/shared/:code/records", h.sharedRecords)
	api.Get("/shared/:code/export", h.sharedExport)

	// Загрузка файла/картинки записи (любой участник). "/uploads" не конфликтует
	// с "/:id<int>" — параметр матчит только числа.
	api.Post("/uploads", employee, h.upload)

	api.Get("", employee, h.listRegistries)
	api.Post("", admin, h.createRegistry)
	api.Get("/:id<int>", employee, h.getRegistry)
	api.Patch("/:id<int>", admin, h.updateRegistry)
	api.Delete("/:id<int>", admin, h.deleteRegistry)
	api.Put("/:id<int>/fields", admin, h.replaceFields)

	api.Get("/:id<int>/shares", employee, h.listShares)
	api.Post("/:id<int>/shares", employee, h.createShare)
	api.Delete("/:id<int>/shares/:shareId<int>", employee, h.revokeShare)

	api.Get("/:id<int>/records", employee, h.listRecords)
	api.Get("/:id<int>/export", employee, h.exportRecords)
	api.Post("/:id<int>/records", employee, h.createRecord)
	api.Post("/:id<int>/records/bulk-delete", employee, h.bulkDeleteRecords)
	api.Get("/:id<int>/records/:rid<int>", employee, h.getRecord)
	api.Patch("/:id<int>/records/:rid<int>", employee, h.updateRecord)
	api.Delete("/:id<int>/records/:rid<int>", employee, h.deleteRecord)

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

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

func recordID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("rid")
	return int64(id)
}

func currentUser(c *fiber.Ctx) *domain.User {
	u, _ := pasetoauth.CurrentUser(c).(*domain.User)
	return u
}

// companyScope — активная компания участника (эндпоинты под RequireRole, так
// что она всегда задана). ok=false — ответ уже записан.
func companyScope(c *fiber.Ctx) (int64, bool) {
	if u := currentUser(c); u != nil && u.CompanyID != nil {
		return *u.CompanyID, true
	}
	c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "BAD_REQUEST", "message": "Нет активной компании",
	})
	return 0, false
}

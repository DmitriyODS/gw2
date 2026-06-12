// Package http — HTTP-транспорт (Fiber): REST /api/auth/*, /api/users/*,
// /api/companies/*, /api/roles и /api/backup/*.
//
// Пути и формы JSON байт-в-байт совместимы с прежними Flask-блюпринтами
// api/{auth,users,companies,roles,backup}.py — фронт не меняется, nginx
// маршрутизирует эти префиксы на сервис вместо Flask (regex-location
// /api/companies/<id>/ai-settings выигрывает у префикса и уходит в aisvc).
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари (is_hidden, активность
// компании, уровень роли) поверх доменного репозитория.
func authSource(users domain.UserRepository) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetByID(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		return &pasetoauth.AuthInfo{
			RoleLevel:     u.Level(),
			IsHidden:      u.IsHidden,
			CompanyActive: u.CompanyActive(),
			User:          u,
		}, nil
	}
}

func NewServer(eps endpoint.Endpoints, verifier *pasetoauth.Verifier,
	users domain.UserRepository, log *slog.Logger) *Server {

	app := fiber.New(fiber.Config{
		AppName:               "gw2-authsvc",
		DisableStartupMessage: true,
		// Лимит тела — под импорт ZIP-бэкапа (в проде фактический потолок —
		// client_max_body_size nginx); аватарка ≤2МБ проверяется в хендлере.
		BodyLimit: 64 * 1024 * 1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	authAPI := app.Group("/api/auth")
	authAPI.Post("/login", h.login)
	authAPI.Post("/refresh", h.refresh)
	authAPI.Post("/logout", auth.RequireToken, h.logout)
	authAPI.Post("/change-default", auth.RequireToken, h.changeDefault)

	usersAPI := app.Group("/api/users")
	usersAPI.Get("", auth.RequireAuth, auth.RequireRole(domain.LevelDirector), h.listUsers)
	usersAPI.Post("", auth.RequireAuth, auth.RequireRole(domain.LevelDirector), h.createUser)
	usersAPI.Get("/directory", auth.RequireAuth, h.directory)
	usersAPI.Get("/directory/:id<int>", auth.RequireAuth, h.directoryUser)
	usersAPI.Get("/me", auth.RequireAuth, h.me)
	usersAPI.Patch("/me", auth.RequireAuth, h.updateMe)
	usersAPI.Post("/me/avatar", auth.RequireAuth, h.uploadAvatar)
	usersAPI.Delete("/me/avatar", auth.RequireAuth, h.deleteAvatar)
	usersAPI.Get("/:id<int>/identicon", h.identicon) // публичный (img src)
	usersAPI.Get("/:id<int>", auth.RequireAuth, auth.RequireRole(domain.LevelDirector), h.getUser)
	usersAPI.Patch("/:id<int>", auth.RequireAuth, auth.RequireRole(domain.LevelDirector), h.updateUser)
	usersAPI.Delete("/:id<int>", auth.RequireAuth, auth.RequireRole(domain.LevelDirector), h.hideUser)
	usersAPI.Patch("/:id<int>/role", auth.RequireAuth, auth.RequireRole(domain.LevelDirector), h.assignRole)
	usersAPI.Post("/:id<int>/reset-password", auth.RequireAuth, auth.RequireRole(domain.LevelDirector), h.resetPassword)

	app.Get("/api/roles", auth.RequireAuth, h.listRoles)

	// Компании: CRUD — Администратор системы; weekend-settings — DIRECTOR+
	// (Руководитель — своей компании, проверка доступа в сервисном слое).
	companiesAPI := app.Group("/api/companies", auth.RequireAuth)
	companiesAPI.Get("", auth.RequireRole(domain.LevelAdmin), h.listCompanies)
	companiesAPI.Post("", auth.RequireRole(domain.LevelAdmin), h.createCompany)
	companiesAPI.Get("/:id<int>", auth.RequireRole(domain.LevelAdmin), h.getCompany)
	companiesAPI.Patch("/:id<int>", auth.RequireRole(domain.LevelAdmin), h.updateCompany)
	companiesAPI.Delete("/:id<int>", auth.RequireRole(domain.LevelAdmin), h.deleteCompany)
	companiesAPI.Patch("/:id<int>/toggle-active", auth.RequireRole(domain.LevelAdmin), h.toggleCompanyActive)
	companiesAPI.Get("/:id<int>/weekend-settings", auth.RequireRole(domain.LevelDirector), h.getWeekendSettings)
	companiesAPI.Put("/:id<int>/weekend-settings", auth.RequireRole(domain.LevelDirector), h.updateWeekendSettings)

	backupAPI := app.Group("/api/backup", auth.RequireAuth, auth.RequireRole(domain.LevelAdmin))
	backupAPI.Get("/export", h.exportBackup)
	backupAPI.Post("/import", h.importBackup)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	eps endpoint.Endpoints
	log *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...}
// (+Extra-поля: retry_after_sec, company_name) с её HTTP-статусом; прочее —
// 500, как Flask-обработчик ошибок.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func badRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "VALIDATION_ERROR", "message": message,
	})
}

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

// currentUser — полный доменный пользователь из Locals (после RequireAuth).
func currentUser(c *fiber.Ctx) *domain.User {
	u, _ := pasetoauth.CurrentUser(c).(*domain.User)
	return u
}

// tokenUserID — id пользователя из Locals (после RequireToken/RequireAuth).
func tokenUserID(c *fiber.Ctx) int64 {
	return pasetoauth.UserID(c)
}

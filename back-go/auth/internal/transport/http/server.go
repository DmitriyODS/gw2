// Package http — HTTP-транспорт (Fiber): REST /api/auth/* и /api/users/*.
//
// Пути и формы JSON байт-в-байт совместимы с прежними Flask-блюпринтами
// api/auth.py и api/users.py — фронт не меняется, nginx маршрутизирует
// эти префиксы на сервис вместо Flask.
package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/token"
)

type Server struct {
	app *fiber.App
}

func NewServer(eps endpoint.Endpoints, verifier *token.Verifier,
	users domain.UserRepository, log *slog.Logger) *Server {

	app := fiber.New(fiber.Config{
		AppName:               "gw2-authsvc",
		DisableStartupMessage: true,
		// Аватарка ≤2МБ проверяется в хендлере; лимит тела — с запасом.
		BodyLimit: 5 * 1024 * 1024,
	})
	auth := &authParser{verifier: verifier, users: users}
	h := &handlers{eps: eps, auth: auth, log: log}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	authAPI := app.Group("/api/auth")
	authAPI.Post("/login", h.login)
	authAPI.Post("/refresh", h.refresh)
	authAPI.Post("/logout", auth.requireToken, h.logout)
	authAPI.Post("/change-default", auth.requireToken, h.changeDefault)

	usersAPI := app.Group("/api/users")
	usersAPI.Get("", auth.requireAuth, auth.requireRole(domain.LevelDirector), h.listUsers)
	usersAPI.Post("", auth.requireAuth, auth.requireRole(domain.LevelDirector), h.createUser)
	usersAPI.Get("/directory", auth.requireAuth, h.directory)
	usersAPI.Get("/directory/:id<int>", auth.requireAuth, h.directoryUser)
	usersAPI.Get("/me", auth.requireAuth, h.me)
	usersAPI.Patch("/me", auth.requireAuth, h.updateMe)
	usersAPI.Post("/me/avatar", auth.requireAuth, h.uploadAvatar)
	usersAPI.Delete("/me/avatar", auth.requireAuth, h.deleteAvatar)
	usersAPI.Get("/:id<int>/identicon", h.identicon) // публичный (img src)
	usersAPI.Get("/:id<int>", auth.requireAuth, auth.requireRole(domain.LevelDirector), h.getUser)
	usersAPI.Patch("/:id<int>", auth.requireAuth, auth.requireRole(domain.LevelDirector), h.updateUser)
	usersAPI.Delete("/:id<int>", auth.requireAuth, auth.requireRole(domain.LevelDirector), h.hideUser)
	usersAPI.Patch("/:id<int>/role", auth.requireAuth, auth.requireRole(domain.LevelDirector), h.assignRole)
	usersAPI.Post("/:id<int>/reset-password", auth.requireAuth, auth.requireRole(domain.LevelDirector), h.resetPassword)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	eps  endpoint.Endpoints
	auth *authParser
	log  *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...}
// (+Extra-поля: retry_after_sec, company_name) с её HTTP-статусом; прочее —
// 500, как Flask-обработчик ошибок.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	if de := domain.AsDomainError(err); de != nil {
		body := fiber.Map{"error": de.Code, "message": de.Message}
		for k, v := range de.Extra {
			body[k] = v
		}
		return c.Status(de.HTTPStatus).JSON(body)
	}
	h.log.Error("http.internal_error", "path", c.Path(), "error", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера",
	})
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

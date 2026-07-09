// Package http — HTTP-транспорт portalsvc (Fiber): REST /api/portal/*.
// Топики ведёт администратор компании, посты/комментарии/реакции — любой
// участник; всё скоупится по активной компании из токена.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/endpoint"
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

	app := httpserver.New(httpserver.Config{
		AppName: "gw2-portalsvc", Log: log, BodyLimit: uploadMaxBytes + 1024*1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	employee := auth.RequireRole(domain.LevelEmployee)
	admin := auth.RequireRole(domain.LevelAdmin)

	api := app.Group("/api/portal", auth.RequireAuth)

	api.Get("/topics", employee, h.listTopics)
	api.Post("/topics", admin, h.createTopic)
	api.Patch("/topics/:id<int>", admin, h.updateTopic)
	api.Delete("/topics/:id<int>", admin, h.deleteTopic)

	api.Get("/posts", employee, h.listPosts)
	api.Post("/posts", employee, h.createPost)
	api.Get("/posts/:id<int>", employee, h.getPost)
	api.Patch("/posts/:id<int>", employee, h.updatePost)
	api.Delete("/posts/:id<int>", employee, h.deletePost)
	api.Post("/posts/:id<int>/pin", employee, h.pinPost)
	api.Delete("/posts/:id<int>/pin", employee, h.unpinPost)
	api.Post("/posts/:id<int>/attachments", employee, h.upload)
	api.Delete("/attachments/:id<int>", employee, h.removeAttachment)
	api.Post("/posts/:id<int>/forward", employee, h.forwardPost)

	api.Get("/posts/:id<int>/comments", employee, h.listComments)
	api.Post("/posts/:id<int>/comments", employee, h.createComment)
	api.Delete("/comments/:id<int>", employee, h.deleteComment)

	api.Post("/posts/:id<int>/reactions", employee, h.addReaction)
	api.Delete("/posts/:id<int>/reactions", employee, h.removeReaction)

	api.Get("/unread", employee, h.unreadCount)
	api.Post("/seen", employee, h.markSeen)

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

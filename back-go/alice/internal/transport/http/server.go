// Package http — HTTP-транспорт alicesvc: публичный вебхук Яндекс.Диалогов.
// Авторизации в мидлварях нет — токен пользователя приходит в теле запроса
// (session.user.access_token, связка аккаунтов) и проверяется в сервисе.
package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/alice/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
)

type Server struct {
	app *fiber.App
}

func NewServer(svc *service.Service, log *slog.Logger) *Server {
	app := httpserver.New(httpserver.Config{AppName: "gw2-alicesvc", Log: log})

	app.Post("/api/alice/webhook", func(c *fiber.Ctx) error {
		var req domain.WebhookRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "VALIDATION", "message": "Некорректное тело запроса",
			})
		}
		return c.JSON(svc.Handle(c.Context(), &req))
	})

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

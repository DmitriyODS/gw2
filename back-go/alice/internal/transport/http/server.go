// Package http — HTTP-транспорт alicesvc: публичный вебхук Яндекс.Диалогов.
// Авторизации в мидлварях нет — токен пользователя приходит в теле запроса
// (session.user.access_token, связка аккаунтов) и проверяется в сервисе.
package http

import (
	"encoding/json"
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
		// json.Unmarshal вместо BodyParser: валидатор Диалогов шлёт запрос
		// БЕЗ заголовка Content-Type, а BodyParser без него отказывает (400).
		// Вебхук ОБЯЗАН всегда отвечать 200 с валидным телом Диалогов — любой
		// не-200 (в т.ч. на пустой/битый пинг валидатора) читается модерацией
		// как «При обращении к серверу возникает ошибка». На нераспарсиваемое
		// тело отдаём штатное приветствие, а не ошибку.
		var req domain.WebhookRequest
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return c.JSON(svc.Fallback())
		}
		return c.JSON(svc.Handle(c.Context(), &req))
	})

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

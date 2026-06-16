// Package http — минимальный HTTP-сервер mailsvc: только /healthz (наружу не
// проксируется, нужен для docker healthcheck). Вся отправка идёт по gRPC.
package http

import "github.com/gofiber/fiber/v2"

type Server struct{ app *fiber.App }

func NewServer() *Server {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

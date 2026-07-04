// Package http — минимальный HTTP-сервер mailsvc: только /healthz (наружу не
// проксируется, нужен для docker healthcheck). Вся отправка идёт по gRPC.
package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
)

type Server struct{ app *fiber.App }

func NewServer() *Server {
	return &Server{app: httpserver.New(httpserver.Config{AppName: "gw2-mailsvc"})}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

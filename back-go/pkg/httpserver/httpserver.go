// Package httpserver — общий Fiber-сетап HTTP-сервисов платформы: единая
// конфигурация (AppName, тихий старт, лимит тела), recover-мидлварь,
// сквозной request-id и стандартный /healthz. Сервисы создают приложение
// через New и вешают только свои маршруты — без дублирования boilerplate.
package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	recoverer "github.com/gofiber/fiber/v2/middleware/recover"
)

// HeaderRequestID — заголовок сквозного идентификатора запроса (принимается
// от прокси, иначе генерируется).
const HeaderRequestID = "X-Request-ID"

// localsKey — ключ request-id в fiber.Ctx.Locals.
const localsKey = "request_id"

type Config struct {
	AppName string
	Log     *slog.Logger // nil — без access-лога
	// BodyLimit — лимит тела запроса в байтах; 0 — дефолт Fiber (4 МБ).
	BodyLimit int
}

// New — Fiber-приложение с общим сетапом: recover, request-id, access-лог
// (Debug; 5xx — Warn) и GET /healthz.
func New(cfg Config) *fiber.App {
	fc := fiber.Config{
		AppName:               cfg.AppName,
		DisableStartupMessage: true,
	}
	if cfg.BodyLimit > 0 {
		fc.BodyLimit = cfg.BodyLimit
	}
	app := fiber.New(fc)
	app.Use(recoverer.New())
	app.Use(requestID())
	if cfg.Log != nil {
		app.Use(accessLog(cfg.Log))
	}
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	return app
}

// RequestID — сквозной идентификатор текущего запроса ("" вне запроса).
func RequestID(c *fiber.Ctx) string {
	id, _ := c.Locals(localsKey).(string)
	return id
}

func requestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get(HeaderRequestID)
		if id == "" || len(id) > 64 {
			id = newID()
		}
		c.Locals(localsKey, id)
		c.Set(HeaderRequestID, id)
		return c.Next()
	}
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// accessLog — лог каждого запроса с request-id: обычные — Debug (в prod при
// уровне Info не шумит), ответы 5xx — Warn, паника/ошибка хендлера — тоже Warn
// (тело ответа формирует ErrorHandler/recover).
func accessLog(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		attrs := []any{
			"method", c.Method(), "path", c.Path(), "status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", RequestID(c),
		}
		switch {
		case err != nil:
			log.Warn("http.request_failed", append(attrs, "error", err)...)
		case status >= 500:
			log.Warn("http.request", attrs...)
		default:
			log.Debug("http.request", attrs...)
		}
		return err
	}
}

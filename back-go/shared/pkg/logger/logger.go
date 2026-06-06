// Package logger — единый slog JSON логгер для всех сервисов. Уровень берётся
// из env-конфига каждого сервиса (LOG_LEVEL=debug|info|warn|error).
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New создаёт JSON-логгер с указанным уровнем. Пишет в stdout —
// docker logs/journalctl сами разруливают ротацию.
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

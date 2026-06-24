// Package storage — единая абстракция файлового хранилища пользовательского
// контента (вложения мессенджера, картинки/файлы реестров и календарей,
// аватарки). Два бэкенда: local (диск, dev) и s3 (объектное хранилище, prod);
// выбор — env STORAGE_BACKEND.
//
// Ключ объекта = относительный путь вида "registry/<hex>.jpg" — он же кладётся
// в БД и отдаётся клиенту по /uploads/<key> (в prod nginx проксирует /uploads/
// на S3-бакет, в dev файлы лежат на диске и раздаются nginx/vite).
package storage

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
)

// Storage — хранилище объектов по строковому ключу (относительному пути).
type Storage interface {
	// Put — записать объект; contentType важен для корректной отдачи из S3.
	Put(ctx context.Context, key string, data []byte, contentType string) error
	// Open — открыть объект на чтение (бэкап/копирование). Закрыть вызывающему.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Copy — серверная копия объекта (пересылка вложений без скачивания).
	Copy(ctx context.Context, srcKey, dstKey string) error
	// Remove — best-effort удаление; ошибки — только warn-лог.
	Remove(ctx context.Context, keys ...string)
	// List — ключи объектов по префиксу (для бэкапа avatars/).
	List(ctx context.Context, prefix string) ([]string, error)
}

// FromEnv — построить хранилище по env. STORAGE_BACKEND: "local" (дефолт) |
// "s3". Для s3 обязательны S3_ENDPOINT/S3_BUCKET/S3_ACCESS_KEY/S3_SECRET_KEY
// (S3_REGION дефолт "ru1", S3_SECURE дефолт true). Для local корень —
// defaultLocalRoot (обычно из UPLOAD_FOLDER сервиса).
func FromEnv(log *slog.Logger, defaultLocalRoot string) Storage {
	switch strings.ToLower(bootstrap.Env("STORAGE_BACKEND", "local")) {
	case "s3":
		st, err := NewS3(S3Config{
			Endpoint:  bootstrap.MustEnv(log, "S3_ENDPOINT"),
			Region:    bootstrap.Env("S3_REGION", "ru1"),
			Bucket:    bootstrap.MustEnv(log, "S3_BUCKET"),
			AccessKey: bootstrap.MustEnv(log, "S3_ACCESS_KEY"),
			SecretKey: bootstrap.MustEnv(log, "S3_SECRET_KEY"),
			Secure:    bootstrap.Env("S3_SECURE", "true") != "false",
		}, log)
		if err != nil {
			log.Error("storage.s3_init_failed", "error", err)
			os.Exit(1)
		}
		log.Info("storage.backend", "kind", "s3", "endpoint", bootstrap.Env("S3_ENDPOINT", ""))
		return st
	default:
		log.Info("storage.backend", "kind", "local", "root", defaultLocalRoot)
		return NewLocal(defaultLocalRoot, log)
	}
}

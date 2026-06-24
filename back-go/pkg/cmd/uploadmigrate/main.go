// Команда uploadmigrate — одноразовый перенос файлов из локального каталога
// (uploads-том) в S3-бакет через pkg/storage. Каждый объект кладётся с
// public-read ACL (Put), Content-Type выводится из расширения. Идемпотентна:
// существующие объекты перезаписываются тем же содержимым.
//
// Запуск (см. scripts/migrate_uploads_s3.sh):
//
//	UPLOAD_FOLDER=/data STORAGE_BACKEND=s3 S3_ENDPOINT=… S3_BUCKET=… \
//	S3_ACCESS_KEY=… S3_SECRET_KEY=… go run ./cmd/uploadmigrate
package main

import (
	"context"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

func main() {
	log := bootstrap.Logger()
	root := bootstrap.Env("UPLOAD_FOLDER", "/data")

	src := storage.NewLocal(root, log)
	dst, err := storage.NewS3(storage.S3Config{
		Endpoint:  bootstrap.MustEnv(log, "S3_ENDPOINT"),
		Region:    bootstrap.Env("S3_REGION", "ru1"),
		Bucket:    bootstrap.MustEnv(log, "S3_BUCKET"),
		AccessKey: bootstrap.MustEnv(log, "S3_ACCESS_KEY"),
		SecretKey: bootstrap.MustEnv(log, "S3_SECRET_KEY"),
		Secure:    bootstrap.Env("S3_SECURE", "true") != "false",
	}, log)
	if err != nil {
		log.Error("migrate.s3_init", "error", err)
		return
	}

	ctx := context.Background()
	keys, err := src.List(ctx, "")
	if err != nil {
		log.Error("migrate.list", "error", err)
		return
	}

	var ok, failed int
	for _, key := range keys {
		rc, err := src.Open(ctx, key)
		if err != nil {
			log.Warn("migrate.open", "key", key, "error", err)
			failed++
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			log.Warn("migrate.read", "key", key, "error", err)
			failed++
			continue
		}
		if err := dst.Put(ctx, key, data, contentType(key)); err != nil {
			log.Warn("migrate.put", "key", key, "error", err)
			failed++
			continue
		}
		ok++
	}
	log.Info("migrate.done", "uploaded", ok, "failed", failed, "total", len(keys))
}

func contentType(key string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(key))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

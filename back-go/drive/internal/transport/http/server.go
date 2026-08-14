// Package http — HTTP-транспорт drivesvc (Fiber): REST /api/drive/*.
//
// Приватные ручки требуют только авторизации (RequireAuth): диск личный и от
// компании не зависит, а скоуп по владельцу и эффективный доступ (шары,
// расшаренные папки-предки) проверяет сервис. Публичные ссылки /shared/*
// идут мимо авторизации — код в адресе и есть capability.
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/drive/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — диск не зависит от компании, поэтому CompanyActive всегда true;
// из БД берём только глобальную активность пользователя.
func authSource(users domain.UserReader) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64, _ pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetUser(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		return &pasetoauth.AuthInfo{IsActive: true, CompanyActive: true, User: u}, nil
	}
}

func NewServer(svc *service.Service, users domain.UserReader, uploads *chunkupload.Manager,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := httpserver.New(httpserver.Config{
		AppName: "gw2-drivesvc", Log: log,
		/* Потолок ОДНОГО запроса: сюда влезает мелкий файл целиком и любая
		   часть крупного. Сам файл может быть куда больше — он приезжает
		   частями (chunkupload), а не одним телом. */
		BodyLimit: chunkupload.ChunkSize + 8<<20,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{svc: svc, log: log}

	// Мимо авторизации пропускаем ТОЛЬКО ссылки по коду. Префикс со слэшем на
	// конце обязателен: без него сюда попадал и «/shared-with-me», а он про
	// личную выдачу и без пользователя бессмыслен.
	api := app.Group("/api/drive", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/drive/shared/") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичные ссылки (без авторизации).
	api.Get("/shared/:code", h.sharedTarget)
	api.Get("/shared/:code/list", h.sharedList)
	api.Get("/shared/:code/download", h.sharedDownload)

	// Обзор: содержимое папки и сквозные выборки (корзина, избранное, недавние).
	api.Get("", h.browse)
	api.Get("/shared-with-me", h.sharedWithMe)
	api.Get("/users", h.searchUsers)

	// Папки (литеральный префикс — до "/:id<int>").
	api.Post("/folders", h.createFolder)
	api.Patch("/folders/:id<int>", h.updateFolder)
	api.Post("/folders/:id<int>/move", h.moveFolder)
	api.Post("/folders/:id<int>/trash", h.trashFolder)
	api.Post("/folders/:id<int>/restore", h.restoreFolder)
	api.Delete("/folders/:id<int>", h.purgeFolder)
	api.Get("/folders/:id<int>/access", h.folderAccessList)
	api.Post("/folders/:id<int>/access", h.shareFolder)
	api.Post("/folders/:id<int>/links", h.createFolderLink)

	// Корзина.
	api.Delete("/trash", h.emptyTrash)

	/* Загрузка по частям (большие файлы) — литеральный префикс до "/files".
	   Механика общая для всей платформы (pkg/chunkupload): сессия живёт в БД,
	   части — объектами в хранилище, поэтому соседние куски могут попасть на
	   разные инстансы, а сборка идёт потоком. */
	chunkupload.Handlers{
		Manager: uploads,
		UserID:  currentUserID,
		Begin:   h.beginUpload,
		Finish:  h.finishUpload,
		Respond: h.respondError,
	}.Mount(api, "/uploads")

	// Файлы.
	api.Post("/files", h.upload)
	api.Get("/files/:id<int>", h.getFile)
	api.Get("/files/:id<int>/download", h.download)
	api.Patch("/files/:id<int>", h.updateFile)
	api.Post("/files/:id<int>/move", h.moveFile)
	api.Post("/files/:id<int>/star", h.starFile)
	api.Post("/files/:id<int>/trash", h.trashFile)
	api.Post("/files/:id<int>/restore", h.restoreFile)
	api.Delete("/files/:id<int>", h.purgeFile)
	api.Get("/files/:id<int>/access", h.fileAccessList)
	api.Post("/files/:id<int>/access", h.shareFile)
	api.Post("/files/:id<int>/links", h.createFileLink)

	// Снятие доступа — общее для файлов и папок (id самой выдачи).
	api.Delete("/access/:id<int>", h.revokeAccess)
	api.Delete("/links/:id<int>", h.deleteLink)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	svc *service.Service
	log *slog.Logger
}

func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func currentUserID(c *fiber.Ctx) int64 { return pasetoauth.UserID(c) }

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

// optionalID — необязательный числовой параметр запроса или тела: nil означает
// «корень диска», и отличить его от нуля обязательно.
func optionalID(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}

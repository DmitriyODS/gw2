// Package http — HTTP-транспорт formsvc (Fiber): REST /api/forms/*.
//
// Форма принадлежит человеку, поэтому ролей компании на роутах нет: доступ
// решает уровень шары, и проверяет его СЕРВИС на входе каждой операции. Здесь
// остаётся только «вошёл ли» — плюс публичные ссылки, которые пускают и гостя.
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/forms/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

// bodyLimit — потолок ОДНОГО запроса. Крупный файл сюда не влезает и не должен:
// всё больше chunkupload.Threshold приезжает частями.
const bodyLimit = chunkupload.ChunkSize + 8*1024*1024

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари: активная компания и роль
// берутся ИЗ ТОКЕНА, из БД — только идентичность и активность компании.
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

func NewServer(svc *service.Service, users domain.UserReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := httpserver.New(httpserver.Config{
		AppName: "gw2-formsvc", Log: log, BodyLimit: bodyLimit,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{svc: svc, auth: auth, log: log}

	// Middleware группы монтируется на весь префикс (Fiber), поэтому публичные
	// ссылки /api/forms/shared/* пропускаем мимо обязательной авторизации:
	// форму заполняет и тот, у кого аккаунта нет вовсе. Токен там разбирают сами
	// хендлеры (см. visitor) — ссылка бывает «только для своих», да и журнал
	// переходов должен знать вошедшего по имени.
	api := app.Group("/api/forms", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/forms/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичное заполнение по коду ссылки.
	api.Get("/shared/:code", h.sharedForm)
	api.Post("/shared/:code/responses", h.sharedSubmit)
	api.Post("/shared/:code/uploads", h.sharedUpload)

	// Загрузка файлов ответа. "/uploads" не конфликтует с "/:id<int>" —
	// параметр матчит только числа.
	api.Post("/uploads", h.upload)
	api.Post("/uploads/init", h.beginUpload)
	api.Post("/uploads/:code/chunk", h.writeChunk)
	api.Post("/uploads/:code/finish", h.finishUpload)
	api.Delete("/uploads/:code", h.cancelUpload)

	api.Get("/search", h.searchForms) // глобальный поиск Hola
	// Кандидаты в адресаты назначения: коллеги и компании самого спрашивающего.
	api.Get("/directory", h.directory)
	api.Get("/companies", h.companies)

	api.Get("", h.listForms)
	api.Post("", h.createForm)
	api.Get("/:id<int>", h.getForm)
	api.Patch("/:id<int>", h.updateForm)
	api.Delete("/:id<int>", h.deleteForm)
	api.Post("/:id<int>/duplicate", h.duplicateForm)
	api.Put("/:id<int>/structure", h.replaceStructure)

	// Заполнение и собственный ответ.
	api.Get("/:id<int>/fill", h.fill)
	api.Post("/:id<int>/responses", h.submit)
	api.Patch("/:id<int>/responses/mine", h.updateMine)

	// Собранные ответы, сводка и контроль исполнения.
	api.Get("/:id<int>/responses", h.listResponses)
	api.Post("/:id<int>/responses/bulk-delete", h.bulkDeleteResponses)
	api.Get("/:id<int>/responses/:rid<int>", h.getResponse)
	api.Delete("/:id<int>/responses/:rid<int>", h.deleteResponse)
	api.Post("/:id<int>/grades", h.publishGrades)
	api.Get("/:id<int>/summary", h.summary)
	api.Get("/:id<int>/progress", h.progress)
	api.Get("/:id<int>/export", h.exportResponses)

	// Внешние ссылки и журнал переходов.
	api.Get("/:id<int>/shares", h.listShares)
	api.Post("/:id<int>/shares", h.createShare)
	api.Patch("/:id<int>/shares/:shareId<int>", h.updateShare)
	api.Delete("/:id<int>/shares/:shareId<int>", h.revokeShare)
	api.Get("/:id<int>/shares/:shareId<int>/visits", h.shareVisits)

	// Адресный доступ и назначения.
	api.Get("/:id<int>/access", h.listUserShares)
	api.Post("/:id<int>/access", h.shareWith)
	api.Delete("/:id<int>/access", h.unshare)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	svc *service.Service
	// auth — нужен публичным роутам: там авторизация необязательна, и токен
	// разбирается вручную, а не мидлварью.
	auth *pasetoauth.Middleware
	log  *slog.Logger
}

func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

func responseID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("rid")
	return int64(id)
}

func shareID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("shareId")
	return int64(id)
}

func currentUser(c *fiber.Ctx) *domain.User {
	u, _ := pasetoauth.CurrentUser(c).(*domain.User)
	return u
}

// userID — кто выполняет запрос (роуты под RequireAuth, поэтому он всегда есть).
func userID(c *fiber.Ctx) int64 {
	if u := currentUser(c); u != nil {
		return u.ID
	}
	return 0
}

// activeCompany — компания сессии; nil, если человек ни в одной не состоит либо
// не выбрал активную. Нужна при СОЗДАНИИ формы: она решает, чья квота платит за
// файлы ответов и что предложить в «назначить компании».
func activeCompany(c *fiber.Ctx) *int64 {
	if u := currentUser(c); u != nil {
		return u.CompanyID
	}
	return nil
}

// visitor — кто открывает публичную ссылку. Адрес берём из заголовков прокси:
// nginx проставляет X-Forwarded-For, и без него в журнале осел бы адрес самого
// прокси, одинаковый для всех.
func (h *handlers) visitor(c *fiber.Ctx) service.Visitor {
	v := service.Visitor{
		IP:        clientIP(c),
		UserAgent: string(c.Request().Header.Peek("User-Agent")),
	}
	if id := h.auth.OptionalUserID(c); id > 0 {
		v.UserID = &id
	}
	return v
}

func clientIP(c *fiber.Ctx) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if raw := string(c.Request().Header.Peek(header)); raw != "" {
			// X-Forwarded-For — цепочка «клиент, прокси1, прокси2».
			if first, _, ok := strings.Cut(raw, ","); ok {
				return strings.TrimSpace(first)
			}
			return strings.TrimSpace(raw)
		}
	}
	return c.IP()
}

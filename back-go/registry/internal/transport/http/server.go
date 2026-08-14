// Package http — HTTP-транспорт registrysvc (Fiber): REST /api/registries/*.
//
// Реестр принадлежит человеку, поэтому ролей компании на роутах нет: доступ
// решает уровень шары, и проверяет его СЕРВИС на входе каждой операции. Здесь
// остаётся только «вошёл ли» — плюс публичные ссылки, которые пускают и гостя.
package http

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/service"
)

// bodyLimit — потолок ОДНОГО запроса. Гигабайтный файл сюда не влезает и не
// должен: всё крупнее chunkupload.Threshold приезжает частями, а часть заведомо
// меньше этого предела.
const bodyLimit = chunkupload.ChunkSize + 8*1024*1024

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари: активная компания и роль
// берутся ИЗ ТОКЕНА, из БД — только идентичность и активность выбранной компании.
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
		AppName: "gw2-registrysvc", Log: log, BodyLimit: bodyLimit,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{svc: svc, auth: auth, log: log}

	// Middleware группы монтируется на весь префикс (Fiber), поэтому публичные
	// ссылки /api/registries/shared/* пропускаем мимо обязательной авторизации.
	// Токен там разбирают сами хендлеры (см. visitor): ссылка бывает «только
	// для своих», да и журнал посещений должен знать вошедшего по имени.
	api := app.Group("/api/registries", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/registries/shared") {
			return c.Next()
		}
		return auth.RequireAuth(c)
	})

	// Публичный доступ по коду ссылки. Уровень доступа проверяет сервис — у
	// транспорта сведений о правах нет.
	api.Get("/shared/:code", h.sharedRegistry)
	api.Get("/shared/:code/records", h.sharedRecords)
	api.Get("/shared/:code/export", h.sharedExport)
	api.Post("/shared/:code/records", h.sharedCreateRecord)
	api.Patch("/shared/:code/records/:rid<int>", h.sharedUpdateRecord)
	api.Delete("/shared/:code/records/:rid<int>", h.sharedDeleteRecord)
	api.Post("/shared/:code/uploads", h.sharedUpload)
	// Учётный реестр по ссылке: выдача — работа с записями, поэтому доступна
	// с уровня edit, как и сами записи.
	api.Get("/shared/:code/records/:rid<int>/issues", h.sharedIssueHistory)
	api.Post("/shared/:code/records/:rid<int>/issue", h.sharedIssue)
	api.Post("/shared/:code/records/:rid<int>/extend", h.sharedExtendIssue)
	api.Post("/shared/:code/records/:rid<int>/return", h.sharedReturnIssue)
	// Уровень «администрирование»: правка самого реестра, а не только записей.
	api.Patch("/shared/:code", h.sharedUpdateRegistry)
	api.Put("/shared/:code/fields", h.sharedReplaceFields)

	// Загрузка файла/картинки записи. "/uploads" не конфликтует с "/:id<int>" —
	// параметр матчит только числа.
	api.Post("/uploads", h.upload)
	api.Post("/uploads/init", h.beginUpload)
	api.Post("/uploads/:code/chunk", h.writeChunk)
	api.Post("/uploads/:code/finish", h.finishUpload)
	api.Delete("/uploads/:code", h.cancelUpload)

	api.Get("/search", h.searchRecords) // глобальный поиск Hola
	// Кандидаты в адресаты шаринга: коллеги и компании самого спрашивающего.
	api.Get("/directory", h.directory)
	api.Get("/companies", h.companies)

	api.Get("", h.listRegistries)
	api.Post("", h.createRegistry)
	api.Get("/:id<int>", h.getRegistry)
	api.Patch("/:id<int>", h.updateRegistry)
	api.Delete("/:id<int>", h.deleteRegistry)
	api.Put("/:id<int>/fields", h.replaceFields)

	// Внешние ссылки и журнал переходов.
	api.Get("/:id<int>/shares", h.listShares)
	api.Post("/:id<int>/shares", h.createShare)
	api.Patch("/:id<int>/shares/:shareId<int>", h.updateShare)
	api.Delete("/:id<int>/shares/:shareId<int>", h.revokeShare)
	api.Get("/:id<int>/shares/:shareId<int>/visits", h.shareVisits)

	// Адресный доступ людям и компаниям.
	api.Get("/:id<int>/access", h.listUserShares)
	api.Post("/:id<int>/access", h.shareWith)
	api.Delete("/:id<int>/access", h.unshare)

	api.Get("/:id<int>/records", h.listRecords)
	api.Get("/:id<int>/export", h.exportRecords)
	api.Post("/:id<int>/records", h.createRecord)
	api.Post("/:id<int>/records/bulk-delete", h.bulkDeleteRecords)
	api.Get("/:id<int>/records/:rid<int>", h.getRecord)
	api.Patch("/:id<int>/records/:rid<int>", h.updateRecord)
	api.Delete("/:id<int>/records/:rid<int>", h.deleteRecord)

	// Учётный реестр: выдача, продление, возврат и история движения позиции.
	api.Get("/:id<int>/records/:rid<int>/issues", h.issueHistory)
	api.Post("/:id<int>/records/:rid<int>/issue", h.issue)
	api.Post("/:id<int>/records/:rid<int>/extend", h.extendIssue)
	api.Post("/:id<int>/records/:rid<int>/return", h.returnIssue)

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

func recordID(c *fiber.Ctx) int64 {
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
// не выбрал активную. Нужна только при СОЗДАНИИ реестра: она решает, чья квота
// платит за файлы и что предложить в «поделиться с компанией».
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

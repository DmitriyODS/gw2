// Package http — HTTP-транспорт (Fiber): REST /api/tasks/*, /api/units/*,
// /api/unit-types/*, /api/departments/*, /api/stages/* и /api/stats/*.
//
// Пути и формы JSON байт-в-байт совместимы с прежними Flask-блюпринтами
// api/{tasks,units,unit_types,departments,stages,stats}.py — фронт не
// меняется, nginx маршрутизирует эти префиксы на сервис вместо Flask.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Активная компания и роль
// в ней берутся ИЗ ТОКЕНА (active), из БД — только идентичность (is_active,
// is_super_admin, ФИО) и активность выбранной компании. Так пользователь
// скоупится по выбранной им компании (claims.CompanyID), а не по принадлежности
// в users (её там больше нет).
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

func NewServer(eps endpoint.Endpoints, users domain.UserReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := httpserver.New(httpserver.Config{AppName: "gw2-tasksvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	employee := auth.RequireRole(domain.LevelEmployee)
	manager := auth.RequireRole(domain.LevelManager)
	admin := auth.RequireRole(domain.LevelAdmin)

	// Вебхук YouGile — публичный (без токена): авторизация через secret
	// в URL. Регистрируется вне auth-группы.
	app.Post("/api/yougile/webhook/:companyId<int>/:secret", h.yougileWebhook)

	ygAPI := app.Group("/api/yougile", auth.RequireAuth)
	ygAPI.Get("/status", h.yougileStatus)
	ygAPI.Post("/account", h.yougileConnect)
	ygAPI.Delete("/account", h.yougileDisconnect)
	ygAPI.Post("/account/rotate", h.yougileRotate)
	ygAPI.Post("/companies/lookup", admin, h.yougileLookupCompanies)
	ygAPI.Get("/projects", admin, h.yougileProjects)
	ygAPI.Get("/boards", admin, h.yougileBoards)
	ygAPI.Get("/columns", admin, h.yougileColumns)
	ygAPI.Get("/company-settings", admin, h.yougileGetSettings)
	ygAPI.Put("/company-settings", admin, h.yougilePutSettings)
	ygAPI.Post("/reset", admin, h.yougileReset)
	ygAPI.Post("/import-task", employee, h.yougileImportTask)
	ygAPI.Post("/export-task", employee, h.yougileExportTask)
	ygAPI.Delete("/tasks/:id<int>/link", employee, h.yougileUnlinkTask)
	ygAPI.Post("/webhook/register", admin, h.yougileRegisterWebhook)

	tasksAPI := app.Group("/api/tasks", auth.RequireAuth)
	tasksAPI.Get("", employee, h.listTasks)
	tasksAPI.Post("", employee, h.createTask)
	tasksAPI.Get("/:id<int>", employee, h.getTask)
	tasksAPI.Patch("/:id<int>", employee, h.updateTask)
	tasksAPI.Delete("/:id<int>", employee, h.deleteTask)
	tasksAPI.Post("/:id<int>/archive", employee, h.archiveTask)
	tasksAPI.Post("/:id<int>/restore", employee, h.restoreTask)
	tasksAPI.Put("/:id<int>/color", employee, h.setTaskColor)
	tasksAPI.Post("/:id<int>/favorite", h.toggleFavorite) // @require_auth — без проверки уровня
	tasksAPI.Get("/:id<int>/units", employee, h.taskUnits)
	tasksAPI.Post("/:id<int>/units", employee, h.createUnit)
	tasksAPI.Patch("/:id<int>/responsible", employee, h.setResponsible)
	tasksAPI.Patch("/:id<int>/stage", employee, h.setStage)
	tasksAPI.Get("/:id<int>/contributors", employee, h.contributors)
	tasksAPI.Get("/:id<int>/comments", employee, h.listComments)
	tasksAPI.Post("/:id<int>/comments", employee, h.createComment)
	tasksAPI.Patch("/:id<int>/comments/:commentId<int>", employee, h.updateComment)
	tasksAPI.Delete("/:id<int>/comments/:commentId<int>", employee, h.deleteComment)

	unitsAPI := app.Group("/api/units", auth.RequireAuth)
	unitsAPI.Get("/active", h.activeUnit)
	unitsAPI.Patch("/:id<int>", employee, h.updateUnit)
	unitsAPI.Delete("/:id<int>", employee, h.deleteUnit)
	unitsAPI.Post("/:id<int>/stop", employee, h.stopUnit)

	typesAPI := app.Group("/api/unit-types", auth.RequireAuth)
	typesAPI.Get("", employee, h.listUnitTypes)
	typesAPI.Post("", manager, h.createUnitType)
	typesAPI.Patch("/:id<int>", manager, h.updateUnitType)
	typesAPI.Delete("/:id<int>", manager, h.deleteUnitType)

	deptsAPI := app.Group("/api/departments", auth.RequireAuth)
	deptsAPI.Get("", employee, h.listDepartments)
	deptsAPI.Post("", manager, h.createDepartment)
	deptsAPI.Patch("/:id<int>", manager, h.updateDepartment)
	deptsAPI.Delete("/:id<int>", manager, h.deleteDepartment)

	stagesAPI := app.Group("/api/stages", auth.RequireAuth)
	stagesAPI.Get("", employee, h.listStages)
	stagesAPI.Post("", manager, h.createStage)
	stagesAPI.Patch("/reorder", manager, h.reorderStages)
	stagesAPI.Patch("/:id<int>", manager, h.updateStage)
	stagesAPI.Delete("/:id<int>", manager, h.deleteStage)

	statsAPI := app.Group("/api/stats", auth.RequireAuth)
	statsAPI.Get("/common", employee, h.statsCommon)
	statsAPI.Get("/extended", employee, h.statsExtended)
	statsAPI.Get("/common/export", manager, h.exportCommon)
	statsAPI.Get("/extended/export", manager, h.exportExtended)
	statsAPI.Get("/user-tasks", employee, h.statsUserTasks)
	statsAPI.Get("/employees", manager, h.statsEmployees)
	statsAPI.Get("/responsibles", employee, h.statsResponsibles)
	statsAPI.Get("/profile", h.statsProfile)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	eps endpoint.Endpoints
	log *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...} с её
// HTTP-статусом; прочее — 500, как Flask-обработчик ошибок.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

// currentUser — полный доменный пользователь из Locals (после RequireAuth).
func currentUser(c *fiber.Ctx) *domain.User {
	u, _ := pasetoauth.CurrentUser(c).(*domain.User)
	return u
}

// scopeBadRequest — форма flask abort(400, description=...).
func scopeBadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "BAD_REQUEST", "message": message,
	})
}

// requireCompanyScope — скоуп компании = АКТИВНАЯ компания из токена. Эндпоинты
// под RequireRole доступны только участникам компании, так что CompanyID всегда
// задан; иначе — нет активной компании. ok=false — ответ уже записан.
func requireCompanyScope(c *fiber.Ctx, u *domain.User) (int64, bool, error) {
	if u != nil && u.CompanyID != nil {
		return *u.CompanyID, true, nil
	}
	return 0, false, scopeBadRequest(c, "Нет активной компании")
}

// optionalCompanyScope — скоуп компании = активная компания из токена. Всегда
// задана для участника компании (эндпоинты под RequireRole).
func optionalCompanyScope(c *fiber.Ctx, u *domain.User) (*int64, bool, error) {
	if u != nil {
		return u.CompanyID, true, nil
	}
	return nil, true, nil
}

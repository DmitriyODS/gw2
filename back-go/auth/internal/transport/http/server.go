// Package http — HTTP-транспорт (Fiber): REST /api/auth/*, /api/users/*,
// /api/companies/*, /api/roles и /api/backup/*.
//
// Пути и формы JSON байт-в-байт совместимы с прежними Flask-блюпринтами
// api/{auth,users,companies,roles,backup}.py — фронт не меняется, nginx
// маршрутизирует эти префиксы на сервис вместо Flask (regex-location
// /api/companies/<id>/ai-settings выигрывает у префикса и уходит в aisvc).
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Активная компания и роль
// в ней — ИЗ ТОКЕНА (active): actor.CompanyID/Role.Level отражают выбранную
// компанию сессии; из БД — активность аккаунта, флаг супер-админа и активность
// выбранной компании.
func authSource(users domain.UserRepository) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64, active pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetByID(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		u.CompanyID = active.CompanyID
		u.Role.Level = active.RoleLevel
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

func NewServer(eps endpoint.Endpoints, verifier *pasetoauth.Verifier,
	users domain.UserRepository, log *slog.Logger) *Server {

	// Лимит тела — под импорт ZIP-бэкапа (в проде фактический потолок —
	// client_max_body_size nginx); аватарка ≤2МБ проверяется в хендлере.
	app := httpserver.New(httpserver.Config{
		AppName: "gw2-authsvc", Log: log, BodyLimit: 64 * 1024 * 1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	authAPI := app.Group("/api/auth")
	authAPI.Post("/login", h.login)
	authAPI.Post("/register", h.register)         // публичная регистрация (без компании)
	authAPI.Get("/suggest-login", h.suggestLogin) // live-подсказка логина по ФИО
	authAPI.Post("/verify-email", h.verifyEmail)  // подтверждение email (код/ссылка) → сессия
	authAPI.Post("/resend-verification", h.resendVerification)
	authAPI.Post("/forgot-password", h.forgotPassword)      // запрос письма со сбросом пароля
	authAPI.Post("/reset-password", h.resetPasswordByToken) // установка нового пароля по токену
	authAPI.Post("/select-company", h.selectCompany)        // завершение login-gate (select-токен в теле)
	authAPI.Post("/switch-company", auth.RequireToken, h.switchCompany)
	authAPI.Post("/refresh", h.refresh)
	authAPI.Post("/logout", auth.RequireToken, h.logout)
	authAPI.Post("/change-default", auth.RequireToken, h.changeDefault)

	usersAPI := app.Group("/api/users")
	// Список всех пользователей платформы — супер-админ.
	usersAPI.Get("", auth.RequireAuth, auth.RequireSuperAdmin, h.listUsers)
	// Заведение сотрудника в своей активной компании — администратор компании.
	usersAPI.Post("", auth.RequireAuth, auth.RequireRole(domain.LevelAdmin), h.createUser)
	// Платформенное управление пользователями (раздел «Пользователи») — супер-админ.
	// Префикс /platform не пересекается с /:id<int> (int-матчер не ловит строку).
	usersAPI.Post("/platform", auth.RequireAuth, auth.RequireSuperAdmin, h.createPlatformUser)
	usersAPI.Patch("/platform/:id<int>", auth.RequireAuth, auth.RequireSuperAdmin, h.updatePlatformUser)
	usersAPI.Post("/platform/:id<int>/reset-password", auth.RequireAuth, auth.RequireSuperAdmin, h.resetPlatformUser)
	usersAPI.Delete("/platform/:id<int>", auth.RequireAuth, auth.RequireSuperAdmin, h.deactivatePlatformUser)
	usersAPI.Get("/directory", auth.RequireAuth, h.directory)
	usersAPI.Get("/directory/:id<int>", auth.RequireAuth, h.directoryUser)
	usersAPI.Get("/me", auth.RequireAuth, h.me)
	usersAPI.Patch("/me", auth.RequireAuth, h.updateMe)
	usersAPI.Post("/me/avatar", auth.RequireAuth, h.uploadAvatar)
	usersAPI.Delete("/me/avatar", auth.RequireAuth, h.deleteAvatar)
	usersAPI.Get("/:id<int>/identicon", h.identicon) // публичный (img src)
	usersAPI.Get("/:id<int>", auth.RequireAuth, auth.RequireRole(domain.LevelAdmin), h.getUser)
	// Управление членом активной компании актора — администратор компании.
	usersAPI.Patch("/:id<int>", auth.RequireAuth, auth.RequireRole(domain.LevelAdmin), h.updateUser)
	usersAPI.Delete("/:id<int>", auth.RequireAuth, auth.RequireRole(domain.LevelAdmin), h.hideUser)
	usersAPI.Patch("/:id<int>/role", auth.RequireAuth, auth.RequireRole(domain.LevelAdmin), h.assignRole)
	usersAPI.Post("/:id<int>/reset-password", auth.RequireAuth, auth.RequireRole(domain.LevelAdmin), h.resetPassword)

	app.Get("/api/roles", auth.RequireAuth, h.listRoles)

	// Компании. Создать может любой авторизованный (станет администратором).
	// Платформенные операции (список всех, вкл/выкл) — супер-админ. Доступ к
	// конкретной компании (карточка/настройки/члены/инвайт) проверяется в
	// сервисе (companyAuthority: супер-админ или администратор этой компании).
	companiesAPI := app.Group("/api/companies", auth.RequireAuth)
	companiesAPI.Get("", auth.RequireSuperAdmin, h.listCompanies)
	// «Мои компании» — где пользователь администратор (раздел «Компании»).
	companiesAPI.Get("/mine", h.listMyCompanies)
	companiesAPI.Post("", h.createCompany)
	companiesAPI.Get("/:id<int>", h.getCompany)
	companiesAPI.Patch("/:id<int>", h.updateCompany)
	companiesAPI.Delete("/:id<int>", h.deleteCompany)
	companiesAPI.Patch("/:id<int>/toggle-active", auth.RequireSuperAdmin, h.toggleCompanyActive)
	companiesAPI.Get("/:id<int>/weekend-settings", h.getWeekendSettings)
	companiesAPI.Put("/:id<int>/weekend-settings", h.updateWeekendSettings)
	companiesAPI.Get("/:id<int>/groove-settings", h.getGrooveSettings)
	companiesAPI.Put("/:id<int>/groove-settings", h.updateGrooveSettings)
	companiesAPI.Get("/:id<int>/members", h.listMembers)
	companiesAPI.Get("/:id<int>/members/candidates", h.companyCandidates)
	companiesAPI.Post("/:id<int>/members", h.addMember)
	companiesAPI.Patch("/:id<int>/members/:userId<int>", h.setMemberRole)
	companiesAPI.Delete("/:id<int>/members/:userId<int>", h.removeMember)
	companiesAPI.Get("/:id<int>/invite", h.companyInvite)
	companiesAPI.Post("/:id<int>/invite", h.regenerateInvite)
	// Создание/редактирование сотрудников в КОНКРЕТНОЙ компании — её создатель.
	companiesAPI.Post("/:id<int>/users", h.createCompanyUser)
	companiesAPI.Patch("/:id<int>/users/:userId<int>", h.updateCompanyMember)
	companiesAPI.Post("/:id<int>/users/:userId<int>/reset-password", h.resetCompanyMember)
	// Email-приглашения в компанию (создатель/супер-админ создаёт; получатель
	// смотрит превью и принимает — токен в пути, не int, не конфликтует с :id).
	companiesAPI.Post("/:id<int>/invites", h.createCompanyInvite)
	companiesAPI.Get("/invites/:token", h.getInvitePreview)
	companiesAPI.Post("/invites/:token/accept", h.acceptCompanyInvite)
	// Вступление по коду — любой авторизованный пользователь.
	companiesAPI.Post("/join/:code", h.joinByInvite)

	backupAPI := app.Group("/api/backup", auth.RequireAuth, auth.RequireSuperAdmin)
	backupAPI.Get("/export", h.exportBackup)
	backupAPI.Post("/import", h.importBackup)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	eps endpoint.Endpoints
	log *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...}
// (+Extra-поля: retry_after_sec, company_name) с её HTTP-статусом; прочее —
// 500, как Flask-обработчик ошибок.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func badRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "VALIDATION_ERROR", "message": message,
	})
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

// tokenUserID — id пользователя из Locals (после RequireToken/RequireAuth).
func tokenUserID(c *fiber.Ctx) int64 {
	return pasetoauth.UserID(c)
}

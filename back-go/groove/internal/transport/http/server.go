// Package http — HTTP-транспорт (Fiber): REST /api/groove/*.
//
// Пути и формы JSON байт-в-байт совместимы с прежним Flask-блюпринтом
// api/groove.py — фронт не меняется, nginx маршрутизирует префикс
// /api/groove на этот сервис вместо Flask.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Активная компания и роль
// в ней — ИЗ ТОКЕНА (active); из БД — активность пользователя, профиль и
// активность выбранной компании.
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
	companies domain.CompanyReader, verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := fiber.New(fiber.Config{
		AppName:               "gw2-groovesvc",
		DisableStartupMessage: true,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := app.Group("/api/groove", auth.RequireAuth)
	// Магазин — без company-scope (глобальный прайс), как во Flask.
	api.Get("/shop", h.getShop)

	scoped := api.Group("", companyScope, grooveGate(companies))
	scoped.Get("/feed", h.getFeed)
	scoped.Post("/feed/:id<int>/reactions", h.toggleReaction)
	scoped.Get("/feed/:id<int>/comments", h.listComments)
	scoped.Post("/feed/:id<int>/comments", h.addComment)
	scoped.Delete("/comments/:id<int>", h.deleteComment)
	scoped.Post("/kudos", h.sendKudos)
	scoped.Get("/live", h.getLive)
	scoped.Post("/zap", h.sendZap)
	scoped.Get("/pet", h.getMyPet)
	scoped.Post("/pet/feed", h.feedPet)
	scoped.Post("/pet/name", h.renamePet)
	scoped.Post("/pet/equip", h.equipItem)
	scoped.Post("/shop/buy", h.buyItem)
	scoped.Post("/shop/buy-species", h.buySpecies)
	scoped.Post("/pet/quest/claim", h.claimQuest)
	scoped.Post("/pet/species", h.switchSpecies)
	scoped.Get("/zoo", h.getZoo)
	scoped.Post("/zoo/:user_id<int>/stroke", h.strokePet)
	scoped.Get("/raid", h.getRaid)
	scoped.Get("/wrapped", h.getWrapped)
	scoped.Post("/wrapped/share", h.shareWrapped)
	scoped.Get("/morning", h.morning)
	scoped.Get("/tv", h.grooveTV)
	scoped.Get("/location", h.getLocation)
	scoped.Put("/location", h.setLocation)
	scoped.Delete("/location", h.deleteLocation)
	scoped.Get("/geo", h.geoSearch)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

const localCompanyID = "companyID"

// companyScope — порт @require_company_scope: если в токене есть активная
// компания — работаем с ней; иначе (активной компании нет, например супер-
// админ) требуется ?company_id=. Вешается после RequireAuth.
func companyScope(c *fiber.Ctx) error {
	info := pasetoauth.Current(c)
	user, _ := info.User.(*domain.User)
	if user != nil && user.CompanyID != nil {
		c.Locals(localCompanyID, *user.CompanyID)
		return c.Next()
	}
	raw := c.Query("company_id")
	if raw == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "BAD_REQUEST", "message": "Требуется указать company_id",
		})
	}
	id, err := parseInt64(raw)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "BAD_REQUEST", "message": "Неверный company_id",
		})
	}
	c.Locals(localCompanyID, id)
	return c.Next()
}

// grooveGate — 403, если компания выключила режим «Мой Groove». Вешается
// после companyScope (читает companyID из Locals). Ошибка чтения настройки →
// пропускаем (fail-open: режим по умолчанию включён).
func grooveGate(companies domain.CompanyReader) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, _ := c.Locals(localCompanyID).(int64)
		if cid != 0 {
			if enabled, err := companies.GrooveEnabled(c.Context(), cid); err == nil && !enabled {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "GROOVE_DISABLED",
					"message": "Режим «Мой Groove» отключён для компании",
				})
			}
		}
		return c.Next()
	}
}

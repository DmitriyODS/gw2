// Package http — HTTP-транспорт (Fiber): REST /api/pets/*.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
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

	app := httpserver.New(httpserver.Config{AppName: "gw2-petsvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, log: log}

	api := app.Group("/api/pets", auth.RequireAuth)

	scoped := api.Group("", companyScope, grooveGate(companies))
	scoped.Get("/shop", h.getShop)
	scoped.Get("/pet", h.getMyPet)
	scoped.Post("/pet/feed", h.feedPet)
	scoped.Post("/pet/name", h.renamePet)
	scoped.Post("/pet/equip", h.equipItem)
	scoped.Post("/pet/species", h.switchSpecies)
	scoped.Delete("/pet/species", h.resetSpecies)
	scoped.Post("/pet/quest/claim", h.claimQuest)
	scoped.Post("/pet/adventure", h.startAdventure)
	scoped.Post("/pet/prestige", h.prestigePet)
	scoped.Get("/season", h.getSeason)
	scoped.Post("/season/claim", h.claimSeasonReward)
	scoped.Get("/house", h.getHouse)
	scoped.Post("/house/buy", h.buyHouseDecor)
	scoped.Post("/house/arrange", h.arrangeHouse)
	scoped.Get("/shop/mystery", h.getMystery)
	scoped.Post("/shop/buy", h.buyItem)
	scoped.Post("/shop/buy-species", h.buySpecies)
	scoped.Post("/walk", h.walkPet)
	scoped.Post("/heal", h.healPet)
	scoped.Post("/stroke/:userId<int>", h.strokePet)
	scoped.Get("/zoo", h.getZoo)
	scoped.Delete("/zoo/:userId<int>", h.deleteZooPet)
	scoped.Get("/rating", h.getRating)
	scoped.Get("/live", h.getLive)
	scoped.Get("/activity", h.getActivityLog)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

const localCompanyID = "companyID"

// companyScope — порт @require_company_scope: если в токене есть активная
// компания — работаем с ней; иначе ?company_id= принимается ТОЛЬКО от
// платформенного супер-админа (обычный пользователь без активной компании
// не должен читать данные произвольной компании). Вешается после RequireAuth.
func companyScope(c *fiber.Ctx) error {
	info := pasetoauth.Current(c)
	user, _ := info.User.(*domain.User)
	if user != nil && user.CompanyID != nil {
		c.Locals(localCompanyID, *user.CompanyID)
		return c.Next()
	}
	if info == nil || !info.IsSuperAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "FORBIDDEN", "message": "Требуется активная компания",
		})
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

// Package http — HTTP-транспорт billingsvc (Fiber): REST /api/billing/*.
//
// Пользовательские ручки требуют только авторизации (подписка личная и от
// активной компании не зависит), административные — супер-админа. Вебхук
// платёжного шлюза публичный: он приходит снаружи и предъявляет секрет
// платежа, а не токен.
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Биллинг от активной
// компании не зависит, поэтому CompanyActive всегда true.
func authSource(identity domain.IdentityReader) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64, _ pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
		u, err := identity.GetUser(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		return &pasetoauth.AuthInfo{
			IsActive:      u.IsActive,
			IsSuperAdmin:  u.IsSuperAdmin,
			CompanyActive: true,
			User:          u,
		}, nil
	}
}

func NewServer(svc *service.Service, identity domain.IdentityReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	app := httpserver.New(httpserver.Config{AppName: "gw2-billingsvc", Log: log})
	auth := pasetoauth.NewMiddleware(verifier, authSource(identity))
	h := &handlers{svc: svc, log: log}

	// Вебхук платёжного шлюза — до авторизационной группы: токена у банка нет.
	app.Post("/api/billing/webhook/payments", h.paymentWebhook)

	api := app.Group("/api/billing", auth.RequireAuth)
	api.Get("/showcase", h.showcase)
	api.Get("/entitlements", h.entitlements)
	api.Get("/storage", h.storage)
	api.Get("/ai", h.aiState)
	api.Post("/quote", h.quote)
	api.Post("/purchase", h.purchase)
	api.Post("/promo/activate", h.activatePromo)
	api.Post("/subscription/auto-renew", h.autoRenew)
	api.Delete("/addons/:id<int>", h.cancelAddon)

	api.Get("/orders", h.orders)
	api.Post("/orders/:id<int>/cancel", h.cancelOrder)

	api.Get("/products", h.products)
	api.Get("/products/:id<int>", h.product)

	api.Get("/my", h.myStore)
	api.Post("/my/products", h.createProduct)
	api.Patch("/my/products/:id<int>", h.updateProduct)
	api.Post("/my/products/:id<int>/submit", h.submitProduct)
	api.Post("/my/products/:id<int>/withdraw", h.withdrawProduct)
	api.Delete("/my/products/:id<int>", h.deleteProduct)
	api.Post("/my/payouts", h.requestPayout)

	// «Аудит платформы» — только супер-админ.
	admin := app.Group("/api/billing/admin", auth.RequireAuth, auth.RequireSuperAdmin)
	admin.Get("/overview", h.adminOverview)
	admin.Get("/settings", h.adminSettings)
	admin.Patch("/settings", h.adminUpdateSettings)
	admin.Patch("/plans/:code", h.adminUpdatePlan)
	admin.Patch("/addons/:code", h.adminUpdateAddon)
	admin.Get("/subscriptions", h.adminSubscriptions)
	admin.Post("/subscriptions/grant", h.adminGrantSubscription)
	admin.Delete("/subscriptions/:userId<int>", h.adminRevokeSubscription)
	admin.Post("/tokens/grant", h.adminGrantTokens)
	admin.Post("/tokens/reset", h.adminResetTokens)
	admin.Get("/promos", h.adminPromos)
	admin.Post("/promos", h.adminCreatePromo)
	admin.Patch("/promos/:id<int>", h.adminUpdatePromo)
	admin.Delete("/promos/:id<int>", h.adminDeletePromo)
	admin.Get("/products", h.adminProducts)
	admin.Post("/products", h.adminCreateProduct)
	admin.Patch("/products/:id<int>", h.adminUpdateProduct)
	admin.Delete("/products/:id<int>", h.adminDeleteProduct)
	admin.Post("/products/:id<int>/review", h.adminReviewProduct)
	admin.Get("/orders", h.adminOrders)
	admin.Post("/orders/:id<int>/confirm", h.adminConfirmOrder)
	admin.Get("/payouts", h.adminPayouts)
	admin.Post("/payouts/:id<int>", h.adminProcessPayout)
	admin.Get("/audit", h.adminAudit)
	admin.Get("/users", h.adminUsers)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

type handlers struct {
	svc *service.Service
	log *slog.Logger
}

func (h *handlers) fail(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func currentUserID(c *fiber.Ctx) int64 { return pasetoauth.UserID(c) }

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

// paging — общий разбор ?limit&offset со здравыми границами.
func paging(c *fiber.Ctx, defLimit int) (limit, offset int) {
	limit = c.QueryInt("limit", defLimit)
	if limit <= 0 || limit > 200 {
		limit = defLimit
	}
	offset = c.QueryInt("offset", 0)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

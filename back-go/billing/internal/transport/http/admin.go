package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/service"
)

// Ручки раздела «Аудит платформы» — только супер-админ (гейт в мидлвари).

// adminOverview — сводка первого экрана: тарифы, аддоны, настройки и лимиты
// линейки одним запросом.
func (h *handlers) adminOverview(c *fiber.Ctx) error {
	ctx := c.Context()
	plans, err := h.svc.Catalog.ListPlans(ctx, false)
	if err != nil {
		return h.fail(c, err)
	}
	addons, err := h.svc.Catalog.ListAddons(ctx, false)
	if err != nil {
		return h.fail(c, err)
	}
	settings, err := h.svc.GetSettings(ctx)
	if err != nil {
		return h.fail(c, err)
	}
	limits := map[string]domain.Limits{}
	for _, code := range domain.PlanCodes {
		limits[code] = domain.LimitsFor(code)
	}
	return c.JSON(fiber.Map{
		"plans": plans, "addons": addons, "settings": settings, "plan_limits": limits,
	})
}

func (h *handlers) adminSettings(c *fiber.Ctx) error {
	s, err := h.svc.GetSettings(c.Context())
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(s)
}

func (h *handlers) adminUpdateSettings(c *fiber.Ctx) error {
	var in domain.Settings
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	out, err := h.svc.UpdateSettings(c.Context(), currentUserID(c), &in)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) adminUpdatePlan(c *fiber.Ctx) error {
	var in domain.Plan
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	in.Code = c.Params("code")
	out, err := h.svc.UpdatePlan(c.Context(), currentUserID(c), &in)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) adminUpdateAddon(c *fiber.Ctx) error {
	var in domain.Addon
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	in.Code = c.Params("code")
	out, err := h.svc.UpdateAddon(c.Context(), currentUserID(c), &in)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) adminSubscriptions(c *fiber.Ctx) error {
	limit, offset := paging(c, 50)
	items, total, err := h.svc.ListSubscriptions(c.Context(), c.Query("search"), c.Query("plan"), limit, offset)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": total})
}

func (h *handlers) adminGrantSubscription(c *fiber.Ctx) error {
	var body struct {
		UserID int64  `json:"user_id"`
		Plan   string `json:"plan"`
		Days   int    `json:"days"`
		Note   string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	sub, err := h.svc.GrantSubscription(c.Context(), currentUserID(c), body.UserID, body.Plan, body.Days, body.Note)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(sub)
}

func (h *handlers) adminRevokeSubscription(c *fiber.Ctx) error {
	userID, _ := c.ParamsInt("userId")
	if err := h.svc.RevokeSubscription(c.Context(), currentUserID(c), int64(userID)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) adminGrantTokens(c *fiber.Ctx) error {
	var body struct {
		UserID int64 `json:"user_id"`
		Tokens int64 `json:"tokens"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	if err := h.svc.GrantTokens(c.Context(), currentUserID(c), body.UserID, body.Tokens); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) adminResetTokens(c *fiber.Ctx) error {
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	if err := h.svc.ResetTokens(c.Context(), currentUserID(c), body.UserID); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) adminPromos(c *fiber.Ctx) error {
	items, err := h.svc.ListPromos(c.Context())
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

// promoInput — тело промокода: даты приходят строками ISO, пустая строка это
// «без ограничения».
type promoInput struct {
	ID           int64   `json:"id"`
	Code         string  `json:"code"`
	Kind         string  `json:"kind"`
	Value        int64   `json:"value"`
	PlanCode     *string `json:"plan_code"`
	AppliesTo    string  `json:"applies_to"`
	MaxUses      int     `json:"max_uses"`
	PerUserLimit int     `json:"per_user_limit"`
	StartsAt     string  `json:"starts_at"`
	ExpiresAt    string  `json:"expires_at"`
	IsActive     bool    `json:"is_active"`
	Comment      string  `json:"comment"`
}

func (in promoInput) toDomain() *domain.Promo {
	p := &domain.Promo{
		ID: in.ID, Code: in.Code, Kind: in.Kind, Value: in.Value, PlanCode: in.PlanCode,
		AppliesTo: in.AppliesTo, MaxUses: in.MaxUses, PerUserLimit: in.PerUserLimit,
		IsActive: in.IsActive, Comment: in.Comment,
	}
	if t := parseTime(in.StartsAt); t != nil {
		p.StartsAt = t
	}
	if t := parseTime(in.ExpiresAt); t != nil {
		p.ExpiresAt = t
	}
	return p
}

func parseTime(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	return nil
}

func (h *handlers) adminCreatePromo(c *fiber.Ctx) error {
	var in promoInput
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	p, err := h.svc.CreatePromo(c.Context(), currentUserID(c), in.toDomain())
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

func (h *handlers) adminUpdatePromo(c *fiber.Ctx) error {
	var in promoInput
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	in.ID = pathID(c)
	p, err := h.svc.UpdatePromo(c.Context(), currentUserID(c), in.toDomain())
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(p)
}

func (h *handlers) adminDeletePromo(c *fiber.Ctx) error {
	if err := h.svc.DeletePromo(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// adminProducts — витрина глазами модератора: очередь на проверку и всё
// опубликованное.
func (h *handlers) adminProducts(c *fiber.Ctx) error {
	items, err := h.svc.ListModeration(c.Context())
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) adminCreateProduct(c *fiber.Ctx) error {
	var in service.ProductInput
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	p, err := h.svc.CreatePlatformProduct(c.Context(), currentUserID(c), in)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

func (h *handlers) adminUpdateProduct(c *fiber.Ctx) error {
	var in service.ProductInput
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	p, err := h.svc.UpdatePlatformProduct(c.Context(), currentUserID(c), pathID(c), in)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(p)
}

func (h *handlers) adminDeleteProduct(c *fiber.Ctx) error {
	if err := h.svc.DeletePlatformProduct(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) adminReviewProduct(c *fiber.Ctx) error {
	var body struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	p, err := h.svc.ReviewProduct(c.Context(), currentUserID(c), pathID(c), body.Approve, body.Reason)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(p)
}

func (h *handlers) adminOrders(c *fiber.Ctx) error {
	limit, offset := paging(c, 50)
	items, total, err := h.svc.ListAllOrders(c.Context(), c.Query("status"), limit, offset)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": total})
}

func (h *handlers) adminConfirmOrder(c *fiber.Ctx) error {
	order, err := h.svc.ConfirmOrder(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(order)
}

func (h *handlers) adminPayouts(c *fiber.Ctx) error {
	items, err := h.svc.ListAllPayouts(c.Context())
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) adminProcessPayout(c *fiber.Ctx) error {
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	if err := h.svc.ProcessPayout(c.Context(), currentUserID(c), pathID(c), body.Status, body.Note); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) adminAudit(c *fiber.Ctx) error {
	limit, offset := paging(c, 50)
	items, total, err := h.svc.ListAudit(c.Context(), c.Query("action"), limit, offset)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": total})
}

func (h *handlers) adminUsers(c *fiber.Ctx) error {
	users, err := h.svc.SearchUsers(c.Context(), c.Query("q"), 20)
	if err != nil {
		return h.fail(c, err)
	}
	items := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		items = append(items, fiber.Map{"id": u.ID, "fio": u.FIO, "login": u.Login, "is_active": u.IsActive})
	}
	return c.JSON(fiber.Map{"items": items})
}

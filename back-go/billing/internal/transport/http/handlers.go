package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/service"
)

// showcase — вкладка «Подписки»: текущий тариф, расход и вся линейка.
func (h *handlers) showcase(c *fiber.Ctx) error {
	data, err := h.svc.Showcase(c.Context(), currentUserID(c))
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(data)
}

// entitlements — лимиты пользователя либо (при ?company_id=) его компании.
func (h *handlers) entitlements(c *fiber.Ctx) error {
	companyID := int64(c.QueryInt("company_id", 0))
	ent, err := h.svc.Entitlements(c.Context(), currentUserID(c), companyID)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(ent)
}

/* Раздел «Настройки → Хранилище». items оставлен ради прежних клиентов
   (карточка подписки читает только разбивку), остальное — детализация:
   лимит, расход и самые крупные файлы. */
func (h *handlers) storage(c *fiber.Ctx) error {
	details, err := h.svc.StorageDetails(c.Context(), currentUserID(c))
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{
		"items":       details.Services,
		"services":    details.Services,
		"files":       details.Files,
		"used_bytes":  details.UsedBytes,
		"limit_bytes": details.LimitBytes,
	})
}

// deleteStorageFiles — удалить выбранные файлы руками их владельцев.
func (h *handlers) deleteStorageFiles(c *fiber.Ctx) error {
	var body struct {
		Keys []string `json:"keys"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	out, err := h.svc.DeleteStorageFiles(c.Context(), currentUserID(c), body.Keys)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(out)
}

// sweepStorage — сверка с владельцами: убрать сирот, доучесть незнакомое,
// пересчитать занятое место.
func (h *handlers) sweepStorage(c *fiber.Ctx) error {
	out, err := h.svc.SweepStorage(c.Context(), currentUserID(c))
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) aiState(c *fiber.Ctx) error {
	ent, usage, err := h.svc.AIState(c.Context(), currentUserID(c))
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{
		"tokens_limit": ent.TokensLimit,
		"tokens_used":  ent.TokensUsed,
		"tokens_left":  ent.TokensLeft,
		"plan":         ent.Plan,
		"plan_name":    ent.PlanName,
		"by_feature":   usage,
	})
}

func (h *handlers) quote(c *fiber.Ctx) error {
	var req service.PurchaseRequest
	if err := c.BodyParser(&req); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	q, err := h.svc.Quote(c.Context(), currentUserID(c), req)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(q)
}

func (h *handlers) purchase(c *fiber.Ctx) error {
	var req service.PurchaseRequest
	if err := c.BodyParser(&req); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	order, err := h.svc.Purchase(c.Context(), currentUserID(c), req)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(order)
}

func (h *handlers) activatePromo(c *fiber.Ctx) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	res, err := h.svc.ActivatePromo(c.Context(), currentUserID(c), body.Code)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(res)
}

func (h *handlers) autoRenew(c *fiber.Ctx) error {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	sub, err := h.svc.SetAutoRenew(c.Context(), currentUserID(c), body.Enabled)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(sub)
}

func (h *handlers) cancelAddon(c *fiber.Ctx) error {
	if err := h.svc.CancelAddon(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) orders(c *fiber.Ctx) error {
	limit, offset := paging(c, 30)
	items, total, err := h.svc.ListOrders(c.Context(), currentUserID(c), limit, offset)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": total})
}

func (h *handlers) cancelOrder(c *fiber.Ctx) error {
	if err := h.svc.CancelOrder(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) products(c *fiber.Ctx) error {
	limit, offset := paging(c, 40)
	items, total, err := h.svc.ListProducts(c.Context(), c.Query("kind"), c.Query("search"),
		currentUserID(c), limit, offset)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": total})
}

func (h *handlers) product(c *fiber.Ctx) error {
	p, err := h.svc.GetProductCard(c.Context(), pathID(c), currentUserID(c))
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(p)
}

func (h *handlers) myStore(c *fiber.Ctx) error {
	data, err := h.svc.MyStore(c.Context(), currentUserID(c))
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(data)
}

func (h *handlers) createProduct(c *fiber.Ctx) error {
	var in service.ProductInput
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	p, err := h.svc.CreateProduct(c.Context(), currentUserID(c), in)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

func (h *handlers) updateProduct(c *fiber.Ctx) error {
	var in service.ProductInput
	if err := c.BodyParser(&in); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	p, err := h.svc.UpdateProduct(c.Context(), currentUserID(c), pathID(c), in)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(p)
}

func (h *handlers) submitProduct(c *fiber.Ctx) error {
	if err := h.svc.SubmitProduct(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) withdrawProduct(c *fiber.Ctx) error {
	if err := h.svc.WithdrawProduct(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) deleteProduct(c *fiber.Ctx) error {
	if err := h.svc.DeleteProduct(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *handlers) requestPayout(c *fiber.Ctx) error {
	var body struct {
		Amount     int64  `json:"amount"`
		Requisites string `json:"requisites"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.fail(c, domain.ErrValidation)
	}
	p, err := h.svc.RequestPayout(c.Context(), currentUserID(c), body.Amount, body.Requisites)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

// paymentWebhook — уведомление платёжного шлюза. Секрет платежа проверяет
// сервис; чужое уведомление получает 403 и ничего не меняет.
func (h *handlers) paymentWebhook(c *fiber.Ctx) error {
	headers := map[string]string{}
	c.Request().Header.VisitAll(func(k, v []byte) { headers[string(k)] = string(v) })
	if err := h.svc.ConfirmPayment(c.Context(), c.Body(), headers); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

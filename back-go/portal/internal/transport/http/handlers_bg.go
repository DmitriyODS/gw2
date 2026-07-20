package http

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// getBackground — рецепт оформления ленты текущего пользователя (или null).
func (h *handlers) getBackground(c *fiber.Ctx) error {
	recipe, err := h.svc.GetBackground(c.Context(), currentUser(c).ID)
	if err != nil {
		return h.respondError(c, err)
	}
	if recipe == nil {
		return c.JSON(fiber.Map{"recipe": nil})
	}
	return c.JSON(fiber.Map{"recipe": recipe})
}

// putBackground — сохранить рецепт оформления.
func (h *handlers) putBackground(c *fiber.Ctx) error {
	var body struct {
		Recipe json.RawMessage `json:"recipe"`
	}
	parseBody(c, &body)
	if len(body.Recipe) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "message": "Пустой рецепт оформления",
		})
	}
	if err := h.svc.SetBackground(c.Context(), currentUser(c).ID, body.Recipe); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// deleteBackground — снять оформление ленты.
func (h *handlers) deleteBackground(c *fiber.Ctx) error {
	if err := h.svc.DeleteBackground(c.Context(), currentUser(c).ID); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

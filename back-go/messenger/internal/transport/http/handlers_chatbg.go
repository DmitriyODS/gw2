package http

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// getChatBackgrounds — весь набор оформления чатов пользователя.
func (h *handlers) getChatBackgrounds(c *fiber.Ctx) error {
	resp, err := h.svc.GetChatBackgrounds(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// putChatBackground — сохранить рецепт. conversation_id отсутствует/null —
// общий дефолт пользователя.
func (h *handlers) putChatBackground(c *fiber.Ctx) error {
	var body struct {
		ConversationID *int64          `json:"conversation_id"`
		Recipe         json.RawMessage `json:"recipe"`
	}
	parseBody(c, &body)
	if len(body.Recipe) == 0 {
		return validationError(c, "recipe", "Missing data for required field.")
	}
	if err := h.svc.SetChatBackground(c.Context(), currentUserID(c), body.ConversationID, body.Recipe); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// deleteChatBackground — снять рецепт. ?conversation_id= отсутствует — дефолт.
func (h *handlers) deleteChatBackground(c *fiber.Ctx) error {
	var convID *int64
	if raw := c.QueryInt("conversation_id", 0); raw != 0 {
		id := int64(raw)
		convID = &id
	}
	if err := h.svc.DeleteChatBackground(c.Context(), currentUserID(c), convID); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

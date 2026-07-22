package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// listFolders — папки текущего пользователя.
func (h *handlers) listFolders(c *fiber.Ctx) error {
	resp, err := h.svc.ListFolders(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createFolder(c *fiber.Ctx) error {
	var in dto.FolderInput
	parseBody(c, &in)
	resp, err := h.svc.CreateFolder(c.Context(), currentUserID(c), in)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) updateFolder(c *fiber.Ctx) error {
	var in dto.FolderInput
	parseBody(c, &in)
	resp, err := h.svc.UpdateFolder(c.Context(), currentUserID(c), pathID(c), in)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteFolder(c *fiber.Ctx) error {
	if err := h.svc.DeleteFolder(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handlers) reorderFolders(c *fiber.Ctx) error {
	var body struct {
		Order []int64 `json:"order"`
	}
	parseBody(c, &body)
	if err := h.svc.ReorderFolders(c.Context(), currentUserID(c), body.Order); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handlers) addFolderItem(c *fiber.Ctx) error {
	var body struct {
		ConversationID int64 `json:"conversation_id"`
	}
	parseBody(c, &body)
	if err := h.svc.AddFolderItem(c.Context(), currentUserID(c), pathID(c), body.ConversationID); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handlers) removeFolderItem(c *fiber.Ctx) error {
	convID, _ := c.ParamsInt("convId")
	if err := h.svc.RemoveFolderItem(c.Context(), currentUserID(c), pathID(c), int64(convID)); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

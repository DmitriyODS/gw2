package http

import (
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

func userIDParam(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("userId")
	return int64(id)
}

// ── Группы ───────────────────────────────────────────────────────

func (h *handlers) createGroup(c *fiber.Ctx) error {
	var body struct {
		Title              string  `json:"title"`
		AvatarAttachmentID *int64  `json:"avatar_attachment_id"`
		MemberIDs          []int64 `json:"member_ids"`
	}
	parseBody(c, &body)
	if strings.TrimSpace(body.Title) == "" {
		return validationError(c, "title", "Missing data for required field.")
	}
	if utf8.RuneCountInString(body.Title) > 120 {
		return validationError(c, "title", "Longer than maximum length 120.")
	}
	resp, err := h.svc.CreateGroup(c.Context(), currentUserID(c), body.Title,
		body.AvatarAttachmentID, body.MemberIDs)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) getGroup(c *fiber.Ctx) error {
	resp, err := h.svc.GetGroup(c.Context(), pathID(c), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) patchGroup(c *fiber.Ctx) error {
	var body struct {
		Title *string `json:"title"`
	}
	parseBody(c, &body)
	if body.Title == nil {
		return validationError(c, "title", "Missing data for required field.")
	}
	if err := h.svc.RenameGroup(c.Context(), pathID(c), currentUserID(c), *body.Title); err != nil {
		return h.respondError(c, err)
	}
	return h.getGroup(c)
}

func (h *handlers) setGroupAvatar(c *fiber.Ctx) error {
	var body struct {
		AvatarAttachmentID *int64 `json:"avatar_attachment_id"`
	}
	parseBody(c, &body)
	if err := h.svc.SetGroupAvatar(c.Context(), pathID(c), currentUserID(c), body.AvatarAttachmentID); err != nil {
		return h.respondError(c, err)
	}
	return h.getGroup(c)
}

func (h *handlers) addGroupMembers(c *fiber.Ctx) error {
	var body struct {
		UserIDs []int64 `json:"user_ids"`
	}
	parseBody(c, &body)
	if len(body.UserIDs) == 0 {
		return validationError(c, "user_ids", "Missing data for required field.")
	}
	if err := h.svc.AddGroupMembers(c.Context(), pathID(c), currentUserID(c), body.UserIDs); err != nil {
		return h.respondError(c, err)
	}
	return h.getGroup(c)
}

func (h *handlers) removeGroupMember(c *fiber.Ctx) error {
	if err := h.svc.RemoveGroupMember(c.Context(), pathID(c), currentUserID(c), userIDParam(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"removed": true})
}

func (h *handlers) patchGroupMember(c *fiber.Ctx) error {
	var body struct {
		Role   *string `json:"role"`
		Rights *struct {
			ManageMembers bool `json:"manage_members"`
			EditInfo      bool `json:"edit_info"`
			PinMessages   bool `json:"pin_messages"`
		} `json:"rights"`
	}
	parseBody(c, &body)
	convID, actorID, memberID := pathID(c), currentUserID(c), userIDParam(c)
	if body.Role != nil {
		if err := h.svc.SetMemberRole(c.Context(), convID, actorID, memberID, *body.Role); err != nil {
			return h.respondError(c, err)
		}
	}
	if body.Rights != nil {
		if err := h.svc.SetMemberRights(c.Context(), convID, actorID, memberID,
			body.Rights.ManageMembers, body.Rights.EditInfo, body.Rights.PinMessages); err != nil {
			return h.respondError(c, err)
		}
	}
	return h.getGroup(c)
}

func (h *handlers) transferOwnership(c *fiber.Ctx) error {
	if err := h.svc.TransferOwnership(c.Context(), pathID(c), currentUserID(c), userIDParam(c)); err != nil {
		return h.respondError(c, err)
	}
	return h.getGroup(c)
}

func (h *handlers) leaveGroup(c *fiber.Ctx) error {
	if err := h.svc.LeaveGroup(c.Context(), pathID(c), currentUserID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"left": true})
}

func (h *handlers) muteGroup(c *fiber.Ctx) error {
	var body struct {
		Muted bool `json:"muted"`
	}
	parseBody(c, &body)
	muted, err := h.svc.SetGroupMute(c.Context(), pathID(c), currentUserID(c), body.Muted)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"muted": muted})
}

func (h *handlers) groupInviteLink(c *fiber.Ctx) error {
	code, err := h.svc.GroupInviteLink(c.Context(), pathID(c), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"code": code})
}

func (h *handlers) revokeGroupInviteLink(c *fiber.Ctx) error {
	if err := h.svc.RevokeGroupInviteLink(c.Context(), pathID(c), currentUserID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"revoked": true})
}

func (h *handlers) groupInvitePreview(c *fiber.Ctx) error {
	resp, err := h.svc.GroupInvitePreview(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) joinGroup(c *fiber.Ctx) error {
	resp, err := h.svc.JoinGroupByCode(c.Context(), c.Params("code"), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) messageReadBy(c *fiber.Ctx) error {
	resp, err := h.svc.ReadBy(c.Context(), pathID(c), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"readers": resp})
}

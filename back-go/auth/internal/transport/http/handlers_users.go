package http

import (
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/avatar"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
)

const avatarMaxBytes = 2 * 1024 * 1024

func (h *handlers) listUsers(c *fiber.Ctx) error {
	resp, err := h.eps.ListUsers(c.Context(), nil)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createUser(c *fiber.Ctx) error {
	var body dto.CreateUserRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	if body.FIO == "" || body.Login == "" || body.RoleID == 0 {
		return badRequest(c, "fio, login и role_id обязательны")
	}
	resp, err := h.eps.CreateUser(c.Context(), endpoint.CreateUserEpRequest{
		Actor: currentUser(c), Body: body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) directory(c *fiber.Ctx) error {
	me := currentUser(c)
	req := dto.DirectoryRequest{
		ActorID: me.ID,
		Query:   c.Query("q"),
	}
	switch c.Query("exclude_self") {
	case "1", "true", "yes":
		req.ExcludeID = me.ID
	}
	// company_id из query учитывается сервисом только для Администратора
	// системы (без своей компании).
	if raw := c.Query("company_id"); raw != "" {
		cid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return badRequest(c, "Неверный company_id")
		}
		req.CompanyID = &cid
	}

	resp, err := h.eps.Directory(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) directoryUser(c *fiber.Ctx) error {
	resp, err := h.eps.DirectoryUser(c.Context(), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) me(c *fiber.Ctx) error {
	resp, err := h.eps.Me(c.Context(), currentUser(c).ID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) updateMe(c *fiber.Ctx) error {
	var body dto.UpdateMeRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	resp, err := h.eps.UpdateMe(c.Context(), endpoint.UpdateMeEpRequest{
		UserID: currentUser(c).ID, Body: body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) uploadAvatar(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "NO_FILE", "message": "Файл не передан",
		})
	}
	if fileHeader.Size > avatarMaxBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "FILE_TOO_LARGE", "message": "Файл превышает 2 МБ",
		})
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close()
	fileBytes, err := io.ReadAll(io.LimitReader(f, avatarMaxBytes+1))
	if err != nil {
		return h.respondError(c, err)
	}
	if len(fileBytes) > avatarMaxBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "FILE_TOO_LARGE", "message": "Файл превышает 2 МБ",
		})
	}

	resp, err := h.eps.UploadAvatar(c.Context(), endpoint.AvatarEpRequest{
		UserID: currentUser(c).ID, File: fileBytes,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteAvatar(c *fiber.Ctx) error {
	resp, err := h.eps.DeleteAvatar(c.Context(), currentUser(c).ID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getUser(c *fiber.Ctx) error {
	resp, err := h.eps.GetUser(c.Context(), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) updateUser(c *fiber.Ctx) error {
	var body dto.UpdateUserRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	resp, err := h.eps.UpdateUser(c.Context(), endpoint.UpdateUserEpRequest{
		Actor: currentUser(c), UserID: pathID(c), Body: body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) hideUser(c *fiber.Ctx) error {
	_, err := h.eps.HideUser(c.Context(), endpoint.ActorRequest{
		Actor: currentUser(c), UserID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Пользователь скрыт"})
}

func (h *handlers) assignRole(c *fiber.Ctx) error {
	var body struct {
		RoleID int64 `json:"role_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.RoleID == 0 {
		return badRequest(c, "role_id обязателен")
	}
	resp, err := h.eps.AssignRole(c.Context(), endpoint.AssignRoleEpRequest{
		Actor: currentUser(c), UserID: pathID(c), RoleID: body.RoleID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) resetPassword(c *fiber.Ctx) error {
	_, err := h.eps.ResetPassword(c.Context(), endpoint.ActorRequest{
		Actor: currentUser(c), UserID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Пароль сброшен"})
}

func (h *handlers) identicon(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "image/png")
	return c.Send(avatar.Identicon(pathID(c)))
}

package http

import (
	"io"

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

// ── Платформенное управление пользователями (раздел «Пользователи», супер-админ) ──

func (h *handlers) createPlatformUser(c *fiber.Ctx) error {
	var body dto.CreateUserRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	if body.FIO == "" || body.Login == "" {
		return badRequest(c, "fio и login обязательны")
	}
	resp, err := h.eps.CreatePlatformUser(c.Context(), endpoint.PlatformCreateEpRequest{Body: body})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updatePlatformUser(c *fiber.Ctx) error {
	var body dto.UpdateUserRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	resp, err := h.eps.UpdatePlatformUser(c.Context(), endpoint.PlatformUpdateEpRequest{
		UserID: pathID(c), Body: body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) resetPlatformUser(c *fiber.Ctx) error {
	_, err := h.eps.ResetPlatformUser(c.Context(), endpoint.ActorRequest{
		Actor: currentUser(c), UserID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Пароль сброшен"})
}

func (h *handlers) deactivatePlatformUser(c *fiber.Ctx) error {
	_, err := h.eps.DeactivatePlatformUser(c.Context(), endpoint.ActorRequest{
		Actor: currentUser(c), UserID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Пользователь удалён"})
}

func (h *handlers) directory(c *fiber.Ctx) error {
	me := currentUser(c)
	// Скоуп — активная компания актора из токена (её члены). Нет активной
	// компании → глобальный поиск всех пользователей (контакты мессенджера).
	req := dto.DirectoryRequest{
		ActorID:   me.ID,
		Query:     c.Query("q"),
		CompanyID: me.CompanyID,
	}
	switch c.Query("exclude_self") {
	case "1", "true", "yes":
		req.ExcludeID = me.ID
	}
	// all=1 — глобальный каталог (для чата с кем угодно), перебивает company-scope.
	switch c.Query("all") {
	case "1", "true", "yes":
		req.CompanyID = nil
	}
	// by=login — глобальный поиск строго по логину (мессенджер: новый собеседник).
	req.LoginOnly = c.Query("by") == "login"

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
	actor := currentUser(c)
	_, err := h.eps.HideUser(c.Context(), endpoint.ActorRequest{
		Actor: actor, UserID: pathID(c),
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
	actor := currentUser(c)
	resp, err := h.eps.AssignRole(c.Context(), endpoint.AssignRoleEpRequest{
		Actor: actor, UserID: pathID(c), RoleID: body.RoleID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) resetPassword(c *fiber.Ctx) error {
	actor := currentUser(c)
	_, err := h.eps.ResetPassword(c.Context(), endpoint.ActorRequest{
		Actor: actor, UserID: pathID(c),
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

// ── Участники компании (multi-company; Администратор системы) ──

func (h *handlers) listMembers(c *fiber.Ctx) error {
	resp, err := h.eps.ListCompanyMembers(c.Context(), endpoint.CompanyActorEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) companyCandidates(c *fiber.Ctx) error {
	resp, err := h.eps.SearchCandidates(c.Context(), endpoint.CandidatesEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), Query: c.Query("q"),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) addMember(c *fiber.Ctx) error {
	var body dto.AddMemberRequest
	if err := c.BodyParser(&body); err != nil || body.UserID == 0 || body.RoleID == 0 {
		return badRequest(c, "user_id и role_id обязательны")
	}
	_, err := h.eps.AddCompanyMember(c.Context(), endpoint.MemberEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), UserID: body.UserID, RoleID: body.RoleID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Сотрудник добавлен в компанию"})
}

func (h *handlers) setMemberRole(c *fiber.Ctx) error {
	userID, _ := c.ParamsInt("userId")
	var body struct {
		RoleID int64 `json:"role_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.RoleID == 0 {
		return badRequest(c, "role_id обязателен")
	}
	_, err := h.eps.SetMemberRole(c.Context(), endpoint.MemberEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), UserID: int64(userID), RoleID: body.RoleID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Роль обновлена"})
}

func (h *handlers) removeMember(c *fiber.Ctx) error {
	userID, _ := c.ParamsInt("userId")
	_, err := h.eps.RemoveMember(c.Context(), endpoint.MemberEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), UserID: int64(userID),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Сотрудник убран из компании"})
}

func (h *handlers) companyInvite(c *fiber.Ctx) error {
	resp, err := h.eps.CompanyInvite(c.Context(), endpoint.CompanyActorEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"code": resp})
}

func (h *handlers) regenerateInvite(c *fiber.Ctx) error {
	resp, err := h.eps.RegenerateInvite(c.Context(), endpoint.CompanyActorEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"code": resp})
}

// ── Сотрудники В КОНКРЕТНОЙ компании (раздел «Компании»; создатель компании) ──

func (h *handlers) createCompanyUser(c *fiber.Ctx) error {
	var body dto.CreateUserRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	if body.FIO == "" || body.Login == "" || body.RoleID == 0 {
		return badRequest(c, "fio, login и role_id обязательны")
	}
	resp, err := h.eps.CreateCompanyUser(c.Context(), endpoint.CompanyUserCreateEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), Body: body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateCompanyMember(c *fiber.Ctx) error {
	userID, _ := c.ParamsInt("userId")
	var body dto.UpdateUserRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	resp, err := h.eps.UpdateCompanyMember(c.Context(), endpoint.CompanyUserUpdateEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), UserID: int64(userID), Body: body,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) resetCompanyMember(c *fiber.Ctx) error {
	userID, _ := c.ParamsInt("userId")
	if _, err := h.eps.ResetCompanyMember(c.Context(), endpoint.CompanyMemberResetEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), UserID: int64(userID),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Пароль сброшен"})
}

// ── Email-приглашения в компанию ──

func (h *handlers) createCompanyInvite(c *fiber.Ctx) error {
	var body struct {
		Email  string `json:"email"`
		RoleID int64  `json:"role_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.Email == "" || body.RoleID == 0 {
		return badRequest(c, "email и role_id обязательны")
	}
	if _, err := h.eps.CreateCompanyInvite(c.Context(), endpoint.CreateInviteEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), Email: body.Email, RoleID: body.RoleID,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Приглашение отправлено"})
}

func (h *handlers) getInvitePreview(c *fiber.Ctx) error {
	resp, err := h.eps.GetInvitePreview(c.Context(), c.Params("token"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) acceptCompanyInvite(c *fiber.Ctx) error {
	resp, err := h.eps.AcceptCompanyInvite(c.Context(), endpoint.AcceptInviteEpRequest{
		UserID: tokenUserID(c), Token: c.Params("token"),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

func (h *handlers) joinByInvite(c *fiber.Ctx) error {
	resp, err := h.eps.JoinByCode(c.Context(), endpoint.JoinEpRequest{
		UserID: tokenUserID(c), Code: c.Params("code"),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

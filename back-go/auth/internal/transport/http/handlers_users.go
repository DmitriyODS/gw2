package http

import (
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/avatar"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
)

const avatarMaxBytes = 2 * 1024 * 1024

// applyAdminScope — у Администратора системы (actor.CompanyID == nil) активная
// компания приходит в ?company_id=; проставляем её в actor на время запроса,
// чтобы company-scoped операции (роль/скрытие/сброс) работали единообразно.
// У обычного актора компания уже в токене — не трогаем.
func applyAdminScope(c *fiber.Ctx, actor *domain.User) {
	if actor == nil || actor.CompanyID != nil {
		return
	}
	if raw := c.Query("company_id"); raw != "" {
		if cid, err := strconv.ParseInt(raw, 10, 64); err == nil {
			actor.CompanyID = &cid
		}
	}
}

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
	// Скоуп — активная компания актора из токена; у Администратора системы её нет,
	// она приходит в ?company_id= (nil → все видимые пользователи всех компаний).
	applyAdminScope(c, me)
	req := dto.DirectoryRequest{
		ActorID:   me.ID,
		Query:     c.Query("q"),
		CompanyID: me.CompanyID,
	}
	switch c.Query("exclude_self") {
	case "1", "true", "yes":
		req.ExcludeID = me.ID
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
	actor := currentUser(c)
	applyAdminScope(c, actor)
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
	applyAdminScope(c, actor)
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
	applyAdminScope(c, actor)
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

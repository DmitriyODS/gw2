package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
)

const (
	refreshCookie = "refresh_token"
	cookieMaxAge  = 30 * 24 * 3600 // 30 дней — синхронно с TTL refresh-токена
)

func setRefreshCookie(c *fiber.Ctx, refreshToken string) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookie,
		Value:    refreshToken,
		MaxAge:   cookieMaxAge,
		HTTPOnly: true,
		Secure:   true,
		SameSite: fiber.CookieSameSiteStrictMode,
		Path:     "/",
	})
}

func clearRefreshCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookie,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Secure:   true,
		SameSite: fiber.CookieSameSiteStrictMode,
		Path:     "/",
	})
}

func (h *handlers) login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil || req.Login == "" || req.Password == "" {
		return badRequest(c, "Логин и пароль обязательны")
	}

	resp, err := h.eps.Login(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

func (h *handlers) selectCompany(c *fiber.Ctx) error {
	var req struct {
		SelectToken string `json:"select_token"`
		CompanyID   int64  `json:"company_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.SelectToken == "" || req.CompanyID == 0 {
		return badRequest(c, "select_token и company_id обязательны")
	}
	resp, err := h.eps.SelectCompany(c.Context(), endpoint.SelectCompanyEpRequest{
		SelectToken: req.SelectToken, CompanyID: req.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

func (h *handlers) switchCompany(c *fiber.Ctx) error {
	var req struct {
		CompanyID int64 `json:"company_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.CompanyID == 0 {
		return badRequest(c, "company_id обязателен")
	}
	resp, err := h.eps.SwitchCompany(c.Context(), endpoint.SwitchCompanyEpRequest{
		UserID: tokenUserID(c), CompanyID: req.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

func (h *handlers) refresh(c *fiber.Ctx) error {
	raw := c.Cookies(refreshCookie)
	if raw == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "INVALID_TOKEN", "message": "Refresh token недействителен",
		})
	}
	resp, err := h.eps.Refresh(c.Context(), raw)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp.(*dto.Session))
}

func (h *handlers) logout(c *fiber.Ctx) error {
	clearRefreshCookie(c)
	return c.JSON(fiber.Map{"message": "Выход выполнен"})
}

func (h *handlers) changeDefault(c *fiber.Ctx) error {
	var req dto.ChangeDefaultRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	if len([]rune(req.NewLogin)) < 3 {
		return badRequest(c, "Логин должен содержать не менее 3 символов")
	}
	if len([]rune(req.NewPassword)) < 8 {
		return badRequest(c, "Пароль должен содержать не менее 8 символов")
	}
	req.UserID = tokenUserID(c)

	resp, err := h.eps.ChangeDefault(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

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

func (h *handlers) register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil || req.FIO == "" || req.Email == "" || req.Password == "" {
		return badRequest(c, "Имя, email и пароль обязательны")
	}
	// Логин может прийти пустым — сервис сгенерирует из ФИО (транслит).
	resp, err := h.eps.Register(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// suggestLogin — live-подсказка свободного логина по ФИО (публичный, без авторизации).
func (h *handlers) suggestLogin(c *fiber.Ctx) error {
	fio := c.Query("fio")
	if fio == "" {
		return c.JSON(fiber.Map{"login": ""})
	}
	resp, err := h.eps.SuggestLogin(c.Context(), fio)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"login": resp})
}

func (h *handlers) verifyEmail(c *fiber.Ctx) error {
	var req dto.VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Неверный формат запроса")
	}
	if req.Token == "" && (req.Email == "" || req.Code == "") {
		return badRequest(c, "Передайте token или email и code")
	}
	resp, err := h.eps.VerifyEmail(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

func (h *handlers) resendVerification(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return badRequest(c, "email обязателен")
	}
	if _, err := h.eps.ResendVerification(c.Context(), req.Email); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) forgotPassword(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return badRequest(c, "email обязателен")
	}
	// Ответ всегда ok (не раскрываем наличие аккаунта).
	if _, err := h.eps.RequestPasswordReset(c.Context(), req.Email); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) resetPasswordByToken(c *fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil || req.Token == "" || req.NewPassword == "" {
		return badRequest(c, "token и new_password обязательны")
	}
	resp, err := h.eps.ResetPasswordByToken(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
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

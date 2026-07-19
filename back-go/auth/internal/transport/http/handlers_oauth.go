// OAuth-провайдер (связка аккаунтов навыка Алисы) и вход через Яндекс ID.
package http

import (
	"encoding/base64"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

// oauthAuthorize — согласие со страницы фронта /oauth/authorize: выпускает
// одноразовый код и возвращает URL возврата к Яндексу.
func (h *handlers) oauthAuthorize(c *fiber.Ctx) error {
	var req dto.OAuthAuthorizeRequest
	if err := c.BodyParser(&req); err != nil || req.ClientID == "" || req.RedirectURI == "" {
		return badRequest(c, "client_id и redirect_uri обязательны")
	}
	var companyID *int64
	if cid := pasetoauth.CompanyID(c); cid > 0 {
		companyID = &cid
	}
	resp, err := h.eps.OAuthAuthorize(c.Context(), endpoint.OAuthAuthorizeEpRequest{
		UserID: tokenUserID(c), CompanyID: companyID, Body: req,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"redirect_url": resp.(string)})
}

// oauthToken — token-эндпоинт для Яндекса (form-urlencoded, публичный).
// Клиентские креды приходят в теле либо Basic-заголовком.
func (h *handlers) oauthToken(c *fiber.Ctx) error {
	req := dto.OAuthTokenRequest{
		GrantType:    c.FormValue("grant_type"),
		Code:         c.FormValue("code"),
		RefreshToken: c.FormValue("refresh_token"),
		ClientID:     c.FormValue("client_id"),
		ClientSecret: c.FormValue("client_secret"),
	}
	if id, secret, ok := parseBasicAuth(c.Get(fiber.HeaderAuthorization)); ok {
		req.ClientID, req.ClientSecret = id, secret
	}
	resp, err := h.eps.OAuthToken(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp.(*dto.OAuthTokens))
}

func parseBasicAuth(header string) (string, string, bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return "", "", false
	}
	id, secret, ok := strings.Cut(string(raw), ":")
	return id, secret, ok
}

// yandexConfig — публичная конфигурация кнопки «Войти с Яндексом».
func (h *handlers) yandexConfig(c *fiber.Ctx) error {
	resp, err := h.eps.YandexConfig(c.Context(), nil)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// yandexCallback — вход/регистрация по коду авторизации Яндекса (как login).
func (h *handlers) yandexCallback(c *fiber.Ctx) error {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil || req.Code == "" {
		return badRequest(c, "code обязателен")
	}
	resp, err := h.eps.YandexLogin(c.Context(), req.Code)
	if err != nil {
		return h.respondError(c, err)
	}
	sess := resp.(*dto.Session)
	setRefreshCookie(c, sess.RefreshToken)
	return c.JSON(sess)
}

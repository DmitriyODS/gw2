package http

import (
	"strconv"
	"strings"

	"aidanwoods.dev/go-paseto"
	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

const localUserID = "userID"

// authParser — валидация PASETO v4.public access-токенов, которые выпускает
// authsvc (общий публичный ключ PASETO_PUBLIC_KEY): claims sub (id строкой),
// type=="access", force_change. Пользователь дополнительно сверяется с БД
// (is_hidden, активность компании) — как в декораторе @require_auth.
type authParser struct {
	public paseto.V4AsymmetricPublicKey
	users  domain.UserReader
}

// parseUserID — извлечь user_id из Bearer-токена; 0 = невалидный/отсутствует.
// forceChange=true — пользователь обязан сменить дефолтный пароль (403).
func (a *authParser) parseUserID(c *fiber.Ctx) (userID int64, forceChange bool) {
	header := c.Get(fiber.HeaderAuthorization)
	if !strings.HasPrefix(header, "Bearer ") {
		return 0, false
	}
	parser := paseto.NewParser() // проверяет exp/iat/nbf
	t, err := parser.ParseV4Public(a.public, strings.TrimPrefix(header, "Bearer "), nil)
	if err != nil {
		return 0, false
	}
	if typ, err := t.GetString("type"); err != nil || typ != "access" {
		return 0, false
	}
	sub, err := t.GetSubject()
	if err != nil {
		return 0, false
	}
	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	var fc bool
	_ = t.Get("force_change", &fc)
	return id, fc
}

// requireAuth — мидлварь обязательной авторизации; кладёт userID в Locals.
func (a *authParser) requireAuth(c *fiber.Ctx) error {
	userID, forceChange := a.parseUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "UNAUTHORIZED", "message": "Требуется авторизация",
		})
	}
	if forceChange {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "FORBIDDEN", "message": "FORCE_PASSWORD_CHANGE",
		})
	}
	user, err := a.users.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера",
		})
	}
	if user == nil || user.IsHidden {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "UNAUTHORIZED", "message": "Пользователь не найден",
		})
	}
	if !user.CompanyActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "FORBIDDEN", "message": "COMPANY_DISABLED",
		})
	}
	c.Locals(localUserID, userID)
	return c.Next()
}

// optionalUserID — для публичных роутов с необязательной авторизацией
// (вход по ссылке): невалидный/чужой токен не ошибка, просто гость.
func (a *authParser) optionalUserID(c *fiber.Ctx) int64 {
	userID, forceChange := a.parseUserID(c)
	if userID == 0 || forceChange {
		return 0
	}
	user, err := a.users.GetUser(c.Context(), userID)
	if err != nil || user == nil || user.IsHidden || !user.CompanyActive {
		return 0
	}
	return userID
}

func currentUserID(c *fiber.Ctx) int64 {
	id, _ := c.Locals(localUserID).(int64)
	return id
}

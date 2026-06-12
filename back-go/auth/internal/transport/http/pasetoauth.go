package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/token"
)

const localUser = "currentUser"

// authParser — проверка PASETO v4.public access-токенов. Пользователь
// дополнительно сверяется с БД (is_hidden, активность компании) — как
// прежний декоратор @require_auth во Flask.
type authParser struct {
	verifier *token.Verifier
	users    domain.UserRepository
}

// tokenUserID — user_id из Bearer-токена; 0 = невалидный/отсутствует.
func (a *authParser) tokenUserID(c *fiber.Ctx) (userID int64, forceChange bool) {
	header := c.Get(fiber.HeaderAuthorization)
	if !strings.HasPrefix(header, "Bearer ") {
		return 0, false
	}
	return a.verifier.ParseAccess(strings.TrimPrefix(header, "Bearer "))
}

func unauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "UNAUTHORIZED", "message": "Требуется авторизация",
	})
}

// requireToken — только валидность токена, без force_change и похода в БД
// (logout, change-default: ими пользуется и тот, кто обязан сменить пароль).
func (a *authParser) requireToken(c *fiber.Ctx) error {
	userID, _ := a.tokenUserID(c)
	if userID == 0 {
		return unauthorized(c)
	}
	c.Locals(localTokenUserID, userID)
	return c.Next()
}

const localTokenUserID = "tokenUserID"

// requireAuth — полная авторизация: токен + force_change-гейт + сверка с БД.
func (a *authParser) requireAuth(c *fiber.Ctx) error {
	userID, forceChange := a.tokenUserID(c)
	if userID == 0 {
		return unauthorized(c)
	}
	if forceChange {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "FORBIDDEN", "message": "FORCE_PASSWORD_CHANGE",
		})
	}
	user, err := a.users.GetByID(c.Context(), userID)
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
	if !user.CompanyActive() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "FORBIDDEN", "message": "COMPANY_DISABLED",
		})
	}
	c.Locals(localUser, user)
	return c.Next()
}

// requireRole — уровень роли не ниже min (вешается после requireAuth).
func (a *authParser) requireRole(min int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if currentUser(c).Level() < min {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "FORBIDDEN", "message": "Недостаточно прав",
			})
		}
		return c.Next()
	}
}

func currentUser(c *fiber.Ctx) *domain.User {
	u, _ := c.Locals(localUser).(*domain.User)
	return u
}

func tokenUserID(c *fiber.Ctx) int64 {
	id, _ := c.Locals(localTokenUserID).(int64)
	return id
}

package pasetoauth

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

const (
	localUserID = "pasetoauth.userID"
	localUser   = "pasetoauth.user"
)

// AuthInfo — срез пользователя для авторизационных проверок мидлвари.
// User — полный доменный пользователь сервиса (any, чтобы pkg не зависел
// от доменов): хендлеры достают его через CurrentUser без второго похода в БД.
type AuthInfo struct {
	RoleLevel     int
	IsHidden      bool
	CompanyActive bool
	User          any
}

// AuthSource — порт сверки с БД: вернуть пользователя для проверки
// (nil — не найден). Пользователь без компании (Администратор системы)
// должен возвращаться с CompanyActive=true.
type AuthSource func(ctx context.Context, userID int64) (*AuthInfo, error)

// Middleware — Fiber-мидлвари авторизации; поведение байт-в-байт повторяет
// прежние Flask-декораторы @require_auth / @require_role.
type Middleware struct {
	verifier *Verifier
	source   AuthSource
}

func NewMiddleware(verifier *Verifier, source AuthSource) *Middleware {
	return &Middleware{verifier: verifier, source: source}
}

func unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "UNAUTHORIZED", "message": message,
	})
}

func forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": "FORBIDDEN", "message": message,
	})
}

// RequireToken — только валидность токена, без force_change-гейта и похода
// в БД (logout, change-default: ими пользуется и тот, кто обязан сменить
// пароль).
func (m *Middleware) RequireToken(c *fiber.Ctx) error {
	claims := m.verifier.FromRequest(c)
	if claims.UserID == 0 {
		return unauthorized(c, "Требуется авторизация")
	}
	c.Locals(localUserID, claims.UserID)
	return c.Next()
}

// RequireAuth — полная авторизация: токен + force_change-гейт + сверка с БД
// (is_hidden, активность компании). Кладёт UserID и AuthInfo в Locals.
func (m *Middleware) RequireAuth(c *fiber.Ctx) error {
	claims := m.verifier.FromRequest(c)
	if claims.UserID == 0 {
		return unauthorized(c, "Требуется авторизация")
	}
	if claims.ForceChange {
		return forbidden(c, "FORCE_PASSWORD_CHANGE")
	}
	info, err := m.source(c.Context(), claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера",
		})
	}
	if info == nil || info.IsHidden {
		return unauthorized(c, "Пользователь не найден")
	}
	if !info.CompanyActive {
		return forbidden(c, "COMPANY_DISABLED")
	}
	c.Locals(localUserID, claims.UserID)
	c.Locals(localUser, info)
	return c.Next()
}

// RequireRole — уровень роли не ниже min (вешается после RequireAuth).
func (m *Middleware) RequireRole(min int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		info := Current(c)
		if info == nil || info.RoleLevel < min {
			return forbidden(c, "Недостаточно прав")
		}
		return c.Next()
	}
}

// OptionalUserID — для публичных роутов с необязательной авторизацией
// (вход по ссылке): невалидный/чужой токен не ошибка, просто гость (0).
func (m *Middleware) OptionalUserID(c *fiber.Ctx) int64 {
	claims := m.verifier.FromRequest(c)
	if claims.UserID == 0 || claims.ForceChange {
		return 0
	}
	info, err := m.source(c.Context(), claims.UserID)
	if err != nil || info == nil || info.IsHidden || !info.CompanyActive {
		return 0
	}
	return claims.UserID
}

// UserID — id пользователя из Locals (после RequireAuth/RequireToken).
func UserID(c *fiber.Ctx) int64 {
	id, _ := c.Locals(localUserID).(int64)
	return id
}

// Current — AuthInfo из Locals (после RequireAuth; nil после RequireToken).
func Current(c *fiber.Ctx) *AuthInfo {
	info, _ := c.Locals(localUser).(*AuthInfo)
	return info
}

// CurrentUser — полный доменный пользователь сервиса из AuthInfo.User
// (nil, если RequireAuth не отработал или сервис его не кладёт).
func CurrentUser(c *fiber.Ctx) any {
	if info := Current(c); info != nil {
		return info.User
	}
	return nil
}

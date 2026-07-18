package pasetoauth

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

const (
	localUserID    = "pasetoauth.userID"
	localUser      = "pasetoauth.user"
	localCompanyID = "pasetoauth.companyID"
)

// AuthInfo — срез пользователя для авторизационных проверок мидлвари.
// User — полный доменный пользователь сервиса (any, чтобы pkg не зависел
// от доменов): хендлеры достают его через CurrentUser без второго похода в БД.
//
// RoleLevel — роль в АКТИВНОЙ компании (0, если активной компании нет).
// IsActive — глобально активный аккаунт (отключённый супер-админом — false).
// IsSuperAdmin — платформенный супер-админ.
// CompanyActive — активна ли ВЫБРАННАЯ компания (true, если компании нет).
type AuthInfo struct {
	RoleLevel     int
	IsActive      bool
	IsSuperAdmin  bool
	CompanyActive bool
	User          any
}

// AuthSource — порт сверки с БД: вернуть пользователя для проверки
// (nil — не найден). active — клеймы access-токена: активная компания и роль
// в ней берутся ИЗ ТОКЕНА (источник истины), реализация сверяет с БД только
// глобальную активность аккаунта и активность ИМЕННО выбранной компании
// (active.CompanyID). Если активной компании нет — CompanyActive=true.
type AuthSource func(ctx context.Context, userID int64, active Claims) (*AuthInfo, error)

// Middleware — Fiber-мидлвари авторизации.
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
// (активность аккаунта, активность выбранной компании). Кладёт UserID и
// AuthInfo в Locals.
func (m *Middleware) RequireAuth(c *fiber.Ctx) error {
	claims := m.verifier.FromRequest(c)
	if claims.UserID == 0 {
		return unauthorized(c, "Требуется авторизация")
	}
	if claims.ForceChange {
		return forbidden(c, "FORCE_PASSWORD_CHANGE")
	}
	info, err := m.source(c.Context(), claims.UserID, claims)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера",
		})
	}
	if info == nil || !info.IsActive {
		return unauthorized(c, "Пользователь не найден")
	}
	if !info.CompanyActive {
		return forbidden(c, "COMPANY_DISABLED")
	}
	c.Locals(localUserID, claims.UserID)
	c.Locals(localUser, info)
	if claims.CompanyID != nil {
		c.Locals(localCompanyID, *claims.CompanyID)
	}
	return c.Next()
}

// RequireRole — уровень роли в АКТИВНОЙ компании не ниже min (вешается после
// RequireAuth). Требует выбранной компании: супер-админ и пользователь без
// активной компании сюда не проходят (RoleLevel == 0) — компанийная
// функциональность доступна только участникам компании.
func (m *Middleware) RequireRole(min int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		info := Current(c)
		if info == nil || info.RoleLevel < min {
			return forbidden(c, "Недостаточно прав")
		}
		return c.Next()
	}
}

// RequireSuperAdmin — только платформенный супер-админ (вешается после
// RequireAuth). Для управления компаниями и просмотра пользователей платформы.
func (m *Middleware) RequireSuperAdmin(c *fiber.Ctx) error {
	info := Current(c)
	if info == nil || !info.IsSuperAdmin {
		return forbidden(c, "Требуется супер-администратор")
	}
	return c.Next()
}

// OptionalUserID — для публичных роутов с необязательной авторизацией
// (вход по ссылке): невалидный/чужой токен не ошибка, просто гость (0).
func (m *Middleware) OptionalUserID(c *fiber.Ctx) int64 {
	claims := m.verifier.FromRequest(c)
	if claims.UserID == 0 || claims.ForceChange {
		return 0
	}
	info, err := m.source(c.Context(), claims.UserID, claims)
	if err != nil || info == nil || !info.IsActive || !info.CompanyActive {
		return 0
	}
	return claims.UserID
}

// UserID — id пользователя из Locals (после RequireAuth/RequireToken).
func UserID(c *fiber.Ctx) int64 {
	id, _ := c.Locals(localUserID).(int64)
	return id
}

// CompanyID — id активной компании из токена (0, если активной компании нет).
// Заполняется в RequireAuth.
func CompanyID(c *fiber.Ctx) int64 {
	id, _ := c.Locals(localCompanyID).(int64)
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

// Package pasetoauth — проверка PASETO v4.public access-токенов Groove Work.
//
// Токены выпускает ТОЛЬКО authsvc (PASETO_PRIVATE_KEY); остальные сервисы
// проверяют подпись по общему публичному ключу PASETO_PUBLIC_KEY —
// скомпрометированный сервис-верификатор не может выпустить токен.
// Клеймы: sub (id строкой), type=="access", exp/iat, force_change.
package pasetoauth

import (
	"strconv"
	"strings"

	"aidanwoods.dev/go-paseto"
	"github.com/gofiber/fiber/v2"
)

// Claims — авторизационные клеймы access-токена. Нулевой UserID — токен
// невалиден/отсутствует.
//
// Идентичность пользователя (UserID) развязана с компаниями. CompanyID/RoleLevel
// описывают АКТИВНУЮ компанию сессии (выбранную при login/switch) и опциональны:
// CompanyID == nil означает, что активной компании нет — это НОРМАЛЬНОЕ состояние
// (мессенджер, профиль, контакты), а не признак админа. RoleLevel значим только
// при CompanyID != nil — это роль пользователя именно в этой компании.
//
// IsSuperAdmin — платформенный супер-админ: отдельный класс, видит все компании и
// пользователей, но к компанийной функциональности (задачи, грувики, YouGile)
// доступа не имеет.
type Claims struct {
	UserID       int64
	ForceChange  bool
	IsSuperAdmin bool
	CompanyID    *int64
	RoleLevel    int
}

// Verifier — проверка подписи и клеймов access-токена.
type Verifier struct {
	public paseto.V4AsymmetricPublicKey
}

func NewVerifier(publicHex string) (*Verifier, error) {
	public, err := paseto.NewV4AsymmetricPublicKeyFromHex(publicHex)
	if err != nil {
		return nil, err
	}
	return &Verifier{public: public}, nil
}

// ParseAccess — клеймы из access-токена; Claims{} (UserID==0), если токен
// невалиден, просрочен или не access-типа.
func (v *Verifier) ParseAccess(raw string) Claims {
	parser := paseto.NewParser() // проверяет exp/iat/nbf
	t, err := parser.ParseV4Public(v.public, raw, nil)
	if err != nil {
		return Claims{}
	}
	if typ, err := t.GetString("type"); err != nil || typ != "access" {
		return Claims{}
	}
	sub, err := t.GetSubject()
	if err != nil {
		return Claims{}
	}
	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || id <= 0 {
		return Claims{}
	}
	var fc, sa bool
	var cid *int64
	var rl int
	_ = t.Get("force_change", &fc)
	_ = t.Get("company_id", &cid)
	_ = t.Get("role_level", &rl)
	_ = t.Get("is_super_admin", &sa)
	return Claims{UserID: id, ForceChange: fc, IsSuperAdmin: sa, CompanyID: cid, RoleLevel: rl}
}

// FromRequest — клеймы из Bearer-заголовка Fiber-запроса.
func (v *Verifier) FromRequest(c *fiber.Ctx) Claims {
	header := c.Get(fiber.HeaderAuthorization)
	if !strings.HasPrefix(header, "Bearer ") {
		return Claims{}
	}
	return v.ParseAccess(strings.TrimPrefix(header, "Bearer "))
}

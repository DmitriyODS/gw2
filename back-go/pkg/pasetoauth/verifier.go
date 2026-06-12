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

// Claims — авторизационные клеймы access-токена. Нулевой UserID —
// токен невалиден/отсутствует.
type Claims struct {
	UserID      int64
	ForceChange bool
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
	var fc bool
	_ = t.Get("force_change", &fc)
	return Claims{UserID: id, ForceChange: fc}
}

// FromRequest — клеймы из Bearer-заголовка Fiber-запроса.
func (v *Verifier) FromRequest(c *fiber.Ctx) Claims {
	header := c.Get(fiber.HeaderAuthorization)
	if !strings.HasPrefix(header, "Bearer ") {
		return Claims{}
	}
	return v.ParseAccess(strings.TrimPrefix(header, "Bearer "))
}

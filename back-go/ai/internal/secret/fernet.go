// Package secret — Fernet-шифрование AI API-ключей компаний. Портировано из
// back/app/utils/ai_secret.py; формат токенов совместим с python
// cryptography.Fernet (ключ — AI_KEY_ENCRYPTION_KEY, base64-32 байта).
//
// Ключ читается один раз при создании. Не задан/некорректен — Encrypt отдаёт
// domain.ErrSecretMisconfigured (hard-fail, как во Flask), Decrypt — ok=false
// (фичи AI тихо выключаются, запросы не роняем).
package secret

import (
	"strings"

	"github.com/fernet/fernet-go"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

type Cipher struct {
	key *fernet.Key // nil — не сконфигурирован
}

var _ domain.SecretCipher = (*Cipher)(nil)

// New — raw из env AI_KEY_ENCRYPTION_KEY; пустой/битый ключ не фатален
// на старте (ровно как lru_cache-инициализация во Flask — упадёт при
// использовании шифрования).
func New(raw string) *Cipher {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &Cipher{}
	}
	key, err := fernet.DecodeKey(raw)
	if err != nil {
		return &Cipher{}
	}
	return &Cipher{key: key}
}

func (c *Cipher) Encrypt(plain string) ([]byte, error) {
	if c.key == nil {
		return nil, domain.ErrSecretMisconfigured
	}
	return fernet.EncryptAndSign([]byte(plain), c.key)
}

// Decrypt — ttl 0: возраст токена не проверяем, как Fernet.decrypt() без ttl.
func (c *Cipher) Decrypt(enc []byte) (string, bool) {
	if c.key == nil || len(enc) == 0 {
		return "", false
	}
	msg := fernet.VerifyAndDecrypt(enc, 0, []*fernet.Key{c.key})
	if msg == nil {
		return "", false
	}
	return string(msg), true
}

// MakeHint — короткая маска ключа для UI: первые 3 + … + последние 4 символа
// (срезы по рунам — как срезы str в Python).
func MakeHint(plain string) string {
	if plain == "" {
		return ""
	}
	r := []rune(plain)
	if len(r) <= 8 {
		// plain[-2:] в Python безопасен и для односимвольной строки.
		return "…" + string(r[max(0, len(r)-2):])
	}
	return string(r[:3]) + "…" + string(r[len(r)-4:])
}

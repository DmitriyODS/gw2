package yougile

import (
	"errors"
	"strings"

	"github.com/fernet/fernet-go"
)

// Шифрование/расшифровка API-ключей YouGile. Ключ Fernet'а — env
// YOUGILE_ENC_KEY (tasksvc). Принципиально отдельная переменная от
// AI_KEY_ENCRYPTION_KEY (живёт в env aisvc): компрометация одного секрета
// не должна снимать защиту со второго; разные жизненные циклы (AI-ключи —
// один на компанию, YouGile-ключи — по одному на каждого юзера).
//
// При отсутствии или некорректном значении переменной — ErrMisconfigured
// при первом обращении (hard-fail, как во Flask): сохранить ключ YG в
// открытом виде или потерять его при сбое ключа шифрования недопустимо.

// ErrMisconfigured — YOUGILE_ENC_KEY не задан или некорректен. Роуты отдают
// 500 ENC_KEY_MISCONFIGURED — видно админу, он знает, что поправить env.
var ErrMisconfigured = errors.New("YOUGILE_ENC_KEY не задан или некорректен")

type Cipher struct {
	key *fernet.Key // nil — не сконфигурирован
}

// NewCipher — raw из env; пустой/битый ключ не фатален на старте (ровно как
// lru_cache-инициализация во Flask — упадёт при использовании шифрования).
func NewCipher(raw string) *Cipher {
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

func (c *Cipher) EncryptKey(plain string) ([]byte, error) {
	if c.key == nil {
		return nil, ErrMisconfigured
	}
	if plain == "" {
		return nil, errors.New("empty yougile key")
	}
	return fernet.EncryptAndSign([]byte(plain), c.key)
}

// DecryptKey — "" без ошибки = ключ не расшифровался (YOUGILE_ENC_KEY
// сменили без миграции — UI попросит переподключение вместо 500).
func (c *Cipher) DecryptKey(enc []byte) (string, error) {
	if len(enc) == 0 {
		return "", nil
	}
	if c.key == nil {
		return "", ErrMisconfigured
	}
	msg := fernet.VerifyAndDecrypt(enc, 0, []*fernet.Key{c.key})
	if msg == nil {
		return "", nil
	}
	return string(msg), nil
}

// MakeFingerprint — последние 4 символа ключа для UI («…X9aQ»).
// Хранится открыто.
func MakeFingerprint(plain string) string {
	if len(plain) >= 4 {
		return plain[len(plain)-4:]
	}
	return plain
}

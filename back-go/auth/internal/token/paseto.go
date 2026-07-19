// Package token — выпуск и проверка PASETO-токенов платформы.
//
// Access-токен — v4.public (Ed25519): подписывает только authsvc, проверяют
// Flask и callsvc по публичному ключу (PASETO_PUBLIC_KEY) — выпустить токен
// они не могут. Refresh-токен — v4.local (XChaCha20-Poly1305) с отдельным
// симметричным ключом: его читает только сам authsvc.
//
// Клеймы access-токена: sub (id строкой), type=access, force_change,
// is_super_admin и опциональный контекст активной компании (company_id,
// company_name, company_settings, role_level). «Нет активной компании» —
// нормальное состояние (мессенджер/профиль), а не признак админа.
package token

import (
	"fmt"
	"strconv"
	"time"

	"aidanwoods.dev/go-paseto"

	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Claims struct {
	UserID          int64
	ForceChange     bool
	IsSuperAdmin    bool
	CompanyID       *int64
	CompanyName     *string
	CompanySettings map[string]any
	RoleLevel       int
}

type Issuer struct {
	secret     paseto.V4AsymmetricSecretKey
	public     paseto.V4AsymmetricPublicKey
	refreshKey paseto.V4SymmetricKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewIssuer: privateHex — 64 байта hex (seed||public Ed25519),
// refreshHex — 32 байта hex симметричного ключа refresh-токенов.
func NewIssuer(privateHex, refreshHex string, accessTTL, refreshTTL time.Duration) (*Issuer, error) {
	secret, err := paseto.NewV4AsymmetricSecretKeyFromHex(privateHex)
	if err != nil {
		return nil, fmt.Errorf("PASETO_PRIVATE_KEY: %w", err)
	}
	refreshKey, err := paseto.V4SymmetricKeyFromHex(refreshHex)
	if err != nil {
		return nil, fmt.Errorf("PASETO_REFRESH_KEY: %w", err)
	}
	return &Issuer{
		secret:     secret,
		public:     secret.Public(),
		refreshKey: refreshKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

// PublicKeyHex — публичный ключ проверки access-токенов (для логов/диагностики).
func (i *Issuer) PublicKeyHex() string { return i.public.ExportHex() }

func (i *Issuer) AccessToken(c Claims) (string, error) {
	t := paseto.NewToken()
	now := time.Now()
	t.SetIssuedAt(now)
	t.SetNotBefore(now)
	t.SetExpiration(now.Add(i.accessTTL))
	t.SetSubject(strconv.FormatInt(c.UserID, 10))
	t.SetString("type", "access")
	if err := setAll(&t, map[string]any{
		"force_change":     c.ForceChange,
		"is_super_admin":   c.IsSuperAdmin,
		"company_id":       c.CompanyID,
		"company_name":     c.CompanyName,
		"company_settings": c.CompanySettings,
		"role_level":       c.RoleLevel,
	}); err != nil {
		return "", err
	}
	return t.V4Sign(i.secret, nil), nil
}

// selectTTL — срок жизни короткого select-токена login-gate (выбор компании
// после ввода пароля, до выдачи полноценной сессии).
// AccessTTL — срок жизни access-токена (expires_in OAuth-провайдера).
func (i *Issuer) AccessTTL() time.Duration { return i.accessTTL }

const selectTTL = 5 * time.Minute

// RefreshToken — refresh-токен несёт user_id и АКТИВНУЮ компанию сессии
// (companyID, nil — активной компании нет): на refresh выбранная компания
// восстанавливается без обращения к localStorage.
func (i *Issuer) RefreshToken(userID int64, companyID *int64) (string, error) {
	t := paseto.NewToken()
	now := time.Now()
	t.SetIssuedAt(now)
	t.SetNotBefore(now)
	t.SetExpiration(now.Add(i.refreshTTL))
	t.SetSubject(strconv.FormatInt(userID, 10))
	t.SetString("type", "refresh")
	if err := t.Set("company_id", companyID); err != nil {
		return "", err
	}
	return t.V4Encrypt(i.refreshKey, nil), nil
}

// ParseRefresh — проверить refresh-токен и вернуть user_id и активную компанию
// (nil — без компании); ошибка на любом дефекте (подпись, срок, не тот тип).
func (i *Issuer) ParseRefresh(raw string) (int64, *int64, error) {
	id, t, err := i.parseLocal(raw, "refresh")
	if err != nil {
		return 0, nil, err
	}
	var cid *int64
	_ = t.Get("company_id", &cid)
	return id, cid, nil
}

// SelectToken — короткий токен этапа выбора компании при логине (>1 компании).
// Шифруется тем же refresh-ключом, отличается типом; в cookie не кладётся.
func (i *Issuer) SelectToken(userID int64) (string, error) {
	t := paseto.NewToken()
	now := time.Now()
	t.SetIssuedAt(now)
	t.SetNotBefore(now)
	t.SetExpiration(now.Add(selectTTL))
	t.SetSubject(strconv.FormatInt(userID, 10))
	t.SetString("type", "select")
	return t.V4Encrypt(i.refreshKey, nil), nil
}

// ParseSelect — проверить select-токен и вернуть user_id.
func (i *Issuer) ParseSelect(raw string) (int64, error) {
	id, _, err := i.parseLocal(raw, "select")
	return id, err
}

// parseLocal — разбор v4.local-токена нужного типа → user_id из subject.
func (i *Issuer) parseLocal(raw, wantType string) (int64, *paseto.Token, error) {
	parser := paseto.NewParser()
	t, err := parser.ParseV4Local(i.refreshKey, raw, nil)
	if err != nil {
		return 0, nil, err
	}
	typ, err := t.GetString("type")
	if err != nil || typ != wantType {
		return 0, nil, fmt.Errorf("not a %s token", wantType)
	}
	sub, err := t.GetSubject()
	if err != nil {
		return 0, nil, err
	}
	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || id <= 0 {
		return 0, nil, fmt.Errorf("bad subject")
	}
	return id, t, nil
}

func setAll(t *paseto.Token, claims map[string]any) error {
	for k, v := range claims {
		if err := t.Set(k, v); err != nil {
			return fmt.Errorf("claim %s: %w", k, err)
		}
	}
	return nil
}

// VerifierFromIssuer — authsvc проверяет access-токены собственным публичным
// ключом той же pkg-реализацией (pasetoauth), что и остальные сервисы.
func VerifierFromIssuer(i *Issuer) *pasetoauth.Verifier {
	v, err := pasetoauth.NewVerifier(i.PublicKeyHex())
	if err != nil { // невозможно: ExportHex отдаёт валидный ключ
		panic(err)
	}
	return v
}

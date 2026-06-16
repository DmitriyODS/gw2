package token

import (
	"strings"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

const (
	testPrivateHex = "b4cbfb43df4ce210727d953e4a713307fa19bb7d9f85041438d9e11b942a37741eb9dbbbbc047c03fd70604e0071f0987e16b28b757225c11f00415d0e20b1a2"
	testRefreshHex = "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"
)

func newTestIssuer(t *testing.T, accessTTL time.Duration) *Issuer {
	t.Helper()
	iss, err := NewIssuer(testPrivateHex, testRefreshHex, accessTTL, time.Hour)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	return iss
}

func TestAccessTokenRoundTrip(t *testing.T) {
	iss := newTestIssuer(t, time.Minute)
	cid := int64(7)
	name := "ООО Ромашка"
	raw, err := iss.AccessToken(Claims{
		UserID: 42, ForceChange: false, CompanyID: &cid, CompanyName: &name,
		CompanySettings: map[string]any{"uses_calls": true}, RoleLevel: 3, IsSuperAdmin: true,
	})
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if !strings.HasPrefix(raw, "v4.public.") {
		t.Fatalf("ожидался v4.public-токен, получено: %s", raw[:20])
	}

	v := VerifierFromIssuer(iss)
	claims := v.ParseAccess(raw)
	if claims.UserID != 42 || claims.ForceChange {
		t.Fatalf("ParseAccess: got (%d, %v), want (42, false)", claims.UserID, claims.ForceChange)
	}
	// Клейм супер-админа называется is_super_admin — verifier поднимает его в IsSuperAdmin.
	if !claims.IsSuperAdmin {
		t.Fatal("IsSuperAdmin не проброшен через клейм is_super_admin")
	}
}

func TestAccessTokenForceChange(t *testing.T) {
	iss := newTestIssuer(t, time.Minute)
	raw, _ := iss.AccessToken(Claims{UserID: 5, ForceChange: true, RoleLevel: 1})
	if c := VerifierFromIssuer(iss).ParseAccess(raw); c.UserID != 5 || !c.ForceChange {
		t.Fatalf("ожидался force_change=true, got (%d, %v)", c.UserID, c.ForceChange)
	}
}

func TestAccessTokenTampered(t *testing.T) {
	iss := newTestIssuer(t, time.Minute)
	raw, _ := iss.AccessToken(Claims{UserID: 1, RoleLevel: 1})
	bad := raw[:len(raw)-3] + "abc"
	if c := VerifierFromIssuer(iss).ParseAccess(bad); c.UserID != 0 {
		t.Fatalf("повреждённый токен принят, user_id=%d", c.UserID)
	}
}

func TestAccessTokenExpired(t *testing.T) {
	iss := newTestIssuer(t, -time.Minute)
	raw, _ := iss.AccessToken(Claims{UserID: 1, RoleLevel: 1})
	if c := VerifierFromIssuer(iss).ParseAccess(raw); c.UserID != 0 {
		t.Fatal("просроченный токен принят")
	}
}

func TestRefreshIsNotAccess(t *testing.T) {
	iss := newTestIssuer(t, time.Minute)
	cid := int64(7)
	refresh, err := iss.RefreshToken(42, &cid)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if !strings.HasPrefix(refresh, "v4.local.") {
		t.Fatalf("ожидался v4.local-токен, получено: %s", refresh[:20])
	}
	// Refresh не проходит как access…
	if c := VerifierFromIssuer(iss).ParseAccess(refresh); c.UserID != 0 {
		t.Fatal("refresh-токен прошёл проверку access")
	}
	// …а access не проходит как refresh.
	access, _ := iss.AccessToken(Claims{UserID: 42, RoleLevel: 1})
	if _, _, err := iss.ParseRefresh(access); err == nil {
		t.Fatal("access-токен прошёл проверку refresh")
	}
	// Сам refresh валиден и несёт активную компанию.
	id, company, err := iss.ParseRefresh(refresh)
	if err != nil || id != 42 {
		t.Fatalf("ParseRefresh: got (%d, %v), want (42, nil)", id, err)
	}
	if company == nil || *company != cid {
		t.Fatalf("ParseRefresh company: got %v, want %d", company, cid)
	}
}

func TestRefreshNoCompany(t *testing.T) {
	iss := newTestIssuer(t, time.Minute)
	refresh, _ := iss.RefreshToken(5, nil)
	id, company, err := iss.ParseRefresh(refresh)
	if err != nil || id != 5 || company != nil {
		t.Fatalf("ParseRefresh: got (%d, %v, %v), want (5, nil, nil)", id, company, err)
	}
}

func TestSelectToken(t *testing.T) {
	iss := newTestIssuer(t, time.Minute)
	sel, err := iss.SelectToken(99)
	if err != nil {
		t.Fatalf("SelectToken: %v", err)
	}
	id, err := iss.ParseSelect(sel)
	if err != nil || id != 99 {
		t.Fatalf("ParseSelect: got (%d, %v), want (99, nil)", id, err)
	}
	// Select-токен не проходит ни как access, ни как refresh.
	if c := VerifierFromIssuer(iss).ParseAccess(sel); c.UserID != 0 {
		t.Fatal("select-токен прошёл проверку access")
	}
	if _, _, err := iss.ParseRefresh(sel); err == nil {
		t.Fatal("select-токен прошёл проверку refresh")
	}
	// А refresh не проходит как select.
	refresh, _ := iss.RefreshToken(99, nil)
	if _, err := iss.ParseSelect(refresh); err == nil {
		t.Fatal("refresh-токен прошёл проверку select")
	}
}

func TestVerifierFromHex(t *testing.T) {
	iss := newTestIssuer(t, time.Minute)
	v, err := pasetoauth.NewVerifier(iss.PublicKeyHex())
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	raw, _ := iss.AccessToken(Claims{UserID: 9, RoleLevel: 2})
	if c := v.ParseAccess(raw); c.UserID != 9 {
		t.Fatalf("проверка по hex-ключу не прошла, user_id=%d", c.UserID)
	}
}

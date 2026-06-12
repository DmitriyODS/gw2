package secret

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

func genKey(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	// Формат Fernet.generate_key() в python — urlsafe base64 от 32 байт.
	return base64.URLEncoding.EncodeToString(raw)
}

func TestRoundTrip(t *testing.T) {
	c := New(genKey(t))
	enc, err := c.Encrypt("sk-proj-secret-key-123")
	if err != nil {
		t.Fatal(err)
	}
	plain, ok := c.Decrypt(enc)
	if !ok || plain != "sk-proj-secret-key-123" {
		t.Fatalf("round-trip: %q, ok=%v", plain, ok)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	enc, err := New(genKey(t)).Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	// Сменили ключ шифрования — расшифровка тихо отдаёт ok=false
	// (InvalidToken → None во Flask), а не ошибку.
	if plain, ok := New(genKey(t)).Decrypt(enc); ok {
		t.Fatalf("чужой ключ не должен расшифровываться: %q", plain)
	}
}

func TestMisconfigured(t *testing.T) {
	for _, raw := range []string{"", "  ", "не-ключ"} {
		c := New(raw)
		if _, err := c.Encrypt("secret"); !errors.Is(err, domain.ErrSecretMisconfigured) {
			t.Fatalf("ключ %q: ожидался ErrSecretMisconfigured, получено %v", raw, err)
		}
		if _, ok := c.Decrypt([]byte("anything")); ok {
			t.Fatalf("ключ %q: Decrypt должен отдавать ok=false", raw)
		}
	}
}

func TestDecryptEmpty(t *testing.T) {
	if _, ok := New(genKey(t)).Decrypt(nil); ok {
		t.Fatal("nil-шифртекст: ok=false (decrypt_api_key(None) → None)")
	}
}

func TestMakeHint(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"a", "…a"},
		{"ab", "…ab"},
		{"12345678", "…78"},          // len <= 8 → … + последние 2
		{"sk-proj-abcd", "sk-…abcd"}, // len > 8 → первые 3 + … + последние 4
	}
	for _, tc := range cases {
		if got := MakeHint(tc.in); got != tc.want {
			t.Errorf("MakeHint(%q) = %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}

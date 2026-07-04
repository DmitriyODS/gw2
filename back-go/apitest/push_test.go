package apitest

import (
	"net/http"
	"testing"
)

// TestPushRegisterUnregister — регистрация/удаление токена устройства
// (RequireToken: нужен валидный токен, но force_change-гейт не применяется).
func TestPushRegisterUnregister(t *testing.T) {
	a := newVerifiedUser(t)
	token := uniq("device-tok-")

	// Без токена — 401.
	r := pushAPI.doJSON(t, http.MethodPost, "/api/push/register", "",
		map[string]any{"token": token, "platform": "android"})
	requireStatus(t, r, 401, "register без токена")

	// Пустой token в теле — 400.
	r = pushAPI.doJSON(t, http.MethodPost, "/api/push/register", a.Token,
		map[string]any{"platform": "android"})
	requireError(t, r, 400, "BAD_REQUEST", "register без token")

	// Валидная регистрация.
	r = pushAPI.doJSON(t, http.MethodPost, "/api/push/register", a.Token,
		map[string]any{"token": token, "platform": "android"})
	requireStatus(t, r, 200, "register")
	if r.Str("status") != "ok" {
		t.Fatalf("register: неожиданный ответ: %s", r.Raw)
	}

	var cnt int
	if err := db.QueryRow(dbCtx(t),
		`SELECT count(*) FROM device_tokens WHERE token=$1 AND user_id=$2`, token, a.ID).Scan(&cnt); err != nil {
		t.Fatalf("проверка device_tokens: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("токен не сохранён (строк: %d)", cnt)
	}

	// Повторная регистрация того же токена идемпотентна (UPSERT, одна строка).
	r = pushAPI.doJSON(t, http.MethodPost, "/api/push/register", a.Token,
		map[string]any{"token": token, "platform": "ios"})
	requireStatus(t, r, 200, "повторный register")
	if err := db.QueryRow(dbCtx(t),
		`SELECT count(*) FROM device_tokens WHERE token=$1`, token).Scan(&cnt); err != nil {
		t.Fatalf("проверка device_tokens после upsert: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("повторная регистрация размножила токен (строк: %d)", cnt)
	}

	// Удаление.
	r = pushAPI.doJSON(t, http.MethodPost, "/api/push/unregister", a.Token,
		map[string]any{"token": token})
	requireStatus(t, r, 200, "unregister")
	if err := db.QueryRow(dbCtx(t),
		`SELECT count(*) FROM device_tokens WHERE token=$1`, token).Scan(&cnt); err != nil {
		t.Fatalf("проверка device_tokens после удаления: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("токен не удалён (строк: %d)", cnt)
	}
}

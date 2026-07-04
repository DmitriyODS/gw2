package apitest

import (
	"fmt"
	"net/http"
	"testing"
)

// Роли компании (фиксированный сид миграций: id == level).
const (
	roleEmployee = 1
	roleManager  = 2
	roleAdmin    = 3
)

// actor — тестовый пользователь: креды + текущая сессия.
type actor struct {
	ID       int64
	FIO      string
	Login    string
	Email    string
	Password string
	Token    string // access-токен текущей сессии
	Refresh  string // refresh-токен (значение cookie)
}

// applySession — обновить токены актора из ответа login/verify/select/switch.
func (a *actor) applySession(t *testing.T, r apiResp) {
	t.Helper()
	a.Token = r.Str("access_token")
	if a.Token == "" {
		t.Fatalf("в ответе нет access_token: %s", r.Raw)
	}
	a.ID = int64(r.Num("user_id"))
	if c := r.Cookie("refresh_token"); c != "" {
		a.Refresh = c
	}
}

// newVerifiedUser — публичная регистрация + подтверждение email кодом из БД.
// Возвращает актора с активной сессией (без компаний).
func newVerifiedUser(t *testing.T) *actor {
	t.Helper()
	a := &actor{
		FIO:      "Тестов Пользователь Апиевич",
		Login:    uniq("user_"),
		Password: "secret-pass-123",
	}
	a.Email = a.Login + "@apitest.local"

	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": a.FIO, "login": a.Login, "email": a.Email, "password": a.Password,
	})
	requireStatus(t, r, 201, "register "+a.Login)
	if r.Str("status") != "verification_required" {
		t.Fatalf("register: ожидался verification_required, тело: %s", r.Raw)
	}

	code, _ := verificationFor(t, a.Email)
	v := authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{
		"email": a.Email, "code": code,
	})
	requireStatus(t, v, 200, "verify-email "+a.Email)
	a.applySession(t, v)
	return a
}

// loginResp — сырой ответ логина (для негативных проверок).
func (a *actor) loginResp(t *testing.T) apiResp {
	t.Helper()
	return authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
		"login": a.Login, "password": a.Password,
	})
}

// mustLogin — логин с обновлением сессии актора (без login-gate).
func (a *actor) mustLogin(t *testing.T) apiResp {
	t.Helper()
	r := a.loginResp(t)
	requireStatus(t, r, 200, "login "+a.Login)
	a.applySession(t, r)
	return r
}

// createCompany — создать компанию и переключить сессию актора на неё.
func (a *actor) createCompany(t *testing.T, name string) int64 {
	t.Helper()
	r := authAPI.doJSON(t, http.MethodPost, "/api/companies", a.Token, map[string]any{"name": name})
	requireStatus(t, r, 201, "создание компании "+name)
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("создание компании: нет id в ответе: %s", r.Raw)
	}
	a.switchCompany(t, id)
	return id
}

// switchCompany — переключить активную компанию сессии.
func (a *actor) switchCompany(t *testing.T, companyID int64) {
	t.Helper()
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/switch-company", a.Token,
		map[string]any{"company_id": companyID})
	requireStatus(t, r, 200, fmt.Sprintf("switch-company %d", companyID))
	a.applySession(t, r)
}

// addToCompany — включить существующего пользователя в компанию с ролью
// (действует создатель компании) и переключить сессию адресата на неё.
func addToCompany(t *testing.T, creator *actor, companyID int64, member *actor, roleID int64) {
	t.Helper()
	r := authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/members", companyID),
		creator.Token, map[string]any{"user_id": member.ID, "role_id": roleID})
	requireStatus(t, r, 201, fmt.Sprintf("добавление %s в компанию %d", member.Login, companyID))
	member.switchCompany(t, companyID)
}

// newMember — новый пользователь, включённый в компанию с заданной ролью
// (сессия уже переключена на компанию).
func newMember(t *testing.T, creator *actor, companyID int64, roleID int64) *actor {
	t.Helper()
	m := newVerifiedUser(t)
	addToCompany(t, creator, companyID, m, roleID)
	return m
}

// newSuperAdmin — платформенный супер-админ (бутстрап напрямую в БД, как
// reset_superadmin_password.sh) + логин.
func newSuperAdmin(t *testing.T) *actor {
	t.Helper()
	a := &actor{
		FIO:      "Супер Админ Платформенный",
		Login:    uniq("root_"),
		Password: "super-secret-99",
	}
	a.Email = a.Login + "@apitest.local"
	err := db.QueryRow(dbCtx(t), `
		INSERT INTO users (fio, login, hash_password, email, is_default_pass,
		                   is_active, is_super_admin, email_verified, created_at)
		VALUES ($1, $2, crypt($3, gen_salt('bf')), $4, FALSE, TRUE, TRUE, TRUE, now())
		RETURNING id`, a.FIO, a.Login, a.Password, a.Email).Scan(&a.ID)
	if err != nil {
		t.Fatalf("бутстрап супер-админа: %v", err)
	}
	a.mustLogin(t)
	return a
}

// ── Хелперы ежедневников ─────────────────────────────────────────

// createDiary — новый ежедневник актора, возвращает id.
func createDiary(t *testing.T, a *actor, name string) int64 {
	t.Helper()
	r := diaryAPI.doJSON(t, http.MethodPost, "/api/diaries", a.Token, map[string]any{"name": name})
	requireStatus(t, r, 201, "создание ежедневника "+name)
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("создание ежедневника: нет id: %s", r.Raw)
	}
	return id
}

// createEntry — новая запись; body поверх обязательных полей.
func createEntry(t *testing.T, a *actor, diaryID int64, date, title string, extra map[string]any) int64 {
	t.Helper()
	body := map[string]any{"entry_date": date, "title": title}
	for k, v := range extra {
		body[k] = v
	}
	r := diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records", diaryID), a.Token, body)
	requireStatus(t, r, 201, "создание записи "+title)
	return int64(r.Num("id"))
}

// entryIDs — id записей из ответа списка {items: [...]} в порядке выдачи.
func entryIDs(r apiResp) []int64 {
	items := r.List("items")
	out := make([]int64, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		id, _ := m["id"].(float64)
		out = append(out, int64(id))
	}
	return out
}

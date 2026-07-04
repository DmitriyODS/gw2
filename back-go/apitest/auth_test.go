package apitest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
)

// ── Регистрация и подтверждение email ────────────────────────────

func TestAuthSuggestLogin(t *testing.T) {
	r := authAPI.doJSON(t, http.MethodGet, "/api/auth/suggest-login?fio=Осиповский+Дмитрий+Сергеевич", "", nil)
	requireStatus(t, r, 200, "suggest-login")
	if got := r.Str("login"); !strings.HasPrefix(got, "osipov.ds") {
		t.Fatalf("suggest-login: %q, ожидался префикс osipov.ds", got)
	}
	// Пустое ФИО — пустая подсказка, не ошибка.
	r = authAPI.doJSON(t, http.MethodGet, "/api/auth/suggest-login", "", nil)
	requireStatus(t, r, 200, "suggest-login без fio")
	if r.Str("login") != "" {
		t.Fatalf("suggest-login без fio: ожидалась пустая строка, получено %q", r.Str("login"))
	}
}

func TestAuthRegistrationFlow(t *testing.T) {
	login := uniq("reg_")
	email := login + "@apitest.local"
	pass := "long-password-1"

	// Регистрация: сессия не выдаётся, только verification_required.
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": "Регистрационный Тест", "login": login, "email": email, "password": pass,
	})
	requireStatus(t, r, 201, "register")
	if r.Str("status") != "verification_required" || r.Str("access_token") != "" {
		t.Fatalf("register: ожидался verification_required без сессии: %s", r.Raw)
	}

	// Повторная регистрация на тот же email → 409.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": "Дубль", "login": uniq("dup_"), "email": email, "password": pass,
	})
	requireError(t, r, 409, "EMAIL_TAKEN", "повторный register на занятый email")

	// Логин до подтверждения → 403 EMAIL_NOT_VERIFIED.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
		"login": login, "password": pass,
	})
	requireError(t, r, 403, "EMAIL_NOT_VERIFIED", "login до верификации")

	// Мгновенная переотправка — тихо ок (троттл 60с), код в БД не меняется.
	codeBefore, tokenBefore := verificationFor(t, email)
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/resend-verification", "", map[string]any{"email": email})
	requireStatus(t, r, 200, "resend-verification")
	codeAfter, tokenAfter := verificationFor(t, email)
	if codeBefore != codeAfter || tokenBefore != tokenAfter {
		t.Fatalf("resend в течение 60с не должен перевыпускать код")
	}

	// Неверный код → 400 INVALID_VERIFICATION.
	wrong := "000000"
	if wrong == codeAfter {
		wrong = "000001"
	}
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{
		"email": email, "code": wrong,
	})
	requireError(t, r, 400, "INVALID_VERIFICATION", "verify-email с неверным кодом")

	// Верный код → полноценная сессия + refresh-cookie.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{
		"email": email, "code": codeAfter,
	})
	requireStatus(t, r, 200, "verify-email")
	if r.Str("access_token") == "" || r.Cookie("refresh_token") == "" {
		t.Fatalf("verify-email: нет сессии или refresh-cookie: %s", r.Raw)
	}

	// Повторное подтверждение тем же кодом → запись удалена → 400.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{
		"email": email, "code": codeAfter,
	})
	requireError(t, r, 400, "INVALID_VERIFICATION", "повторный verify-email")

	// Теперь логин работает.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
		"login": login, "password": pass,
	})
	requireStatus(t, r, 200, "login после верификации")
}

func TestAuthVerifyEmailByTokenLink(t *testing.T) {
	login := uniq("tok_")
	email := login + "@apitest.local"
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": "Токеновый Тест", "login": login, "email": email, "password": "long-password-1",
	})
	requireStatus(t, r, 201, "register")

	// Мусорный токен → 400.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{"token": "garbage"})
	requireError(t, r, 400, "INVALID_VERIFICATION", "verify-email с мусорным токеном")

	_, token := verificationFor(t, email)
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{"token": token})
	requireStatus(t, r, 200, "verify-email по токену-ссылке")
	if r.Str("access_token") == "" {
		t.Fatalf("verify-email по токену: нет сессии: %s", r.Raw)
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
		code int
	}{
		{"без email", map[string]any{"fio": "А", "login": uniq("v_"), "password": "12345678"}, 400},
		{"кривой email", map[string]any{"fio": "А", "login": uniq("v_"), "email": "not-an-email", "password": "12345678"}, 400},
		{"короткий пароль", map[string]any{"fio": "А", "login": uniq("v_"), "email": uniq("v") + "@apitest.local", "password": "short"}, 400},
		{"короткий логин", map[string]any{"fio": "А", "login": "ab", "email": uniq("v") + "@apitest.local", "password": "12345678"}, 400},
		{"без тела", nil, 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", tc.body)
			requireStatus(t, r, tc.code, "register: "+tc.name)
		})
	}
	// Битый JSON → 400, не 500.
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", "{broken json")
	requireStatus(t, r, 400, "register с битым JSON")
}

// ── Логин / refresh / logout ─────────────────────────────────────

func TestAuthLoginRefreshLogout(t *testing.T) {
	u := newVerifiedUser(t)

	// Неверный пароль → 401 INVALID_CREDENTIALS.
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
		"login": u.Login, "password": "wrong-password",
	})
	requireError(t, r, 401, "INVALID_CREDENTIALS", "login с неверным паролем")

	// Несуществующий логин → 401 (не раскрываем наличие аккаунта).
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
		"login": uniq("ghost_"), "password": "whatever-123",
	})
	requireError(t, r, 401, "INVALID_CREDENTIALS", "login несуществующего")

	// Пустое тело → 400.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{})
	requireStatus(t, r, 400, "login с пустым телом")

	// Валидный логин: сессия + HttpOnly refresh-cookie.
	r = u.mustLogin(t)
	if int64(r.Num("user_id")) != u.ID || r.Bool("force_change") {
		t.Fatalf("login: неожиданные клеймы: %s", r.Raw)
	}
	var httpOnly bool
	for _, c := range r.Cookies {
		if c.Name == "refresh_token" {
			httpOnly = c.HttpOnly
		}
	}
	if !httpOnly {
		t.Fatalf("refresh-cookie должна быть HttpOnly")
	}

	// Refresh по cookie → новый access-токен.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/refresh", "", nil,
		withCookie("refresh_token", u.Refresh))
	requireStatus(t, r, 200, "refresh")
	if r.Str("access_token") == "" {
		t.Fatalf("refresh: нет access_token: %s", r.Raw)
	}

	// Refresh без cookie → 401; с мусорной cookie → 401.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/refresh", "", nil)
	requireError(t, r, 401, "INVALID_TOKEN", "refresh без cookie")
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/refresh", "", nil,
		withCookie("refresh_token", "v4.local.garbage"))
	requireError(t, r, 401, "INVALID_TOKEN", "refresh с мусорной cookie")

	// Access-токен в роли refresh-cookie → 401 (не тот тип токена).
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/refresh", "", nil,
		withCookie("refresh_token", u.Token))
	requireError(t, r, 401, "INVALID_TOKEN", "refresh access-токеном")

	// Logout гасит cookie.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/logout", u.Token, nil)
	requireStatus(t, r, 200, "logout")
	for _, c := range r.Cookies {
		if c.Name == "refresh_token" && c.Value != "" {
			t.Fatalf("logout: refresh-cookie не очищена")
		}
	}
}

func TestAuthTokenNegatives(t *testing.T) {
	// Без токена → 401.
	r := authAPI.doJSON(t, http.MethodGet, "/api/users/me", "", nil)
	requireError(t, r, 401, "UNAUTHORIZED", "me без токена")

	// Мусорный токен → 401.
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", "v4.public.garbage", nil)
	requireError(t, r, 401, "UNAUTHORIZED", "me с мусорным токеном")

	// Просроченный, но корректно подписанный dev-ключом токен → 401.
	u := newVerifiedUser(t)
	expired := expiredAccessToken(t, u.ID)
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", expired, nil)
	requireError(t, r, 401, "UNAUTHORIZED", "me с просроченным токеном")

	// Токен с испорченной подписью → 401.
	tampered := u.Token[:len(u.Token)-4] + "AAAA"
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", tampered, nil)
	requireError(t, r, 401, "UNAUTHORIZED", "me с испорченной подписью")

	// Diarysvc проверяет те же токены тем же публичным ключом.
	d := diaryAPI.doJSON(t, http.MethodGet, "/api/diaries", expired, nil)
	requireError(t, d, 401, "UNAUTHORIZED", "diaries с просроченным токеном")
}

// expiredAccessToken — просроченный access-токен, подписанный dev-ключом
// authsvc: проверяем exp-гейт, а не подпись.
func expiredAccessToken(t *testing.T, userID int64) string {
	t.Helper()
	secret, err := paseto.NewV4AsymmetricSecretKeyFromHex(pasetoPrivateKey)
	if err != nil {
		t.Fatalf("dev private key: %v", err)
	}
	tok := paseto.NewToken()
	past := time.Now().Add(-time.Hour)
	tok.SetIssuedAt(past)
	tok.SetNotBefore(past)
	tok.SetExpiration(past.Add(15 * time.Minute))
	tok.SetSubject(strconv.FormatInt(userID, 10))
	tok.SetString("type", "access")
	return tok.V4Sign(secret, nil)
}

func TestAuthVerificationExpiryAndAttempts(t *testing.T) {
	// Просроченный код → 400 VERIFICATION_EXPIRED.
	login := uniq("exp_")
	email := login + "@apitest.local"
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": "Просроченный", "login": login, "email": email, "password": "long-password-1",
	})
	requireStatus(t, r, 201, "register")
	if _, err := db.Exec(dbCtx(t), `
		UPDATE email_verifications SET expires_at = now() - interval '1 minute'
		 WHERE user_id = (SELECT id FROM users WHERE login = $1)`, login); err != nil {
		t.Fatalf("просрочка кода: %v", err)
	}
	code, token := verificationFor(t, email)
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{
		"email": email, "code": code,
	})
	requireError(t, r, 400, "VERIFICATION_EXPIRED", "verify просроченным кодом")
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{"token": token})
	requireError(t, r, 400, "VERIFICATION_EXPIRED", "verify просроченным токеном")

	// Лимит попыток кода: после 5 неверных даже верный код → 429.
	login2 := uniq("att_")
	email2 := login2 + "@apitest.local"
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": "Лимит Попыток", "login": login2, "email": email2, "password": "long-password-1",
	})
	requireStatus(t, r, 201, "register")
	realCode, _ := verificationFor(t, email2)
	wrong := "999999"
	if wrong == realCode {
		wrong = "999998"
	}
	for i := 0; i < 5; i++ {
		r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{
			"email": email2, "code": wrong,
		})
		requireError(t, r, 400, "INVALID_VERIFICATION", fmt.Sprintf("неверный код №%d", i+1))
	}
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/verify-email", "", map[string]any{
		"email": email2, "code": realCode,
	})
	requireError(t, r, 429, "TOO_MANY_ATTEMPTS", "верный код после 5 неудач")
}

func TestAuthResendAfterCooldown(t *testing.T) {
	login := uniq("cool_")
	email := login + "@apitest.local"
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": "Кулдаун", "login": login, "email": email, "password": "long-password-1",
	})
	requireStatus(t, r, 201, "register")
	_, tokenBefore := verificationFor(t, email)

	// Отматываем троттл в БД вместо ожидания 60 секунд.
	if _, err := db.Exec(dbCtx(t), `
		UPDATE email_verifications SET last_sent_at = now() - interval '2 minutes'
		 WHERE user_id = (SELECT id FROM users WHERE login = $1)`, login); err != nil {
		t.Fatalf("отмотка троттла: %v", err)
	}
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/resend-verification", "", map[string]any{"email": email})
	requireStatus(t, r, 200, "resend после кулдауна")
	_, tokenAfter := verificationFor(t, email)
	if tokenBefore == tokenAfter {
		t.Fatalf("resend после кулдауна должен перевыпустить код")
	}
}

func TestAuthResetTokenExpired(t *testing.T) {
	u := newVerifiedUser(t)
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/forgot-password", "", map[string]any{"email": u.Email})
	requireStatus(t, r, 200, "forgot-password")
	token := resetTokenFor(t, u.Email)
	if _, err := db.Exec(dbCtx(t),
		`UPDATE password_resets SET expires_at = now() - interval '1 minute' WHERE token = $1`, token); err != nil {
		t.Fatalf("просрочка токена: %v", err)
	}
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/reset-password", "",
		map[string]any{"token": token, "new_password": "whatever-pass-1"})
	requireError(t, r, 400, "INVALID_RESET", "reset просроченным токеном")
}

func TestAuthDeactivatedUserLogin(t *testing.T) {
	root := newSuperAdmin(t)
	u := newVerifiedUser(t)

	r := authAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/users/platform/%d", u.ID), root.Token, nil)
	requireStatus(t, r, 200, "деактивация пользователя")

	// Логин деактивированного — 401, без раскрытия причины.
	r = u.loginResp(t)
	requireError(t, r, 401, "INVALID_CREDENTIALS", "login деактивированного")

	// Его старый access-токен тоже перестаёт работать.
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", u.Token, nil)
	requireError(t, r, 401, "UNAUTHORIZED", "me деактивированного")
}

func TestAuthRefreshAfterMembershipLoss(t *testing.T) {
	creator := newVerifiedUser(t)
	companyID := creator.createCompany(t, "Потеря членства "+uniq("L"))

	worker := newVerifiedUser(t)
	r := authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/members", companyID),
		creator.Token, map[string]any{"user_id": worker.ID, "role_id": roleEmployee})
	requireStatus(t, r, 201, "добавление работника")

	// Логин работника: единственная компания автоактивна.
	r = worker.mustLogin(t)
	if int64(r.Num("company_id")) != companyID {
		t.Fatalf("ожидалась автоактивная компания %d: %s", companyID, r.Raw)
	}

	// Исключение из компании: refresh не падает, а даёт сессию без компании.
	r = authAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, worker.ID), creator.Token, nil)
	requireStatus(t, r, 200, "исключение работника")
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/refresh", "", nil,
		withCookie("refresh_token", worker.Refresh))
	requireStatus(t, r, 200, "refresh после исключения")
	if r.JSON["company_id"] != nil || r.Num("role_level") != 0 {
		t.Fatalf("refresh после исключения: ожидалась сессия без компании: %s", r.Raw)
	}
}

func TestAuthRoleGuards(t *testing.T) {
	creator := newVerifiedUser(t)
	companyID := creator.createCompany(t, "Гарды "+uniq("G"))

	// Свою роль менять нельзя.
	r := authAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/users/%d/role", creator.ID),
		creator.Token, map[string]any{"role_id": roleEmployee})
	requireError(t, r, 422, "SELF_ROLE_CHANGE", "смена собственной роли")

	// Роль не-участнику активной компании → 404.
	outsider := newVerifiedUser(t)
	r = authAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/users/%d/role", outsider.ID),
		creator.Token, map[string]any{"role_id": roleEmployee})
	requireError(t, r, 404, "NOT_FOUND", "роль не-участнику")

	// Несуществующая роль → 404 ROLE_NOT_FOUND.
	worker := newVerifiedUser(t)
	r = authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/members", companyID),
		creator.Token, map[string]any{"user_id": worker.ID, "role_id": 99})
	requireError(t, r, 404, "ROLE_NOT_FOUND", "несуществующая роль")

	// Настройки выходных: доступ только администратору компании.
	r = authAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/companies/%d/weekend-settings", companyID),
		creator.Token, map[string]any{"weekend_days": []int{5, 6}})
	requireStatus(t, r, 200, "weekend-settings создателем")
	r = authAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/companies/%d/weekend-settings", companyID),
		outsider.Token, map[string]any{"weekend_days": []int{5, 6}})
	requireError(t, r, 403, "FORBIDDEN", "weekend-settings посторонним")
	// Значение вне 0..6 → 400 VALIDATION_ERROR (формы marshmallow).
	r = authAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/companies/%d/weekend-settings", companyID),
		creator.Token, map[string]any{"weekend_days": []int{7}})
	requireError(t, r, 400, "VALIDATION_ERROR", "weekend-settings вне диапазона")
}

func TestAuthSuggestLoginCollision(t *testing.T) {
	fio := "Коллизионов Тест Иванович"
	r := authAPI.doJSON(t, http.MethodGet, "/api/auth/suggest-login?fio="+urlQuery(fio), "", nil)
	requireStatus(t, r, 200, "suggest до занятия")
	s1 := r.Str("login")

	// Занимаем предложенный логин — подсказка обязана смениться.
	rr := authAPI.doJSON(t, http.MethodPost, "/api/auth/register", "", map[string]any{
		"fio": fio, "login": s1, "email": uniq("col") + "@apitest.local", "password": "long-password-1",
	})
	requireStatus(t, rr, 201, "register с предложенным логином")

	r = authAPI.doJSON(t, http.MethodGet, "/api/auth/suggest-login?fio="+urlQuery(fio), "", nil)
	requireStatus(t, r, 200, "suggest после занятия")
	if r.Str("login") == s1 {
		t.Fatalf("после занятия %q подсказка должна смениться", s1)
	}
}

// ── Брутфорс-щит ─────────────────────────────────────────────────

func TestAuthBruteForceShield(t *testing.T) {
	u := newVerifiedUser(t)

	// 4 неудачи — ещё 401.
	for i := 0; i < 4; i++ {
		r := authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
			"login": u.Login, "password": "wrong-password",
		})
		requireError(t, r, 401, "INVALID_CREDENTIALS", fmt.Sprintf("неудача №%d", i+1))
	}
	// 5-я — блокировка с retry_after_sec.
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
		"login": u.Login, "password": "wrong-password",
	})
	requireError(t, r, 429, "TOO_MANY_ATTEMPTS", "5-я неудача")
	if r.Num("retry_after_sec") <= 0 {
		t.Fatalf("429 без retry_after_sec: %s", r.Raw)
	}

	// Пока блокировка активна — даже верный пароль получает 429.
	r = u.loginResp(t)
	requireError(t, r, 429, "TOO_MANY_ATTEMPTS", "верный пароль во время блокировки")
}

// ── Сброс пароля по email ────────────────────────────────────────

func TestAuthForgotResetPassword(t *testing.T) {
	u := newVerifiedUser(t)

	// Неизвестный email — всегда ok (не раскрываем аккаунт).
	r := authAPI.doJSON(t, http.MethodPost, "/api/auth/forgot-password", "",
		map[string]any{"email": uniq("nobody_") + "@apitest.local"})
	requireStatus(t, r, 200, "forgot-password незнакомого email")

	// Свой email → токен в password_resets.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/forgot-password", "",
		map[string]any{"email": u.Email})
	requireStatus(t, r, 200, "forgot-password")
	token := resetTokenFor(t, u.Email)

	// Короткий новый пароль → 400.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/reset-password", "",
		map[string]any{"token": token, "new_password": "short"})
	requireError(t, r, 400, "VALIDATION_ERROR", "reset-password с коротким паролем")

	// Мусорный токен → 400 INVALID_RESET.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/reset-password", "",
		map[string]any{"token": "deadbeef", "new_password": "new-password-9"})
	requireError(t, r, 400, "INVALID_RESET", "reset-password с мусорным токеном")

	// Смена по токену → 200 + login для префилла.
	newPass := "brand-new-pass-7"
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/reset-password", "",
		map[string]any{"token": token, "new_password": newPass})
	requireStatus(t, r, 200, "reset-password")
	if r.Str("login") != u.Login {
		t.Fatalf("reset-password: ожидался login %q, получен %q", u.Login, r.Str("login"))
	}

	// Старый пароль больше не работает, новый — работает.
	r = u.loginResp(t)
	requireError(t, r, 401, "INVALID_CREDENTIALS", "login со старым паролем")
	u.Password = newPass
	u.mustLogin(t)

	// Повторное использование того же токена → 400 (одноразовый).
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/reset-password", "",
		map[string]any{"token": token, "new_password": "yet-another-pass-1"})
	requireError(t, r, 400, "INVALID_RESET", "повторный reset тем же токеном")
}

// ── Компании: создание, login-gate, выбор ────────────────────────

func TestCompanyCreateAndLoginGate(t *testing.T) {
	u := newVerifiedUser(t)

	// Сессия без компаний: company_id null, роль 0.
	if r := u.mustLogin(t); r.JSON["company_id"] != nil || r.Num("role_level") != 0 {
		t.Fatalf("login без компаний: ожидалась сессия без company_id: %s", r.Raw)
	}

	// Создание компании: создатель становится администратором.
	c1 := u.createCompany(t, "Компания "+uniq("A"))
	me := authAPI.doJSON(t, http.MethodGet, "/api/users/me", u.Token, nil)
	requireStatus(t, me, 200, "me после создания компании")

	// Дубль имени → 409.
	name2 := "Компания " + uniq("B")
	r := authAPI.doJSON(t, http.MethodPost, "/api/companies", u.Token, map[string]any{"name": name2})
	requireStatus(t, r, 201, "вторая компания")
	c2 := int64(r.Num("id"))
	r = authAPI.doJSON(t, http.MethodPost, "/api/companies", u.Token, map[string]any{"name": name2})
	requireError(t, r, 409, "DUPLICATE", "компания с дублем имени")

	// Одна компания — автоактивна; две — login-gate.
	r = u.loginResp(t)
	requireStatus(t, r, 200, "login с двумя компаниями")
	if !r.Bool("needs_company_selection") || r.Str("select_token") == "" {
		t.Fatalf("ожидался login-gate: %s", r.Raw)
	}
	if len(r.List("companies")) != 2 {
		t.Fatalf("login-gate: ожидались 2 компании: %s", r.Raw)
	}
	selectToken := r.Str("select_token")

	// select-company чужой компании → 403 NOT_A_MEMBER.
	other := newVerifiedUser(t)
	cOther := other.createCompany(t, "Чужая "+uniq("C"))
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/select-company", "", map[string]any{
		"select_token": selectToken, "company_id": cOther,
	})
	requireError(t, r, 403, "NOT_A_MEMBER", "select-company чужой компании")

	// select-company своей → полноценная сессия с ролью администратора.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/select-company", "", map[string]any{
		"select_token": selectToken, "company_id": c1,
	})
	requireStatus(t, r, 200, "select-company")
	if int64(r.Num("company_id")) != c1 || r.Num("role_level") != roleAdmin {
		t.Fatalf("select-company: ожидалась компания %d с ролью 3: %s", c1, r.Raw)
	}
	u.applySession(t, r)

	// Мусорный select-токен → 401.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/select-company", "", map[string]any{
		"select_token": "garbage", "company_id": c1,
	})
	requireError(t, r, 401, "INVALID_TOKEN", "select-company с мусорным токеном")

	// switch-company на вторую свою → 200; на чужую → 403.
	u.switchCompany(t, c2)
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/switch-company", u.Token,
		map[string]any{"company_id": cOther})
	requireError(t, r, 403, "NOT_A_MEMBER", "switch-company в чужую компанию")

	// Refresh сохраняет активную компанию сессии.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/refresh", "", nil,
		withCookie("refresh_token", u.Refresh))
	requireStatus(t, r, 200, "refresh с активной компанией")
	if int64(r.Num("company_id")) != c2 {
		t.Fatalf("refresh: ожидалась активная компания %d: %s", c2, r.Raw)
	}
	_ = me
}

// ── Членство, роли, гарды ────────────────────────────────────────

func TestCompanyMembershipAndRoles(t *testing.T) {
	creator := newVerifiedUser(t)
	companyID := creator.createCompany(t, "Членства "+uniq("M"))

	// Ссылки-приглашения изначально нет; перевыпуск — только создатель.
	r := authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/invite", companyID), creator.Token, nil)
	requireStatus(t, r, 200, "перевыпуск инвайт-кода")
	code := r.Str("code")
	if code == "" {
		t.Fatalf("нет invite_code: %s", r.Raw)
	}

	// Вступление по коду: любой авторизованный становится Сотрудником.
	worker := newVerifiedUser(t)
	r = authAPI.doJSON(t, http.MethodPost, "/api/companies/join/"+code, worker.Token, nil)
	requireStatus(t, r, 200, "join по коду")
	if int64(r.Num("company_id")) != companyID || r.Num("role_level") != roleEmployee {
		t.Fatalf("join: ожидалась роль Сотрудник в компании %d: %s", companyID, r.Raw)
	}
	worker.applySession(t, r)

	// Мусорный код → 404.
	r = authAPI.doJSON(t, http.MethodPost, "/api/companies/join/nonexistent", worker.Token, nil)
	requireError(t, r, 404, "INVALID_INVITE", "join по мусорному коду")

	// Создатель повышает работника до Менеджера (адресный member-роут).
	r = authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, worker.ID),
		creator.Token, map[string]any{"role_id": roleManager})
	requireStatus(t, r, 200, "смена роли работника")

	// Обычный участник (после повышения до админа ниже — пока менеджер) не
	// управляет участниками: не создатель → 403.
	stranger := newVerifiedUser(t)
	strangerCo := stranger.createCompany(t, "Посторонняя "+uniq("S"))
	_ = strangerCo
	r = authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, worker.ID),
		stranger.Token, map[string]any{"role_id": roleEmployee})
	requireError(t, r, 403, "FORBIDDEN", "смена роли не-создателем")

	// Чужая карточка компании → 403 (и не 200 с данными).
	r = authAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/companies/%d", companyID), stranger.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "чужая компания")

	// Несуществующая компания → 404 (для супер-админа) / 403 постороннему.
	r = authAPI.doJSON(t, http.MethodGet, "/api/companies/99999999", stranger.Token, nil)
	if r.Status != 403 && r.Status != 404 {
		t.Fatalf("несуществующая компания: ожидался 403/404, получен %d: %s", r.Status, r.Raw)
	}

	// Повышаем работника до Администратора и проверяем «не-создатель-админ
	// участниками не управляет, но настройки видит».
	r = authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, worker.ID),
		creator.Token, map[string]any{"role_id": roleAdmin})
	requireStatus(t, r, 200, "повышение работника до администратора")

	worker.switchCompany(t, companyID) // перевыпуск токена с новой ролью
	r = authAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/companies/%d", companyID), worker.Token, nil)
	requireStatus(t, r, 200, "карточка компании для администратора-не-создателя")
	r = authAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/companies/%d/users", companyID), worker.Token,
		map[string]any{"fio": "Не должен создаться", "login": uniq("no_"), "role_id": roleEmployee})
	requireError(t, r, 403, "FORBIDDEN", "создание сотрудника не-создателем")

	// Защита последнего администратора: понизить работника можно (создатель
	// остаётся админом), а потом создателя — нельзя.
	r = authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, worker.ID),
		creator.Token, map[string]any{"role_id": roleEmployee})
	requireStatus(t, r, 200, "понижение работника")
	r = authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, creator.ID),
		creator.Token, map[string]any{"role_id": roleEmployee})
	requireError(t, r, 422, "LAST_ADMIN", "понижение последнего администратора")
	r = authAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, creator.ID), creator.Token, nil)
	requireError(t, r, 422, "LAST_ADMIN", "удаление последнего администратора")

	// Удаление обычного участника создателем.
	r = authAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/companies/%d/members/%d", companyID, worker.ID), creator.Token, nil)
	requireStatus(t, r, 200, "удаление участника")
	r = authAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/companies/%d/members", companyID), creator.Token, nil)
	requireStatus(t, r, 200, "список участников")
	if ids := directoryIDs(t, r); ids[worker.ID] || !ids[creator.ID] {
		t.Fatalf("после удаления работник не должен быть в списке участников: %s", r.Raw)
	}
}

func TestCompanyEmployeeCRUDAndForceChange(t *testing.T) {
	creator := newVerifiedUser(t)
	creator.createCompany(t, "Сотрудники "+uniq("E")) // активная компания для /api/users

	// Администратор активной компании заводит сотрудника без пароля →
	// дефолтный <login>123 и force_change при входе.
	empLogin := uniq("emp_")
	r := authAPI.doJSON(t, http.MethodPost, "/api/users", creator.Token, map[string]any{
		"fio": "Сотрудник Заведённый", "login": empLogin, "role_id": roleEmployee,
	})
	requireStatus(t, r, 201, "создание сотрудника")
	empID := int64(r.Num("id"))
	if !r.Bool("is_default_pass") {
		t.Fatalf("новый сотрудник должен иметь is_default_pass: %s", r.Raw)
	}

	// Роль выше своей назначить нельзя — проверяем на равной (можно) и через
	// не-админа (403 самим роутом RequireRole).
	emp := &actor{Login: empLogin, Password: empLogin + "123"}
	lr := emp.loginResp(t)
	requireStatus(t, lr, 200, "login сотрудника с дефолтным паролем")
	if !lr.Bool("force_change") {
		t.Fatalf("ожидался force_change: %s", lr.Raw)
	}
	emp.applySession(t, lr)

	// Любое API под force_change → 403 FORCE_PASSWORD_CHANGE.
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", emp.Token, nil)
	requireStatus(t, r, 403, "me под force_change")
	if !strings.Contains(string(r.Raw), "FORCE_PASSWORD_CHANGE") {
		t.Fatalf("ожидался FORCE_PASSWORD_CHANGE: %s", r.Raw)
	}

	// change-default: несовпадение паролей → 400; успех → сессия без force_change.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/change-default", emp.Token, map[string]any{
		"new_login": empLogin, "new_password": "fresh-pass-123", "confirm_password": "other",
	})
	requireError(t, r, 400, "PASSWORDS_MISMATCH", "change-default с несовпадением")
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/change-default", emp.Token, map[string]any{
		"new_login": empLogin, "new_password": "fresh-pass-123", "confirm_password": "fresh-pass-123",
	})
	requireStatus(t, r, 200, "change-default")
	if r.Bool("force_change") {
		t.Fatalf("после change-default force_change должен сняться: %s", r.Raw)
	}
	emp.Password = "fresh-pass-123"
	emp.applySession(t, r)

	// Повторный change-default → 422 ALREADY_CHANGED.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/change-default", emp.Token, map[string]any{
		"new_login": empLogin, "new_password": "fresh-pass-456", "confirm_password": "fresh-pass-456",
	})
	requireError(t, r, 422, "ALREADY_CHANGED", "повторный change-default")

	// Сотрудник (роль 1) не может ни заводить сотрудников, ни смотреть
	// админскую карточку пользователя.
	r = authAPI.doJSON(t, http.MethodPost, "/api/users", emp.Token, map[string]any{
		"fio": "Нельзя", "login": uniq("nope_"), "role_id": roleEmployee,
	})
	requireError(t, r, 403, "FORBIDDEN", "создание сотрудника сотрудником")
	r = authAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/users/%d", creator.ID), emp.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "карточка пользователя сотрудником")

	// Сброс пароля сотрудника администратором → снова force_change.
	r = authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/users/%d/reset-password", empID), creator.Token, nil)
	requireStatus(t, r, 200, "reset-password сотрудника")
	lr = authAPI.doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]any{
		"login": empLogin, "password": empLogin + "123",
	})
	requireStatus(t, lr, 200, "login после сброса")
	if !lr.Bool("force_change") {
		t.Fatalf("после сброса ожидался force_change: %s", lr.Raw)
	}

	// Самого себя исключить нельзя.
	r = authAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/users/%d", creator.ID), creator.Token, nil)
	requireError(t, r, 422, "SELF_HIDE", "исключение самого себя")
}

func TestCompanyEmailInvites(t *testing.T) {
	creator := newVerifiedUser(t)
	companyID := creator.createCompany(t, "Инвайты "+uniq("I"))

	invitee := newVerifiedUser(t)

	// Email-инвайт с ролью Менеджер (создатель).
	r := authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/invites", companyID),
		creator.Token, map[string]any{"email": invitee.Email, "role_id": roleManager})
	requireStatus(t, r, 201, "создание email-инвайта")

	token := inviteTokenFor(t, companyID, invitee.Email)

	// Превью без авторизации не бывает — роут в companiesAPI (RequireAuth).
	r = authAPI.doJSON(t, http.MethodGet, "/api/companies/invites/"+token, invitee.Token, nil)
	requireStatus(t, r, 200, "превью инвайта")
	if r.Str("role_name") == "" {
		t.Fatalf("превью без роли: %s", r.Raw)
	}

	// Принятие: членство с ролью из инвайта + сессия на компанию.
	r = authAPI.doJSON(t, http.MethodPost, "/api/companies/invites/"+token+"/accept", invitee.Token, nil)
	requireStatus(t, r, 200, "принятие инвайта")
	if int64(r.Num("company_id")) != companyID || r.Num("role_level") != roleManager {
		t.Fatalf("инвайт: ожидалась роль Менеджер в компании %d: %s", companyID, r.Raw)
	}

	// Токен погашен → повторное принятие 404.
	r = authAPI.doJSON(t, http.MethodPost, "/api/companies/invites/"+token+"/accept", invitee.Token, nil)
	requireError(t, r, 404, "INVALID_INVITE", "повторное принятие инвайта")

	// Инвайт может слать только создатель/супер-админ.
	r = authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/invites", companyID),
		invitee.Token, map[string]any{"email": uniq("x") + "@apitest.local", "role_id": roleEmployee})
	requireError(t, r, 403, "FORBIDDEN", "email-инвайт не-создателем")
}

// TestCompanyLastAdminBypass — гард «последнего администратора» обязан
// действовать на ВСЕХ путях смены роли, включая повторное добавление
// существующего участника и принятие email-инвайта (оба апсертят роль).
func TestCompanyLastAdminBypass(t *testing.T) {
	// Путь 1: POST /members по своему id с ролью Сотрудник.
	creator := newVerifiedUser(t)
	companyID := creator.createCompany(t, "Последний админ "+uniq("X"))
	r := authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/members", companyID),
		creator.Token, map[string]any{"user_id": creator.ID, "role_id": roleEmployee})
	requireError(t, r, 422, "LAST_ADMIN", "повторное добавление себя с ролью ниже")
	// Роль не должна была понизиться.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/switch-company", creator.Token,
		map[string]any{"company_id": companyID})
	requireStatus(t, r, 200, "switch после попытки понижения")
	if r.Num("role_level") != roleAdmin {
		t.Fatalf("роль создателя понижена в обход гарда: %s", r.Raw)
	}

	// Путь 2: email-инвайт на собственный адрес с ролью Сотрудник.
	creator2 := newVerifiedUser(t)
	companyID2 := creator2.createCompany(t, "Последний админ "+uniq("Y"))
	r = authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/invites", companyID2),
		creator2.Token, map[string]any{"email": creator2.Email, "role_id": roleEmployee})
	requireStatus(t, r, 201, "инвайт на собственный email")
	token := inviteTokenFor(t, companyID2, creator2.Email)
	r = authAPI.doJSON(t, http.MethodPost, "/api/companies/invites/"+token+"/accept", creator2.Token, nil)
	requireError(t, r, 422, "LAST_ADMIN", "принятие инвайта с понижением последнего админа")
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/switch-company", creator2.Token,
		map[string]any{"company_id": companyID2})
	requireStatus(t, r, 200, "switch после инвайта")
	if r.Num("role_level") != roleAdmin {
		t.Fatalf("роль создателя понижена инвайтом в обход гарда: %s", r.Raw)
	}
}

// ── Профиль (me) ─────────────────────────────────────────────────

func TestUsersUpdateMe(t *testing.T) {
	u := newVerifiedUser(t)

	// Телефон нормализуется к +7…, мусорный — 400.
	r := authAPI.doJSON(t, http.MethodPatch, "/api/users/me", u.Token,
		map[string]any{"phone": "8 (912) 345-67-89"})
	requireStatus(t, r, 200, "смена телефона")
	if r.Str("phone") != "+79123456789" {
		t.Fatalf("нормализация телефона: %s", r.Raw)
	}
	r = authAPI.doJSON(t, http.MethodPatch, "/api/users/me", u.Token,
		map[string]any{"phone": "12345"})
	requireError(t, r, 400, "VALIDATION_ERROR", "мусорный телефон")

	// Email занят другим → 409.
	other := newVerifiedUser(t)
	r = authAPI.doJSON(t, http.MethodPatch, "/api/users/me", u.Token,
		map[string]any{"email": strings.ToUpper(other.Email)})
	requireError(t, r, 409, "EMAIL_TAKEN", "занятый email (без учёта регистра)")

	// Смена пароля: неверный текущий → 400; верный → login новым паролем.
	r = authAPI.doJSON(t, http.MethodPatch, "/api/users/me", u.Token, map[string]any{
		"current_password": "wrong", "new_password": "changed-pass-1", "confirm_password": "changed-pass-1",
	})
	requireError(t, r, 400, "WRONG_PASSWORD", "смена пароля с неверным текущим")
	r = authAPI.doJSON(t, http.MethodPatch, "/api/users/me", u.Token, map[string]any{
		"current_password": u.Password, "new_password": "changed-pass-1", "confirm_password": "changed-pass-1",
	})
	requireStatus(t, r, 200, "смена пароля")
	u.Password = "changed-pass-1"
	u.mustLogin(t)

	// Идентикон — публичный PNG.
	r = authAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/users/%d/identicon", u.ID), "", nil)
	requireStatus(t, r, 200, "identicon")
	if ct := r.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("identicon: content-type %q", ct)
	}
}

// ── Каталог пользователей ────────────────────────────────────────

func TestUsersDirectory(t *testing.T) {
	creator := newVerifiedUser(t)
	companyID := creator.createCompany(t, "Каталог "+uniq("D"))

	colleague := newVerifiedUser(t)
	r := authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/members", companyID),
		creator.Token, map[string]any{"user_id": colleague.ID, "role_id": roleEmployee})
	requireStatus(t, r, 201, "добавление участника по id")

	outsider := newVerifiedUser(t)

	// Скоуп активной компании: коллега виден, посторонний — нет.
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/directory", creator.Token, nil)
	requireStatus(t, r, 200, "directory компании")
	ids := directoryIDs(t, r)
	if !ids[colleague.ID] || ids[outsider.ID] {
		t.Fatalf("directory компании: ожидался коллега %d без постороннего %d: %s",
			colleague.ID, outsider.ID, r.Raw)
	}

	// Глобальный каталог ?all=1 — виден и посторонний.
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/directory?all=1", creator.Token, nil)
	requireStatus(t, r, 200, "directory all=1")
	ids = directoryIDs(t, r)
	if !ids[outsider.ID] {
		t.Fatalf("directory all=1: посторонний %d должен быть виден", outsider.ID)
	}

	// exclude_self.
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/directory?exclude_self=1", creator.Token, nil)
	requireStatus(t, r, 200, "directory exclude_self")
	if directoryIDs(t, r)[creator.ID] {
		t.Fatalf("directory exclude_self: собственный профиль не должен возвращаться")
	}

	// Поиск по имени.
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/directory?all=1&q="+outsider.Login, creator.Token, nil)
	requireStatus(t, r, 200, "directory поиск")
	ids = directoryIDs(t, r)
	if !ids[outsider.ID] || len(ids) != 1 {
		t.Fatalf("directory поиск по %q: ожидался ровно один результат: %s", outsider.Login, r.Raw)
	}
}

// directoryIDs — множество id из ответа-массива каталога.
func directoryIDs(t *testing.T, r apiResp) map[int64]bool {
	t.Helper()
	var arr []map[string]any
	if err := jsonUnmarshal(r.Raw, &arr); err != nil {
		t.Fatalf("directory: ответ не массив: %s", r.Raw)
	}
	out := map[int64]bool{}
	for _, u := range arr {
		if id, ok := u["id"].(float64); ok {
			out[int64(id)] = true
		}
	}
	return out
}

// ── Платформенные роуты (супер-админ) ────────────────────────────

func TestSuperAdminPlatformScopes(t *testing.T) {
	root := newSuperAdmin(t)
	regular := newVerifiedUser(t)
	companyID := regular.createCompany(t, "Платформа "+uniq("P"))

	// Список всех пользователей платформы — только супер-админ.
	r := authAPI.doJSON(t, http.MethodGet, "/api/users", root.Token, nil)
	requireStatus(t, r, 200, "список пользователей платформы")
	r = authAPI.doJSON(t, http.MethodGet, "/api/users", regular.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "список пользователей не-админом")

	// Список всех компаний — только супер-админ.
	r = authAPI.doJSON(t, http.MethodGet, "/api/companies", root.Token, nil)
	requireStatus(t, r, 200, "список компаний платформы")
	r = authAPI.doJSON(t, http.MethodGet, "/api/companies", regular.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "список компаний не-админом")

	// Выключение компании: члены получают COMPANY_DISABLED.
	r = authAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/companies/%d/toggle-active", companyID),
		root.Token, map[string]any{"is_active": false})
	requireStatus(t, r, 200, "выключение компании")
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", regular.Token, nil)
	requireStatus(t, r, 403, "me при выключенной компании")
	if !strings.Contains(string(r.Raw), "COMPANY_DISABLED") {
		t.Fatalf("ожидался COMPANY_DISABLED: %s", r.Raw)
	}
	// switch-company в отключённую тоже закрыт.
	r = authAPI.doJSON(t, http.MethodPost, "/api/auth/switch-company", regular.Token,
		map[string]any{"company_id": companyID})
	requireError(t, r, 403, "COMPANY_DISABLED", "switch в отключённую компанию")

	// Включаем обратно — доступ восстановлен.
	r = authAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/companies/%d/toggle-active", companyID),
		root.Token, map[string]any{"is_active": true})
	requireStatus(t, r, 200, "включение компании")
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", regular.Token, nil)
	requireStatus(t, r, 200, "me после включения компании")

	// Супер-админа нельзя изменять через компанийные ручки.
	r = authAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/companies/%d/members", companyID),
		regular.Token, map[string]any{"user_id": root.ID, "role_id": roleAdmin})
	requireError(t, r, 422, "SUPER_ADMIN", "добавление супер-админа в компанию")
}

// ── Роли ─────────────────────────────────────────────────────────

func TestRolesList(t *testing.T) {
	u := newVerifiedUser(t)
	r := authAPI.doJSON(t, http.MethodGet, "/api/roles", u.Token, nil)
	requireStatus(t, r, 200, "список ролей")
	var roles []map[string]any
	if err := jsonUnmarshal(r.Raw, &roles); err != nil || len(roles) != 3 {
		t.Fatalf("ожидались ровно 3 роли: %s", r.Raw)
	}
}

package service

import (
	"context"
	"sync"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

// fakeSessions — реестр входов в памяти.
type fakeSessions struct {
	mu      sync.Mutex
	nextID  int64
	items   map[int64]*domain.Session
	revoked map[int64]bool
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{items: map[int64]*domain.Session{}, revoked: map[int64]bool{}}
}

func (f *fakeSessions) Create(_ context.Context, s *domain.Session) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	cp := *s
	cp.ID = f.nextID
	f.items[cp.ID] = &cp
	return cp.ID, nil
}

func (f *fakeSessions) Get(_ context.Context, id int64) (*domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revoked[id] {
		return nil, nil
	}
	return f.items[id], nil
}

func (f *fakeSessions) Touch(_ context.Context, id, userID int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.items[id]
	return s != nil && s.UserID == userID && !f.revoked[id], nil
}

func (f *fakeSessions) ListActive(_ context.Context, userID int64) ([]*domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.Session
	for id, s := range f.items {
		if s.UserID == userID && !f.revoked[id] {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSessions) Revoke(_ context.Context, id, userID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s := f.items[id]; s != nil && s.UserID == userID {
		f.revoked[id] = true
	}
	return nil
}

func (f *fakeSessions) RevokeOthers(_ context.Context, userID, keepID int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for id, s := range f.items {
		if s.UserID != userID || id == keepID || f.revoked[id] {
			continue
		}
		f.revoked[id] = true
		n++
	}
	return n, nil
}

func (f *fakeSessions) SetCity(_ context.Context, id int64, city string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s := f.items[id]; s != nil {
		s.City = city
	}
	return nil
}

// withSessions — сервис с включённым реестром входов (гео выключено: сеть в
// тестах не нужна).
func withSessions(t *testing.T) (*Service, *fakeRepo, *fakeSessions) {
	t.Helper()
	svc, repo, _ := newTestService(t)
	sessions := newFakeSessions()
	svc.WithSessions(sessions, nil)
	return svc, repo, sessions
}

const chromeUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

func reqCtx(ua, ip, refresh string) context.Context {
	return domain.WithSessionMeta(context.Background(), domain.SessionMeta{
		UserAgent: ua, IP: ip, Refresh: refresh,
	})
}

func loginAs(t *testing.T, svc *Service, ctx context.Context, login string) *dto.Session {
	t.Helper()
	sess, err := svc.Login(ctx, dto.LoginRequest{Login: login, Password: "secret123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	return sess
}

// Вход заводит карточку устройства, а её id уезжает в refresh-токен — иначе
// завершать сеанс было бы нечем.
func TestLoginCreatesSession(t *testing.T) {
	svc, repo, _ := withSessions(t)
	cid := int64(1)
	employee(repo, "ivanov", &cid)

	sess := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")

	_, _, sid, err := svc.tokens.ParseRefresh(sess.RefreshToken)
	if err != nil || sid == 0 {
		t.Fatalf("refresh без сессии: sid=%d, err=%v", sid, err)
	}

	items, err := svc.ListSessions(reqCtx(chromeUA, "8.8.8.8", sess.RefreshToken), sess.UserID)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("сеансов: %d, ожидался 1", len(items))
	}
	if !items[0].Current {
		t.Fatal("сеанс, из которого пришёл запрос, не помечен текущим")
	}
	if items[0].Title != "Веб-приложение" || items[0].Platform != domain.PlatformWeb {
		t.Fatalf("карточка веб-сеанса: %+v", items[0])
	}
}

// Смена активной компании — та же сессия: перевыпуск пары токенов не должен
// плодить карточки устройств.
func TestSwitchCompanyKeepsSession(t *testing.T) {
	svc, repo, _ := withSessions(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)

	// Вторая компания — уже после входа: иначе login ушёл бы в выбор компании
	// (login-gate) и refresh-токена на руках бы не было.
	first := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")
	_ = repo.AddMembership(context.Background(), u.ID, 2, 1)
	_, _, sid1, _ := svc.tokens.ParseRefresh(first.RefreshToken)

	second, err := svc.SwitchCompany(reqCtx(chromeUA, "8.8.8.8", first.RefreshToken), u.ID, 2)
	if err != nil {
		t.Fatalf("SwitchCompany: %v", err)
	}
	_, _, sid2, _ := svc.tokens.ParseRefresh(second.RefreshToken)
	if sid1 != sid2 {
		t.Fatalf("сессия сменилась: %d → %d", sid1, sid2)
	}

	items, _ := svc.ListSessions(reqCtx(chromeUA, "8.8.8.8", second.RefreshToken), u.ID)
	if len(items) != 1 {
		t.Fatalf("сеансов: %d, ожидался 1", len(items))
	}
}

// Вход с того же устройства под другим аккаунтом получает свою карточку:
// чужой refresh-cookie переиспользовать нельзя.
func TestLoginOtherUserStartsOwnSession(t *testing.T) {
	svc, repo, sessions := withSessions(t)
	cid := int64(1)
	employee(repo, "ivanov", &cid)
	petrov := employee(repo, "petrov", &cid)

	first := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")
	second := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", first.RefreshToken), "petrov")

	_, _, sid1, _ := svc.tokens.ParseRefresh(first.RefreshToken)
	_, _, sid2, _ := svc.tokens.ParseRefresh(second.RefreshToken)
	if sid1 == sid2 {
		t.Fatal("вход другого пользователя переиспользовал чужую сессию")
	}
	own, _ := sessions.ListActive(context.Background(), petrov.ID)
	if len(own) != 1 {
		t.Fatalf("сеансов у petrov: %d, ожидался 1", len(own))
	}
}

// Завершённый сеанс отвергает refresh — вот ради чего всё затевалось.
func TestRevokedSessionBlocksRefresh(t *testing.T) {
	svc, repo, _ := withSessions(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)

	sess := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")
	ctx := reqCtx(chromeUA, "8.8.8.8", sess.RefreshToken)

	if _, err := svc.Refresh(ctx, sess.RefreshToken); err != nil {
		t.Fatalf("живая сессия не обновилась: %v", err)
	}

	items, _ := svc.ListSessions(ctx, u.ID)
	if err := svc.RevokeSession(ctx, u.ID, items[0].ID); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	_, err := svc.Refresh(ctx, sess.RefreshToken)
	wantCode(t, err, "INVALID_TOKEN")

	if left, _ := svc.ListSessions(ctx, u.ID); len(left) != 0 {
		t.Fatalf("завершённый сеанс остался в списке: %d", len(left))
	}
}

// Чужой сеанс завершить нельзя: id из другого аккаунта не находится, и сеанс
// жертвы продолжает работать. Ответ при этом — отказ, а не «ок»: мнимый успех
// в списке устройств означал бы, что человек считает чужой вход отключённым.
func TestRevokeForeignSessionDoesNothing(t *testing.T) {
	svc, repo, _ := withSessions(t)
	cid := int64(1)
	ivanov := employee(repo, "ivanov", &cid)
	petrov := employee(repo, "petrov", &cid)

	victim := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")
	victimCtx := reqCtx(chromeUA, "8.8.8.8", victim.RefreshToken)
	items, _ := svc.ListSessions(victimCtx, ivanov.ID)

	err := svc.RevokeSession(context.Background(), petrov.ID, items[0].ID)
	wantCode(t, err, "NOT_FOUND")

	if _, err := svc.Refresh(victimCtx, victim.RefreshToken); err != nil {
		t.Fatalf("чужой отзыв убил сеанс: %v", err)
	}
}

/* Смена пароля обрывает ПРОЧИЕ входы: человек, меняющий пароль, рассчитывает,
   что чужие устройства перестали работать. Текущее — остаётся: выкидывать
   самого себя из настроек незачем. */
func TestPasswordChangeRevokesOtherSessions(t *testing.T) {
	svc, repo, _ := withSessions(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)

	phone := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")
	laptop := loginAs(t, svc, reqCtx(chromeUA, "9.9.9.9", ""), "ivanov")
	laptopCtx := reqCtx(chromeUA, "9.9.9.9", laptop.RefreshToken)

	newPass := "new-secret-123"
	confirm := newPass
	current := "secret123"
	if _, err := svc.UpdateMe(laptopCtx, u.ID, dto.UpdateMeRequest{
		CurrentPassword: &current, NewPassword: &newPass, ConfirmPassword: &confirm,
	}); err != nil {
		t.Fatalf("UpdateMe: %v", err)
	}

	// Чужое устройство больше не обновит токен...
	if _, err := svc.Refresh(reqCtx(chromeUA, "8.8.8.8", phone.RefreshToken), phone.RefreshToken); err == nil {
		t.Fatal("после смены пароля прежний вход обязан перестать работать")
	}
	// ...а то, с которого меняли, продолжает работать.
	if _, err := svc.Refresh(laptopCtx, laptop.RefreshToken); err != nil {
		t.Fatalf("текущий сеанс не должен обрываться: %v", err)
	}
}

// Refresh-токен, выпущенный до реестра входов, продолжает работать и получает
// свою карточку вместе с новым refresh (иначе вход «повис» бы без сессии).
func TestLegacyRefreshUpgradesToSession(t *testing.T) {
	svc, repo, _ := withSessions(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)

	legacy, err := svc.tokens.RefreshToken(u.ID, &cid, 0)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	ctx := reqCtx(chromeUA, "8.8.8.8", legacy)

	sess, err := svc.Refresh(ctx, legacy)
	if err != nil {
		t.Fatalf("легаси-refresh отвергнут: %v", err)
	}
	if sess.RefreshToken == "" {
		t.Fatal("легаси-refresh не перевыпущен — карточка сеанса не появится никогда")
	}
	_, _, sid, _ := svc.tokens.ParseRefresh(sess.RefreshToken)
	if sid == 0 {
		t.Fatal("перевыпущенный refresh снова без сессии")
	}
	if items, _ := svc.ListSessions(ctx, u.ID); len(items) != 1 {
		t.Fatalf("сеансов: %d, ожидался 1", len(items))
	}
}

// Выход из системы гасит именно свой сеанс, чужие не трогает.
func TestLogoutRevokesCurrentSession(t *testing.T) {
	svc, repo, _ := withSessions(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)

	phone := loginAs(t, svc, reqCtx("Mozilla/5.0 (Linux; Android 14; Pixel 8) GrooveWorkApp", "8.8.8.8", ""), "ivanov")
	web := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")

	webCtx := reqCtx(chromeUA, "8.8.8.8", web.RefreshToken)
	if err := svc.RevokeCurrentSession(webCtx, u.ID); err != nil {
		t.Fatalf("RevokeCurrentSession: %v", err)
	}

	items, _ := svc.ListSessions(webCtx, u.ID)
	if len(items) != 1 {
		t.Fatalf("сеансов: %d, ожидался 1 (телефон)", len(items))
	}
	if items[0].Title != "Pixel 8, приложение" || items[0].Platform != domain.PlatformMobile {
		t.Fatalf("карточка мобильного сеанса: %+v", items[0])
	}
	if _, err := svc.Refresh(webCtx, web.RefreshToken); err == nil {
		t.Fatal("refresh после выхода принят")
	}
	phoneCtx := reqCtx("", "", phone.RefreshToken)
	if _, err := svc.Refresh(phoneCtx, phone.RefreshToken); err != nil {
		t.Fatalf("выход на одном устройстве убил сеанс на другом: %v", err)
	}
}

// Реестр выключен (nil) — авторизация работает по-прежнему, без карточек.
func TestSessionsDisabledKeepsAuthWorking(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	employee(repo, "ivanov", &cid)

	sess := loginAs(t, svc, reqCtx(chromeUA, "8.8.8.8", ""), "ivanov")
	if _, err := svc.Refresh(context.Background(), sess.RefreshToken); err != nil {
		t.Fatalf("refresh без реестра: %v", err)
	}
	items, err := svc.ListSessions(context.Background(), sess.UserID)
	if err != nil || len(items) != 0 {
		t.Fatalf("ListSessions без реестра: %v, %d", err, len(items))
	}
}

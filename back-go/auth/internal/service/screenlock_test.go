package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

/* Экран блокировки: пин закрывает приложение, но сессию не рвёт. Проверяем
   правила, ради которых он и заведён: снять экран можно пином или паролем,
   чужой код не подходит, а выключить блокировку без секрета нельзя — иначе её
   отключил бы любой, кто подошёл к запертому экрану. */
func TestScreenLockPinAndPassword(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)
	ctx := context.Background()

	state, err := svc.SetScreenLock(ctx, u.ID, dto.ScreenLockRequest{Pin: "2468"})
	if err != nil {
		t.Fatalf("SetScreenLock: %v", err)
	}
	if !state.Enabled {
		t.Fatal("блокировка не включилась")
	}

	if err := svc.UnlockScreen(ctx, u.ID, "2468"); err != nil {
		t.Fatalf("верный пин не снял экран: %v", err)
	}
	// Пароль аккаунта — запасной путь: пин забывается чаще пароля.
	if err := svc.UnlockScreen(ctx, u.ID, "secret123"); err != nil {
		t.Fatalf("пароль не снял экран: %v", err)
	}
	wantCode(t, svc.UnlockScreen(ctx, u.ID, "1111"), "WRONG_PIN")
	wantCode(t, svc.UnlockScreen(ctx, u.ID, ""), "WRONG_PIN")
}

func TestScreenLockRejectsBadPin(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)
	ctx := context.Background()

	for _, pin := range []string{"123", "abcd", "12345678901"} {
		_, err := svc.SetScreenLock(ctx, u.ID, dto.ScreenLockRequest{Pin: pin})
		wantCode(t, err, "BAD_PIN")
	}
	// Включить блокировку без пина нельзя: снимать её было бы нечем.
	minutes := 5
	_, err := svc.SetScreenLock(ctx, u.ID, dto.ScreenLockRequest{AfterMin: &minutes})
	wantCode(t, err, "BAD_PIN")
}

func TestDisableScreenLockNeedsSecret(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)
	ctx := context.Background()

	if _, err := svc.SetScreenLock(ctx, u.ID, dto.ScreenLockRequest{Pin: "2468"}); err != nil {
		t.Fatalf("SetScreenLock: %v", err)
	}
	wantCode(t, svc.DisableScreenLock(ctx, u.ID, "0000"), "WRONG_PIN")

	if err := svc.DisableScreenLock(ctx, u.ID, "2468"); err != nil {
		t.Fatalf("DisableScreenLock: %v", err)
	}
	state, err := svc.ScreenLockState(ctx, u.ID)
	if err != nil || state.Enabled {
		t.Fatalf("блокировка не снялась: %+v (%v)", state, err)
	}
}

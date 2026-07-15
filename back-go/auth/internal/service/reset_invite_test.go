package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/token"
)

// svcWithExtras — сервис с доступом к фейкам стораджей сброса/инвайтов и компаний.
func svcWithExtras(t *testing.T) (*Service, *fakeRepo, *fakeCompanies, *fakePasswordResets, *fakeCompanyInvites) {
	t.Helper()
	iss, err := token.NewIssuer(testPrivateHex, testRefreshHex, 15*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	repo := newFakeRepo()
	companies := newFakeCompanies()
	resets := newFakePasswordResets()
	invites := newFakeCompanyInvites()
	svc := New(repo, companies, fakeBackup{}, newFakeThrottle(), iss, &fakeAvatars{},
		newFakeVerifications(), resets, invites, newFakeDeviceLinks(), fakeMail{}, "http://x", slog.Default())
	return svc, repo, companies, resets, invites
}

func TestPasswordResetFlow(t *testing.T) {
	svc, repo, _, resets, _ := svcWithExtras(t)
	ctx := context.Background()
	email := "user@example.com"
	u := repo.add(&domain.User{FIO: "Иван Петров", Login: "ivan.p", Email: &email})

	if err := svc.RequestPasswordReset(ctx, email); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}
	r, _ := resets.GetByUserID(ctx, u.ID)
	if r == nil {
		t.Fatal("токен сброса не создан")
	}

	res, err := svc.ResetPasswordByToken(ctx, dto.ResetPasswordRequest{Token: r.Token, NewPassword: "newpass12"})
	if err != nil {
		t.Fatalf("ResetPasswordByToken: %v", err)
	}
	if res.Login != "ivan.p" {
		t.Errorf("ожидался логин ivan.p, получено %q", res.Login)
	}
	if r2, _ := resets.GetByUserID(ctx, u.ID); r2 != nil {
		t.Error("токен не погашен после сброса")
	}
	uu, _ := repo.GetByID(ctx, u.ID)
	if uu.HashPassword != "hash:newpass12" {
		t.Errorf("пароль не обновлён: %q", uu.HashPassword)
	}
}

func TestPasswordResetUnknownEmailSilent(t *testing.T) {
	svc, _, _, resets, _ := svcWithExtras(t)
	if err := svc.RequestPasswordReset(context.Background(), "nobody@example.com"); err != nil {
		t.Fatalf("на несуществующий email должно быть тихо ok, получено %v", err)
	}
	if len(resets.m) != 0 {
		t.Error("для несуществующего email токен не должен создаваться")
	}
}

func TestCompanyInviteAcceptFlow(t *testing.T) {
	svc, repo, _, _, invites := svcWithExtras(t)
	ctx := context.Background()
	creator := repo.add(&domain.User{FIO: "Создатель", Login: "creator"})
	company, err := svc.CreateCompany(ctx, creator, dto.CompanyCreate{Name: "Acme"})
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	// Приглашаем на роль Менеджера (id=2).
	if err := svc.CreateCompanyInvite(ctx, creator, company.ID, "bob@example.com", 2); err != nil {
		t.Fatalf("CreateCompanyInvite: %v", err)
	}
	var tok string
	for k := range invites.byToken {
		tok = k
	}
	if tok == "" {
		t.Fatal("инвайт не создан")
	}

	bob := repo.add(&domain.User{FIO: "Боб", Login: "bob"})
	sess, err := svc.AcceptCompanyInvite(ctx, bob.ID, tok)
	if err != nil {
		t.Fatalf("AcceptCompanyInvite: %v", err)
	}
	if sess.CompanyID == nil || *sess.CompanyID != company.ID || sess.RoleLevel != domain.LevelManager {
		t.Fatalf("сессия не переключена на компанию с ролью менеджера: %+v", sess)
	}
	if m, _ := repo.GetMembership(ctx, bob.ID, company.ID); m == nil || m.Role.Level != domain.LevelManager {
		t.Fatalf("членство с ролью менеджера не создано: %+v", m)
	}
	if len(invites.byToken) != 0 {
		t.Error("инвайт не погашен после принятия")
	}
}

func TestCompanyInviteNonCreatorForbidden(t *testing.T) {
	svc, repo, _, _, _ := svcWithExtras(t)
	ctx := context.Background()
	creator := repo.add(&domain.User{FIO: "Создатель", Login: "creator"})
	company, _ := svc.CreateCompany(ctx, creator, dto.CompanyCreate{Name: "Acme"})

	// Не-создатель (даже администратор) не может приглашать.
	other := repo.add(&domain.User{FIO: "Другой", Login: "other"})
	_ = repo.AddMembership(ctx, other.ID, company.ID, 3)
	wantCode(t, svc.CreateCompanyInvite(ctx, other, company.ID, "x@e.com", 1), "FORBIDDEN")
}

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

func TestPersonalCompanyName(t *testing.T) {
	cases := map[string]string{
		"Иванов Иван Иванович": "Иванов Иван — личное",
		"Иванов Иван":          "Иванов Иван — личное",
		"Иванов":               "Иванов — личное",
		"   ":                  "Моя компания — личное",
	}
	for fio, want := range cases {
		if got := PersonalCompanyName(fio); got != want {
			t.Errorf("PersonalCompanyName(%q) = %q, ждали %q", fio, got, want)
		}
	}
}

/* Вход человека без компаний заводит ему личную и сразу делает её активной:
   к компании привязана вся работа, и без неё половина разделов отвечает
   «нужна активная компания». */
func TestLoginCreatesPersonalCompany(t *testing.T) {
	svc, repo, _ := newTestService(t)
	employee(repo, "odinokiy", nil) // членств нет вовсе

	sess, err := svc.Login(context.Background(), dto.LoginRequest{
		Login: "odinokiy", Password: "secret123",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if sess.CompanyID == nil {
		t.Fatal("после входа не появилось активной компании")
	}
	// Имя проверяем у самой компании: фейк членств не связывает их с записью
	// компании, и в клеймах сессии оно осталось бы пустым.
	company, err := svc.companies.GetCompany(context.Background(), *sess.CompanyID)
	if err != nil || company == nil {
		t.Fatalf("созданная компания не найдена: %v", err)
	}
	if !strings.HasSuffix(company.Name, "— личное") {
		t.Fatalf("компания названа не как личная: %q", company.Name)
	}
}

// Второй вход не плодит компании: она заводится, только когда нет ни одной.
func TestPersonalCompanyCreatedOnce(t *testing.T) {
	svc, repo, _ := newTestService(t)
	employee(repo, "odinokiy", nil)
	ctx := context.Background()

	first, err := svc.Login(ctx, dto.LoginRequest{Login: "odinokiy", Password: "secret123"})
	if err != nil {
		t.Fatalf("первый вход: %v", err)
	}
	second, err := svc.Login(ctx, dto.LoginRequest{Login: "odinokiy", Password: "secret123"})
	if err != nil {
		t.Fatalf("второй вход: %v", err)
	}
	if first.CompanyID == nil || second.CompanyID == nil || *first.CompanyID != *second.CompanyID {
		t.Fatalf("компания сменилась между входами: %v → %v", first.CompanyID, second.CompanyID)
	}
}

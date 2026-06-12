package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

func companyService(t *testing.T) (*Service, *fakeRepo, *fakeCompanies) {
	t.Helper()
	svc, repo, _ := newTestService(t)
	companies := newFakeCompanies()
	svc.companies = companies
	return svc, repo, companies
}

func TestCreateCompanyDuplicateName(t *testing.T) {
	svc, _, companies := companyService(t)
	companies.CreateCompany(context.Background(), &domain.Company{Name: "Рога и копыта"})

	_, err := svc.CreateCompany(context.Background(), dto.CompanyCreate{
		Name: "Рога и копыта", IsActive: true,
	})
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "DUPLICATE" || de.HTTPStatus != 409 {
		t.Fatalf("ожидался DUPLICATE 409, получено %v", err)
	}
}

func TestCreateCompanyBindsDirectorWithoutCompany(t *testing.T) {
	svc, repo, companies := companyService(t)
	director := employee(repo, "boss", nil)

	out, err := svc.CreateCompany(context.Background(), dto.CompanyCreate{
		Name: "Новая", DirectorID: &director.ID, IsActive: true,
		Settings: map[string]any{"uses_calls": false},
	})
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	if director.CompanyID == nil || *director.CompanyID != out.ID {
		t.Fatal("директор без компании должен быть привязан к созданной")
	}
	// merge с DEFAULT_SETTINGS: переданный ключ поверх, остальные — дефолты.
	c := companies.companies[out.ID]
	if c.Settings["uses_calls"] != false || c.Settings["uses_yougile"] != false {
		t.Fatalf("settings = %v", c.Settings)
	}
}

func TestCreateCompanyDirectorNotFound(t *testing.T) {
	svc, _, _ := companyService(t)
	missing := int64(999)
	_, err := svc.CreateCompany(context.Background(), dto.CompanyCreate{
		Name: "X", DirectorID: &missing, IsActive: true,
	})
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "DIRECTOR_NOT_FOUND" || de.HTTPStatus != 404 {
		t.Fatalf("ожидался DIRECTOR_NOT_FOUND 404, получено %v", err)
	}
}

func TestWeekendSettingsAccess(t *testing.T) {
	svc, repo, companies := companyService(t)
	c := &domain.Company{Name: "А"}
	companies.CreateCompany(context.Background(), c)
	otherID := c.ID + 100
	stranger := employee(repo, "dir", &otherID)
	stranger.Role = domain.Role{ID: 3, Name: "Руководитель", Level: domain.LevelDirector}

	_, err := svc.GetWeekendSettings(context.Background(), stranger, c.ID)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "FORBIDDEN" {
		t.Fatalf("чужая компания: ожидался FORBIDDEN, получено %v", err)
	}

	insider := employee(repo, "dir2", &c.ID)
	got, err := svc.GetWeekendSettings(context.Background(), insider, c.ID)
	if err != nil {
		t.Fatalf("GetWeekendSettings: %v", err)
	}
	if len(got.WeekendDays) != 2 || got.WeekendDays[0] != 5 || got.WeekendDays[1] != 6 {
		t.Fatalf("дефолт выходных = %v", got.WeekendDays)
	}
}

func TestUpdateWeekendSettingsSortsAndDedupes(t *testing.T) {
	svc, repo, companies := companyService(t)
	c := &domain.Company{Name: "А", Settings: domain.DefaultCompanySettings()}
	companies.CreateCompany(context.Background(), c)
	admin := employee(repo, "root", nil)
	admin.IsRootAdmin = true

	got, err := svc.UpdateWeekendSettings(context.Background(), admin, c.ID, []int{6, 4, 6, 5})
	if err != nil {
		t.Fatalf("UpdateWeekendSettings: %v", err)
	}
	if len(got.WeekendDays) != 3 || got.WeekendDays[0] != 4 || got.WeekendDays[2] != 6 {
		t.Fatalf("дни = %v", got.WeekendDays)
	}
	// Снова читаем — настройки сохранены поверх остальных ключей.
	again, _ := svc.GetWeekendSettings(context.Background(), admin, c.ID)
	if len(again.WeekendDays) != 3 {
		t.Fatalf("после сохранения = %v", again.WeekendDays)
	}
	if companies.companies[c.ID].Settings["uses_calls"] != true {
		t.Fatal("остальные settings не должны теряться")
	}
}

func TestWeekendDaysGarbageFallsBack(t *testing.T) {
	if got := weekendDays(map[string]any{"weekend_days": "мусор"}); len(got) != 2 {
		t.Fatalf("мусор → дефолт, получено %v", got)
	}
	// Элементы вне 0..6 отфильтровываются (не дефолт!).
	if got := weekendDays(map[string]any{"weekend_days": []any{float64(9), float64(0)}}); len(got) != 1 || got[0] != 0 {
		t.Fatalf("фильтрация диапазона: %v", got)
	}
	// Пустой список — валидный ответ [] (компания без выходных «наоборот»).
	if got := weekendDays(map[string]any{"weekend_days": []any{}}); len(got) != 0 {
		t.Fatalf("пустой список → [], получено %v", got)
	}
}

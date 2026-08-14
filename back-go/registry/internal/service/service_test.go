package service

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

const (
	ownerID   = 42 // владелец тестового реестра
	companyID = 7  // компания, в которой он заведён
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRepo — in-memory реализация порта для тестов бизнес-логики.
type fakeRepo struct {
	reg        *domain.Registry
	share      *domain.Share
	fields     []domain.Field
	records    map[int64]*domain.Record
	userShares []*domain.UserShare
	issues     map[int64]*domain.Issue
	lastSearch string
	lastFilter domain.RecordListFilter
	lastExport domain.ExportFilter
	lastDelete domain.ExportFilter
	nextID     int64
}

func (f *fakeRepo) ListRegistries(_ domain.Ctx, _ int64, _ []int64, _ string) ([]*domain.Registry, error) {
	return []*domain.Registry{f.reg}, nil
}
func (f *fakeRepo) GetRegistry(_ domain.Ctx, id int64) (*domain.Registry, error) {
	if f.reg != nil && f.reg.ID == id {
		return f.reg, nil
	}
	return nil, nil
}

// AccessOf — владелец получает всё, остальные — по своим шарам.
func (f *fakeRepo) AccessOf(_ domain.Ctx, registryID, userID int64, companyIDs []int64) (string, error) {
	if f.reg == nil || f.reg.ID != registryID {
		return domain.AccessNone, nil
	}
	if f.reg.OwnerID == userID {
		return domain.AccessOwner, nil
	}
	best := domain.AccessNone
	for _, sh := range f.userShares {
		if sh.RegistryID != registryID {
			continue
		}
		if (sh.UserID != nil && *sh.UserID == userID) ||
			(sh.CompanyID != nil && slices.Contains(companyIDs, *sh.CompanyID)) {
			best = domain.BestAccess(best, sh.Access)
		}
	}
	return best, nil
}

func (f *fakeRepo) Audience(_ domain.Ctx, _ int64) ([]int64, error) {
	return []int64{ownerID}, nil
}
func (f *fakeRepo) ListUserShares(_ domain.Ctx, _ int64) ([]*domain.UserShare, error) {
	return f.userShares, nil
}
func (f *fakeRepo) PutUserShare(_ domain.Ctx, s *domain.UserShare) error {
	f.userShares = append(f.userShares, s)
	return nil
}
func (f *fakeRepo) DeleteUserShare(_ domain.Ctx, _ int64, _, _ *int64) error { return nil }

func (f *fakeRepo) CreateRegistry(_ domain.Ctx, r *domain.Registry) error { r.ID = 1; return nil }
func (f *fakeRepo) UpdateRegistry(_ domain.Ctx, _ int64, name string, _ int, sectionFieldID *int64, accounting bool) error {
	f.reg.Name = name
	f.reg.SectionFieldID = sectionFieldID
	f.reg.Accounting = accounting
	return nil
}
func (f *fakeRepo) DeleteRegistry(_ domain.Ctx, _ int64) error               { return nil }
func (f *fakeRepo) NextRegistryPosition(_ domain.Ctx, _ int64) (int, error)  { return 1, nil }
func (f *fakeRepo) ListFields(_ domain.Ctx, _ int64) ([]domain.Field, error) { return f.fields, nil }
func (f *fakeRepo) FieldsByRegistries(_ domain.Ctx, _ []int64) (map[int64][]domain.Field, error) {
	return map[int64][]domain.Field{f.reg.ID: f.fields}, nil
}
func (f *fakeRepo) ReplaceFields(_ domain.Ctx, _ int64, fields []domain.Field) ([]int64, error) {
	f.fields = fields
	return []int64{99}, nil // имитируем удаление поля 99
}
func (f *fakeRepo) ListRecords(_ domain.Ctx, filter domain.RecordListFilter) ([]*domain.Record, int, error) {
	f.lastFilter = filter
	return nil, 0, nil
}
func (f *fakeRepo) SearchRecords(_ domain.Ctx, _ int64, _ []int64, _ string, _ int) ([]*domain.SearchHit, error) {
	return nil, nil
}
func (f *fakeRepo) GetRecord(_ domain.Ctx, id int64) (*domain.Record, error) {
	return f.records[id], nil
}
func (f *fakeRepo) CreateRecord(_ domain.Ctx, r *domain.Record, searchText string) error {
	f.nextID++
	r.ID = f.nextID
	f.lastSearch = searchText
	if f.records == nil {
		f.records = map[int64]*domain.Record{}
	}
	f.records[r.ID] = r
	return nil
}
func (f *fakeRepo) UpdateRecord(_ domain.Ctx, id int64, data map[string]any, searchText string) error {
	f.lastSearch = searchText
	if r := f.records[id]; r != nil {
		r.Data = data
	}
	return nil
}
func (f *fakeRepo) DeleteRecord(_ domain.Ctx, _ int64) error { return nil }
func (f *fakeRepo) DeleteRecords(_ domain.Ctx, filter domain.ExportFilter) ([]*domain.Record, error) {
	f.lastDelete = filter
	out := []*domain.Record{}
	if len(filter.IDs) > 0 {
		for _, id := range filter.IDs {
			if r := f.records[id]; r != nil {
				out = append(out, r)
			}
		}
		return out, nil
	}
	excluded := map[int64]bool{}
	for _, id := range filter.Exclude {
		excluded[id] = true
	}
	all, _ := f.AllRecords(nil, 0)
	for _, r := range all {
		if !excluded[r.ID] {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeRepo) RecordsForExport(_ domain.Ctx, filter domain.ExportFilter) ([]*domain.Record, error) {
	f.lastExport = filter
	return f.AllRecords(nil, 0)
}
func (f *fakeRepo) CreateShare(_ domain.Ctx, s *domain.Share) error {
	s.ID = 1
	f.share = s
	return nil
}
func (f *fakeRepo) ListShares(_ domain.Ctx, _ int64) ([]*domain.Share, error) { return nil, nil }
func (f *fakeRepo) GetShareByCode(_ domain.Ctx, code string) (*domain.Share, error) {
	if f.share != nil && f.share.Code == code {
		return f.share, nil
	}
	return nil, nil
}
func (f *fakeRepo) UpdateShare(_ domain.Ctx, _, _ int64, _, _ string, _ bool) error { return nil }
func (f *fakeRepo) DeleteShare(_ domain.Ctx, _, _ int64) error                      { return nil }
func (f *fakeRepo) LogVisit(_ domain.Ctx, _ *domain.ShareVisit) error               { return nil }
func (f *fakeRepo) ListVisits(_ domain.Ctx, _ int64, _ int) ([]*domain.ShareVisit, error) {
	return nil, nil
}
func (f *fakeRepo) AllRecords(_ domain.Ctx, _ int64) ([]*domain.Record, error) {
	out := []*domain.Record{}
	for _, r := range f.records {
		out = append(out, r)
	}
	return out, nil
}

// ── Учётные выдачи ──

func (f *fakeRepo) OpenIssues(_ domain.Ctx, recordIDs []int64) (map[int64]*domain.Issue, error) {
	out := map[int64]*domain.Issue{}
	for _, id := range recordIDs {
		if issue := f.issues[id]; issue != nil && issue.ReturnedAt == nil {
			out[id] = issue
		}
	}
	return out, nil
}
func (f *fakeRepo) IssueHistory(_ domain.Ctx, recordID int64) ([]*domain.Issue, error) {
	if issue := f.issues[recordID]; issue != nil {
		return []*domain.Issue{issue}, nil
	}
	return nil, nil
}
func (f *fakeRepo) CreateIssue(_ domain.Ctx, i *domain.Issue, _ string) error {
	if f.issues == nil {
		f.issues = map[int64]*domain.Issue{}
	}
	if open := f.issues[i.RecordID]; open != nil && open.ReturnedAt == nil {
		return domain.ErrAlreadyIssued
	}
	i.ID = i.RecordID
	i.IssuedAt = time.Now()
	f.issues[i.RecordID] = i
	return nil
}
func (f *fakeRepo) ExtendIssue(_ domain.Ctx, issueID int64, dueAt *time.Time, _ string, _ *int64) error {
	if issue := f.issues[issueID]; issue != nil {
		issue.DueAt = dueAt
	}
	return nil
}
func (f *fakeRepo) ReturnIssue(_ domain.Ctx, issueID int64, at time.Time, _ string, _ *int64) (bool, error) {
	issue := f.issues[issueID]
	if issue == nil || issue.ReturnedAt != nil {
		return false, nil
	}
	issue.ReturnedAt = &at
	return true, nil
}

// Раздел «Хранилище»: записи отбираются по компании их реестра.
func (f *fakeRepo) RecordsOfCompanies(_ domain.Ctx, companyIDs []int64) ([]*domain.RecordScope, error) {
	out := []*domain.RecordScope{}
	if f.reg == nil || f.reg.CompanyID == nil || !slices.Contains(companyIDs, *f.reg.CompanyID) {
		return out, nil
	}
	for _, r := range f.records {
		out = append(out, &domain.RecordScope{
			Record: r, RegistryID: f.reg.ID, RegistryName: f.reg.Name, CompanyID: *f.reg.CompanyID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Record.ID < out[j].Record.ID })
	return out, nil
}

func (f *fakeRepo) CountOwned(_ domain.Ctx, _ int64) (int, error) {
	if f.reg == nil {
		return 0, nil
	}
	return 1, nil
}

type fakeBus struct{ events []string }

func (b *fakeBus) Publish(_ domain.Ctx, event string, _ []string, _ any) {
	b.events = append(b.events, event)
}

type fakeFiles struct{ removed []string }

func (f *fakeFiles) SaveFor(_ context.Context, _, _ int64, _ string, _ []byte) (string, error) {
	return "registry/x", nil
}

func (f *fakeFiles) SaveStreamFor(_ context.Context, _, _ int64, _ string, r io.Reader, _ int64) (string, error) {
	_, _ = io.Copy(io.Discard, r)
	return "registry/stream", nil
}

func (f *fakeFiles) RemoveFor(_ context.Context, _, _ int64, paths []string) {
	f.removed = append(f.removed, paths...)
}

func (f *fakeFiles) Remove(paths []string) {
	f.removed = append(f.removed, paths...)
}

// fakeUsers — идентичность: владелец состоит в компании companyID.
type fakeUsers struct{ companies map[int64][]int64 }

func (u *fakeUsers) GetUser(_ domain.Ctx, id int64) (*domain.User, error) {
	return &domain.User{ID: id, IsActive: true}, nil
}
func (u *fakeUsers) CompanyActive(_ domain.Ctx, _ *int64) (bool, error) { return true, nil }
func (u *fakeUsers) CompaniesOf(_ domain.Ctx, userID int64) ([]int64, error) {
	if u.companies == nil {
		return []int64{}, nil
	}
	return u.companies[userID], nil
}
func (u *fakeUsers) CompanyMembers(_ domain.Ctx, _ int64) ([]int64, error) { return nil, nil }
func (u *fakeUsers) SearchDirectory(_ domain.Ctx, _ []int64, _ string, _ int) ([]*domain.User, error) {
	return nil, nil
}
func (u *fakeUsers) CompanyName(_ domain.Ctx, _ int64) (string, error) {
	return "Компания", nil
}

func newTestService(fields []domain.Field) (*Service, *fakeRepo, *fakeBus) {
	company := int64(companyID)
	repo := &fakeRepo{
		reg:    &domain.Registry{ID: 1, OwnerID: ownerID, CompanyID: &company, Name: "Тест"},
		fields: fields,
	}
	bus := &fakeBus{}
	users := &fakeUsers{companies: map[int64][]int64{ownerID: {companyID}}}
	svc := New(Deps{Repo: repo, Users: users, Files: &fakeFiles{}, Bus: bus, Log: discardLogger()})
	return svc, repo, bus
}

func TestCreateRecord_BuildsSearchTextAndValidates(t *testing.T) {
	fields := []domain.Field{
		{ID: 10, Label: "Имя", Type: domain.FieldText},
		{ID: 11, Label: "Код", Type: domain.FieldNumber, Config: map[string]any{"pattern": `^\d{3}$`}},
		{ID: 12, Label: "Категория", Type: domain.FieldSelect, Config: map[string]any{"options": []any{"A", "B"}}},
		{ID: 13, Label: "Обложка", Type: domain.FieldImage},
	}
	svc, repo, bus := newTestService(fields)

	rec, err := svc.CreateRecord(context.Background(), ownerID, 1, map[string]any{
		"10": "Привет",
		"11": "123",
		"12": "A",
		"13": map[string]any{"path": "registry/x.png"},
		"99": "мусор-неизвестное-поле",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := rec.Data["99"]; ok {
		t.Error("неизвестное поле не должно сохраняться")
	}
	// search_text включает текст/число/select, но не картинку.
	want := "Привет 123 A"
	if repo.lastSearch != want {
		t.Errorf("search_text = %q, want %q", repo.lastSearch, want)
	}
	if len(bus.events) != 1 || bus.events[0] != "record:created" {
		t.Errorf("ожидалось событие record:created, получено %v", bus.events)
	}
}

func TestCreateRecord_NumberPatternRejected(t *testing.T) {
	fields := []domain.Field{
		{ID: 11, Label: "Код", Type: domain.FieldNumber, Config: map[string]any{"pattern": `^\d{3}$`}},
	}
	svc, _, _ := newTestService(fields)
	_, err := svc.CreateRecord(context.Background(), ownerID, 1, map[string]any{"11": "abc"})
	if err == nil {
		t.Fatal("ожидалась ошибка валидации по маске числа")
	}
	if de := domain.AsDomainError(err); de == nil || de.HTTPStatus != 400 {
		t.Errorf("ожидалась VALIDATION 400, получено %v", err)
	}
}

func TestCreateRecord_SelectOptionRejected(t *testing.T) {
	fields := []domain.Field{
		{ID: 12, Label: "Категория", Type: domain.FieldSelect, Config: map[string]any{"options": []any{"A", "B"}}},
	}
	svc, _, _ := newTestService(fields)
	_, err := svc.CreateRecord(context.Background(), ownerID, 1, map[string]any{"12": "Z"})
	if err == nil {
		t.Fatal("ожидалась ошибка: вариант вне options")
	}
}

func TestReplaceFields_StripsRemovedFieldData(t *testing.T) {
	fields := []domain.Field{{ID: 10, Label: "Имя", Type: domain.FieldText}}
	svc, repo, _ := newTestService(fields)
	repo.records = map[int64]*domain.Record{
		5: {ID: 5, RegistryID: 1, Data: map[string]any{"10": "Аня", "99": "удалится"}},
	}

	_, err := svc.ReplaceFields(context.Background(), ownerID, 1, []domain.Field{
		{ID: 10, Label: "Имя", Type: domain.FieldText},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := repo.records[5].Data["99"]; ok {
		t.Error("данные удалённого поля 99 должны быть вычищены из записи")
	}
	if repo.records[5].Data["10"] != "Аня" {
		t.Error("данные оставшегося поля должны сохраниться")
	}
}

// Клиенту, которому нужны все записи (печать QR-кодов), большая страница
// ЗАЖИМАЕТСЯ до потолка, а не сбрасывается в дефолт: иначе он молча получал 30
// первых записей и не находил остальные.
func TestListRecords_PerPageClampedNotReset(t *testing.T) {
	svc, repo, _ := newTestService(nil)
	cases := []struct {
		asked, want int
	}{
		{asked: 500, want: maxPerPage},
		{asked: maxPerPage, want: maxPerPage},
		{asked: 50, want: 50},
		{asked: 0, want: 30},
		{asked: -5, want: 30},
	}
	for _, c := range cases {
		if _, err := svc.ListRecords(context.Background(), ownerID, 1, RecordListParams{PerPage: c.asked}); err != nil {
			t.Fatalf("ListRecords(per_page=%d): %v", c.asked, err)
		}
		if repo.lastFilter.PerPage != c.want {
			t.Errorf("per_page=%d → в репозиторий ушло %d, ожидалось %d",
				c.asked, repo.lastFilter.PerPage, c.want)
		}
	}
}

func TestSectionFieldOnlySelect(t *testing.T) {
	fields := []domain.Field{
		{ID: 10, Label: "Имя", Type: domain.FieldText},
		{ID: 12, Label: "Тип изделия", Type: domain.FieldSelect, Config: map[string]any{"options": []any{"A", "B"}}},
	}
	svc, repo, _ := newTestService(fields)
	ctx := context.Background()
	id := func(v int64) *int64 { return &v }

	// Подразделами становится только списковое поле своего реестра.
	for _, bad := range []*int64{id(10), id(777)} {
		if _, err := svc.UpdateRegistry(ctx, ownerID, 1, RegistryPatch{
			Name: "Тест", SectionFieldID: bad, SectionFieldSet: true,
		}); err != domain.ErrSectionFieldInvalid {
			t.Fatalf("поле %d подразделами быть не должно, получено %v", *bad, err)
		}
	}

	reg, err := svc.UpdateRegistry(ctx, ownerID, 1, RegistryPatch{
		Name: "Тест", SectionFieldID: id(12), SectionFieldSet: true,
	})
	if err != nil || reg.SectionFieldID == nil || *reg.SectionFieldID != 12 {
		t.Fatalf("списковое поле должно стать источником подразделов: %v %+v", err, reg)
	}

	// Переименование без ключа section_field_id настройку не сбрасывает.
	reg, err = svc.UpdateRegistry(ctx, ownerID, 1, RegistryPatch{Name: "Другое имя"})
	if err != nil || reg.SectionFieldID == nil || *reg.SectionFieldID != 12 {
		t.Fatalf("переименование сбросило подразделы: %v %+v", err, reg)
	}

	// Вкладка уходит в репозиторий как условие по этому полю.
	if _, err := svc.ListRecords(ctx, ownerID, 1, RecordListParams{Section: "A"}); err != nil {
		t.Fatalf("ListRecords с подразделом: %v", err)
	}
	if repo.lastFilter.SectionFieldID != 12 || repo.lastFilter.SectionValue != "A" {
		t.Errorf("фильтр подраздела не доехал: %+v", repo.lastFilter)
	}

	// Выгрузка идёт тем же фильтром — файл не должен расходиться с экраном.
	if _, _, err := svc.ExportRecords(ctx, ownerID, 1, ExportParams{
		FieldIDs:  []int64{12},
		Selection: BulkParams{All: true, Section: "A"},
	}); err != nil {
		t.Fatalf("выгрузка с подразделом: %v", err)
	}
	if repo.lastExport.SectionFieldID != 12 || repo.lastExport.SectionValue != "A" {
		t.Errorf("подраздел не доехал до выгрузки: %+v", repo.lastExport)
	}

	// Подразделы выключили — вкладка больше не фильтрует.
	if _, err := svc.UpdateRegistry(ctx, ownerID, 1, RegistryPatch{Name: "Тест", SectionFieldSet: true}); err != nil {
		t.Fatalf("выключение подразделов: %v", err)
	}
	if _, err := svc.ListRecords(ctx, ownerID, 1, RecordListParams{Section: "A"}); err != nil {
		t.Fatalf("ListRecords без подразделов: %v", err)
	}
	if repo.lastFilter.SectionFieldID != 0 {
		t.Errorf("после выключения подразделов фильтр остался: %+v", repo.lastFilter)
	}
}

// Посторонний реестра не видит вовсе, а его существование не раскрывается:
// на чтение отвечаем «не найден», а не «нет прав».
func TestAccess_StrangerSeesNothing(t *testing.T) {
	svc, _, _ := newTestService(nil)
	const stranger = 999
	if _, err := svc.GetRegistry(context.Background(), stranger, 1); err != domain.ErrRegistryNotFound {
		t.Errorf("ожидалась ErrRegistryNotFound для постороннего, получено %v", err)
	}
}

// Уровни доступа вложены друг в друга: смотрящий не пишет, пишущий не правит
// структуру, а удалить реестр может только владелец.
func TestAccess_LevelsAreNested(t *testing.T) {
	ctx := context.Background()
	const guest = 1000

	cases := []struct {
		access                       string
		canRead, canWrite, canStruct bool
	}{
		{domain.AccessView, true, false, false},
		{domain.AccessEdit, true, true, false},
		{domain.AccessAdmin, true, true, true},
	}
	for _, c := range cases {
		t.Run(c.access, func(t *testing.T) {
			svc, repo, _ := newTestService([]domain.Field{{ID: 10, Label: "Имя", Type: domain.FieldText}})
			user := int64(guest)
			repo.userShares = []*domain.UserShare{{RegistryID: 1, UserID: &user, Access: c.access}}

			_, err := svc.GetRegistry(ctx, guest, 1)
			if (err == nil) != c.canRead {
				t.Errorf("чтение при %s: %v", c.access, err)
			}
			_, err = svc.CreateRecord(ctx, guest, 1, map[string]any{"10": "Х"})
			if (err == nil) != c.canWrite {
				t.Errorf("запись при %s: %v", c.access, err)
			}
			_, err = svc.ReplaceFields(ctx, guest, 1, []domain.Field{{ID: 10, Label: "Имя", Type: domain.FieldText}})
			if (err == nil) != c.canStruct {
				t.Errorf("правка структуры при %s: %v", c.access, err)
			}
			// Удаление реестра — только владельцу, даже администратору нельзя.
			if err := svc.DeleteRegistry(ctx, guest, 1); err != domain.ErrOwnerOnly {
				t.Errorf("удаление при %s должно требовать владельца, получено %v", c.access, err)
			}
		})
	}
}

// Доступ приходит человеку несколькими путями сразу — берётся сильнейший.
func TestAccess_BestOfPersonalAndCompany(t *testing.T) {
	svc, repo, _ := newTestService(nil)
	const guest = 1001
	user, company := int64(guest), int64(companyID)
	repo.userShares = []*domain.UserShare{
		{RegistryID: 1, CompanyID: &company, Access: domain.AccessView},
		{RegistryID: 1, UserID: &user, Access: domain.AccessEdit},
	}
	svc.users.(*fakeUsers).companies[guest] = []int64{companyID}

	reg, err := svc.GetRegistry(context.Background(), guest, 1)
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	if reg.MyAccess != domain.AccessEdit {
		t.Errorf("личная шара сильнее компанийной: получено %q", reg.MyAccess)
	}
}

// Учётный реестр: позицию нельзя выдать дважды, а после возврата — можно снова.
func TestAccounting_IssueReturnCycle(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newTestService([]domain.Field{{ID: 10, Label: "Имя", Type: domain.FieldText}})
	repo.reg.Accounting = true
	repo.records = map[int64]*domain.Record{5: {ID: 5, RegistryID: 1, Data: map[string]any{}}}

	due := time.Now().Add(48 * time.Hour)
	if _, err := svc.Issue(ctx, ownerID, 1, 5, IssueParams{HolderName: "Иванов", DueAt: &due}); err != nil {
		t.Fatalf("выдача: %v", err)
	}
	if _, err := svc.Issue(ctx, ownerID, 1, 5, IssueParams{HolderName: "Петров"}); err != domain.ErrAlreadyIssued {
		t.Errorf("повторная выдача должна отбиваться, получено %v", err)
	}
	if _, err := svc.Return(ctx, ownerID, 1, 5, ""); err != nil {
		t.Fatalf("возврат: %v", err)
	}
	if _, err := svc.Return(ctx, ownerID, 1, 5, ""); err != domain.ErrNotIssued {
		t.Errorf("повторный возврат должен отбиваться, получено %v", err)
	}
	if _, err := svc.Issue(ctx, ownerID, 1, 5, IssueParams{HolderName: "Петров"}); err != nil {
		t.Errorf("после возврата позицию можно выдать снова: %v", err)
	}
}

// Состояние позиции считает сервер: просрочка не должна зависеть от часов
// клиента.
func TestIssueState(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	past := now.Add(-72 * time.Hour)
	future := now.Add(24 * time.Hour)

	cases := []struct {
		name  string
		issue *domain.Issue
		want  string
		days  int
	}{
		{"на месте", nil, domain.StockIn, 0},
		{"выдана", &domain.Issue{DueAt: &future}, domain.StockIssued, 0},
		{"без срока", &domain.Issue{}, domain.StockNoDue, 0},
		{"просрочена", &domain.Issue{DueAt: &past}, domain.StockOverdue, 4},
		{"вернули", &domain.Issue{DueAt: &past, ReturnedAt: &now}, domain.StockIn, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.issue.State(now); got != c.want {
				t.Errorf("State = %q, want %q", got, c.want)
			}
			if got := c.issue.OverdueDays(now); got != c.days {
				t.Errorf("OverdueDays = %d, want %d", got, c.days)
			}
		})
	}
}

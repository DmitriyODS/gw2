package service

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/spaces"
)

const (
	authorID   = 42 // автор тестового календаря
	companyID  = 7  // команда-пространство
	adminID    = 43 // администратор команды
	memberID   = 44 // рядовой участник
	strangerID = 99
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRepo — in-memory реализация порта для тестов бизнес-логики.
type fakeRepo struct {
	cal        *domain.Calendar
	fields     []domain.Field
	entries    map[int64]*domain.Entry
	lastSearch string
	nextID     int64
	users      *fakeUsers
}

// AccessOf — зеркало accessExpr: хозяин личного календаря и автор/админ
// календаря команды — владельцы, участникам — team_access.
func (f *fakeRepo) AccessOf(ctx domain.Ctx, calendarID, userID int64) (string, error) {
	if f.cal == nil || f.cal.ID != calendarID {
		return domain.AccessNone, nil
	}
	if f.cal.CompanyID == nil {
		if f.cal.OwnerID == userID {
			return domain.AccessOwner, nil
		}
		return domain.AccessNone, nil
	}
	role, _ := f.users.TeamRole(ctx, userID, *f.cal.CompanyID)
	switch {
	case role.Admin, role.Member && f.cal.OwnerID == userID:
		return domain.AccessOwner, nil
	case role.Member:
		return f.cal.TeamAccess, nil
	}
	return domain.AccessNone, nil
}

func (f *fakeRepo) Audience(_ domain.Ctx, _ int64) ([]int64, error) { return []int64{authorID}, nil }
func (f *fakeRepo) CountOwned(_ domain.Ctx, _ int64) (int, error)   { return 1, nil }
func (f *fakeRepo) MoveCalendar(_ domain.Ctx, _, ownerID int64, companyID *int64, teamAccess string) error {
	f.cal.OwnerID, f.cal.CompanyID, f.cal.TeamAccess = ownerID, companyID, teamAccess
	return nil
}

// fakeUsers — кто в какой команде и кто ею управляет.
type fakeUsers struct {
	members map[int64][]int64
	admins  map[int64][]int64
}

func (u *fakeUsers) GetUser(_ domain.Ctx, id int64) (*domain.User, error) {
	return &domain.User{ID: id, IsActive: true}, nil
}
func (u *fakeUsers) CompanyActive(_ domain.Ctx, _ *int64) (bool, error) { return true, nil }
func (u *fakeUsers) TeamRole(_ domain.Ctx, userID, companyID int64) (spaces.Role, error) {
	return spaces.Role{
		Member: slices.Contains(u.members[userID], companyID),
		Admin:  slices.Contains(u.admins[userID], companyID),
	}, nil
}

func (f *fakeRepo) ListCalendars(_ domain.Ctx, _ int64) ([]*domain.Calendar, error) {
	return []*domain.Calendar{f.cal}, nil
}
func (f *fakeRepo) GetCalendar(_ domain.Ctx, id int64) (*domain.Calendar, error) {
	if f.cal != nil && f.cal.ID == id {
		return f.cal, nil
	}
	return nil, nil
}
func (f *fakeRepo) CreateCalendar(_ domain.Ctx, c *domain.Calendar) error       { c.ID = 1; return nil }
func (f *fakeRepo) UpdateCalendar(_ domain.Ctx, _ int64, _ string, _ int) error { return nil }
func (f *fakeRepo) DeleteCalendar(_ domain.Ctx, _ int64) error                  { return nil }
func (f *fakeRepo) NextCalendarPosition(_ domain.Ctx, _ int64) (int, error)     { return 1, nil }
func (f *fakeRepo) ListFields(_ domain.Ctx, _ int64) ([]domain.Field, error)    { return f.fields, nil }
func (f *fakeRepo) FieldsByCalendars(_ domain.Ctx, _ []int64) (map[int64][]domain.Field, error) {
	return map[int64][]domain.Field{f.cal.ID: f.fields}, nil
}
func (f *fakeRepo) ReplaceFields(_ domain.Ctx, _ int64, fields []domain.Field) ([]int64, error) {
	f.fields = fields
	return []int64{99}, nil // имитируем удаление поля 99
}
func (f *fakeRepo) ListEntries(_ domain.Ctx, _ domain.EntryListFilter) ([]*domain.Entry, error) {
	return nil, nil
}
func (f *fakeRepo) GetEntry(_ domain.Ctx, id int64) (*domain.Entry, error) {
	return f.entries[id], nil
}
func (f *fakeRepo) CreateEntry(_ domain.Ctx, e *domain.Entry, searchText string) error {
	f.nextID++
	e.ID = f.nextID
	f.lastSearch = searchText
	if f.entries == nil {
		f.entries = map[int64]*domain.Entry{}
	}
	f.entries[e.ID] = e
	return nil
}
func (f *fakeRepo) UpdateEntry(_ domain.Ctx, id int64, _ any, data map[string]any, searchText string) error {
	f.lastSearch = searchText
	if e := f.entries[id]; e != nil {
		e.Data = data
	}
	return nil
}
func (f *fakeRepo) DeleteEntry(_ domain.Ctx, _ int64) error { return nil }
func (f *fakeRepo) DeleteEntries(_ domain.Ctx, _ int64, ids []int64) (int64, error) {
	return int64(len(ids)), nil
}
func (f *fakeRepo) EntriesForExport(_ domain.Ctx, _ domain.EntryListFilter, _ []int64) ([]*domain.Entry, error) {
	return f.AllEntries(nil, 0)
}
func (f *fakeRepo) CreateShare(_ domain.Ctx, s *domain.Share) error              { s.ID = 1; return nil }
func (f *fakeRepo) ListShares(_ domain.Ctx, _ int64) ([]*domain.Share, error)    { return nil, nil }
func (f *fakeRepo) GetShareByCode(_ domain.Ctx, _ string) (*domain.Share, error) { return nil, nil }
func (f *fakeRepo) DeleteShare(_ domain.Ctx, _, _ int64) error                   { return nil }
func (f *fakeRepo) AllEntries(_ domain.Ctx, _ int64) ([]*domain.Entry, error) {
	out := []*domain.Entry{}
	for _, e := range f.entries {
		out = append(out, e)
	}
	return out, nil
}

func (f *fakeRepo) AgendaEntries(ctx domain.Ctx, userID int64, from, to time.Time, limit int) ([]domain.AgendaRow, int, error) {
	out := []domain.AgendaRow{}
	total := 0
	cal := f.cal
	access, _ := f.AccessOf(ctx, cal.ID, userID)
	for _, e := range f.entries {
		if access == domain.AccessNone || cal.ID != e.CalendarID ||
			e.EventAt.Before(from) || !e.EventAt.Before(to) {
			continue
		}
		total++
		if len(out) >= limit {
			continue
		}
		out = append(out, domain.AgendaRow{
			CalendarID: e.CalendarID, CalendarName: cal.Name, EntryID: e.ID,
			EventAt: e.EventAt, Data: e.Data,
		})
	}
	return out, total, nil
}

type fakeBus struct{ events []string }

func (b *fakeBus) Publish(_ domain.Ctx, event string, _ []string, _ any) {
	b.events = append(b.events, event)
}

type fakeFiles struct{ removed, moved []string }

func (f *fakeFiles) SaveFor(_ context.Context, _, _ int64, _ string, _ []byte) (string, error) {
	return "calendar/x", nil
}

func (f *fakeFiles) SaveStreamFor(_ context.Context, _, _ int64, _ string, r io.Reader, _ int64) (string, error) {
	// Поток вычитываем целиком: незакрытая часть осталась бы висеть.
	_, _ = io.Copy(io.Discard, r)
	return "calendar/x", nil
}

func (f *fakeFiles) RemoveFor(_ context.Context, _, _ int64, paths []string) {
	f.removed = append(f.removed, paths...)
}

func (f *fakeFiles) MoveFor(_ context.Context, _, _ int64, paths []string) error {
	f.moved = append(f.moved, paths...)
	return nil
}

func (f *fakeFiles) Remove(paths []string) {
	f.removed = append(f.removed, paths...)
}

// Раздел «Хранилище»: календарь команды — по списку команд, личный — по хозяину.
func (f *fakeRepo) EntriesForQuota(_ domain.Ctx, userID int64, companyIDs []int64) ([]*domain.EntryScope, error) {
	out := []*domain.EntryScope{}
	if f.cal == nil {
		return out, nil
	}
	var company int64
	if f.cal.CompanyID != nil {
		company = *f.cal.CompanyID
	}
	if !(company != 0 && slices.Contains(companyIDs, company)) && !(company == 0 && f.cal.OwnerID == userID) {
		return out, nil
	}
	for _, e := range f.entries {
		out = append(out, &domain.EntryScope{
			Entry: e, CalendarID: f.cal.ID, CalendarName: f.cal.Name, CompanyID: company,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Entry.ID < out[j].Entry.ID })
	return out, nil
}

func newTestService(fields []domain.Field) (*Service, *fakeRepo, *fakeBus) {
	company := int64(companyID)
	users := &fakeUsers{
		members: map[int64][]int64{authorID: {companyID}, adminID: {companyID}, memberID: {companyID}},
		admins:  map[int64][]int64{adminID: {companyID}},
	}
	repo := &fakeRepo{
		cal: &domain.Calendar{ID: 1, OwnerID: authorID, CompanyID: &company,
			TeamAccess: domain.AccessEdit, Name: "Тест"},
		fields: fields,
		users:  users,
	}
	bus := &fakeBus{}
	svc := New(Deps{Repo: repo, Users: users, Files: &fakeFiles{}, Bus: bus, Log: discardLogger()})
	return svc, repo, bus
}

func TestCreateEntry_BuildsSearchTextAndValidates(t *testing.T) {
	fields := []domain.Field{
		{ID: 10, Label: "Имя", Type: domain.FieldText},
		{ID: 11, Label: "Код", Type: domain.FieldNumber, Config: map[string]any{"pattern": `^\d{3}$`}},
		{ID: 12, Label: "Категория", Type: domain.FieldSelect, Config: map[string]any{"options": []any{"A", "B"}}},
		{ID: 13, Label: "Фото", Type: domain.FieldImage},
	}
	svc, repo, bus := newTestService(fields)

	at := time.Date(2026, 6, 23, 10, 30, 45, 0, time.UTC)
	e, err := svc.CreateEntry(context.Background(), authorID, 1, at, map[string]any{
		"10": "Привет",
		"11": "123",
		"12": "A",
		"13": map[string]any{"path": "calendar/x.png"},
		"99": "мусор-неизвестное-поле",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := e.Data["99"]; ok {
		t.Error("неизвестное поле не должно сохраняться")
	}
	// Секунды у event_at срезаются.
	if e.EventAt.Second() != 0 {
		t.Errorf("event_at должен быть без секунд, получено %v", e.EventAt)
	}
	want := "Привет 123 A"
	if repo.lastSearch != want {
		t.Errorf("search_text = %q, want %q", repo.lastSearch, want)
	}
	if len(bus.events) != 1 || bus.events[0] != "entry:created" {
		t.Errorf("ожидалось событие entry:created, получено %v", bus.events)
	}
}

func TestCreateEntry_RequiresEventAt(t *testing.T) {
	svc, _, _ := newTestService(nil)
	_, err := svc.CreateEntry(context.Background(), authorID, 1, time.Time{}, map[string]any{})
	if err != domain.ErrEventAtRequired {
		t.Errorf("ожидалась ErrEventAtRequired, получено %v", err)
	}
}

func TestCreateEntry_NumberPatternRejected(t *testing.T) {
	fields := []domain.Field{
		{ID: 11, Label: "Код", Type: domain.FieldNumber, Config: map[string]any{"pattern": `^\d{3}$`}},
	}
	svc, _, _ := newTestService(fields)
	_, err := svc.CreateEntry(context.Background(), authorID, 1, time.Now(), map[string]any{"11": "abc"})
	if err == nil {
		t.Fatal("ожидалась ошибка валидации по маске числа")
	}
	if de := domain.AsDomainError(err); de == nil || de.HTTPStatus != 400 {
		t.Errorf("ожидалась VALIDATION 400, получено %v", err)
	}
}

func TestReplaceFields_StripsRemovedFieldData(t *testing.T) {
	fields := []domain.Field{{ID: 10, Label: "Имя", Type: domain.FieldText}}
	svc, repo, _ := newTestService(fields)
	repo.entries = map[int64]*domain.Entry{
		5: {ID: 5, CalendarID: 1, Data: map[string]any{"10": "Аня", "99": "удалится"}},
	}

	_, err := svc.ReplaceFields(context.Background(), authorID, 1, []domain.Field{
		{ID: 10, Label: "Имя", Type: domain.FieldText},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := repo.entries[5].Data["99"]; ok {
		t.Error("данные удалённого поля 99 должны быть вычищены из записи")
	}
	if repo.entries[5].Data["10"] != "Аня" {
		t.Error("данные оставшегося поля должны сохраниться")
	}
}

// Календарь команды: участники ведут записи, структуру и удаление решают
// автор и администраторы, посторонний календаря не видит вовсе.
func TestCalendarTeamAccess(t *testing.T) {
	svc, _, _ := newTestService(nil)
	ctx := context.Background()
	if _, err := svc.GetCalendar(ctx, strangerID, 1); err != domain.ErrCalendarNotFound {
		t.Errorf("посторонний не должен видеть календарь команды, получено %v", err)
	}
	if _, err := svc.CreateEntry(ctx, memberID, 1, time.Now(), map[string]any{}); err != nil {
		t.Errorf("участник ведёт записи: %v", err)
	}
	if _, err := svc.UpdateCalendar(ctx, memberID, 1, "Новое"); err != domain.ErrForbidden {
		t.Errorf("рядовой участник не меняет структуру, получено %v", err)
	}
	if err := svc.DeleteCalendar(ctx, memberID, 1); err != domain.ErrOwnerOnly {
		t.Errorf("рядовой участник не удаляет календарь, получено %v", err)
	}
	if _, err := svc.UpdateCalendar(ctx, adminID, 1, "Новое"); err != nil {
		t.Errorf("администратор команды меняет структуру: %v", err)
	}
}

// Забирая календарь к себе, человек становится хозяином, а файлы записей
// переезжают на его квоту; в чужую команду календарь не положить.
func TestMoveCalendar(t *testing.T) {
	fields := []domain.Field{{ID: 13, Label: "Фото", Type: domain.FieldImage}}
	svc, repo, _ := newTestService(fields)
	repo.entries = map[int64]*domain.Entry{5: {ID: 5, CalendarID: 1,
		Data: map[string]any{"13": map[string]any{"path": "calendar/a.png", "name": "a.png"}}}}
	ctx := context.Background()

	foreign := int64(companyID + 1)
	if _, err := svc.MoveCalendar(ctx, adminID, 1, &foreign, ""); err != domain.ErrNotTeamMember {
		t.Fatalf("перенос в чужую команду должен отбиваться, получено %v", err)
	}
	cal, err := svc.MoveCalendar(ctx, adminID, 1, nil, "")
	if err != nil {
		t.Fatalf("перенос к себе: %v", err)
	}
	if cal.CompanyID != nil || cal.OwnerID != adminID || cal.MyAccess != domain.AccessOwner {
		t.Errorf("календарь не стал личным: %+v", cal)
	}
	if moved := svc.files.(*fakeFiles).moved; len(moved) != 1 || moved[0] != "calendar/a.png" {
		t.Errorf("файлы не переехали на новую квоту: %v", moved)
	}
}

// Повестка дня для живой плитки: заголовок события считает сервер по первому
// полю «в таблице», за пределы периода записи не попадают.
func TestAgenda_TitleFromTableFieldAndPeriod(t *testing.T) {
	fields := []domain.Field{
		{ID: 10, Label: "Заметка", Type: domain.FieldText},
		{ID: 11, Label: "Тема", Type: domain.FieldText, ShowInTable: true},
	}
	svc, repo, _ := newTestService(fields)
	repo.entries = map[int64]*domain.Entry{
		1: {ID: 1, CalendarID: 1, EventAt: time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC),
			Data: map[string]any{"10": "не заголовок", "11": "Планёрка"}},
		2: {ID: 2, CalendarID: 1, EventAt: time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC),
			Data: map[string]any{"11": "Следующий месяц"}},
	}

	from := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	res, err := svc.Agenda(context.Background(), memberID, from, from.AddDate(0, 0, 1), 10)
	if err != nil {
		t.Fatalf("Agenda: %v", err)
	}
	if res.Total != 1 || len(res.Items) != 1 {
		t.Fatalf("ожидалось одно событие за день, получено total=%d items=%d", res.Total, len(res.Items))
	}
	if res.Items[0].Title != "Планёрка" || res.Items[0].CalendarName != "Тест" {
		t.Fatalf("неожиданный элемент повестки: %+v", res.Items[0])
	}

	// Посторонний повестку не получает.
	other, err := svc.Agenda(context.Background(), strangerID, from, from.AddDate(0, 0, 1), 10)
	if err != nil || other.Total != 0 {
		t.Fatalf("посторонний не должен видеть события: %+v, %v", other, err)
	}
}

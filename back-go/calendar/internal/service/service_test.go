package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
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

type fakeBus struct{ events []string }

func (b *fakeBus) Publish(_ domain.Ctx, event string, _ []string, _ any) {
	b.events = append(b.events, event)
}

func newTestService(fields []domain.Field) (*Service, *fakeRepo, *fakeBus) {
	repo := &fakeRepo{
		cal:    &domain.Calendar{ID: 1, CompanyID: 7, Name: "Тест"},
		fields: fields,
	}
	bus := &fakeBus{}
	return New(Deps{Repo: repo, Bus: bus, Log: discardLogger()}), repo, bus
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
	e, err := svc.CreateEntry(context.Background(), 7, 1, 42, at, map[string]any{
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
	_, err := svc.CreateEntry(context.Background(), 7, 1, 42, time.Time{}, map[string]any{})
	if err != domain.ErrEventAtRequired {
		t.Errorf("ожидалась ErrEventAtRequired, получено %v", err)
	}
}

func TestCreateEntry_NumberPatternRejected(t *testing.T) {
	fields := []domain.Field{
		{ID: 11, Label: "Код", Type: domain.FieldNumber, Config: map[string]any{"pattern": `^\d{3}$`}},
	}
	svc, _, _ := newTestService(fields)
	_, err := svc.CreateEntry(context.Background(), 7, 1, 42, time.Now(), map[string]any{"11": "abc"})
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

	_, err := svc.ReplaceFields(context.Background(), 7, 1, []domain.Field{
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

func TestCalendarScopedToCompany(t *testing.T) {
	svc, _, _ := newTestService(nil)
	// Календарь принадлежит компании 7 — чужая компания 99 не видит его.
	if _, err := svc.GetCalendar(context.Background(), 99, 1); err != domain.ErrCalendarNotFound {
		t.Errorf("ожидалась ErrCalendarNotFound для чужой компании, получено %v", err)
	}
}

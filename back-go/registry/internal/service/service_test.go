package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRepo — in-memory реализация порта для тестов бизнес-логики.
type fakeRepo struct {
	reg        *domain.Registry
	fields     []domain.Field
	records    map[int64]*domain.Record
	lastSearch string
	nextID     int64
}

func (f *fakeRepo) ListRegistries(_ domain.Ctx, _ int64) ([]*domain.Registry, error) {
	return []*domain.Registry{f.reg}, nil
}
func (f *fakeRepo) GetRegistry(_ domain.Ctx, id int64) (*domain.Registry, error) {
	if f.reg != nil && f.reg.ID == id {
		return f.reg, nil
	}
	return nil, nil
}
func (f *fakeRepo) CreateRegistry(_ domain.Ctx, r *domain.Registry) error       { r.ID = 1; return nil }
func (f *fakeRepo) UpdateRegistry(_ domain.Ctx, _ int64, _ string, _ int) error { return nil }
func (f *fakeRepo) DeleteRegistry(_ domain.Ctx, _ int64) error                  { return nil }
func (f *fakeRepo) NextRegistryPosition(_ domain.Ctx, _ int64) (int, error)     { return 1, nil }
func (f *fakeRepo) ListFields(_ domain.Ctx, _ int64) ([]domain.Field, error)    { return f.fields, nil }
func (f *fakeRepo) FieldsByRegistries(_ domain.Ctx, _ []int64) (map[int64][]domain.Field, error) {
	return map[int64][]domain.Field{f.reg.ID: f.fields}, nil
}
func (f *fakeRepo) ReplaceFields(_ domain.Ctx, _ int64, fields []domain.Field) ([]int64, error) {
	f.fields = fields
	return []int64{99}, nil // имитируем удаление поля 99
}
func (f *fakeRepo) ListRecords(_ domain.Ctx, _ domain.RecordListFilter) ([]*domain.Record, int, error) {
	return nil, 0, nil
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
func (f *fakeRepo) DeleteRecords(_ domain.Ctx, _ int64, ids []int64) (int64, error) {
	return int64(len(ids)), nil
}
func (f *fakeRepo) RecordsForExport(_ domain.Ctx, _ int64, _ string, _ []int64) ([]*domain.Record, error) {
	return f.AllRecords(nil, 0)
}
func (f *fakeRepo) CreateShare(_ domain.Ctx, s *domain.Share) error              { s.ID = 1; return nil }
func (f *fakeRepo) ListShares(_ domain.Ctx, _ int64) ([]*domain.Share, error)    { return nil, nil }
func (f *fakeRepo) GetShareByCode(_ domain.Ctx, _ string) (*domain.Share, error) { return nil, nil }
func (f *fakeRepo) DeleteShare(_ domain.Ctx, _, _ int64) error                   { return nil }
func (f *fakeRepo) AllRecords(_ domain.Ctx, _ int64) ([]*domain.Record, error) {
	out := []*domain.Record{}
	for _, r := range f.records {
		out = append(out, r)
	}
	return out, nil
}

type fakeBus struct{ events []string }

func (b *fakeBus) Publish(_ domain.Ctx, event string, _ []string, _ any) {
	b.events = append(b.events, event)
}

type fakeFiles struct{ removed []string }

func (f *fakeFiles) Save(_ string, _ []byte) (string, error) { return "registry/x", nil }
func (f *fakeFiles) Remove(paths []string)                   { f.removed = append(f.removed, paths...) }

func newTestService(fields []domain.Field) (*Service, *fakeRepo, *fakeBus) {
	repo := &fakeRepo{
		reg:    &domain.Registry{ID: 1, CompanyID: 7, Name: "Тест"},
		fields: fields,
	}
	bus := &fakeBus{}
	return New(Deps{Repo: repo, Files: &fakeFiles{}, Bus: bus, Log: discardLogger()}), repo, bus
}

func TestCreateRecord_BuildsSearchTextAndValidates(t *testing.T) {
	fields := []domain.Field{
		{ID: 10, Label: "Имя", Type: domain.FieldText},
		{ID: 11, Label: "Код", Type: domain.FieldNumber, Config: map[string]any{"pattern": `^\d{3}$`}},
		{ID: 12, Label: "Категория", Type: domain.FieldSelect, Config: map[string]any{"options": []any{"A", "B"}}},
		{ID: 13, Label: "Фото", Type: domain.FieldImage},
	}
	svc, repo, bus := newTestService(fields)

	rec, err := svc.CreateRecord(context.Background(), 7, 1, 42, map[string]any{
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
	_, err := svc.CreateRecord(context.Background(), 7, 1, 42, map[string]any{"11": "abc"})
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
	_, err := svc.CreateRecord(context.Background(), 7, 1, 42, map[string]any{"12": "Z"})
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

	_, err := svc.ReplaceFields(context.Background(), 7, 1, []domain.Field{
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

func TestRegistryScopedToCompany(t *testing.T) {
	svc, _, _ := newTestService(nil)
	// Реестр принадлежит компании 7 — чужая компания 99 не видит его.
	if _, err := svc.GetRegistry(context.Background(), 99, 1); err != domain.ErrRegistryNotFound {
		t.Errorf("ожидалась ErrRegistryNotFound для чужой компании, получено %v", err)
	}
}

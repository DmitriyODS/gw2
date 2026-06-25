package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeRepo — in-memory реализация порта для тестов бизнес-логики.
type fakeRepo struct {
	diaries    map[int64]*domain.Diary
	entries    map[int64]*domain.Entry
	members    map[int64]map[int64]bool // diaryID → set(userID)
	lastSearch string
	nextID     int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		diaries: map[int64]*domain.Diary{},
		entries: map[int64]*domain.Entry{},
		members: map[int64]map[int64]bool{},
	}
}

func (f *fakeRepo) ListOwned(_ domain.Ctx, ownerID int64) ([]*domain.Diary, error) {
	out := []*domain.Diary{}
	for _, d := range f.diaries {
		if d.OwnerID == ownerID {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeRepo) ListShared(_ domain.Ctx, userID int64) ([]*domain.Diary, error) {
	out := []*domain.Diary{}
	for id, set := range f.members {
		if set[userID] {
			if d := f.diaries[id]; d != nil {
				out = append(out, d)
			}
		}
	}
	return out, nil
}
func (f *fakeRepo) GetDiary(_ domain.Ctx, id int64) (*domain.Diary, error) { return f.diaries[id], nil }
func (f *fakeRepo) CreateDiary(_ domain.Ctx, d *domain.Diary) error {
	f.nextID++
	d.ID = f.nextID
	f.diaries[d.ID] = d
	return nil
}
func (f *fakeRepo) UpdateDiary(_ domain.Ctx, id int64, name string) error {
	if d := f.diaries[id]; d != nil {
		d.Name = name
	}
	return nil
}
func (f *fakeRepo) DeleteDiary(_ domain.Ctx, id int64) error        { delete(f.diaries, id); return nil }
func (f *fakeRepo) NextPosition(_ domain.Ctx, _ int64) (int, error) { return 1, nil }

func (f *fakeRepo) ListEntries(_ domain.Ctx, fl domain.EntryListFilter) ([]*domain.Entry, error) {
	out := []*domain.Entry{}
	for _, e := range f.entries {
		if e.DiaryID == fl.DiaryID && e.Done == fl.Archived {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeRepo) GetEntry(_ domain.Ctx, id int64) (*domain.Entry, error) { return f.entries[id], nil }
func (f *fakeRepo) CreateEntry(_ domain.Ctx, e *domain.Entry, searchText string) error {
	f.nextID++
	e.ID = f.nextID
	f.lastSearch = searchText
	f.entries[e.ID] = e
	return nil
}
func (f *fakeRepo) UpdateEntry(_ domain.Ctx, e *domain.Entry, searchText string) error {
	f.lastSearch = searchText
	f.entries[e.ID] = e
	return nil
}
func (f *fakeRepo) SetEntryDone(_ domain.Ctx, id int64, done bool) error {
	if e := f.entries[id]; e != nil {
		e.Done = done
	}
	return nil
}
func (f *fakeRepo) SetEntryTask(_ domain.Ctx, id int64, taskID *int64) error {
	if e := f.entries[id]; e != nil {
		e.LinkedTaskID = taskID
	}
	return nil
}
func (f *fakeRepo) DeleteEntry(_ domain.Ctx, id int64) error { delete(f.entries, id); return nil }
func (f *fakeRepo) DeleteEntries(_ domain.Ctx, _ int64, ids []int64) (int64, error) {
	for _, id := range ids {
		delete(f.entries, id)
	}
	return int64(len(ids)), nil
}
func (f *fakeRepo) EntriesForExport(c domain.Ctx, fl domain.EntryListFilter, _ []int64) ([]*domain.Entry, error) {
	return f.ListEntries(c, fl)
}
func (f *fakeRepo) CreateShare(_ domain.Ctx, s *domain.Share) error              { s.ID = 1; return nil }
func (f *fakeRepo) ListShares(_ domain.Ctx, _ int64) ([]*domain.Share, error)    { return nil, nil }
func (f *fakeRepo) GetShareByCode(_ domain.Ctx, _ string) (*domain.Share, error) { return nil, nil }
func (f *fakeRepo) DeleteShare(_ domain.Ctx, _, _ int64) error                   { return nil }
func (f *fakeRepo) ListMembers(_ domain.Ctx, _ int64) ([]*domain.Member, error)  { return nil, nil }
func (f *fakeRepo) MemberIDs(_ domain.Ctx, diaryID int64) ([]int64, error) {
	out := []int64{}
	for uid := range f.members[diaryID] {
		out = append(out, uid)
	}
	return out, nil
}
func (f *fakeRepo) HasMember(_ domain.Ctx, diaryID, userID int64) (bool, error) {
	return f.members[diaryID][userID], nil
}
func (f *fakeRepo) AddMember(_ domain.Ctx, diaryID, userID int64) error {
	if f.members[diaryID] == nil {
		f.members[diaryID] = map[int64]bool{}
	}
	f.members[diaryID][userID] = true
	return nil
}
func (f *fakeRepo) RemoveMember(_ domain.Ctx, diaryID, userID int64) error {
	delete(f.members[diaryID], userID)
	return nil
}

type fakeUsers struct{}

func (fakeUsers) GetUser(_ domain.Ctx, id int64) (*domain.User, error) {
	return &domain.User{ID: id, FIO: "Тест", IsActive: true}, nil
}

type fakeBus struct {
	events    []string
	lastRooms []string
}

func (b *fakeBus) Publish(_ domain.Ctx, event string, rooms []string, _ any) {
	b.events = append(b.events, event)
	b.lastRooms = rooms
}

func newTestService() (*Service, *fakeRepo, *fakeBus) {
	repo := newFakeRepo()
	repo.diaries[1] = &domain.Diary{ID: 1, OwnerID: 7, Name: "Личный"}
	repo.nextID = 1
	bus := &fakeBus{}
	return New(Deps{Repo: repo, Users: fakeUsers{}, Bus: bus, Log: discardLogger()}), repo, bus
}

func TestCreateEntry_BuildsSearchTextAndValidates(t *testing.T) {
	svc, repo, bus := newTestService()
	at := time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC)
	e, err := svc.CreateEntry(context.Background(), 7, 1, EntryInput{Date: at, Title: "Купить хлеб", Description: "В пекарне"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.lastSearch != "купить хлеб в пекарне" {
		t.Errorf("search_text = %q", repo.lastSearch)
	}
	if e.Done {
		t.Error("новая запись не должна быть в архиве")
	}
	if len(bus.events) != 1 || bus.events[0] != "diary_entry:created" {
		t.Errorf("ожидалось diary_entry:created, получено %v", bus.events)
	}
}

func TestCreateEntry_RequiresTitleAndDate(t *testing.T) {
	svc, _, _ := newTestService()
	if _, err := svc.CreateEntry(context.Background(), 7, 1, EntryInput{Date: time.Now(), Title: "  "}); err != domain.ErrTitleRequired {
		t.Errorf("ожидалась ErrTitleRequired, получено %v", err)
	}
	if _, err := svc.CreateEntry(context.Background(), 7, 1, EntryInput{Title: "Есть"}); err != domain.ErrDateRequired {
		t.Errorf("ожидалась ErrDateRequired, получено %v", err)
	}
}

func TestDiaryScopedToOwner(t *testing.T) {
	svc, _, _ := newTestService()
	// Чужой пользователь без доступа не видит ежедневник.
	if _, err := svc.GetDiary(context.Background(), 99, 1); err != domain.ErrDiaryNotFound {
		t.Errorf("ожидалась ErrDiaryNotFound для чужого, получено %v", err)
	}
	// Чужой не может создать запись.
	if _, err := svc.CreateEntry(context.Background(), 99, 1, EntryInput{Date: time.Now(), Title: "Чужое"}); err != domain.ErrDiaryNotFound {
		t.Errorf("ожидалась ErrDiaryNotFound при записи чужого, получено %v", err)
	}
}

func TestSharedMemberReadOnly(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.members[1] = map[int64]bool{42: true} // ежедневник 1 открыт пользователю 42
	// Адресат видит ежедневник (read-only).
	d, err := svc.GetDiary(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("адресат должен видеть ежедневник: %v", err)
	}
	if !d.Shared {
		t.Error("для адресата Shared должен быть true (read-only)")
	}
	// Но не может его менять.
	if _, err := svc.UpdateDiary(context.Background(), 42, 1, "Хочу переименовать"); err != domain.ErrDiaryNotFound {
		t.Errorf("адресат не должен править чужой ежедневник, получено %v", err)
	}
	// И видит его в списке «Поделились».
	shared, _ := svc.ListShared(context.Background(), 42)
	if len(shared) != 1 || shared[0].ID != 1 {
		t.Errorf("ежедневник должен быть в «Поделились», получено %v", shared)
	}
}

func TestSetDone_MovesBetweenTabs(t *testing.T) {
	svc, _, _ := newTestService()
	at := time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC)
	e, _ := svc.CreateEntry(context.Background(), 7, 1, EntryInput{Date: at, Title: "Задача"})
	if _, err := svc.SetDone(context.Background(), 7, 1, e.ID, true); err != nil {
		t.Fatalf("setDone: %v", err)
	}
	active, _ := svc.ListEntries(context.Background(), 7, 1, ListParams{Archived: false})
	archived, _ := svc.ListEntries(context.Background(), 7, 1, ListParams{Archived: true})
	if len(active.Items) != 0 {
		t.Error("выполненная запись не должна быть в активных")
	}
	if len(archived.Items) != 1 {
		t.Error("выполненная запись должна быть в архиве")
	}
}

func TestEntryEventsReachMembers(t *testing.T) {
	svc, repo, bus := newTestService()
	repo.members[1] = map[int64]bool{42: true}
	at := time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC)
	if _, err := svc.CreateEntry(context.Background(), 7, 1, EntryInput{Date: at, Title: "Видно адресату"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	wantOwner, wantMember := "user_7", "user_42"
	var hasOwner, hasMember bool
	for _, r := range bus.lastRooms {
		if r == wantOwner {
			hasOwner = true
		}
		if r == wantMember {
			hasMember = true
		}
	}
	if !hasOwner || !hasMember {
		t.Errorf("событие должно идти владельцу и адресату, комнаты %v", bus.lastRooms)
	}
}

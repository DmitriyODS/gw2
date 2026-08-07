package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/reminder/internal/domain"
)

// ── фейки портов ────────────────────────────────────────────────

type fakeRepo struct {
	items  map[int64]*domain.Reminder
	nextID int64
}

func newRepo() *fakeRepo { return &fakeRepo{items: map[int64]*domain.Reminder{}, nextID: 1} }

func (f *fakeRepo) List(_ domain.Ctx, ownerID int64, scope domain.ListScope) ([]*domain.Reminder, error) {
	out := []*domain.Reminder{}
	for _, r := range f.items {
		if r.OwnerID != ownerID {
			continue
		}
		if (scope == domain.ScopeActive && !r.Active) || (scope == domain.ScopeDone && r.Active) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeRepo) Upcoming(_ domain.Ctx, ownerID int64, until time.Time, _ int) ([]*domain.Reminder, error) {
	out := []*domain.Reminder{}
	for _, r := range f.items {
		if r.OwnerID == ownerID && r.Active && !r.RemindAt.After(until) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) ByLink(_ domain.Ctx, ownerID int64, kind string, recordID int64) ([]*domain.Reminder, error) {
	out := []*domain.Reminder{}
	for _, r := range f.items {
		if r.OwnerID == ownerID && r.Link.Kind == kind && r.Link.RecordID == recordID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) Get(_ domain.Ctx, id int64) (*domain.Reminder, error) { return f.items[id], nil }

func (f *fakeRepo) Create(_ domain.Ctx, r *domain.Reminder) error {
	r.ID = f.nextID
	f.nextID++
	r.CreatedAt, r.UpdatedAt = time.Now(), time.Now()
	f.items[r.ID] = r
	return nil
}

func (f *fakeRepo) Update(_ domain.Ctx, r *domain.Reminder) error {
	r.UpdatedAt = time.Now()
	f.items[r.ID] = r
	return nil
}

func (f *fakeRepo) Delete(_ domain.Ctx, id int64) error { delete(f.items, id); return nil }

// ClaimDue — как в postgres: забирает наступившие и сразу гасит active.
func (f *fakeRepo) ClaimDue(_ domain.Ctx, now time.Time, _ int) ([]*domain.Reminder, error) {
	out := []*domain.Reminder{}
	for _, r := range f.items {
		if r.Active && !r.RemindAt.After(now) {
			r.Active = false
			out = append(out, r)
		}
	}
	return out, nil
}

type event struct {
	name    string
	rooms   []string
	payload any
}

type fakeBus struct{ events []event }

func (b *fakeBus) Publish(_ domain.Ctx, name string, rooms []string, payload any) {
	b.events = append(b.events, event{name: name, rooms: rooms, payload: payload})
}

func (b *fakeBus) count(name string) int {
	n := 0
	for _, e := range b.events {
		if e.name == name {
			n++
		}
	}
	return n
}

func newService() (*Service, *fakeRepo, *fakeBus) {
	repo, bus := newRepo(), &fakeBus{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(Deps{Repo: repo, Bus: bus, Log: log}), repo, bus
}

func ctx() context.Context { return context.Background() }

// ── тесты ───────────────────────────────────────────────────────

func TestCreateRequiresTitleAndTime(t *testing.T) {
	s, _, _ := newService()
	if _, err := s.Create(ctx(), &domain.Reminder{OwnerID: 1, RemindAt: time.Now()}); err == nil {
		t.Fatal("напоминание без текста должно отклоняться")
	}
	if _, err := s.Create(ctx(), &domain.Reminder{OwnerID: 1, Title: "Позвонить"}); err == nil {
		t.Fatal("напоминание без времени должно отклоняться")
	}
}

func TestForeignReminderIsNotFound(t *testing.T) {
	s, _, _ := newService()
	mine, err := s.Create(ctx(), &domain.Reminder{OwnerID: 1, Title: "Моё", RemindAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("создание: %v", err)
	}
	if _, err := s.Get(ctx(), 2, mine.ID); err == nil {
		t.Fatal("чужое напоминание не должно быть видно")
	}
	if err := s.Delete(ctx(), 2, mine.ID); err == nil {
		t.Fatal("чужое напоминание не должно удаляться")
	}
}

func TestOneShotFiresOnceAndGoesToJournal(t *testing.T) {
	s, repo, bus := newService()
	r, _ := s.Create(ctx(), &domain.Reminder{
		OwnerID: 7, Title: "Забрать посылку", RemindAt: time.Now().Add(-time.Minute),
	})
	s.fireDue(ctx())
	if bus.count("reminder:fire") != 1 {
		t.Fatalf("ожидалось одно срабатывание, получено %d", bus.count("reminder:fire"))
	}
	if repo.items[r.ID].Active {
		t.Fatal("разовое напоминание должно стать неактивным")
	}
	s.fireDue(ctx()) // второй проход не должен ничего доставить повторно
	if bus.count("reminder:fire") != 1 {
		t.Fatalf("повторная доставка: %d", bus.count("reminder:fire"))
	}
}

func TestFireGoesOnlyToOwnerRoom(t *testing.T) {
	s, _, bus := newService()
	_, _ = s.Create(ctx(), &domain.Reminder{OwnerID: 42, Title: "Планёрка", RemindAt: time.Now().Add(-time.Second)})
	s.fireDue(ctx())
	for _, e := range bus.events {
		if e.name != "reminder:fire" {
			continue
		}
		if len(e.rooms) != 1 || e.rooms[0] != "user_42" {
			t.Fatalf("событие должно идти только владельцу, комнаты %v", e.rooms)
		}
	}
}

func TestRepeatingReminderReschedulesForward(t *testing.T) {
	s, repo, _ := newService()
	// Просрочено на трое суток: следующий срок обязан быть в будущем, а не
	// пачкой пропущенных.
	r, _ := s.Create(ctx(), &domain.Reminder{
		OwnerID: 3, Title: "Зарядка", RemindAt: time.Now().Add(-72 * time.Hour),
		Timezone: "Europe/Moscow", Repeat: domain.Repeat{Kind: domain.RepeatDaily, Interval: 1},
	})
	s.fireDue(ctx())
	got := repo.items[r.ID]
	if !got.Active {
		t.Fatal("повторяющееся напоминание должно остаться активным")
	}
	if !got.RemindAt.After(time.Now()) {
		t.Fatalf("следующий срок должен быть в будущем, получено %s", got.RemindAt)
	}
	if got.LastFiredAt == nil {
		t.Fatal("должно проставиться время последнего срабатывания")
	}
}

func TestSnoozeMovesTimeAndValidatesRange(t *testing.T) {
	s, repo, _ := newService()
	r, _ := s.Create(ctx(), &domain.Reminder{OwnerID: 5, Title: "Выпить воды", RemindAt: time.Now().Add(-time.Minute)})
	s.fireDue(ctx())
	if _, err := s.Snooze(ctx(), 5, r.ID, 0); err == nil {
		t.Fatal("нулевая отсрочка должна отклоняться")
	}
	if _, err := s.Snooze(ctx(), 5, r.ID, 10); err != nil {
		t.Fatalf("отложить: %v", err)
	}
	got := repo.items[r.ID]
	if !got.Active || !got.RemindAt.After(time.Now()) {
		t.Fatalf("после отсрочки напоминание должно снова ждать, получено %+v", got)
	}
}

func TestCompleteSkipsToNextOccurrence(t *testing.T) {
	s, repo, _ := newService()
	r, _ := s.Create(ctx(), &domain.Reminder{
		OwnerID: 9, Title: "Отчёт", RemindAt: time.Now().Add(time.Hour),
		Timezone: "Europe/Moscow", Repeat: domain.Repeat{Kind: domain.RepeatWeekly, Interval: 1},
	})
	before := repo.items[r.ID].RemindAt
	if _, err := s.Complete(ctx(), 9, r.ID); err != nil {
		t.Fatalf("готово: %v", err)
	}
	got := repo.items[r.ID]
	if !got.Active || !got.RemindAt.After(before) {
		t.Fatalf("после «готово» повтор должен уехать вперёд, было %s стало %s", before, got.RemindAt)
	}
}

func TestLinkedRequiresKnownKind(t *testing.T) {
	s, _, _ := newService()
	if _, err := s.ByLink(ctx(), 1, "tasks", 5); err == nil {
		t.Fatal("неизвестная привязка должна отклоняться")
	}
	if _, err := s.ByLink(ctx(), 1, domain.LinkDiary, 5); err != nil {
		t.Fatalf("привязка к ежедневнику: %v", err)
	}
}

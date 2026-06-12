package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// ── Python-паритет дат: квесты и болезни должны совпасть с прежним Flask ──

func TestPythonOrdinal(t *testing.T) {
	cases := []struct {
		date time.Time
		want int
	}{
		{time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC), 739779},
		{time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC), 730486},
		{time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), 739616},
		{time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), 1},
	}
	for _, c := range cases {
		if got := pythonOrdinal(c.date); got != c.want {
			t.Errorf("pythonOrdinal(%s) = %d, want %d", c.date.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestPickQuestTemplateMatchesPython(t *testing.T) {
	// Эталоны посчитаны CPython: (user_id*1009 + toordinal) % 6.
	if got := pickQuestTemplate(77, time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)); got != domain.QuestTemplates[2] {
		t.Errorf("quest(77, 2026-06-12): got %q, want %q", got.Kind, domain.QuestTemplates[2].Kind)
	}
	if got := pickQuestTemplate(5, time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)); got != domain.QuestTemplates[1] {
		t.Errorf("quest(5, 2025-12-31): got %q, want %q", got.Kind, domain.QuestTemplates[1].Kind)
	}
}

func TestWorkingDaysBetween(t *testing.T) {
	weekend := []int{5, 6}                              // Сб+Вс
	mon := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)  // понедельник
	fri := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC) // пятница
	nextMon := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	if got := workingDaysBetween(mon, fri, weekend); got != 4 {
		t.Errorf("пн→пт: %d, want 4", got)
	}
	// Выходные в интервале не считаются.
	if got := workingDaysBetween(fri, nextMon, weekend); got != 1 {
		t.Errorf("пт→пн: %d, want 1", got)
	}
	if got := workingDaysBetween(fri, fri, weekend); got != 0 {
		t.Errorf("пустой интервал: %d, want 0", got)
	}
	// Все дни — выходные: вечный 0 (никто не заболеет).
	if got := workingDaysBetween(mon, nextMon, []int{0, 1, 2, 3, 4, 5, 6}); got != 0 {
		t.Errorf("7 выходных: %d, want 0", got)
	}
}

func TestPlural(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "день"}, {2, "дня"}, {5, "дней"}, {11, "дней"},
		{21, "день"}, {104, "дня"}, {111, "дней"},
	}
	for _, c := range cases {
		if got := plural(c.n, "день", "дня", "дней"); got != c.want {
			t.Errorf("plural(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestFirstName(t *testing.T) {
	if got := firstName("Иванов Пётр Сергеевич"); got != "Пётр" {
		t.Errorf("got %q", got)
	}
	if got := firstName("Мадонна"); got != "Мадонна" {
		t.Errorf("got %q", got)
	}
	if got := firstName(""); got != "коллега" {
		t.Errorf("got %q", got)
	}
}

func TestResolvePeriod(t *testing.T) {
	p := resolvePeriod("this_week")
	if p.Label != "эта неделя" {
		t.Errorf("label: %q", p.Label)
	}
	if d := p.End.Sub(p.Start); d != 7*24*time.Hour {
		t.Errorf("длина недели: %v", d)
	}
	if got := resolvePeriod("yesterday"); got.End.Sub(got.Start) != 24*time.Hour {
		t.Errorf("вчера: %v", got.End.Sub(got.Start))
	}
	// Мусор → дефолт this_week.
	if got := resolvePeriod("nonsense"); got.Label != "эта неделя" {
		t.Errorf("дефолт: %q", got.Label)
	}
}

func TestApplyRecovery(t *testing.T) {
	now := time.Now()
	pet := &domain.Pet{SickSince: &now, Recovery: 0}
	if applyRecovery(pet, 1) {
		t.Error("выздоровел слишком рано")
	}
	if applyRecovery(pet, 1) {
		t.Error("выздоровел слишком рано")
	}
	if !applyRecovery(pet, 1) {
		t.Error("должен был выздороветь на 3-м очке")
	}
	if pet.SickSince != nil || pet.Recovery != 0 {
		t.Error("после выздоровления болезнь не снята")
	}
	// Здоровому recovery не идёт.
	if applyRecovery(pet, 5) {
		t.Error("здоровый питомец «выздоровел»")
	}
}

// ── FeedPet на фейках портов ───────────────────────────────────────

type fakePets struct {
	domain.PetRepo
	pet *domain.Pet
}

func (f *fakePets) GetOrCreate(_ context.Context, userID, companyID int64) (*domain.Pet, error) {
	if f.pet == nil {
		f.pet = &domain.Pet{UserID: userID, CompanyID: companyID,
			Name: "Грувик", Species: "egg",
			Accessories: []string{}, UnlockedSpecies: []string{}}
	}
	return f.pet, nil
}
func (f *fakePets) GetPet(_ context.Context, userID int64) (*domain.Pet, error) {
	return f.pet, nil
}
func (f *fakePets) SavePet(_ context.Context, p *domain.Pet) error {
	f.pet = p
	return nil
}
func (f *fakePets) FinishedUnitsForUser(context.Context, int64, time.Time, int) ([]domain.FinishedUnit, error) {
	return nil, nil
}

type fakeFeed struct {
	domain.FeedRepo
	kinds []string
}

func (f *fakeFeed) CreateEvent(_ context.Context, companyID int64, userID *int64,
	kind string, payload map[string]any) (*domain.FeedEvent, error) {
	f.kinds = append(f.kinds, kind)
	return &domain.FeedEvent{ID: int64(len(f.kinds)), CompanyID: companyID,
		UserID: userID, Kind: kind, Payload: payload, CreatedAt: time.Now()}, nil
}
func (f *fakeFeed) GetEvent(context.Context, int64) (*domain.FeedEvent, error) {
	return nil, nil // makeBotComment тихо выходит
}

type fakeDaily struct {
	domain.Daily
	denyBudget bool
}

func (f *fakeDaily) TakeBudget(_ context.Context, _ int64, _ string, want, _ int) int {
	if f.denyBudget {
		return 0
	}
	return want
}
func (f *fakeDaily) Left(context.Context, int64, string, int) int { return 1 }
func (f *fakeDaily) GetCache(context.Context, string) string      { return "" }

type fakePub struct {
	domain.EventPublisher
	events []string
}

func (f *fakePub) Publish(_ context.Context, event string, _ []string, _ any) {
	f.events = append(f.events, event)
}

type fakeAI struct{ domain.AIClient }

func (fakeAI) Enabled(context.Context, int64) bool { return false }

func newTestService(pets *fakePets, daily *fakeDaily, pub *fakePub, feed *fakeFeed) *Service {
	return New(feed, pets, nil, nil, nil, nil, nil, daily, pub, fakeAI{}, nil, nil,
		slog.New(slog.DiscardHandler))
}

func TestFeedPetHappyPath(t *testing.T) {
	pets := &fakePets{}
	daily := &fakeDaily{}
	pub := &fakePub{}
	feed := &fakeFeed{}
	svc := newTestService(pets, daily, pub, feed)

	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Beans = 10

	data, err := svc.FeedPet(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if data.Beans != 10-domain.FeedCost {
		t.Errorf("beans = %d", data.Beans)
	}
	if data.XP != domain.FeedXP {
		t.Errorf("xp = %d", data.XP)
	}
	if data.FeedStreak != 1 {
		t.Errorf("streak = %d", data.FeedStreak)
	}
	if data.Phrase == nil || *data.Phrase == "" {
		t.Error("нет реплики кормления")
	}
	if data.Evolved == nil || *data.Evolved {
		t.Error("неожиданная эволюция")
	}
	if len(pub.events) == 0 || pub.events[len(pub.events)-1] != "pet:update" {
		t.Errorf("события: %v", pub.events)
	}
}

func TestFeedPetNoBeans(t *testing.T) {
	pets := &fakePets{}
	svc := newTestService(pets, &fakeDaily{}, &fakePub{}, &fakeFeed{})
	pets.GetOrCreate(context.Background(), 1, 10) // beans = 0

	_, err := svc.FeedPet(context.Background(), 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "NO_BEANS" {
		t.Fatalf("ожидался NO_BEANS, got %v", err)
	}
}

func TestFeedPetDailyCap(t *testing.T) {
	pets := &fakePets{}
	svc := newTestService(pets, &fakeDaily{denyBudget: true}, &fakePub{}, &fakeFeed{})
	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Beans = 10

	_, err := svc.FeedPet(context.Background(), 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "FED_ENOUGH" || de.HTTPStatus != 429 {
		t.Fatalf("ожидался FED_ENOUGH/429, got %v", err)
	}
}

func TestFeedPetEvolves(t *testing.T) {
	pets := &fakePets{}
	feed := &fakeFeed{}
	svc := newTestService(pets, &fakeDaily{}, &fakePub{}, feed)
	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Beans = 10
	pet.XP = domain.StageXP[1] - domain.FeedXP + 1 // эволюция после кормления

	data, err := svc.FeedPet(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if data.Stage != 1 {
		t.Errorf("stage = %d, want 1", data.Stage)
	}
	if data.Species != "fox" { // нет юнитов → fox
		t.Errorf("species = %q", data.Species)
	}
	if data.Evolved == nil || !*data.Evolved {
		t.Error("эволюция не отмечена")
	}
	found := false
	for _, k := range feed.kinds {
		if k == "pet_evolved" {
			found = true
		}
	}
	if !found {
		t.Errorf("нет события pet_evolved: %v", feed.kinds)
	}
}

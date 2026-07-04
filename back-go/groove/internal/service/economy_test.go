package service

// Тесты экономики Groove: начисление грувов с дневными капами,
// кормление, прямой XP за работу и эволюция.

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// Расширения фейков из service_test.go под рейд и рейтинг.

func (f *fakePets) ListCompanyPets(context.Context, int64) ([]*domain.Pet, error) {
	return f.company, nil
}
func (f *fakePets) GetRaid(context.Context, int64, time.Time) (*domain.Raid, error) {
	return f.raid, nil
}
func (f *fakePets) CountClosedBetween(context.Context, int64, time.Time, time.Time) (int, error) {
	return f.closedBetween, nil
}

func (f *fakeFeed) CountUserEvents(context.Context, int64, int64, string, time.Time) (int, error) {
	return f.myClosed, nil
}

func (f *fakeFeed) KudosWeekCounts(context.Context, int64, time.Time) (map[int64]int, error) {
	return f.kudosWeek, nil
}

// capDaily — фейк дневных капов с реальным учётом бюджета (в отличие от
// fakeDaily, который выдаёт всё подряд).
type capDaily struct {
	domain.Daily
	used map[string]int
}

func (f *capDaily) TakeBudget(_ context.Context, _ int64, source string, want, cap int) int {
	if want <= 0 {
		return 0
	}
	if f.used == nil {
		f.used = map[string]int{}
	}
	granted := min(want, cap-f.used[source])
	if granted <= 0 {
		return 0
	}
	f.used[source] += granted
	return granted
}
func (f *capDaily) Left(_ context.Context, _ int64, source string, cap int) int {
	return max(0, cap-f.used[source])
}
func (f *capDaily) GetCache(context.Context, string) string { return "" }

func newEconomyService(pets *fakePets, daily *capDaily) (*Service, *fakePub, *fakeFeed) {
	pub := &fakePub{}
	feed := &fakeFeed{}
	return New(feed, pets, nil, nil, nil, nil, nil, daily, pub, fakeAI{}, nil, nil,
		slog.New(slog.DiscardHandler)), pub, feed
}

// ── AwardBeans: дневные капы по источникам ─────────────────────────

func TestAwardBeansRespectsDailyCap(t *testing.T) {
	pets := &fakePets{}
	svc, pub, _ := newEconomyService(pets, &capDaily{})
	ctx := context.Background()
	pets.GetOrCreate(ctx, 1, 10)

	// Источник «unit», кап 15: 10 + 10 → второй раз урезается до 5.
	if got := svc.AwardBeans(ctx, 1, 10, "unit", 10); got != 10 {
		t.Errorf("первое начисление: %d, want 10", got)
	}
	if got := svc.AwardBeans(ctx, 1, 10, "unit", 10); got != domain.DailyCaps["unit"]-10 {
		t.Errorf("второе начисление: %d, want %d", got, domain.DailyCaps["unit"]-10)
	}
	if got := svc.AwardBeans(ctx, 1, 10, "unit", 1); got != 0 {
		t.Errorf("сверх капа: %d, want 0", got)
	}
	if pets.pet.Beans != domain.DailyCaps["unit"] {
		t.Errorf("beans = %d, want %d", pets.pet.Beans, domain.DailyCaps["unit"])
	}
	// Каждое успешное начисление эмитит pet:update.
	if len(pub.events) != 2 {
		t.Errorf("события: %v", pub.events)
	}
}

func TestAwardBeansUnknownSourceDefaultCap(t *testing.T) {
	pets := &fakePets{}
	svc, _, _ := newEconomyService(pets, &capDaily{})
	ctx := context.Background()
	pets.GetOrCreate(ctx, 1, 10)

	if got := svc.AwardBeans(ctx, 1, 10, "mystery", 99); got != domain.DefaultDailyCap {
		t.Errorf("неизвестный источник: %d, want %d", got, domain.DefaultDailyCap)
	}
}

// ── Кормление: стоимость, XP, лимит ────────────────────────────────

func TestFeedPetCostAndXP(t *testing.T) {
	pets := &fakePets{}
	daily := &capDaily{}
	svc, _, _ := newEconomyService(pets, daily)
	ctx := context.Background()
	pet, _ := pets.GetOrCreate(ctx, 1, 10)
	pet.Beans = domain.FeedCost * (domain.FeedDailyMax + 2)

	for i := 1; i <= domain.FeedDailyMax; i++ {
		if _, err := svc.FeedPet(ctx, 1, 10); err != nil {
			t.Fatalf("кормление %d: %v", i, err)
		}
	}
	if pets.pet.XP != domain.FeedXP*domain.FeedDailyMax {
		t.Errorf("xp = %d, want %d", pets.pet.XP, domain.FeedXP*domain.FeedDailyMax)
	}
	wantBeans := domain.FeedCost * 2
	if pets.pet.Beans != wantBeans {
		t.Errorf("beans = %d, want %d", pets.pet.Beans, wantBeans)
	}
	// Седьмое кормление — сверх дневного лимита.
	_, err := svc.FeedPet(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "FED_ENOUGH" {
		t.Fatalf("ожидался FED_ENOUGH, got %v", err)
	}
}

// ── Прямой XP за работу ────────────────────────────────────────────

func TestAwardXPForUnitMinutes(t *testing.T) {
	pets := &fakePets{}
	svc, pub, _ := newEconomyService(pets, &capDaily{})
	ctx := context.Background()
	pets.GetOrCreate(ctx, 1, 10)

	// 30 минут юнита → 10 XP (по 1 за каждые 3 минуты).
	got := svc.AwardXP(ctx, 1, 10, "xp_unit", 30/domain.XPUnitMinutesPer, domain.XPUnitDailyCap)
	if got != 10 {
		t.Errorf("granted = %d, want 10", got)
	}
	if pets.pet.XP != 10 {
		t.Errorf("xp = %d, want 10", pets.pet.XP)
	}
	if len(pub.events) == 0 || pub.events[len(pub.events)-1] != "pet:update" {
		t.Errorf("события: %v", pub.events)
	}
}

func TestAwardXPDailyCap(t *testing.T) {
	pets := &fakePets{}
	svc, _, _ := newEconomyService(pets, &capDaily{})
	ctx := context.Background()
	pets.GetOrCreate(ctx, 1, 10)

	svc.AwardXP(ctx, 1, 10, "xp_unit", 35, domain.XPUnitDailyCap)
	if got := svc.AwardXP(ctx, 1, 10, "xp_unit", 20, domain.XPUnitDailyCap); got != 5 {
		t.Errorf("второе начисление: %d, want 5 (остаток капа)", got)
	}
	if got := svc.AwardXP(ctx, 1, 10, "xp_unit", 1, domain.XPUnitDailyCap); got != 0 {
		t.Errorf("сверх капа: %d, want 0", got)
	}
	if pets.pet.XP != domain.XPUnitDailyCap {
		t.Errorf("xp = %d, want %d", pets.pet.XP, domain.XPUnitDailyCap)
	}
}

func TestAwardXPFedBoost(t *testing.T) {
	pets := &fakePets{}
	svc, _, _ := newEconomyService(pets, &capDaily{})
	ctx := context.Background()
	pet, _ := pets.GetOrCreate(ctx, 1, 10)
	fed := todayMSK()
	pet.LastFedDate = &fed // сегодня кормлен → сытость ×1.5

	if got := svc.AwardXP(ctx, 1, 10, "xp_unit", 10, domain.XPUnitDailyCap); got != 15 {
		t.Errorf("granted = %d, want 15 (10 × 1.5)", got)
	}
	if pets.pet.XP != 15 {
		t.Errorf("xp = %d, want 15", pets.pet.XP)
	}
}

func TestAwardXPFrozenWhileSick(t *testing.T) {
	pets := &fakePets{}
	daily := &capDaily{}
	svc, _, _ := newEconomyService(pets, daily)
	ctx := context.Background()
	pet, _ := pets.GetOrCreate(ctx, 1, 10)
	now := time.Now()
	pet.SickSince = &now

	if got := svc.AwardXP(ctx, 1, 10, "xp_unit", 10, domain.XPUnitDailyCap); got != 0 {
		t.Errorf("больной питомец получил XP: %d", got)
	}
	if pets.pet.XP != 0 {
		t.Errorf("xp = %d, want 0", pets.pet.XP)
	}
	// Бюджет капа не тратится впустую.
	if daily.used["xp_unit"] != 0 {
		t.Errorf("бюджет потрачен: %d", daily.used["xp_unit"])
	}
}

func TestAwardXPTriggersEvolution(t *testing.T) {
	pets := &fakePets{}
	svc, _, feed := newEconomyService(pets, &capDaily{})
	ctx := context.Background()
	pet, _ := pets.GetOrCreate(ctx, 1, 10)
	pet.XP = domain.StageXP[1] - 5

	if got := svc.AwardXP(ctx, 1, 10, "xp_task", domain.XPTaskClosed, domain.XPTaskDailyCap); got != domain.XPTaskClosed {
		t.Errorf("granted = %d", got)
	}
	if pets.pet.Stage != 1 {
		t.Errorf("stage = %d, want 1", pets.pet.Stage)
	}
	hasEvolved := false
	for _, k := range feed.kinds {
		if k == "pet_evolved" {
			hasEvolved = true
		}
	}
	if !hasEvolved {
		t.Errorf("нет события pet_evolved: %v", feed.kinds)
	}
}

// ── Хук юнита: грувы + XP одним завершением ────────────────────────

type fakeCompanies struct{ domain.CompanyReader }

func (fakeCompanies) GrooveEnabled(context.Context, int64) (bool, error) { return true, nil }

func TestOnUnitStoppedAwardsBeansAndXP(t *testing.T) {
	pets := &fakePets{}
	daily := &capDaily{}
	pub := &fakePub{}
	feed := &fakeFeed{}
	svc := New(feed, pets, nil, fakeCompanies{}, nil, nil, nil, daily, pub, fakeAI{},
		nil, nil, slog.New(slog.DiscardHandler))
	ctx := context.Background()
	pets.GetOrCreate(ctx, 1, 10)

	svc.OnUnitStopped(ctx, UnitHook{CompanyID: 10, UserID: 1, UnitID: 7,
		UnitName: "Юнит", Minutes: 30})

	if pets.pet.Beans != 30/5 {
		t.Errorf("beans = %d, want %d", pets.pet.Beans, 30/5)
	}
	if pets.pet.XP != 30/domain.XPUnitMinutesPer {
		t.Errorf("xp = %d, want %d", pets.pet.XP, 30/domain.XPUnitMinutesPer)
	}
	// Машинных событий ленты юниты больше не создают.
	if len(feed.kinds) != 0 {
		t.Errorf("события ленты: %v", feed.kinds)
	}
}

// ── Рейд: личный вклад и рейтинг ───────────────────────────────────

func TestGetRaidStateMyClosed(t *testing.T) {
	pets := &fakePets{raid: &domain.Raid{ID: 1, CompanyID: 10,
		WeekStart: weekStartMSK(), Boss: "Багоблин", Target: 10, Reward: "helmet"},
		closedBetween: 4}
	feed := &fakeFeed{myClosed: 3}
	svc := New(feed, pets, nil, nil, nil, nil, nil, &capDaily{}, &fakePub{}, fakeAI{},
		nil, nil, slog.New(slog.DiscardHandler))

	raid, err := svc.GetRaidState(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("GetRaidState: %v", err)
	}
	if raid.Progress != 4 {
		t.Errorf("progress = %d, want 4", raid.Progress)
	}
	if raid.MyClosed != 3 {
		t.Errorf("my_closed = %d, want 3", raid.MyClosed)
	}
	// Без зрителя (ТВ) личный вклад не считается.
	raid, _ = svc.GetRaidState(context.Background(), 10, 0)
	if raid.MyClosed != 0 {
		t.Errorf("my_closed без зрителя = %d, want 0", raid.MyClosed)
	}
}

func TestGetRating(t *testing.T) {
	mk := func(id int64, stage, xp int) *domain.Pet {
		return &domain.Pet{UserID: id, CompanyID: 10, Name: "Грувик", Species: "fox",
			Stage: stage, XP: xp, User: &domain.UserRef{ID: id, FIO: "Сотрудник"},
			Accessories: []string{}, UnlockedSpecies: []string{}}
	}
	pets := &fakePets{company: []*domain.Pet{mk(3, 4, 600), mk(1, 2, 200), mk(2, 1, 50)}}
	feed := &fakeFeed{kudosWeek: map[int64]int{2: 4}}
	svc := New(feed, pets, nil, nil, nil, nil, nil, &capDaily{}, &fakePub{},
		fakeAI{}, nil, nil, slog.New(slog.DiscardHandler))

	out, err := svc.GetRating(context.Background(), 10, 2)
	if err != nil {
		t.Fatalf("GetRating: %v", err)
	}
	items := out["items"].([]map[string]any)
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	if items[0]["position"] != 1 || items[0]["xp"] != 600 {
		t.Errorf("топ-1: %v", items[0])
	}
	if items[0]["kudos_week"] != 0 {
		t.Errorf("kudos_week топ-1: %v", items[0]["kudos_week"])
	}
	me, _ := out["me"].(map[string]any)
	if me == nil || me["position"] != 3 {
		t.Errorf("me: %v", me)
	}
	// Счётчик признания: кудосы, полученные с начала недели.
	if me != nil && me["kudos_week"] != 4 {
		t.Errorf("kudos_week me: %v", me["kudos_week"])
	}
	if out["total"] != 3 {
		t.Errorf("total = %v", out["total"])
	}
}

// ── Эволюция: порог стадии ─────────────────────────────────────────

func TestEvolutionThresholds(t *testing.T) {
	pets := &fakePets{}
	svc, _, feed := newEconomyService(pets, &capDaily{})
	ctx := context.Background()
	pet, _ := pets.GetOrCreate(ctx, 1, 10)
	pet.Beans = 10
	pet.XP = domain.StageXP[2] - domain.FeedXP // ровно до порога 2-й стадии

	data, err := svc.FeedPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if data.Stage != 2 {
		t.Errorf("stage = %d, want 2", data.Stage)
	}
	if data.XP != domain.StageXP[2] {
		t.Errorf("xp = %d, want %d", data.XP, domain.StageXP[2])
	}
	hasEvolved := false
	for _, k := range feed.kinds {
		if k == "pet_evolved" {
			hasEvolved = true
		}
	}
	if !hasEvolved {
		t.Errorf("нет события pet_evolved: %v", feed.kinds)
	}
}

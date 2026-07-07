package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
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

// ── Фейки портов (без БД/Redis, как в calendar/diary) ──────────────

type fakePets struct {
	domain.PetRepo
	byUser      map[int64]*domain.Pet
	company     []*domain.Pet
	weeklyKudos map[int64]int
	strokes     map[string]int

	saves          int // вызовы full-row SavePet
	adjustCalls    int // вызовы атомарного AdjustBalances
	evolutionSaves int // вызовы узкого SaveEvolution
}

func strokeKey(ownerID, strokerID int64, day time.Time) string {
	return strconvI64(ownerID) + ":" + strconvI64(strokerID) + ":" + day.Format("2006-01-02")
}

func (f *fakePets) GetOrCreate(_ context.Context, userID, companyID int64) (*domain.Pet, error) {
	if f.byUser == nil {
		f.byUser = map[int64]*domain.Pet{}
	}
	if f.byUser[userID] == nil {
		f.byUser[userID] = &domain.Pet{UserID: userID, CompanyID: companyID,
			Name: "Питомец", Species: "egg",
			Accessories: []string{}, UnlockedSpecies: []string{}}
	}
	return f.byUser[userID], nil
}
func (f *fakePets) GetPet(_ context.Context, userID int64) (*domain.Pet, error) {
	if f.byUser == nil {
		return nil, nil
	}
	return f.byUser[userID], nil
}
func (f *fakePets) SavePet(_ context.Context, p *domain.Pet) error {
	if f.byUser == nil {
		f.byUser = map[int64]*domain.Pet{}
	}
	f.byUser[p.UserID] = p
	f.saves++
	return nil
}
func (f *fakePets) AdjustBalances(_ context.Context, userID int64, deltaKudos, deltaXP int) (int, int, error) {
	p := f.byUser[userID]
	if p == nil {
		return 0, 0, errNoPet
	}
	p.Kudos = max(0, p.Kudos+deltaKudos)
	p.XP += deltaXP
	f.adjustCalls++
	return p.Kudos, p.XP, nil
}
func (f *fakePets) SaveEvolution(_ context.Context, p *domain.Pet) error {
	cur := f.byUser[p.UserID]
	if cur == nil {
		return errNoPet
	}
	cur.Stage, cur.Species, cur.Personality = p.Stage, p.Species, p.Personality
	cur.UnlockedSpecies = append([]string{}, p.UnlockedSpecies...)
	f.evolutionSaves++
	return nil
}
func (f *fakePets) StartAdventure(_ context.Context, userID int64, until time.Time, place string) (bool, error) {
	p := f.byUser[userID]
	if p == nil {
		return false, errNoPet
	}
	// Guard как в SQL: не болен и не в пути.
	if p.SickSince != nil || p.AdventureUntil != nil {
		return false, nil
	}
	u, pl := until, place
	p.AdventureUntil, p.AdventurePlace = &u, &pl
	return true, nil
}

// FinishAdventure — RETURNING-семантика реального репозитория: true ровно
// один раз, только для истёкшего приключения.
func (f *fakePets) FinishAdventure(_ context.Context, userID int64, now time.Time) (string, bool, error) {
	p := f.byUser[userID]
	if p == nil || p.AdventureUntil == nil || p.AdventureUntil.After(now) {
		return "", false, nil
	}
	place := ""
	if p.AdventurePlace != nil {
		place = *p.AdventurePlace
	}
	p.AdventureUntil, p.AdventurePlace = nil, nil
	return place, true, nil
}

func (f *fakePets) FinishedUnitsForUser(context.Context, int64, time.Time, int) ([]domain.FinishedUnit, error) {
	return nil, nil
}
func (f *fakePets) ListCompanyPets(context.Context, int64) ([]*domain.Pet, error) {
	return f.company, nil
}
func (f *fakePets) LastUnitEndByUsers(context.Context, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}
func (f *fakePets) AddWeeklyKudos(_ context.Context, userID int64, _, _, amount int) error {
	if f.weeklyKudos == nil {
		f.weeklyKudos = map[int64]int{}
	}
	f.weeklyKudos[userID] += amount
	return nil
}
func (f *fakePets) WeeklyKudosCounts(context.Context, int64, int, int) (map[int64]int, error) {
	if f.weeklyKudos == nil {
		return map[int64]int{}, nil
	}
	return f.weeklyKudos, nil
}
func (f *fakePets) StrokesToday(_ context.Context, ownerID, strokerID int64, day time.Time) (int, error) {
	if f.strokes == nil {
		return 0, nil
	}
	return f.strokes[strokeKey(ownerID, strokerID, day)], nil
}
func (f *fakePets) RecordStroke(_ context.Context, ownerID, strokerID int64, day time.Time) error {
	if f.strokes == nil {
		f.strokes = map[string]int{}
	}
	f.strokes[strokeKey(ownerID, strokerID, day)]++
	return nil
}
func (f *fakePets) StrokesTodayByStroker(_ context.Context, strokerID int64, day time.Time) (map[int64]int, error) {
	out := map[int64]int{}
	for ownerID, p := range f.byUser {
		_ = p
		key := strokeKey(ownerID, strokerID, day)
		if n := f.strokes[key]; n > 0 {
			out[ownerID] = n
		}
	}
	return out, nil
}

// errNoPet — фейковая инфраструктурная ошибка «питомец не найден».
var errNoPet = domain.NewError("NO_PET", "нет питомца", 500)

type fakeShop struct {
	domain.ShopRepo
	items     map[string]*domain.ShopItem
	purchases map[int64]int
	// forceSoldOut — имитация гонки: превентивный CountPurchases ещё видит
	// остаток, но атомарный RecordPurchase уже отдаёт SOLD_OUT.
	forceSoldOut bool
}

func newFakeShop() *fakeShop { return &fakeShop{items: map[string]*domain.ShopItem{}} }

func (f *fakeShop) ListActiveItems(_ context.Context, now time.Time) ([]*domain.ShopItem, error) {
	var out []*domain.ShopItem
	for _, it := range f.items {
		if it.Active(now) {
			out = append(out, it)
		}
	}
	return out, nil
}
func (f *fakeShop) GetItem(_ context.Context, key string) (*domain.ShopItem, error) {
	return f.items[key], nil
}
func (f *fakeShop) CountPurchases(_ context.Context, itemID, _ int64) (int, error) {
	if f.purchases == nil {
		return 0, nil
	}
	return f.purchases[itemID], nil
}
func (f *fakeShop) RecordPurchase(_ context.Context, itemID, _, _ int64, quota *int) error {
	if f.purchases == nil {
		f.purchases = map[int64]int{}
	}
	if quota != nil && (f.forceSoldOut || f.purchases[itemID] >= *quota) {
		return domain.ErrSoldOut
	}
	f.purchases[itemID]++
	return nil
}

type fakeActivity struct {
	domain.ActivityRepo
	entries []*domain.ActivityLogEntry
}

func (f *fakeActivity) Append(_ context.Context, petUserID int64, kind string, payload map[string]any) error {
	f.entries = append(f.entries, &domain.ActivityLogEntry{
		PetUserID: petUserID, Kind: kind, Payload: payload, CreatedAt: time.Now(),
	})
	return nil
}
func (f *fakeActivity) ListForPet(_ context.Context, petUserID int64, _ int) ([]*domain.ActivityLogEntry, error) {
	var out []*domain.ActivityLogEntry
	for _, e := range f.entries {
		if e.PetUserID == petUserID {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeActivity) hasKind(kind string) bool {
	for _, e := range f.entries {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

type fakeUsers struct {
	domain.UserReader
	members map[int64]map[int64]bool // userID → companyID → member
}

func (f *fakeUsers) IsCompanyMember(_ context.Context, userID, companyID int64) (bool, error) {
	if f.members == nil {
		return true, nil
	}
	m, ok := f.members[userID]
	if !ok {
		return false, nil
	}
	return m[companyID], nil
}
func (f *fakeUsers) GetUser(context.Context, int64) (*domain.User, error) { return nil, nil }
func (f *fakeUsers) CompanyActive(context.Context, *int64) (bool, error)  { return true, nil }

type fakeCompanies struct{ domain.CompanyReader }

func (fakeCompanies) GrooveEnabled(context.Context, int64) (bool, error) { return true, nil }
func (fakeCompanies) ActiveCompanyIDs(context.Context) ([]int64, error)  { return nil, nil }
func (fakeCompanies) WeekendDays(context.Context, int64) ([]int, error) {
	return append([]int{}, domain.DefaultWeekend...), nil
}

type fakeWork struct {
	domain.WorkReader
	units []*domain.ActiveUnit
}

func (f *fakeWork) ListActiveUnits(context.Context, int64) ([]*domain.ActiveUnit, error) {
	return f.units, nil
}

// capDaily — фейк дневных капов с реальным учётом бюджета.
type capDaily struct {
	domain.Daily
	used  map[string]int
	cache map[string]string
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
func (f *capDaily) GetCache(_ context.Context, key string) string {
	if f.cache == nil {
		return ""
	}
	return f.cache[key]
}
func (f *capDaily) SetCache(_ context.Context, key, value string, _ time.Duration) {
	if f.cache == nil {
		f.cache = map[string]string{}
	}
	f.cache[key] = value
}
func (f *capDaily) Exists(_ context.Context, key string) bool {
	if f.cache == nil {
		return false
	}
	_, ok := f.cache[key]
	return ok
}

// fakeDaily — фейк, выдающий бюджет целиком (или ничего при denyBudget).
type fakeDaily struct {
	domain.Daily
	denyBudget bool
	cache      map[string]string
}

func (f *fakeDaily) TakeBudget(_ context.Context, _ int64, _ string, want, _ int) int {
	if f.denyBudget {
		return 0
	}
	return want
}
func (f *fakeDaily) Left(context.Context, int64, string, int) int { return 1 }
func (f *fakeDaily) GetCache(_ context.Context, key string) string {
	if f.cache == nil {
		return ""
	}
	return f.cache[key]
}
func (f *fakeDaily) SetCache(_ context.Context, key, value string, _ time.Duration) {
	if f.cache == nil {
		f.cache = map[string]string{}
	}
	f.cache[key] = value
}
func (f *fakeDaily) Exists(_ context.Context, key string) bool {
	if f.cache == nil {
		return false
	}
	_, ok := f.cache[key]
	return ok
}

type fakePub struct {
	domain.EventPublisher
	events []string
}

func (f *fakePub) Publish(_ context.Context, event string, _ []string, _ any) {
	f.events = append(f.events, event)
}

// testEnv — окружение сервиса на фейках всех портов.
type testEnv struct {
	pets     *fakePets
	shop     *fakeShop
	activity *fakeActivity
	users    *fakeUsers
	daily    *capDaily
	pub      *fakePub
	svc      *Service
}

func newEnv() *testEnv {
	pets := &fakePets{}
	shop := newFakeShop()
	activity := &fakeActivity{}
	users := &fakeUsers{}
	daily := &capDaily{}
	pub := &fakePub{}
	svc := New(pets, shop, activity, users, fakeCompanies{}, &fakeWork{}, daily, pub,
		nil, slog.New(slog.DiscardHandler))
	return &testEnv{pets: pets, shop: shop, activity: activity, users: users, daily: daily, pub: pub, svc: svc}
}

// newTestService — окружение с fakeDaily (выдаёт бюджет целиком), как раньше
// использовалось для тестов кормления.
func newTestService(pets *fakePets, daily *fakeDaily, pub *fakePub, activity *fakeActivity) *Service {
	return New(pets, newFakeShop(), activity, &fakeUsers{}, fakeCompanies{}, &fakeWork{}, daily, pub,
		nil, slog.New(slog.DiscardHandler))
}

// ── FeedPet на фейках портов ───────────────────────────────────────

func TestFeedPetHappyPath(t *testing.T) {
	pets := &fakePets{}
	daily := &fakeDaily{}
	pub := &fakePub{}
	activity := &fakeActivity{}
	svc := newTestService(pets, daily, pub, activity)

	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Kudos = 10

	data, err := svc.FeedPet(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if data.Kudos != 10-domain.FeedCost {
		t.Errorf("kudos = %d", data.Kudos)
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
	if !activity.hasKind("fed") {
		t.Errorf("нет записи истории fed: %+v", activity.entries)
	}
}

func TestFeedPetNoKudos(t *testing.T) {
	pets := &fakePets{}
	svc := newTestService(pets, &fakeDaily{}, &fakePub{}, &fakeActivity{})
	pets.GetOrCreate(context.Background(), 1, 10) // kudos = 0

	_, err := svc.FeedPet(context.Background(), 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "NO_KUDOS" {
		t.Fatalf("ожидался NO_KUDOS, got %v", err)
	}
}

func TestFeedPetDailyCap(t *testing.T) {
	pets := &fakePets{}
	svc := newTestService(pets, &fakeDaily{denyBudget: true}, &fakePub{}, &fakeActivity{})
	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Kudos = 10

	_, err := svc.FeedPet(context.Background(), 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "FED_ENOUGH" || de.HTTPStatus != 429 {
		t.Fatalf("ожидался FED_ENOUGH/429, got %v", err)
	}
}

func TestFeedPetEvolves(t *testing.T) {
	pets := &fakePets{}
	activity := &fakeActivity{}
	svc := newTestService(pets, &fakeDaily{}, &fakePub{}, activity)
	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Kudos = 10
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
	if !activity.hasKind("evolved") {
		t.Errorf("нет записи истории evolved: %+v", activity.entries)
	}
}

// ── Квест дня: прогресс и награда ───────────────────────────────────

func TestClaimQuestRewardsKudos(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	env.svc.ensureTodayQuest(pet)
	target := 0
	if pet.QuestTarget != nil {
		target = *pet.QuestTarget
	}
	pet.QuestProgress = target
	env.pets.SavePet(ctx, pet)

	data, err := env.svc.ClaimQuest(ctx, 1, 10)
	if err != nil {
		t.Fatalf("ClaimQuest: %v", err)
	}
	if data.Kudos != domain.QuestRewardKudos {
		t.Errorf("kudos = %d, want %d", data.Kudos, domain.QuestRewardKudos)
	}
	if _, err := env.svc.ClaimQuest(ctx, 1, 10); domain.AsDomainError(err) == nil {
		t.Fatal("повторный клейм должен быть ALREADY_CLAIMED")
	}
}

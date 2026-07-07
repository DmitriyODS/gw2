package service

// Тесты экономики питомцев: начисление кудосов с дневными капами, кормление,
// прямой XP за работу, эволюция, прогулка/лечение/поглаживание и магазин.

import (
	"context"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

func intp(n int) *int { return &n }

// ── AwardKudos: дневные капы по источникам ─────────────────────────

func TestAwardKudosRespectsDailyCap(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	// Источник «unit», кап 15: 10 + 10 → второй раз урезается до 5.
	if got := env.svc.AwardKudos(ctx, 1, 10, "unit", 10); got != 10 {
		t.Errorf("первое начисление: %d, want 10", got)
	}
	if got := env.svc.AwardKudos(ctx, 1, 10, "unit", 10); got != domain.DailyCaps["unit"]-10 {
		t.Errorf("второе начисление: %d, want %d", got, domain.DailyCaps["unit"]-10)
	}
	if got := env.svc.AwardKudos(ctx, 1, 10, "unit", 1); got != 0 {
		t.Errorf("сверх капа: %d, want 0", got)
	}
	if env.pets.byUser[1].Kudos != domain.DailyCaps["unit"] {
		t.Errorf("kudos = %d, want %d", env.pets.byUser[1].Kudos, domain.DailyCaps["unit"])
	}
	// Каждое успешное начисление эмитит pet:update, и идёт в счётчик недели.
	if len(env.pub.events) != 2 {
		t.Errorf("события: %v", env.pub.events)
	}
	if env.pets.weeklyKudos[1] != domain.DailyCaps["unit"] {
		t.Errorf("weekly kudos = %d, want %d", env.pets.weeklyKudos[1], domain.DailyCaps["unit"])
	}
}

// Начисления из хуков обязаны идти атомарным AdjustBalances, а не full-row
// SavePet: иначе конкурентная покупка/кормление перетирается устаревшим
// снимком (lost-update).
func TestAwardKudosUsesAtomicAdjustNotSavePet(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)
	env.pets.saves = 0

	if got := env.svc.AwardKudos(ctx, 1, 10, "unit", 5); got != 5 {
		t.Fatalf("granted = %d, want 5", got)
	}
	if env.pets.adjustCalls != 1 {
		t.Errorf("adjustCalls = %d, want 1", env.pets.adjustCalls)
	}
	if env.pets.saves != 0 {
		t.Errorf("AwardKudos не должен звать full-row SavePet, saves = %d", env.pets.saves)
	}
}

func TestAwardXPUsesAtomicAdjustAndNarrowEvolutionSave(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.XP = domain.StageXP[1] - 1 // следующий XP эволюционирует
	env.pets.saves = 0

	if got := env.svc.AwardXP(ctx, 1, 10, "xp_task", domain.XPTaskClosed, domain.XPTaskDailyCap); got != domain.XPTaskClosed {
		t.Fatalf("granted = %d", got)
	}
	if env.pets.adjustCalls != 1 {
		t.Errorf("adjustCalls = %d, want 1", env.pets.adjustCalls)
	}
	if env.pets.evolutionSaves != 1 {
		t.Errorf("evolutionSaves = %d, want 1", env.pets.evolutionSaves)
	}
	if env.pets.saves != 0 {
		t.Errorf("AwardXP не должен звать full-row SavePet, saves = %d", env.pets.saves)
	}
	if env.pets.byUser[1].Stage != 1 {
		t.Errorf("stage = %d, want 1", env.pets.byUser[1].Stage)
	}
}

func TestAwardKudosUnknownSourceDefaultCap(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	if got := env.svc.AwardKudos(ctx, 1, 10, "mystery", 99); got != domain.DefaultDailyCap {
		t.Errorf("неизвестный источник: %d, want %d", got, domain.DefaultDailyCap)
	}
}

// ── Прямой XP за работу ────────────────────────────────────────────

func TestAwardXPForUnitMinutes(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	// 30 минут юнита → 10 XP (по 1 за каждые 3 минуты).
	got := env.svc.AwardXP(ctx, 1, 10, "xp_unit", 30/domain.XPUnitMinutesPer, domain.XPUnitDailyCap)
	if got != 10 {
		t.Errorf("granted = %d, want 10", got)
	}
	if env.pets.byUser[1].XP != 10 {
		t.Errorf("xp = %d, want 10", env.pets.byUser[1].XP)
	}
	if len(env.pub.events) == 0 || env.pub.events[len(env.pub.events)-1] != "pet:update" {
		t.Errorf("события: %v", env.pub.events)
	}
}

func TestAwardXPDailyCap(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	env.svc.AwardXP(ctx, 1, 10, "xp_unit", 35, domain.XPUnitDailyCap)
	if got := env.svc.AwardXP(ctx, 1, 10, "xp_unit", 20, domain.XPUnitDailyCap); got != 5 {
		t.Errorf("второе начисление: %d, want 5 (остаток капа)", got)
	}
	if got := env.svc.AwardXP(ctx, 1, 10, "xp_unit", 1, domain.XPUnitDailyCap); got != 0 {
		t.Errorf("сверх капа: %d, want 0", got)
	}
	if env.pets.byUser[1].XP != domain.XPUnitDailyCap {
		t.Errorf("xp = %d, want %d", env.pets.byUser[1].XP, domain.XPUnitDailyCap)
	}
}

func TestAwardXPFedBoost(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	fed := todayMSK()
	pet.LastFedDate = &fed // сегодня кормлен → сытость ×1.5

	if got := env.svc.AwardXP(ctx, 1, 10, "xp_unit", 10, domain.XPUnitDailyCap); got != 15 {
		t.Errorf("granted = %d, want 15 (10 × 1.5)", got)
	}
	if env.pets.byUser[1].XP != 15 {
		t.Errorf("xp = %d, want 15", env.pets.byUser[1].XP)
	}
}

func TestAwardXPFrozenWhileSick(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	now := time.Now()
	pet.SickSince = &now

	if got := env.svc.AwardXP(ctx, 1, 10, "xp_unit", 10, domain.XPUnitDailyCap); got != 0 {
		t.Errorf("больной питомец получил XP: %d", got)
	}
	if env.pets.byUser[1].XP != 0 {
		t.Errorf("xp = %d, want 0", env.pets.byUser[1].XP)
	}
	// Бюджет капа не тратится впустую.
	if env.daily.used["xp_unit"] != 0 {
		t.Errorf("бюджет потрачен: %d", env.daily.used["xp_unit"])
	}
}

func TestAwardXPTriggersEvolution(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.XP = domain.StageXP[1] - 5

	if got := env.svc.AwardXP(ctx, 1, 10, "xp_task", domain.XPTaskClosed, domain.XPTaskDailyCap); got != domain.XPTaskClosed {
		t.Errorf("granted = %d", got)
	}
	if env.pets.byUser[1].Stage != 1 {
		t.Errorf("stage = %d, want 1", env.pets.byUser[1].Stage)
	}
	if !env.activity.hasKind("evolved") {
		t.Errorf("нет записи истории evolved: %v", env.activity.entries)
	}
}

// ── Хук юнита: кудосы + XP одним завершением ───────────────────────

func TestOnUnitStoppedAwardsKudosAndXP(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	env.svc.OnUnitStopped(ctx, UnitHook{CompanyID: 10, UserID: 1, UnitID: 7,
		UnitName: "Юнит", Minutes: 30})

	if env.pets.byUser[1].Kudos != 30/5 {
		t.Errorf("kudos = %d, want %d", env.pets.byUser[1].Kudos, 30/5)
	}
	if env.pets.byUser[1].XP != 30/domain.XPUnitMinutesPer {
		t.Errorf("xp = %d, want %d", env.pets.byUser[1].XP, 30/domain.XPUnitMinutesPer)
	}
}

func TestOnTaskClosedNoHeroNoAward(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	env.svc.OnTaskClosed(ctx, 10, 0, 5, "Задача")
	if env.pets.byUser[1].Kudos != 0 {
		t.Errorf("без героя не должно быть начисления: kudos = %d", env.pets.byUser[1].Kudos)
	}
}

// ── Прогулка ─────────────────────────────────────────────────────────

func TestWalkPetChargesKudosAndGrantsXP(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = domain.WalkCost

	data, err := env.svc.WalkPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("WalkPet: %v", err)
	}
	if data.Kudos != 0 {
		t.Errorf("kudos = %d, want 0", data.Kudos)
	}
	if data.XP != domain.WalkXP {
		t.Errorf("xp = %d, want %d", data.XP, domain.WalkXP)
	}
	if !env.activity.hasKind("walked") {
		t.Error("нет записи истории walked")
	}
}

func TestWalkPetDailyCap(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = domain.WalkCost * (domain.WalkDailyMax + 1)

	for i := 0; i < domain.WalkDailyMax; i++ {
		if _, err := env.svc.WalkPet(ctx, 1, 10); err != nil {
			t.Fatalf("прогулка %d: %v", i, err)
		}
	}
	_, err := env.svc.WalkPet(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "WALKED_ENOUGH" {
		t.Fatalf("ожидался WALKED_ENOUGH, got %v", err)
	}
}

func TestWalkPetNoKudos(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	_, err := env.svc.WalkPet(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "NO_KUDOS" {
		t.Fatalf("ожидался NO_KUDOS, got %v", err)
	}
}

// ── Лечение ──────────────────────────────────────────────────────────

func TestHealPetRequiresSick(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = domain.HealCost

	_, err := env.svc.HealPet(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "NOT_SICK" {
		t.Fatalf("ожидался NOT_SICK, got %v", err)
	}
}

func TestHealPetRecovers(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = domain.HealCost * domain.RecoveryTarget
	now := time.Now()
	pet.SickSince = &now
	pet.Recovery = domain.RecoveryTarget - domain.HealRecoveryPoints

	data, err := env.svc.HealPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("HealPet: %v", err)
	}
	if data.Recovered == nil || !*data.Recovered {
		t.Error("ожидалось выздоровление")
	}
	if data.Sick {
		t.Error("питомец всё ещё болен")
	}
	if !env.activity.hasKind("healed") || !env.activity.hasKind("recovered") {
		t.Errorf("нет записей истории: %v", env.activity.entries)
	}
}

// ── Поглаживание чужого питомца ──────────────────────────────────────

func TestStrokePetChargesStrokerAndBoostsOwner(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.users.members = map[int64]map[int64]bool{
		1: {10: true}, 2: {10: true},
	}
	stroker, _ := env.pets.GetOrCreate(ctx, 2, 10)
	stroker.Kudos = domain.StrokeCost
	owner, _ := env.pets.GetOrCreate(ctx, 1, 10)
	ownerXPBefore := owner.XP

	if _, err := env.svc.StrokePet(ctx, 2, 1, 10); err != nil {
		t.Fatalf("StrokePet: %v", err)
	}
	if env.pets.byUser[2].Kudos != 0 {
		t.Errorf("у гладящего должны списаться кудосы: %d", env.pets.byUser[2].Kudos)
	}
	if env.pets.byUser[1].XP != ownerXPBefore+domain.StrokeMoodXP {
		t.Errorf("владельцу должен начислиться XP настроения: %d", env.pets.byUser[1].XP)
	}
	used, _ := env.pets.StrokesToday(ctx, 1, 2, todayMSK())
	if used != 1 {
		t.Errorf("StrokesToday = %d, want 1", used)
	}
	if !env.activity.hasKind("stroked_by") {
		t.Error("нет записи истории stroked_by")
	}
}

func TestStrokePetDailyLimitPerPet(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.users.members = map[int64]map[int64]bool{1: {10: true}, 2: {10: true}}
	stroker, _ := env.pets.GetOrCreate(ctx, 2, 10)
	stroker.Kudos = domain.StrokeCost * (domain.StrokeDailyMaxPerPet + 1)
	env.pets.GetOrCreate(ctx, 1, 10)

	for i := 0; i < domain.StrokeDailyMaxPerPet; i++ {
		if _, err := env.svc.StrokePet(ctx, 2, 1, 10); err != nil {
			t.Fatalf("поглаживание %d: %v", i, err)
		}
	}
	_, err := env.svc.StrokePet(ctx, 2, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "STROKED_ENOUGH" {
		t.Fatalf("ожидался STROKED_ENOUGH (лимит %d/день), got %v", domain.StrokeDailyMaxPerPet, err)
	}
}

func TestStrokePetSelfForbidden(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	_, err := env.svc.StrokePet(ctx, 1, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "SELF_STROKE" {
		t.Fatalf("ожидался SELF_STROKE, got %v", err)
	}
}

func TestStrokePetRequiresCompanyMembership(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.users.members = map[int64]map[int64]bool{2: {10: true}} // владелец не член компании

	_, err := env.svc.StrokePet(ctx, 2, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "USER_NOT_FOUND" {
		t.Fatalf("ожидался USER_NOT_FOUND, got %v", err)
	}
}

// ── Магазин: постоянные/лимитированные/достижимые товары ────────────

func TestBuyItemLimitedQuotaSoldOut(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.shop.items["party"] = &domain.ShopItem{
		ID: 1, Key: "party", Kind: "accessory", Rarity: "common",
		PriceKudos: 10, UnlockKind: "shop", LimitedQuota: intp(1),
	}
	env.shop.purchases = map[int64]int{1: 1} // тираж уже выбран

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 10
	_, err := env.svc.BuyItem(ctx, 1, 10, "party")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "SOLD_OUT" {
		t.Fatalf("ожидался SOLD_OUT, got %v", err)
	}
}

// Гонка тиража: превентивный COUNT ещё видит остаток, но атомарный
// RecordPurchase (COUNT+INSERT в одной транзакции под локом товара) уже
// отдаёт SOLD_OUT — покупка отклоняется, кудосы не сохраняются.
func TestBuyItemLimitedQuotaRaceSoldOut(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.shop.items["party"] = &domain.ShopItem{
		ID: 1, Key: "party", Kind: "accessory", Rarity: "common",
		PriceKudos: 10, UnlockKind: "shop", LimitedQuota: intp(1),
	}
	env.shop.forceSoldOut = true // конкурент успел выкупить между COUNT и INSERT

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 10
	env.pets.saves = 0

	_, err := env.svc.BuyItem(ctx, 1, 10, "party")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "SOLD_OUT" {
		t.Fatalf("ожидался SOLD_OUT из атомарного RecordPurchase, got %v", err)
	}
	if env.pets.saves != 0 {
		t.Errorf("при SOLD_OUT питомец не должен сохраняться (кудосы не списаны), saves = %d", env.pets.saves)
	}
}

func TestBuyItemAchievementNotPurchasable(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.shop.items["legend"] = &domain.ShopItem{
		ID: 2, Key: "legend", Kind: "accessory", Rarity: "legendary",
		PriceKudos: 0, UnlockKind: "achievement",
	}
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 100

	_, err := env.svc.BuyItem(ctx, 1, 10, "legend")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "ACHIEVEMENT_ONLY" {
		t.Fatalf("ожидался ACHIEVEMENT_ONLY, got %v", err)
	}
}

func TestBuyItemSuccessEquipsAndLogs(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.shop.items["cap"] = &domain.ShopItem{
		ID: 3, Key: "cap", Kind: "accessory", Rarity: "common",
		PriceKudos: 10, UnlockKind: "shop",
	}
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 10

	data, err := env.svc.BuyItem(ctx, 1, 10, "cap")
	if err != nil {
		t.Fatalf("BuyItem: %v", err)
	}
	if data.Kudos != 0 || data.Hat == nil || *data.Hat != "cap" {
		t.Errorf("питомец после покупки: %+v", data)
	}
	if !env.activity.hasKind("item_bought") {
		t.Error("нет записи истории item_bought")
	}
	// Повторная покупка — уже куплено.
	if _, err := env.svc.BuyItem(ctx, 1, 10, "cap"); domain.AsDomainError(err) == nil {
		t.Fatal("ожидался ALREADY_OWNED")
	}
}

func TestGetMysteryItemOncePerDay(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.shop.items["party"] = &domain.ShopItem{
		ID: 1, Key: "party", Kind: "accessory", Rarity: "common",
		PriceKudos: 10, UnlockKind: "shop",
	}
	env.pets.GetOrCreate(ctx, 1, 10)

	item, err := env.svc.GetMysteryItem(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMysteryItem: %v", err)
	}
	if item.Key != "party" {
		t.Errorf("key = %q, want party (единственный кандидат)", item.Key)
	}
	if _, err := env.svc.GetMysteryItem(ctx, 1, 10); domain.AsDomainError(err) == nil {
		t.Fatal("повторный вызов должен быть ALREADY_TAKEN")
	}

	// Витрина отражает состояние мистери-слота (mystery_taken).
	shop, err := env.svc.GetShopState(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetShopState: %v", err)
	}
	if !shop.MysteryTaken {
		t.Error("mystery_taken после получения сюрприза должен быть true")
	}
	shop2, err := env.svc.GetShopState(ctx, 2, 10)
	if err != nil {
		t.Fatalf("GetShopState (другой пользователь): %v", err)
	}
	if shop2.MysteryTaken {
		t.Error("mystery_taken чужого пользователя должен быть false")
	}
}

func TestGetMysteryItemExcludesLimitedAndAchievements(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.shop.items["limited"] = &domain.ShopItem{
		ID: 1, Key: "limited", Kind: "accessory", Rarity: "epic",
		PriceKudos: 50, UnlockKind: "shop", LimitedQuota: intp(5),
	}
	env.shop.items["achievement"] = &domain.ShopItem{
		ID: 2, Key: "achievement", Kind: "accessory", Rarity: "legendary",
		PriceKudos: 0, UnlockKind: "achievement",
	}
	env.pets.GetOrCreate(ctx, 1, 10)

	_, err := env.svc.GetMysteryItem(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "NO_ITEM" {
		t.Fatalf("пул должен быть пуст (лимитированные/достижимые исключены), got %v", err)
	}
}

// ── Рейтинг и «в эфире» ───────────────────────────────────────────────

func TestGetRatingUsesWeeklyKudos(t *testing.T) {
	env := newEnv()
	mk := func(id int64, stage, xp int) *domain.Pet {
		return &domain.Pet{UserID: id, CompanyID: 10, Name: "Питомец", Species: "fox",
			Stage: stage, XP: xp, User: &domain.UserRef{ID: id, FIO: "Сотрудник"},
			Accessories: []string{}, UnlockedSpecies: []string{}}
	}
	env.pets.company = []*domain.Pet{mk(3, 4, 600), mk(1, 2, 200), mk(2, 1, 50)}
	env.pets.weeklyKudos = map[int64]int{2: 4}

	out, err := env.svc.GetRating(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("GetRating: %v", err)
	}
	items := out["items"].([]map[string]any)
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	// Топ недели — по кудосам ISO-недели: user 2 (4 кудоса) выше более
	// прокачанных; ничьи (0 кудосов) — в порядке stage/XP.
	if items[0]["position"] != 1 || items[0]["kudos_week"] != 4 {
		t.Errorf("топ-1: %v", items[0])
	}
	if items[1]["xp"] != 600 || items[2]["xp"] != 200 {
		t.Errorf("ничьи не в порядке stage/XP: %v, %v", items[1], items[2])
	}
	me, _ := out["me"].(map[string]any)
	if me == nil || me["position"] != 3 {
		t.Errorf("me: %v", me)
	}
	if out["total"] != 3 {
		t.Errorf("total = %v", out["total"])
	}
}

func TestGetLiveListsActiveUnits(t *testing.T) {
	env := newEnv()
	env.svc.work = &fakeWork{units: []*domain.ActiveUnit{
		{ID: 1, Name: "Юнит", TaskID: 5, StartedAt: time.Now(),
			User: &domain.UserRef{ID: 1, FIO: "Сотрудник"}},
	}}
	live, err := env.svc.GetLive(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetLive: %v", err)
	}
	if len(live.Items) != 1 || live.Items[0].UnitID != 1 {
		t.Errorf("live: %+v", live.Items)
	}
}

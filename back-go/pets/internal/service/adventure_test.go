package service

import (
	"context"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ── Старт приключения ───────────────────────────────────────────────

func TestStartAdventureHappyPath(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	data, err := env.svc.StartAdventure(ctx, 1, 10)
	if err != nil {
		t.Fatalf("StartAdventure: %v", err)
	}
	if data.AdventureUntil == nil || data.AdventurePlace == nil {
		t.Fatalf("в DTO нет полей приключения: %+v", data)
	}
	if !containsStr(domain.AdventurePlaces, *data.AdventurePlace) {
		t.Errorf("незнакомая локация: %q", *data.AdventurePlace)
	}
	pet := env.pets.byUser[1]
	if pet.AdventureUntil == nil {
		t.Fatal("приключение не сохранено в репозитории")
	}
	dur := time.Until(*pet.AdventureUntil)
	if dur < time.Duration(domain.AdventureMinMinutes-1)*time.Minute ||
		dur > time.Duration(domain.AdventureMaxMinutes+1)*time.Minute {
		t.Errorf("длительность вне 2–4 часов: %v", dur)
	}
	if !env.activity.hasKind("adventure_started") {
		t.Errorf("нет записи истории adventure_started: %+v", env.activity.entries)
	}
	if len(env.pub.events) == 0 || env.pub.events[len(env.pub.events)-1] != "pet:update" {
		t.Errorf("события: %v", env.pub.events)
	}
}

func TestStartAdventureSickPet(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	now := time.Now()
	pet.SickSince = &now

	_, err := env.svc.StartAdventure(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "PET_SICK" {
		t.Fatalf("ожидался PET_SICK, got %v", err)
	}
}

func TestStartAdventureAlreadyAway(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	if _, err := env.svc.StartAdventure(ctx, 1, 10); err != nil {
		t.Fatalf("первый старт: %v", err)
	}
	_, err := env.svc.StartAdventure(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "PET_AWAY" || de.HTTPStatus != 422 {
		t.Fatalf("ожидался PET_AWAY/422, got %v", err)
	}
}

func TestStartAdventureDailyCap(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)
	// Кап стартов исчерпан (источник 'adventure' в daily).
	env.daily.TakeBudget(ctx, 1, "adventure", domain.AdventureDailyMax, domain.AdventureDailyMax)

	_, err := env.svc.StartAdventure(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "ADVENTURE_LIMIT" || de.HTTPStatus != 429 {
		t.Fatalf("ожидался ADVENTURE_LIMIT/429, got %v", err)
	}
}

// ── Ленивый возврат и награда ───────────────────────────────────────

// sendPetAway — питомец в приключении, срок уже истёк (для тестов возврата).
func sendPetAway(t *testing.T, env *testEnv, userID int64) {
	t.Helper()
	pet, _ := env.pets.GetOrCreate(context.Background(), userID, 10)
	until := time.Now().UTC().Add(-time.Minute)
	place := domain.AdventurePlaces[0]
	pet.AdventureUntil, pet.AdventurePlace = &until, &place
}

func TestAdventureReturnAwardsOnce(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	sendPetAway(t, env, 1)

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	r := data.AdventureReward
	if r == nil {
		t.Fatal("возврат не принёс награду")
	}
	if r.Kudos < domain.AdventureKudosMin || r.Kudos > domain.AdventureKudosMax {
		t.Errorf("кудосы вне диапазона: %d", r.Kudos)
	}
	if r.XP < domain.AdventureXPMin || r.XP > domain.AdventureXPMax {
		t.Errorf("XP вне диапазона: %d", r.XP)
	}
	if r.Place != domain.AdventurePlaces[0] {
		t.Errorf("локация награды: %q", r.Place)
	}
	if data.AdventureUntil != nil || data.AdventurePlace != nil {
		t.Error("поля приключения не очищены после возврата")
	}
	if data.Kudos != r.Kudos || data.XP != r.XP {
		t.Errorf("балансы не совпали с наградой: kudos=%d xp=%d, награда %+v",
			data.Kudos, data.XP, r)
	}
	// Начисление — через атомарный AdjustBalances, не full-row SavePet.
	if env.pets.adjustCalls != 1 {
		t.Errorf("adjustCalls = %d, want 1", env.pets.adjustCalls)
	}
	if !env.activity.hasKind("adventure_returned") {
		t.Errorf("нет записи истории adventure_returned: %+v", env.activity.entries)
	}
	// Кудосы приключения попадают в недельный счётчик признания.
	if env.pets.weeklyKudos[1] != r.Kudos {
		t.Errorf("weeklyKudos = %d, want %d", env.pets.weeklyKudos[1], r.Kudos)
	}

	// Второй GET: возврат уже зафиксирован — награды нет, балансы прежние.
	data2, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("повторный GetMyPet: %v", err)
	}
	if data2.AdventureReward != nil {
		t.Error("повторный GET начислил награду второй раз")
	}
	if data2.Kudos != data.Kudos || data2.XP != data.XP {
		t.Errorf("балансы изменились без начисления: %d/%d → %d/%d",
			data.Kudos, data.XP, data2.Kudos, data2.XP)
	}
	if env.pets.adjustCalls != 1 {
		t.Errorf("adjustCalls после второго GET = %d, want 1", env.pets.adjustCalls)
	}
}

// ── Гейты платных действий ──────────────────────────────────────────

func TestAdventureBlocksPaidActions(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 100
	if _, err := env.svc.StartAdventure(ctx, 1, 10); err != nil {
		t.Fatalf("StartAdventure: %v", err)
	}

	if _, err := env.svc.FeedPet(ctx, 1, 10, ""); domain.AsDomainError(err) == nil ||
		domain.AsDomainError(err).Code != "PET_AWAY" {
		t.Errorf("FeedPet в пути: %v", err)
	}
	if _, err := env.svc.WalkPet(ctx, 1, 10); domain.AsDomainError(err) == nil ||
		domain.AsDomainError(err).Code != "PET_AWAY" {
		t.Errorf("WalkPet в пути: %v", err)
	}
	if _, err := env.svc.HealPet(ctx, 1, 10); domain.AsDomainError(err) == nil ||
		domain.AsDomainError(err).Code != "PET_AWAY" {
		t.Errorf("HealPet в пути: %v", err)
	}

	// Чужой питомец в пути — поглаживание тоже недоступно.
	stroker, _ := env.pets.GetOrCreate(ctx, 2, 10)
	stroker.Kudos = 10
	if _, err := env.svc.StrokePet(ctx, 2, 1, 10); domain.AsDomainError(err) == nil ||
		domain.AsDomainError(err).Code != "PET_AWAY" {
		t.Errorf("StrokePet чужого в пути: %v", err)
	}
}

// Истёкшее приключение платное действие НЕ блокирует: гейт сначала лениво
// фиксирует возврат (с наградой), затем действие проходит.
func TestExpiredAdventureDoesNotBlockAndReturns(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	sendPetAway(t, env, 1)
	env.pets.byUser[1].Kudos = 100

	data, err := env.svc.WalkPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("WalkPet после истёкшего приключения: %v", err)
	}
	if data.AdventureUntil != nil {
		t.Error("поля приключения не очищены")
	}
	if !env.activity.hasKind("adventure_returned") {
		t.Errorf("возврат не зафиксирован: %+v", env.activity.entries)
	}
}

package service

// Тесты потребностей: ленивое убывание шкал, болезни по «своей» шкале,
// рецепты лечения (верный/неверный) и побег заброшенного питомца.

import (
	"context"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// hoursAgo — момент в прошлом (стартовая точка убывания потребностей).
func hoursAgo(h int) time.Time { return time.Now().UTC().Add(-time.Duration(h) * time.Hour) }

// ── Убывание и болезни ──────────────────────────────────────────────

func TestNeedsDecayOverTime(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.NeedsAt = hoursAgo(5) // 10 тиков по 30 минут

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if want := domain.NeedMax - 10*2; data.Needs.Satiety != want {
		t.Errorf("сытость = %d, want %d", data.Needs.Satiety, want)
	}
	if want := domain.NeedMax - 10; data.Needs.Energy != want {
		t.Errorf("энергия = %d, want %d", data.Needs.Energy, want)
	}
}

// Дробный хвост меньше тика не сгорает: частый поллинг клиента не должен
// останавливать убывание (needs_at сдвигается только на целые тики).
func TestNeedsDecayKeepsSubTickRemainder(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	start := hoursAgo(1).Add(-20 * time.Minute) // 2 тика + 20 минут
	pet.NeedsAt = start

	if _, err := env.svc.GetMyPet(ctx, 1, 10); err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if got := pet.NeedsAt.Sub(start); got != 2*domain.NeedTick {
		t.Errorf("needs_at сдвинут на %v, want %v (хвост в 20 минут должен дожить)", got, 2*domain.NeedTick)
	}
	if want := domain.NeedMax - 2*2; pet.Needs.Satiety != want {
		t.Errorf("сытость = %d, want %d", pet.Needs.Satiety, want)
	}
}

// Пустая сытость — истощение: «не кормишь — болеет», причём независимо от
// работы и выходных.
func TestEmptySatietyCausesHunger(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.NeedsAt = hoursAgo(30) // сытость: 100 − 60 тиков × 2 → 0

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if !data.Sick || data.Ailment == nil || *data.Ailment != domain.AilmentHunger {
		t.Fatalf("ожидалось истощение, got sick=%v ailment=%v", data.Sick, data.Ailment)
	}
	if !env.activity.hasKind("sickness_started") {
		t.Error("нет записи истории sickness_started")
	}
	// Владелец должен узнать о болезни пушем (pushsvc слушает pet:sick).
	if !env.pub.has("pet:sick") {
		t.Errorf("нет события pet:sick: %v", env.pub.events)
	}
}

// Каждая запущенная шкала ведёт в СВОЮ болезнь.
func TestNeedsCauseOwnAilments(t *testing.T) {
	cases := []struct {
		name    string
		needs   domain.NeedValues
		ailment string
	}{
		{"голод", domain.NeedValues{Satiety: 0, Energy: 50, Hygiene: 50, Social: 50}, domain.AilmentHunger},
		{"простуда", domain.NeedValues{Satiety: 50, Energy: 0, Hygiene: 50, Social: 50}, domain.AilmentCold},
		{"грязь", domain.NeedValues{Satiety: 50, Energy: 50, Hygiene: 0, Social: 50}, domain.AilmentGrime},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newEnv()
			ctx := context.Background()
			pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
			pet.Needs = c.needs

			data, err := env.svc.GetMyPet(ctx, 1, 10)
			if err != nil {
				t.Fatalf("GetMyPet: %v", err)
			}
			if data.Ailment == nil || *data.Ailment != c.ailment {
				t.Fatalf("ailment = %v, want %q", data.Ailment, c.ailment)
			}
		})
	}
}

// Пустое общение болезни не даёт — только роняет настроение (и множитель XP).
func TestEmptySocialDoesNotCauseSickness(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Needs = domain.NeedValues{Satiety: 50, Energy: 50, Hygiene: 50, Social: 0}

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if data.Sick {
		t.Errorf("одиночество не должно быть болезнью: ailment = %v", data.Ailment)
	}
	if data.MoodFactor >= 1 {
		t.Errorf("настроение должно тормозить XP: factor = %v", data.MoodFactor)
	}
}

// ── Рецепты лечения ─────────────────────────────────────────────────

// Верный рецепт: одно купание поднимает грязнулю на ноги.
func TestBathCuresGrime(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = domain.BathCost
	pet.Needs.Hygiene = 0
	pet.Fall(domain.AilmentGrime, time.Now().UTC())

	data, err := env.svc.BathPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("BathPet: %v", err)
	}
	if data.Recovered == nil || !*data.Recovered || data.Sick {
		t.Errorf("купание должно вылечить грязнулю: %+v", data)
	}
	if data.Needs.Hygiene != domain.NeedGains[domain.ActionBath][domain.NeedHygiene] {
		t.Errorf("чистота = %d", data.Needs.Hygiene)
	}
	if data.Kudos != 0 {
		t.Errorf("кудосы не списаны: %d", data.Kudos)
	}
}

// Сон бесплатен и лечит простуду, но не бесконечен.
func TestSleepCuresColdAndIsFree(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Needs.Energy = 0
	pet.Fall(domain.AilmentCold, time.Now().UTC())
	pet.Recovery = domain.RecoveryTarget - domain.CureFor(domain.AilmentCold, domain.ActionSleep)

	data, err := env.svc.SleepPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("SleepPet: %v", err)
	}
	if data.Recovered == nil || !*data.Recovered {
		t.Error("сон должен вылечить простуду")
	}
	if data.Kudos != 0 {
		t.Errorf("сон бесплатен, kudos = %d", data.Kudos)
	}

	for i := 1; i < domain.SleepDailyMax; i++ {
		if _, err := env.svc.SleepPet(ctx, 1, 10); err != nil {
			t.Fatalf("сон %d: %v", i, err)
		}
	}
	_, err = env.svc.SleepPet(ctx, 1, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "SLEPT_ENOUGH" {
		t.Fatalf("ожидался SLEPT_ENOUGH, got %v", err)
	}
}

// Неверный рецепт: работа лечит хандру, но истощённому не помогает —
// голодного надо кормить.
func TestWorkCuresOnlyBlues(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Fall(domain.AilmentHunger, time.Now().UTC())

	env.svc.AddRecovery(ctx, 1, 10, 1)
	if env.pets.byUser[1].Recovery != 0 {
		t.Errorf("работа не должна лечить истощение: recovery = %d", env.pets.byUser[1].Recovery)
	}

	pet.Fall(domain.AilmentBlues, time.Now().UTC())
	env.svc.AddRecovery(ctx, 1, 10, 1)
	if env.pets.byUser[1].Recovery != domain.CureFor(domain.AilmentBlues, domain.ActionWork) {
		t.Errorf("работа должна лечить хандру: recovery = %d", env.pets.byUser[1].Recovery)
	}
}

// Бульон истощённому — почти всё лечение, при простуде — символическая помощь.
func TestSickFeedFollowsRecipe(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 10
	pet.Needs.Satiety = 0
	pet.Fall(domain.AilmentHunger, time.Now().UTC())

	data, err := env.svc.FeedPet(ctx, 1, 10, "")
	if err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if data.Recovery != domain.CureFor(domain.AilmentHunger, domain.ActionFeed) {
		t.Errorf("recovery = %d, want %d", data.Recovery,
			domain.CureFor(domain.AilmentHunger, domain.ActionFeed))
	}
	if data.Needs.Satiety != domain.SickFeedSatiety {
		t.Errorf("бульон должен питать: сытость = %d", data.Needs.Satiety)
	}
	if data.Kudos != 10-domain.SickFeedCost {
		t.Errorf("цена бульона: kudos = %d", data.Kudos)
	}
}

// ── Побег ───────────────────────────────────────────────────────────

// Заброшенный питомец уходит: прогресс с нуля, имущество остаётся, хозяину —
// событие (его pushsvc превращает в уведомление).
func TestRunawayResetsProgressKeepsProperty(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Stage, pet.XP, pet.Species = 4, 700, "owl"
	pet.Kudos, pet.Generation = 500, 2
	pet.HouseOwned = []string{"sofa"}
	pet.Accessories = []string{"cap"}
	pet.Fall(domain.AilmentBlues, time.Now().UTC().AddDate(0, 0, -domain.RunawaySickDays-1))

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if data.Runaway == nil || data.Runaway.Ailment != domain.AilmentBlues {
		t.Fatalf("ожидался побег: %+v", data.Runaway)
	}
	if data.Stage != 0 || data.XP != 0 || data.Species != "egg" {
		t.Errorf("прогресс не сброшен: stage=%d xp=%d species=%s", data.Stage, data.XP, data.Species)
	}
	if data.Sick {
		t.Error("новое яйцо не может быть больным")
	}
	if data.Kudos != 500 || len(data.HouseOwned) != 1 || len(data.Accessories) != 1 {
		t.Errorf("имущество должно уцелеть: %+v", data)
	}
	if data.Generation != 2 {
		t.Errorf("поколения престижа не сбрасываются: %d", data.Generation)
	}
	if !env.pub.has("pet:runaway") {
		t.Errorf("нет события pet:runaway: %v", env.pub.events)
	}
	if !env.activity.hasKind("ran_away") {
		t.Error("нет записи истории ran_away")
	}

	// Повторный GET побега не повторяет.
	again, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet (повтор): %v", err)
	}
	if again.Runaway != nil {
		t.Error("побег зафиксирован дважды")
	}
}

// До срока питомец не уходит, но получает предупреждение.
func TestRunawayWarnsBeforeLeaving(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Stage = 3
	sickFor := domain.RunawaySickDays - domain.RunawayWarnDays + 1
	pet.Fall(domain.AilmentBlues, time.Now().UTC().AddDate(0, 0, -sickFor))

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if data.Runaway != nil {
		t.Fatal("рано сбежал")
	}
	if data.Stage != 3 {
		t.Errorf("прогресс тронут: stage = %d", data.Stage)
	}
	if data.RunawayInDays == nil || *data.RunawayInDays != domain.RunawaySickDays-sickFor {
		t.Errorf("предупреждение о побеге: %v", data.RunawayInDays)
	}
}

// Здоровому питомцу побег не грозит, сколько бы он ни жил.
func TestHealthyPetNeverRunsAway(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Stage, pet.XP = 5, 1000

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if data.Runaway != nil || data.Stage != 5 {
		t.Errorf("здоровый питомец не должен сбегать: %+v", data.Runaway)
	}
	if data.RunawayInDays != nil {
		t.Errorf("здоровому не место предупреждению: %v", data.RunawayInDays)
	}
}

// ── Отпуск владельца ────────────────────────────────────────────────

// В отпуске показатели заморожены: шкалы не тают (needs_at лишь сдвигается),
// болезнь не наступает, а её таймер продлевается — после отпуска побег
// отсчитывается с того же места.
func TestVacationFreezesNeedsAndSickness(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.OwnerOnVacation = true
	pet.NeedsAt = hoursAgo(30) // без отпуска сытость дошла бы до 0 и истощения

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if !data.OnVacation {
		t.Error("DTO не помечен on_vacation")
	}
	if data.Needs.Satiety != domain.NeedMax || data.Sick {
		t.Fatalf("показатели не заморожены: satiety=%d sick=%v", data.Needs.Satiety, data.Sick)
	}
	if time.Since(pet.NeedsAt) >= domain.NeedTick {
		t.Errorf("needs_at не сдвинут к текущему моменту: %v", pet.NeedsAt)
	}
}

// Больной питомец отпускника не сбегает, и таймер болезни стоит на паузе.
func TestVacationPausesRunaway(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.OwnerOnVacation = true
	pet.Stage, pet.XP = 3, 500
	sickStart := hoursAgo(24 * (domain.RunawaySickDays + 2))
	pet.Fall(domain.AilmentBlues, sickStart)
	pet.NeedsAt = sickStart

	data, err := env.svc.GetMyPet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetMyPet: %v", err)
	}
	if data.Runaway != nil || data.Stage != 3 {
		t.Fatalf("в отпуске питомец сбежал: %+v", data.Runaway)
	}
	if !pet.SickSince.After(sickStart) {
		t.Error("SickSince не продлён заморозкой")
	}
}

// В отпуске уход и поглаживания закрыты, начисления хуков не растят баланс.
func TestVacationBlocksActionsAndAwards(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.OwnerOnVacation = true
	pet.Kudos = 100

	if _, err := env.svc.FeedPet(ctx, 1, 10, ""); domain.AsDomainError(err) == nil ||
		domain.AsDomainError(err).Code != "PET_ON_VACATION" {
		t.Fatalf("кормление в отпуске: %v", err)
	}
	if _, err := env.svc.StartAdventure(ctx, 1, 10); domain.AsDomainError(err) == nil ||
		domain.AsDomainError(err).Code != "PET_ON_VACATION" {
		t.Fatalf("приключение в отпуске: %v", err)
	}
	if granted := env.svc.AwardKudos(ctx, 1, 10, "unit", 5); granted != 0 {
		t.Errorf("AwardKudos в отпуске начислил %d", granted)
	}
	if granted := env.svc.AwardXP(ctx, 1, 10, "xp_unit", 5, 40); granted != 0 {
		t.Errorf("AwardXP в отпуске начислил %d", granted)
	}
	if pet.Kudos != 100 {
		t.Errorf("баланс изменился: %d", pet.Kudos)
	}

	// Коллега тоже не погладит отпускника.
	stroker, _ := env.pets.GetOrCreate(ctx, 2, 10)
	stroker.Kudos = 10
	if _, err := env.svc.StrokePet(ctx, 2, 1, 10); domain.AsDomainError(err) == nil ||
		domain.AsDomainError(err).Code != "PET_ON_VACATION" {
		t.Fatalf("поглаживание отпускника: %v", err)
	}
}

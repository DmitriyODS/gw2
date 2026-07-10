package service

// Тесты развития после максимальной формы: престиж-поколения, сезонный
// трек наград и домик.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ── Престиж ─────────────────────────────────────────────────────────

func TestPrestigeRequiresMaxStage(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Stage = domain.MaxStage - 1

	if _, err := env.svc.PrestigePet(ctx, 1, 10); err == nil {
		t.Fatal("престиж не «Легенды» должен отклоняться")
	}
}

func TestPrestigeResetsStageKeepsWealth(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Stage = domain.MaxStage
	pet.XP = domain.StageXP[domain.MaxStage] + 100
	pet.Kudos = 77
	pet.Species = "cat"
	pet.UnlockedSpecies = []string{"cat"}
	pet.HouseOwned = []string{"sofa"}

	data, err := env.svc.PrestigePet(ctx, 1, 10)
	if err != nil {
		t.Fatalf("PrestigePet: %v", err)
	}
	if data.Generation != 2 {
		t.Errorf("generation = %d, want 2", data.Generation)
	}
	if data.Stage != 0 || data.XP != 0 || data.Species != "egg" {
		t.Errorf("после перерождения: stage=%d xp=%d species=%s", data.Stage, data.XP, data.Species)
	}
	// Богатство не сгорает: кудосы, купленные виды, домик.
	if data.Kudos != 77 {
		t.Errorf("kudos = %d, want 77", data.Kudos)
	}
	if !containsStr(data.UnlockedSpecies, "cat") {
		t.Errorf("купленный вид потерян: %v", data.UnlockedSpecies)
	}
	if !containsStr(data.HouseOwned, "sofa") {
		t.Errorf("домик потерян: %v", data.HouseOwned)
	}
	// Эксклюзив второго поколения разблокирован.
	if want := domain.PrestigeSpecies[2]; !containsStr(data.UnlockedSpecies, want) {
		t.Errorf("нет эксклюзива поколения %q: %v", want, data.UnlockedSpecies)
	}
	if !env.activity.hasKind("prestige") {
		t.Error("нет записи prestige в истории")
	}
}

func TestPrestigeBlockedWhenSickOrAway(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Stage = domain.MaxStage
	now := time.Now().UTC()
	pet.SickSince = &now

	if _, err := env.svc.PrestigePet(ctx, 1, 10); err == nil {
		t.Fatal("больной питомец не должен перерождаться")
	}

	pet.SickSince = nil
	until := now.Add(time.Hour)
	pet.AdventureUntil = &until
	if _, err := env.svc.PrestigePet(ctx, 1, 10); err == nil {
		t.Fatal("питомец в приключении не должен перерождаться")
	}
}

// ── Сезонный трек ───────────────────────────────────────────────────

func TestAwardKudosFeedsSeasonalCounter(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 1, 10)

	env.svc.AwardKudos(ctx, 1, 10, "unit", 10)

	season, err := env.svc.GetSeason(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetSeason: %v", err)
	}
	if season.Kudos != 10 {
		t.Errorf("сезонные кудосы = %d, want 10", season.Kudos)
	}
	if len(season.Rewards) != len(domain.SeasonTrack) {
		t.Errorf("порогов %d, want %d", len(season.Rewards), len(domain.SeasonTrack))
	}
}

func TestClaimSeasonRewardFlow(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)

	first := domain.SeasonTrack[0] // decor garland на пороге 40

	// Не достигнут → NOT_REACHED.
	if _, err := env.svc.ClaimSeasonReward(ctx, 1, 10, first.Threshold); err == nil {
		t.Fatal("недостигнутый порог не должен отдаваться")
	}

	env.pets.AddSeasonalKudos(ctx, 1, seasonKey(time.Now()), first.Threshold)

	season, err := env.svc.ClaimSeasonReward(ctx, 1, 10, first.Threshold)
	if err != nil {
		t.Fatalf("ClaimSeasonReward: %v", err)
	}
	if !season.Rewards[0].Claimed {
		t.Error("порог не отмечен забранным")
	}
	if !containsStr(pet.HouseOwned, first.Key) {
		t.Errorf("декор-награда не выдана: %v", pet.HouseOwned)
	}
	// Повторный клейм — ALREADY_CLAIMED.
	if _, err := env.svc.ClaimSeasonReward(ctx, 1, 10, first.Threshold); err == nil {
		t.Fatal("двойной клейм должен отклоняться")
	}
	if !env.activity.hasKind("season_reward") {
		t.Error("нет записи season_reward в истории")
	}
}

func TestClaimSeasonKudosReward(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)

	var kudosReward *domain.SeasonReward
	for i := range domain.SeasonTrack {
		if domain.SeasonTrack[i].Kind == "kudos" {
			kudosReward = &domain.SeasonTrack[i]
			break
		}
	}
	if kudosReward == nil {
		t.Skip("в треке нет kudos-награды")
	}
	env.pets.AddSeasonalKudos(ctx, 1, seasonKey(time.Now()), kudosReward.Threshold)

	if _, err := env.svc.ClaimSeasonReward(ctx, 1, 10, kudosReward.Threshold); err != nil {
		t.Fatalf("ClaimSeasonReward: %v", err)
	}
	if pet.Kudos != kudosReward.Amount {
		t.Errorf("kudos = %d, want %d", pet.Kudos, kudosReward.Amount)
	}
}

// ── Домик ───────────────────────────────────────────────────────────

func TestBuyHouseDecorSpendsKudos(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 200

	house, err := env.svc.BuyHouseDecor(ctx, 1, 10, "sofa")
	if err != nil {
		t.Fatalf("BuyHouseDecor: %v", err)
	}
	if pet.Kudos != 200-domain.HouseDecor["sofa"] {
		t.Errorf("kudos = %d", pet.Kudos)
	}
	var sofa bool
	for _, d := range house.Catalog {
		if d.Key == "sofa" && d.Owned {
			sofa = true
		}
	}
	if !sofa {
		t.Error("sofa не отмечен купленным в каталоге")
	}
	// Повторная покупка и покупка без кудосов отклоняются.
	if _, err := env.svc.BuyHouseDecor(ctx, 1, 10, "sofa"); err == nil {
		t.Fatal("повторная покупка должна отклоняться")
	}
	pet.Kudos = 0
	if _, err := env.svc.BuyHouseDecor(ctx, 1, 10, "piano"); err == nil {
		t.Fatal("покупка без кудосов должна отклоняться")
	}
}

func TestBuyHouseDecorSeasonOnlyRejected(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 1000

	// Награды трека (цена 0) не продаются.
	if _, err := env.svc.BuyHouseDecor(ctx, 1, 10, "fireplace"); err == nil {
		t.Fatal("сезонный декор не должен продаваться")
	}
	if _, err := env.svc.BuyHouseDecor(ctx, 1, 10, "no_such_key"); err == nil {
		t.Fatal("неизвестный декор должен отклоняться")
	}
}

func TestArrangeHouseValidation(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.HouseOwned = []string{"sofa", "plant", "chair"}

	house, err := env.svc.ArrangeHouse(ctx, 1, 10, []domain.HouseItem{
		{Key: "plant", X: 20, Y: 30},
		{Key: "sofa", X: 150, Y: -10}, // координаты зажимаются в 0..100
	})
	if err != nil {
		t.Fatalf("ArrangeHouse: %v", err)
	}
	if len(house.Placed) != 2 {
		t.Errorf("placed = %v", house.Placed)
	}
	if house.Placed[1].X != 100 || house.Placed[1].Y != 0 {
		t.Errorf("координаты не зажаты: %+v", house.Placed[1])
	}

	// Не купленное, дубли и превышение лимита отклоняются.
	if _, err := env.svc.ArrangeHouse(ctx, 1, 10, []domain.HouseItem{{Key: "piano"}}); err == nil {
		t.Fatal("расстановка некупленного должна отклоняться")
	}
	if _, err := env.svc.ArrangeHouse(ctx, 1, 10,
		[]domain.HouseItem{{Key: "sofa"}, {Key: "sofa"}}); err == nil {
		t.Fatal("дубли должны отклоняться")
	}
	many := make([]domain.HouseItem, domain.HousePlacedMax+1)
	for i := range many {
		many[i] = domain.HouseItem{Key: "sofa"}
	}
	if _, err := env.svc.ArrangeHouse(ctx, 1, 10, many); err == nil {
		t.Fatal("превышение лимита должно отклоняться")
	}
}

// HouseItem принимает обе формы: объект с координатами и легаси-строку
// (старые данные/клиенты) — та получает дефолтное место, а не 400.
func TestHouseItemUnmarshalLegacyString(t *testing.T) {
	var items []domain.HouseItem
	raw := `["sofa", {"key": "plant", "x": 10, "y": 20}]`
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 2 || items[0].Key != "sofa" || items[1].X != 10 {
		t.Errorf("items = %+v", items)
	}
	if items[0].X <= 0 || items[0].Y <= 0 {
		t.Errorf("легаси-строка без дефолтных координат: %+v", items[0])
	}
}

// ── Ключ сезона ─────────────────────────────────────────────────────

func TestSeasonKeyQuarters(t *testing.T) {
	cases := []struct {
		month time.Month
		want  string
	}{
		{time.January, "2026-Q1"}, {time.March, "2026-Q1"},
		{time.April, "2026-Q2"}, {time.July, "2026-Q3"}, {time.December, "2026-Q4"},
	}
	for _, c := range cases {
		at := time.Date(2026, c.month, 15, 12, 0, 0, 0, domain.MSK)
		if got := seasonKey(at); got != c.want {
			t.Errorf("seasonKey(%v) = %s, want %s", c.month, got, c.want)
		}
	}
	// Конец сезона — первый день следующего квартала.
	end := seasonEnd(time.Date(2026, time.July, 10, 0, 0, 0, 0, domain.MSK))
	if end.Month() != time.October || end.Day() != 1 {
		t.Errorf("seasonEnd = %v", end)
	}
}

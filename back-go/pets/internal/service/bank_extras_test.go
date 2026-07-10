package service

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ── Копилки-цели ────────────────────────────────────────────────────

func TestGoalLifecycle(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 100

	bank, err := env.svc.CreateGoal(ctx, 1, 10, "На дракона", "🐉", 50)
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if len(bank.Goals) != 1 || bank.Goals[0].Title != "На дракона" {
		t.Fatalf("копилка не создана: %+v", bank.Goals)
	}
	goalID := bank.Goals[0].ID

	// Пополнение двигает кошелёк → копилку.
	bank, err = env.svc.GoalDeposit(ctx, 1, 10, goalID, 30)
	if err != nil {
		t.Fatalf("GoalDeposit: %v", err)
	}
	if bank.Kudos != 70 || bank.Goals[0].Saved != 30 {
		t.Errorf("после пополнения: kudos=%d saved=%d", bank.Kudos, bank.Goals[0].Saved)
	}
	if bank.GoalAchieved != nil {
		t.Error("цель не должна быть достигнута")
	}
	// Достижение цели — разовый маркер goal_achieved.
	bank, err = env.svc.GoalDeposit(ctx, 1, 10, goalID, 20)
	if err != nil {
		t.Fatalf("GoalDeposit(достижение): %v", err)
	}
	if bank.GoalAchieved == nil || !bank.Goals[0].Achieved {
		t.Error("достижение цели не отмечено")
	}
	if !env.activity.hasKind("goal_achieved") {
		t.Error("нет записи истории goal_achieved")
	}
	// Повторное пополнение достигнутой — без повторного маркера.
	bank, _ = env.svc.GoalDeposit(ctx, 1, 10, goalID, 5)
	if bank.GoalAchieved != nil {
		t.Error("повторный goal_achieved")
	}

	// Снятие возвращает в кошелёк.
	bank, err = env.svc.GoalWithdraw(ctx, 1, 10, goalID, 15)
	if err != nil {
		t.Fatalf("GoalWithdraw: %v", err)
	}
	if bank.Kudos != 60 || bank.Goals[0].Saved != 40 {
		t.Errorf("после снятия: kudos=%d saved=%d", bank.Kudos, bank.Goals[0].Saved)
	}
	// Удаление возвращает остаток.
	bank, err = env.svc.DeleteGoal(ctx, 1, 10, goalID)
	if err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	if bank.Kudos != 100 || len(bank.Goals) != 0 {
		t.Errorf("после удаления: kudos=%d goals=%d", bank.Kudos, len(bank.Goals))
	}
	kinds := env.bank.kinds(1)
	if !slices.Contains(kinds, "goal_deposit") || !slices.Contains(kinds, "goal_withdraw") {
		t.Errorf("леджер копилки: %v", kinds)
	}
}

func TestGoalGuards(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 10

	if _, err := env.svc.CreateGoal(ctx, 1, 10, "", "", 50); errCode(err) != "VALIDATION" {
		t.Errorf("пустое название: %v", err)
	}
	if _, err := env.svc.CreateGoal(ctx, 1, 10, "x", "", 0); errCode(err) != "VALIDATION" {
		t.Errorf("нулевая цель: %v", err)
	}
	for i := 0; i < domain.GoalsMax; i++ {
		if _, err := env.svc.CreateGoal(ctx, 1, 10, "цель", "", 10); err != nil {
			t.Fatalf("копилка %d: %v", i, err)
		}
	}
	if _, err := env.svc.CreateGoal(ctx, 1, 10, "лишняя", "", 10); errCode(err) != "GOALS_LIMIT" {
		t.Errorf("лимит копилок: %v", err)
	}
	goalID := env.bank.goals[0].ID
	if _, err := env.svc.GoalDeposit(ctx, 1, 10, goalID, 999); errCode(err) != "NO_KUDOS" {
		t.Errorf("пополнение без средств: %v", err)
	}
	if _, err := env.svc.GoalWithdraw(ctx, 1, 10, goalID, 5); errCode(err) != "NO_SAVINGS" {
		t.Errorf("снятие из пустой: %v", err)
	}
	if _, err := env.svc.DeleteGoal(ctx, 1, 10, 999); errCode(err) != "NOT_FOUND" {
		t.Errorf("удаление чужой/несуществующей: %v", err)
	}
}

// ── Благотворительные сборы ─────────────────────────────────────────

func TestFundLifecycle(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	creator, _ := env.pets.GetOrCreate(ctx, 1, 10)
	creator.Kudos = 100
	donor, _ := env.pets.GetOrCreate(ctx, 2, 10)
	donor.Kudos = 100

	// Сотрудник (роль 1) создать сбор не может.
	if _, err := env.svc.CreateFund(ctx, 1, 10, 1, "На пиццу", "", "🍕", 50); errCode(err) != "FORBIDDEN" {
		t.Errorf("сбор от сотрудника: %v", err)
	}
	bank, err := env.svc.CreateFund(ctx, 1, 10, 2, "На пиццу", "отметим релиз", "🍕", 50)
	if err != nil {
		t.Fatalf("CreateFund: %v", err)
	}
	if len(bank.Funds) != 1 || bank.Funds[0].Status != "active" {
		t.Fatalf("сбор не создан: %+v", bank.Funds)
	}
	fundID := bank.Funds[0].ID
	if !slices.Contains(env.pub.events, "bank:fund") {
		t.Errorf("нет события bank:fund: %v", env.pub.events)
	}

	// Взносы двух коллег; второй закрывает цель.
	bank, err = env.svc.DonateFund(ctx, 1, 10, fundID, 30)
	if err != nil {
		t.Fatalf("DonateFund: %v", err)
	}
	if bank.Kudos != 70 || bank.Funds[0].Collected != 30 || bank.FundCompleted != nil {
		t.Errorf("после первого взноса: kudos=%d collected=%d", bank.Kudos, bank.Funds[0].Collected)
	}
	bank, err = env.svc.DonateFund(ctx, 2, 10, fundID, 20)
	if err != nil {
		t.Fatalf("DonateFund(финиш): %v", err)
	}
	if bank.FundCompleted == nil || bank.Funds[0].Status != "done" {
		t.Errorf("закрытие цели не отмечено: %+v", bank.Funds[0])
	}
	// В завершённый сбор взнос не проходит.
	if _, err := env.svc.DonateFund(ctx, 1, 10, fundID, 5); errCode(err) != "FUND_CLOSED" {
		t.Errorf("взнос в завершённый: %v", err)
	}
	if !slices.Contains(env.bank.kinds(1), "charity") || !slices.Contains(env.bank.kinds(2), "charity") {
		t.Errorf("леджер благотворительности: %v / %v", env.bank.kinds(1), env.bank.kinds(2))
	}
}

func TestFundCloseAuthority(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	env.pets.GetOrCreate(ctx, 1, 10)
	env.pets.GetOrCreate(ctx, 2, 10)

	bank, err := env.svc.CreateFund(ctx, 1, 10, 2, "Сбор", "", "", 100)
	if err != nil {
		t.Fatalf("CreateFund: %v", err)
	}
	fundID := bank.Funds[0].ID

	// Не создатель и не админ — нельзя.
	if _, err := env.svc.CloseFund(ctx, 2, 10, fundID, 2); errCode(err) != "FORBIDDEN" {
		t.Errorf("закрытие чужого сбора менеджером: %v", err)
	}
	// Админ компании — можно.
	bank, err = env.svc.CloseFund(ctx, 2, 10, fundID, 3)
	if err != nil {
		t.Fatalf("CloseFund админом: %v", err)
	}
	if bank.Funds[0].Status != "closed" {
		t.Errorf("сбор не закрыт: %+v", bank.Funds[0])
	}
	// Повторное закрытие — NOT_FOUND (активного нет).
	if _, err := env.svc.CloseFund(ctx, 1, 10, fundID, 3); errCode(err) != "NOT_FOUND" {
		t.Errorf("повторное закрытие: %v", err)
	}
}

// ── Досрочный возврат из приключения ────────────────────────────────

func TestRecallAdventure(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 150

	// Дома — возвращать некого.
	if _, err := env.svc.RecallAdventure(ctx, 1, 10); errCode(err) != "PET_HOME" {
		t.Errorf("возврат домашнего: %v", err)
	}

	until := time.Now().Add(2 * time.Hour)
	place := "лес"
	pet.AdventureUntil, pet.AdventurePlace = &until, &place

	data, err := env.svc.RecallAdventure(ctx, 1, 10)
	if err != nil {
		t.Fatalf("RecallAdventure: %v", err)
	}
	if data.AdventureUntil != nil {
		t.Error("питомец не вернулся")
	}
	if data.Kudos != 150-domain.AdventureRecallCost {
		t.Errorf("kudos = %d", data.Kudos)
	}
	if data.AdventureReward != nil {
		t.Error("досрочный возврат не должен приносить награду")
	}
	if !env.activity.hasKind("adventure_recalled") {
		t.Error("нет записи истории adventure_recalled")
	}
	if !slices.Contains(env.bank.kinds(1), "adventure_recall") {
		t.Errorf("леджер возврата: %v", env.bank.kinds(1))
	}

	// Не хватает кудосов.
	pet.Kudos = 10
	pet.AdventureUntil, pet.AdventurePlace = &until, &place
	if _, err := env.svc.RecallAdventure(ctx, 1, 10); errCode(err) != "NO_KUDOS" {
		t.Errorf("возврат без средств: %v", err)
	}

	// Истёкшее приключение возвращается бесплатно с наградой.
	pet.Kudos = 200
	expired := time.Now().Add(-time.Hour)
	pet.AdventureUntil = &expired
	data, err = env.svc.RecallAdventure(ctx, 1, 10)
	if err != nil {
		t.Fatalf("RecallAdventure(истёкшее): %v", err)
	}
	if data.AdventureReward == nil {
		t.Error("награда за истёкшее приключение не выдана")
	}
	if data.Kudos < 200 {
		t.Errorf("бесплатный возврат списал кудосы: %d", data.Kudos)
	}
}

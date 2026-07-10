package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/dto"
)

// Кудо-банк 2.0: копилки-цели (личные суб-счета «коплю на мечту») и
// благотворительные сборы компании. Копилки процент не приносят (процент —
// у вклада), сборы — общая цель: собранное считается потраченным.

// CreateGoal — новая копилка-цель (лимит GoalsMax на пользователя).
func (s *Service) CreateGoal(ctx context.Context, userID, companyID int64,
	title, emoji string, target int) (*dto.BankDTO, error) {

	title = strings.TrimSpace(title)
	if title == "" || runeLen(title) > domain.GoalTitleMax {
		return nil, domain.NewError("VALIDATION",
			"Название копилки — от 1 до "+strconv.Itoa(domain.GoalTitleMax)+" символов", 422)
	}
	if target < 1 || target > domain.GoalTargetMax {
		return nil, domain.NewError("VALIDATION",
			"Цель копилки — от 1 до "+strconv.Itoa(domain.GoalTargetMax)+" кудосов", 422)
	}
	emoji = strings.TrimSpace(emoji)
	if emoji == "" || runeLen(emoji) > 4 {
		emoji = "🎯"
	}
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	goals, err := s.bank.ListGoals(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(goals) >= domain.GoalsMax {
		return nil, domain.NewError("GOALS_LIMIT",
			"Копилок может быть не больше "+strconv.Itoa(domain.GoalsMax), 422)
	}
	g := &domain.BankGoal{UserID: userID, CompanyID: companyID, Title: title, Emoji: emoji, Target: target}
	if err := s.bank.CreateGoal(ctx, g); err != nil {
		return nil, err
	}
	return s.GetBank(ctx, userID, companyID)
}

// GoalDeposit — кошелёк → копилка; достижение цели отмечается разово
// (goal_achieved в ответе — фронт празднует конфетти).
func (s *Service) GoalDeposit(ctx context.Context, userID, companyID, goalID int64, amount int) (*dto.BankDTO, error) {
	if amount < 1 {
		return nil, domain.NewError("VALIDATION", "Сумма должна быть положительной", 422)
	}
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	goal, achievedNow, ok, err := s.bank.GoalDeposit(ctx, userID, goalID, amount)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов для пополнения копилки", 422)
	}
	if achievedNow {
		s.appendActivity(ctx, userID, "goal_achieved", map[string]any{
			"title": goal.Title, "emoji": goal.Emoji, "target": goal.Target,
		})
	}
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	bank, err := s.GetBank(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if achievedNow {
		bank.GoalAchieved = dto.NewGoal(goal)
	}
	return bank, nil
}

// GoalWithdraw — копилка → кошелёк (частично или целиком).
func (s *Service) GoalWithdraw(ctx context.Context, userID, companyID, goalID int64, amount int) (*dto.BankDTO, error) {
	if amount < 1 {
		return nil, domain.NewError("VALIDATION", "Сумма должна быть положительной", 422)
	}
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	_, ok, err := s.bank.GoalWithdraw(ctx, userID, goalID, amount)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NO_SAVINGS", "В копилке нет такой суммы", 422)
	}
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	return s.GetBank(ctx, userID, companyID)
}

// DeleteGoal — удаление копилки; остаток возвращается в кошелёк.
func (s *Service) DeleteGoal(ctx context.Context, userID, companyID, goalID int64) (*dto.BankDTO, error) {
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	_, ok, err := s.bank.DeleteGoal(ctx, userID, goalID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NOT_FOUND", "Копилка не найдена", 404)
	}
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	return s.GetBank(ctx, userID, companyID)
}

// CreateFund — благотворительный сбор компании: создаёт менеджер (роль ≥2).
func (s *Service) CreateFund(ctx context.Context, userID, companyID int64, userLevel int,
	title, description, emoji string, target int) (*dto.BankDTO, error) {

	if userLevel < 2 {
		return nil, domain.NewError("FORBIDDEN", "Сборы создаёт менеджер или администратор", 403)
	}
	title = strings.TrimSpace(title)
	if title == "" || runeLen(title) > domain.FundTitleMax {
		return nil, domain.NewError("VALIDATION",
			"Название сбора — от 1 до "+strconv.Itoa(domain.FundTitleMax)+" символов", 422)
	}
	description = strings.TrimSpace(description)
	if runeLen(description) > domain.FundDescriptionMax {
		return nil, domain.NewError("VALIDATION",
			"Описание — не длиннее "+strconv.Itoa(domain.FundDescriptionMax)+" символов", 422)
	}
	if target < 1 || target > domain.FundTargetMax {
		return nil, domain.NewError("VALIDATION",
			"Цель сбора — от 1 до "+strconv.Itoa(domain.FundTargetMax)+" кудосов", 422)
	}
	emoji = strings.TrimSpace(emoji)
	if emoji == "" || runeLen(emoji) > 4 {
		emoji = "💝"
	}
	f := &domain.BankFund{CompanyID: companyID, CreatedBy: &userID,
		Title: title, Description: description, Emoji: emoji, Target: target, Status: "active"}
	if err := s.bank.CreateFund(ctx, f); err != nil {
		return nil, err
	}
	s.emitFundUpdate(ctx, companyID, f, "created")
	return s.GetBank(ctx, userID, companyID)
}

// DonateFund — пожертвование в сбор; закрытие цели празднуется событием
// всей компании (fund_completed в ответе — конфетти донору-финишеру).
func (s *Service) DonateFund(ctx context.Context, userID, companyID, fundID int64, amount int) (*dto.BankDTO, error) {
	if amount < 1 {
		return nil, domain.NewError("VALIDATION", "Сумма должна быть положительной", 422)
	}
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	fund, fundOK, ok, completedNow, err := s.bank.Donate(ctx, userID, fundID, companyID, amount)
	if err != nil {
		return nil, err
	}
	if !fundOK {
		return nil, domain.NewError("FUND_CLOSED", "Сбор уже завершён", 422)
	}
	if !ok {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов для взноса", 422)
	}
	s.appendActivity(ctx, userID, "charity_donated", map[string]any{
		"title": fund.Title, "amount": amount,
	})
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	s.emitFundUpdate(ctx, companyID, fund, map[bool]string{true: "completed", false: "donated"}[completedNow])

	bank, err := s.GetBank(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if completedNow {
		bank.FundCompleted = dto.NewFund(fund)
	}
	return bank, nil
}

// CloseFund — досрочное закрытие сбора: создатель сбора или администратор
// компании (собранное не возвращается — благотворительность).
func (s *Service) CloseFund(ctx context.Context, userID, companyID, fundID int64, userLevel int) (*dto.BankDTO, error) {
	funds, err := s.bank.ListFunds(ctx, companyID, userID, domain.FundsFinishedShown)
	if err != nil {
		return nil, err
	}
	var fund *domain.BankFund
	for _, f := range funds {
		if f.ID == fundID {
			fund = f
			break
		}
	}
	if fund == nil || fund.Status != "active" {
		return nil, domain.NewError("NOT_FOUND", "Активный сбор не найден", 404)
	}
	isCreator := fund.CreatedBy != nil && *fund.CreatedBy == userID
	if !isCreator && userLevel < 3 {
		return nil, domain.NewError("FORBIDDEN", "Закрыть сбор может его создатель или администратор", 403)
	}
	ok, err := s.bank.CloseFund(ctx, fundID, companyID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NOT_FOUND", "Активный сбор не найден", 404)
	}
	fund.Status = "closed"
	s.emitFundUpdate(ctx, companyID, fund, "closed")
	return s.GetBank(ctx, userID, companyID)
}

// GetBankStats — динамика прихода/расхода по дням и структура по видам
// операций за окно BankStatsDays.
func (s *Service) GetBankStats(ctx context.Context, userID int64) (*dto.BankStatsDTO, error) {
	days, err := s.bank.DailyTotals(ctx, userID, domain.BankStatsDays)
	if err != nil {
		return nil, err
	}
	kinds, err := s.bank.KindTotals(ctx, userID, domain.BankStatsDays)
	if err != nil {
		return nil, err
	}
	return dto.NewBankStats(days, kinds), nil
}

// emitFundUpdate — событие сбора всей компании (комната all, клиент фильтрует
// по company_id — как pet:deleted).
func (s *Service) emitFundUpdate(ctx context.Context, companyID int64, f *domain.BankFund, action string) {
	s.pub.Publish(ctx, "bank:fund", []string{"all"}, map[string]any{
		"company_id": companyID, "action": action, "fund": dto.NewFund(f),
	})
}

// runeLen — длина в рунах (валидация пользовательских строк).
func runeLen(s string) int { return len([]rune(s)) }

package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/dto"
)

// Кудо-банк: переводы кудосов коллегам, выписка (леджер), вклад под
// ежедневный процент и кредит. Условия (ставка/комиссия/лимиты) зависят от
// уровня клиента — суммарно заработанных кудосов за всё время (loyalty-tiers).

const bankTransferSource = "bank_transfer" // Redis-источник дневного лимита переводов

// bankTier — текущий уровень клиента + заработанное (fail-open: ошибка
// чтения леджера не роняет банк, просто стартовый уровень).
func (s *Service) bankTier(ctx context.Context, userID int64) (domain.BankTier, *domain.BankTier, int) {
	earned, err := s.bank.LifetimeEarned(ctx, userID)
	if err != nil {
		s.log.Warn("pets.bank_tier_failed", "user_id", userID, "error", err)
		earned = 0
	}
	tier, next := domain.TierFor(earned)
	return tier, next, earned
}

// GetBank — сводка банка: балансы, уровень с прогрессом, месячные обороты,
// остаток дневного лимита переводов и топ щедрости компании. Заодно лениво
// начисляет проценты по вкладу (целые прошедшие сутки).
func (s *Service) GetBank(ctx context.Context, userID, companyID int64) (*dto.BankDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	tier, next, earned := s.bankTier(ctx, userID)

	interest, err := s.bank.AccrueSavings(ctx, userID, companyID, tier.SavingsRatePct)
	if err != nil {
		s.log.Warn("pets.bank_accrue_failed", "user_id", userID, "error", err)
	} else if interest > 0 {
		s.appendActivity(ctx, userID, "bank_interest", map[string]any{"amount": interest})
		// Свежий снимок после атомарного начисления — не мутируем локальную копию.
		if fresh, err := s.pets.GetPet(ctx, userID); err == nil && fresh != nil {
			pet = fresh
		}
	}

	monthIn, monthOut, err := s.bank.MonthlyTotals(ctx, userID)
	if err != nil {
		return nil, err
	}
	top, err := s.bank.TopGenerous(ctx, companyID, 3)
	if err != nil {
		return nil, err
	}
	goals, err := s.bank.ListGoals(ctx, userID)
	if err != nil {
		return nil, err
	}
	funds, err := s.bank.ListFunds(ctx, companyID, userID, domain.FundsFinishedShown)
	if err != nil {
		return nil, err
	}
	for _, f := range funds {
		if donors, err := s.bank.FundTopDonors(ctx, f.ID, 3); err == nil {
			f.TopDonors = donors
		}
	}
	d := dto.NewBank(pet, tier, next, earned, monthIn, monthOut, top)
	d.Goals = dto.NewGoals(goals)
	d.Funds = dto.NewFunds(funds)
	d.TransferLeftToday = s.daily.Left(ctx, userID, bankTransferSource, tier.TransferDailyCap)
	if interest > 0 {
		d.InterestPaid = &interest
	}
	return d, nil
}

// GetBankLedger — выписка операций (keyset-пагинация по id вниз).
func (s *Service) GetBankLedger(ctx context.Context, userID, beforeID int64) (*dto.LedgerDTO, error) {
	entries, err := s.bank.ListLedger(ctx, userID, beforeID, domain.LedgerPageSize+1)
	if err != nil {
		return nil, err
	}
	return dto.NewLedger(entries, domain.LedgerPageSize), nil
}

// TransferKudos — перевод коллеге по компании: списание/зачисление и обе
// записи выписки атомарны (одна транзакция), дневной лимит — по уровню
// клиента, получателю уходит отдельное событие kudos:received.
func (s *Service) TransferKudos(ctx context.Context, fromID, toID, companyID int64,
	amount int, comment string) (*dto.BankDTO, error) {

	comment = strings.TrimSpace(comment)
	if fromID == toID {
		return nil, domain.NewError("SELF_TRANSFER", "Себе переводить незачем — они и так ваши", 422)
	}
	tier, _, _ := s.bankTier(ctx, fromID)
	if amount < 1 || amount > tier.TransferMax {
		return nil, domain.NewError("VALIDATION",
			"Сумма перевода — от 1 до "+strconv.Itoa(tier.TransferMax)+" кудосов", 422)
	}
	toMember, err := s.users.IsCompanyMember(ctx, toID, companyID)
	if err != nil {
		return nil, err
	}
	fromMember, err := s.users.IsCompanyMember(ctx, fromID, companyID)
	if err != nil {
		return nil, err
	}
	if !toMember || !fromMember {
		return nil, domain.NewError("USER_NOT_FOUND", "Сотрудник не найден", 404)
	}
	if left := s.daily.Left(ctx, fromID, bankTransferSource, tier.TransferDailyCap); amount > left {
		return nil, domain.NewError("TRANSFER_LIMIT",
			"Дневной лимит переводов исчерпан (осталось "+strconv.Itoa(left)+")", 429)
	}
	// Питомец получателя обязан существовать до зачисления.
	if _, err := s.pets.GetOrCreate(ctx, toID, companyID); err != nil {
		return nil, err
	}
	_, ok, err := s.bank.Transfer(ctx, fromID, toID, companyID, amount, comment)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов для перевода", 422)
	}
	s.daily.TakeBudget(ctx, fromID, bankTransferSource, amount, tier.TransferDailyCap)

	s.appendActivity(ctx, fromID, "kudos_sent", map[string]any{"to_id": toID, "amount": amount})
	s.appendActivity(ctx, toID, "kudos_received", map[string]any{
		"from_id": fromID, "amount": amount, "comment": comment,
	})

	// Свежие балансы обоим (pet:update) + адресный тост получателю.
	if fromPet, err := s.pets.GetPet(ctx, fromID); err == nil && fromPet != nil {
		s.emitPetUpdate(ctx, fromPet)
	}
	if toPet, err := s.pets.GetPet(ctx, toID); err == nil && toPet != nil {
		s.emitPetUpdate(ctx, toPet)
	}
	sender, _ := s.users.GetUser(ctx, fromID)
	payload := map[string]any{"amount": amount, "comment": comment, "company_id": companyID}
	if sender != nil {
		payload["from"] = &domain.UserRef{ID: sender.ID, FIO: sender.FIO, AvatarPath: sender.AvatarPath}
	}
	s.pub.Publish(ctx, "kudos:received", []string{userRoom(toID)}, payload)

	return s.GetBank(ctx, fromID, companyID)
}

// BankDeposit — кошелёк → вклад. При активном кредите закрыт (иначе
// арбитраж «кредит → вклад под процент»).
func (s *Service) BankDeposit(ctx context.Context, userID, companyID int64, amount int) (*dto.BankDTO, error) {
	if amount < 1 {
		return nil, domain.NewError("VALIDATION", "Сумма должна быть положительной", 422)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if pet.BankLoan > 0 {
		return nil, domain.NewError("LOAN_ACTIVE", "Сначала погасите кредит — вклад с долгом не открыть", 422)
	}
	_, _, ok, err := s.bank.DepositSavings(ctx, userID, amount)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов для вклада", 422)
	}
	s.appendActivity(ctx, userID, "bank_deposit", map[string]any{"amount": amount})
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	return s.GetBank(ctx, userID, companyID)
}

// BankWithdraw — вклад → кошелёк (с предварительным начислением процентов —
// внутри GetBank ниже они уже посчитаны атомарно в AccrueSavings).
func (s *Service) BankWithdraw(ctx context.Context, userID, companyID int64, amount int) (*dto.BankDTO, error) {
	if amount < 1 {
		return nil, domain.NewError("VALIDATION", "Сумма должна быть положительной", 422)
	}
	// Сначала капитализируем накопленное — снятие не должно «сжигать» процент.
	tier, _, _ := s.bankTier(ctx, userID)
	if _, err := s.bank.AccrueSavings(ctx, userID, companyID, tier.SavingsRatePct); err != nil {
		s.log.Warn("pets.bank_accrue_failed", "user_id", userID, "error", err)
	}
	_, _, ok, err := s.bank.WithdrawSavings(ctx, userID, amount)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NO_SAVINGS", "На вкладе нет такой суммы", 422)
	}
	s.appendActivity(ctx, userID, "bank_withdraw", map[string]any{"amount": amount})
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	return s.GetBank(ctx, userID, companyID)
}

// BankTakeLoan — кредит: тело сразу на кошелёк, долг = тело + комиссия
// уровня; один активный кредит на питомца.
func (s *Service) BankTakeLoan(ctx context.Context, userID, companyID int64, amount int) (*dto.BankDTO, error) {
	tier, _, _ := s.bankTier(ctx, userID)
	if amount < 1 || amount > tier.LoanMax {
		return nil, domain.NewError("VALIDATION",
			"Сумма кредита — от 1 до "+strconv.Itoa(tier.LoanMax)+" кудосов", 422)
	}
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	debt := amount + (amount*tier.LoanFeePct+99)/100 // комиссия вверх до целого
	_, ok, err := s.bank.TakeLoan(ctx, userID, amount, debt)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("LOAN_ACTIVE", "Кредит уже взят — сначала погасите его", 422)
	}
	s.appendActivity(ctx, userID, "loan_taken", map[string]any{"amount": amount, "debt": debt})
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	return s.GetBank(ctx, userID, companyID)
}

// BankRepayLoan — погашение с кошелька; сумма сверх долга клампится
// (кнопка «Погасить всё» не требует точной цифры).
func (s *Service) BankRepayLoan(ctx context.Context, userID, companyID int64, amount int) (*dto.BankDTO, error) {
	if amount < 1 {
		return nil, domain.NewError("VALIDATION", "Сумма должна быть положительной", 422)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if pet.BankLoan <= 0 {
		return nil, domain.NewError("NO_LOAN", "Активного кредита нет", 422)
	}
	pay := min(amount, pet.BankLoan)
	_, loanLeft, ok, err := s.bank.RepayLoan(ctx, userID, pay)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов для погашения", 422)
	}
	s.appendActivity(ctx, userID, "loan_repaid", map[string]any{"amount": pay, "left": loanLeft})
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	return s.GetBank(ctx, userID, companyID)
}

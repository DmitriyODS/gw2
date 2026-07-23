package service

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ── Фейк кудо-банка (балансы — в fakePets, леджер — в памяти) ───────

type fakeBank struct {
	pets    *fakePets
	entries []*domain.LedgerEntry
	nextID  int64

	goals      []*domain.BankGoal
	nextGoalID int64
	funds      []*domain.BankFund
	nextFundID int64
	donations  map[int64]map[int64]int // fundID → userID → сумма
}

var _ domain.BankRepo = (*fakeBank)(nil)

func (f *fakeBank) add(e *domain.LedgerEntry) {
	f.nextID++
	e.ID = f.nextID
	e.CreatedAt = time.Now()
	f.entries = append(f.entries, e)
}

func (f *fakeBank) AppendLedger(_ context.Context, e *domain.LedgerEntry) error {
	f.add(e)
	return nil
}

func (f *fakeBank) ListLedger(_ context.Context, userID, beforeID int64, limit int) ([]*domain.LedgerEntry, error) {
	var out []*domain.LedgerEntry
	for i := len(f.entries) - 1; i >= 0 && len(out) < limit; i-- {
		e := f.entries[i]
		if e.UserID != userID || (beforeID != 0 && e.ID >= beforeID) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (f *fakeBank) Transfer(_ context.Context, fromID, toID, companyID int64, amount int, comment string) (int, bool, error) {
	from, to := f.pets.byUser[fromID], f.pets.byUser[toID]
	if from == nil || to == nil {
		return 0, false, errNoPet
	}
	if from.Kudos < amount {
		return 0, false, nil
	}
	from.Kudos -= amount
	to.Kudos += amount
	f.add(&domain.LedgerEntry{UserID: fromID, CompanyID: companyID, Delta: -amount,
		Kind: "transfer_out", CounterpartyID: &toID, Comment: comment})
	f.add(&domain.LedgerEntry{UserID: toID, CompanyID: companyID, Delta: amount,
		Kind: "transfer_in", CounterpartyID: &fromID, Comment: comment})
	return from.Kudos, true, nil
}

func (f *fakeBank) MonthlyTotals(_ context.Context, userID int64) (int, int, error) {
	in, out := 0, 0
	for _, e := range f.entries {
		if e.UserID != userID {
			continue
		}
		if e.Delta > 0 {
			in += e.Delta
		} else {
			out -= e.Delta
		}
	}
	return in, out, nil
}

func (f *fakeBank) LifetimeEarned(_ context.Context, userID int64) (int, error) {
	earned := 0
	for _, e := range f.entries {
		if e.UserID != userID || e.Delta <= 0 || slices.Contains(domain.LedgerEarnExcluded, e.Kind) {
			continue
		}
		earned += e.Delta
	}
	return earned, nil
}

func (f *fakeBank) TopGenerous(context.Context, int64, int) ([]domain.GenerousEntry, error) {
	return nil, nil
}

func (f *fakeBank) DepositSavings(_ context.Context, userID int64, amount int) (int, int, bool, error) {
	p := f.pets.byUser[userID]
	if p == nil {
		return 0, 0, false, errNoPet
	}
	if p.Kudos < amount || p.BankLoan > 0 {
		return 0, 0, false, nil
	}
	p.Kudos -= amount
	if p.BankSavings == 0 {
		now := time.Now()
		p.BankSavingsAccruedAt = &now
	}
	p.BankSavings += amount
	f.add(&domain.LedgerEntry{UserID: userID, CompanyID: p.CompanyID, Delta: -amount, Kind: "bank_deposit"})
	return p.Kudos, p.BankSavings, true, nil
}

func (f *fakeBank) WithdrawSavings(_ context.Context, userID int64, amount int) (int, int, bool, error) {
	p := f.pets.byUser[userID]
	if p == nil {
		return 0, 0, false, errNoPet
	}
	if p.BankSavings < amount {
		return 0, 0, false, nil
	}
	p.BankSavings -= amount
	p.Kudos += amount
	if p.BankSavings == 0 {
		p.BankSavingsAccruedAt = nil
	}
	f.add(&domain.LedgerEntry{UserID: userID, CompanyID: p.CompanyID, Delta: amount, Kind: "bank_withdraw"})
	return p.Kudos, p.BankSavings, true, nil
}

func (f *fakeBank) AccrueSavings(_ context.Context, userID, _ int64, ratePct int) (int, error) {
	p := f.pets.byUser[userID]
	if p == nil || p.BankSavings <= 0 || p.BankSavingsAccruedAt == nil {
		return 0, nil
	}
	days := int(time.Since(*p.BankSavingsAccruedAt).Hours() / 24)
	if days < 1 {
		return 0, nil
	}
	interest := p.BankSavings * ratePct / 100 * days
	p.BankSavings += interest
	at := p.BankSavingsAccruedAt.Add(time.Duration(days) * 24 * time.Hour)
	p.BankSavingsAccruedAt = &at
	if interest > 0 {
		f.add(&domain.LedgerEntry{UserID: userID, CompanyID: p.CompanyID, Delta: interest, Kind: "bank_interest"})
	}
	return interest, nil
}

func (f *fakeBank) TakeLoan(_ context.Context, userID int64, amount, debt int) (int, bool, error) {
	p := f.pets.byUser[userID]
	if p == nil {
		return 0, false, errNoPet
	}
	if p.BankLoan != 0 {
		return 0, false, nil
	}
	p.Kudos += amount
	p.BankLoan = debt
	f.add(&domain.LedgerEntry{UserID: userID, CompanyID: p.CompanyID, Delta: amount, Kind: "loan_taken"})
	return p.Kudos, true, nil
}

func (f *fakeBank) RepayLoan(_ context.Context, userID int64, amount int) (int, int, bool, error) {
	p := f.pets.byUser[userID]
	if p == nil {
		return 0, 0, false, errNoPet
	}
	if p.Kudos < amount || p.BankLoan < amount {
		return 0, 0, false, nil
	}
	p.Kudos -= amount
	p.BankLoan -= amount
	f.add(&domain.LedgerEntry{UserID: userID, CompanyID: p.CompanyID, Delta: -amount, Kind: "loan_repaid"})
	return p.Kudos, p.BankLoan, true, nil
}

// ── Копилки-цели ────────────────────────────────────────────────────

func (f *fakeBank) ListGoals(_ context.Context, userID int64) ([]*domain.BankGoal, error) {
	var out []*domain.BankGoal
	for _, g := range f.goals {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeBank) CreateGoal(_ context.Context, g *domain.BankGoal) error {
	f.nextGoalID++
	g.ID = f.nextGoalID
	g.CreatedAt = time.Now()
	f.goals = append(f.goals, g)
	return nil
}

func (f *fakeBank) findGoal(userID, goalID int64) *domain.BankGoal {
	for _, g := range f.goals {
		if g.ID == goalID && g.UserID == userID {
			return g
		}
	}
	return nil
}

func (f *fakeBank) GoalDeposit(_ context.Context, userID, goalID int64, amount int) (*domain.BankGoal, bool, bool, error) {
	p := f.pets.byUser[userID]
	g := f.findGoal(userID, goalID)
	if p == nil || g == nil || p.Kudos < amount {
		return nil, false, false, nil
	}
	p.Kudos -= amount
	g.Saved += amount
	achievedNow := false
	if g.AchievedAt == nil && g.Saved >= g.Target {
		now := time.Now()
		g.AchievedAt = &now
		achievedNow = true
	}
	f.add(&domain.LedgerEntry{UserID: userID, CompanyID: g.CompanyID, Delta: -amount, Kind: "goal_deposit"})
	return g, achievedNow, true, nil
}

func (f *fakeBank) GoalWithdraw(_ context.Context, userID, goalID int64, amount int) (*domain.BankGoal, bool, error) {
	p := f.pets.byUser[userID]
	g := f.findGoal(userID, goalID)
	if p == nil || g == nil || g.Saved < amount {
		return nil, false, nil
	}
	g.Saved -= amount
	p.Kudos += amount
	f.add(&domain.LedgerEntry{UserID: userID, CompanyID: g.CompanyID, Delta: amount, Kind: "goal_withdraw"})
	return g, true, nil
}

func (f *fakeBank) DeleteGoal(_ context.Context, userID, goalID int64) (int, bool, error) {
	g := f.findGoal(userID, goalID)
	if g == nil {
		return 0, false, nil
	}
	refund := g.Saved
	if refund > 0 {
		f.pets.byUser[userID].Kudos += refund
		f.add(&domain.LedgerEntry{UserID: userID, CompanyID: g.CompanyID, Delta: refund, Kind: "goal_withdraw"})
	}
	for i, x := range f.goals {
		if x == g {
			f.goals = append(f.goals[:i], f.goals[i+1:]...)
			break
		}
	}
	return refund, true, nil
}

// ── Благотворительные сборы ─────────────────────────────────────────

func (f *fakeBank) ListFunds(_ context.Context, companyID, viewerID int64, _ int) ([]*domain.BankFund, error) {
	var out []*domain.BankFund
	for _, fd := range f.funds {
		if fd.CompanyID == companyID {
			fd.MyDonated = f.donations[fd.ID][viewerID]
			out = append(out, fd)
		}
	}
	return out, nil
}

func (f *fakeBank) CreateFund(_ context.Context, fd *domain.BankFund) error {
	f.nextFundID++
	fd.ID = f.nextFundID
	fd.CreatedAt = time.Now()
	f.funds = append(f.funds, fd)
	return nil
}

func (f *fakeBank) findFund(fundID int64) *domain.BankFund {
	for _, fd := range f.funds {
		if fd.ID == fundID {
			return fd
		}
	}
	return nil
}

func (f *fakeBank) Donate(_ context.Context, userID, fundID, companyID int64, amount int) (*domain.BankFund, bool, bool, bool, error) {
	fd := f.findFund(fundID)
	if fd == nil || fd.CompanyID != companyID || fd.Status != "active" {
		return nil, false, false, false, nil
	}
	p := f.pets.byUser[userID]
	if p == nil || p.Kudos < amount {
		return nil, true, false, false, nil
	}
	p.Kudos -= amount
	fd.Collected += amount
	completedNow := false
	if fd.Collected >= fd.Target {
		fd.Status = "done"
		now := time.Now()
		fd.FinishedAt = &now
		completedNow = true
	}
	if f.donations == nil {
		f.donations = map[int64]map[int64]int{}
	}
	if f.donations[fundID] == nil {
		f.donations[fundID] = map[int64]int{}
	}
	f.donations[fundID][userID] += amount
	f.add(&domain.LedgerEntry{UserID: userID, CompanyID: companyID, Delta: -amount, Kind: "charity"})
	return fd, true, true, completedNow, nil
}

func (f *fakeBank) CloseFund(_ context.Context, fundID, companyID int64) (bool, error) {
	fd := f.findFund(fundID)
	if fd == nil || fd.CompanyID != companyID || fd.Status != "active" {
		return false, nil
	}
	fd.Status = "closed"
	now := time.Now()
	fd.FinishedAt = &now
	return true, nil
}

func (f *fakeBank) FundTopDonors(_ context.Context, fundID int64, _ int) ([]domain.GenerousEntry, error) {
	var out []domain.GenerousEntry
	for uid, amount := range f.donations[fundID] {
		out = append(out, domain.GenerousEntry{User: &domain.UserRef{ID: uid}, Sent: amount})
	}
	return out, nil
}

// ── Статистика ──────────────────────────────────────────────────────

func (f *fakeBank) DailyTotals(_ context.Context, userID int64, _ int) ([]domain.BankDayStat, error) {
	s := domain.BankDayStat{Day: time.Now()}
	for _, e := range f.entries {
		if e.UserID != userID {
			continue
		}
		if e.Delta > 0 {
			s.In += e.Delta
		} else {
			s.Out -= e.Delta
		}
	}
	return []domain.BankDayStat{s}, nil
}

func (f *fakeBank) KindTotals(_ context.Context, userID int64, _ int) ([]domain.BankKindStat, error) {
	byKind := map[string]*domain.BankKindStat{}
	for _, e := range f.entries {
		if e.UserID != userID {
			continue
		}
		k := byKind[e.Kind]
		if k == nil {
			k = &domain.BankKindStat{Kind: e.Kind}
			byKind[e.Kind] = k
		}
		if e.Delta > 0 {
			k.In += e.Delta
		} else {
			k.Out -= e.Delta
		}
	}
	var out []domain.BankKindStat
	for _, k := range byKind {
		out = append(out, *k)
	}
	return out, nil
}

func (f *fakeBank) DeleteLedger(_ context.Context, userID int64) error {
	kept := f.entries[:0]
	for _, e := range f.entries {
		if e.UserID != userID {
			kept = append(kept, e)
		}
	}
	f.entries = kept
	return nil
}

func (f *fakeBank) kinds(userID int64) []string {
	var out []string
	for _, e := range f.entries {
		if e.UserID == userID {
			out = append(out, e.Kind)
		}
	}
	return out
}

// ── Переводы ────────────────────────────────────────────────────────

func TestTransferKudosHappyPath(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	from, _ := env.pets.GetOrCreate(ctx, 1, 10)
	from.Kudos = 50
	env.pets.GetOrCreate(ctx, 2, 10)

	bank, err := env.svc.TransferKudos(ctx, 1, 2, 10, 15, "спасибо за ревью")
	if err != nil {
		t.Fatalf("TransferKudos: %v", err)
	}
	if from.Kudos != 35 {
		t.Errorf("баланс отправителя = %d", from.Kudos)
	}
	if env.pets.byUser[2].Kudos != 15 {
		t.Errorf("баланс получателя = %d", env.pets.byUser[2].Kudos)
	}
	if bank.Kudos != 35 {
		t.Errorf("сводка банка kudos = %d", bank.Kudos)
	}
	// Выписка: расход у отправителя, приход у получателя.
	if !slices.Contains(env.bank.kinds(1), "transfer_out") || !slices.Contains(env.bank.kinds(2), "transfer_in") {
		t.Errorf("леджер: %v / %v", env.bank.kinds(1), env.bank.kinds(2))
	}
	// История активности обеим сторонам, события: 2×pet:update + kudos:received.
	if !env.activity.hasKind("kudos_sent") || !env.activity.hasKind("kudos_received") {
		t.Error("нет записей активности о переводе")
	}
	if !slices.Contains(env.pub.events, "kudos:received") {
		t.Errorf("события: %v", env.pub.events)
	}
	// Переводы не двигают счётчики признания (weekly).
	if env.pets.weeklyKudos[2] != 0 {
		t.Errorf("weekly получателя = %d", env.pets.weeklyKudos[2])
	}
}

func TestTransferKudosGuards(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	from, _ := env.pets.GetOrCreate(ctx, 1, 10)
	from.Kudos = 5
	env.pets.GetOrCreate(ctx, 2, 10)

	if _, err := env.svc.TransferKudos(ctx, 1, 1, 10, 5, ""); errCode(err) != "SELF_TRANSFER" {
		t.Errorf("перевод себе: %v", err)
	}
	if _, err := env.svc.TransferKudos(ctx, 1, 2, 10, 10, ""); errCode(err) != "NO_KUDOS" {
		t.Errorf("перевод без средств: %v", err)
	}
	if _, err := env.svc.TransferKudos(ctx, 1, 2, 10, 0, ""); errCode(err) != "VALIDATION" {
		t.Errorf("нулевая сумма: %v", err)
	}
	tier, _ := domain.TierFor(0)
	if _, err := env.svc.TransferKudos(ctx, 1, 2, 10, tier.TransferMax+1, ""); errCode(err) != "VALIDATION" {
		t.Errorf("сумма сверх лимита уровня: %v", err)
	}
	// Не член компании — 404.
	env.users.members = map[int64]map[int64]bool{1: {10: true}, 2: {10: false}}
	from.Kudos = 50
	if _, err := env.svc.TransferKudos(ctx, 1, 2, 10, 5, ""); errCode(err) != "USER_NOT_FOUND" {
		t.Errorf("перевод вне компании: %v", err)
	}
}

func TestTransferKudosDailyCapByTier(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	from, _ := env.pets.GetOrCreate(ctx, 1, 10)
	from.Kudos = 1000
	env.pets.GetOrCreate(ctx, 2, 10)

	tier, _ := domain.TierFor(0)
	// Съедаем дневной лимит переводов переводами по максимуму за раз.
	sent := 0
	for sent+tier.TransferMax <= tier.TransferDailyCap {
		if _, err := env.svc.TransferKudos(ctx, 1, 2, 10, tier.TransferMax, ""); err != nil {
			t.Fatalf("перевод в пределах лимита: %v", err)
		}
		sent += tier.TransferMax
	}
	rest := tier.TransferDailyCap - sent
	if rest > 0 {
		if _, err := env.svc.TransferKudos(ctx, 1, 2, 10, rest, ""); err != nil {
			t.Fatalf("добор лимита: %v", err)
		}
	}
	if _, err := env.svc.TransferKudos(ctx, 1, 2, 10, 1, ""); errCode(err) != "TRANSFER_LIMIT" {
		t.Errorf("перевод сверх дневного лимита: %v", err)
	}
}

// ── Вклад и проценты ────────────────────────────────────────────────

func TestBankDepositWithdraw(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 100

	bank, err := env.svc.BankDeposit(ctx, 1, 10, 60)
	if err != nil {
		t.Fatalf("BankDeposit: %v", err)
	}
	if bank.Kudos != 40 || bank.Savings != 60 {
		t.Errorf("после вклада: kudos=%d savings=%d", bank.Kudos, bank.Savings)
	}
	if _, err := env.svc.BankDeposit(ctx, 1, 10, 100); errCode(err) != "NO_KUDOS" {
		t.Errorf("вклад без средств: %v", err)
	}

	bank, err = env.svc.BankWithdraw(ctx, 1, 10, 25)
	if err != nil {
		t.Fatalf("BankWithdraw: %v", err)
	}
	if bank.Kudos != 65 || bank.Savings != 35 {
		t.Errorf("после снятия: kudos=%d savings=%d", bank.Kudos, bank.Savings)
	}
	if _, err := env.svc.BankWithdraw(ctx, 1, 10, 999); errCode(err) != "NO_SAVINGS" {
		t.Errorf("снятие сверх вклада: %v", err)
	}
	kinds := env.bank.kinds(1)
	if !slices.Contains(kinds, "bank_deposit") || !slices.Contains(kinds, "bank_withdraw") {
		t.Errorf("леджер вклада: %v", kinds)
	}
}

func TestBankSavingsInterestLazyAccrual(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 0
	pet.BankSavings = 200
	twoDaysAgo := time.Now().Add(-49 * time.Hour)
	pet.BankSavingsAccruedAt = &twoDaysAgo

	bank, err := env.svc.GetBank(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetBank: %v", err)
	}
	// Стартовый уровень: 10%/день → 200*10/100=20 кудосов × 2 суток = 40.
	if bank.Savings != 240 {
		t.Errorf("savings после начисления = %d", bank.Savings)
	}
	if bank.InterestPaid == nil || *bank.InterestPaid != 40 {
		t.Errorf("interest_paid = %v", bank.InterestPaid)
	}
	// Повторный вызов сразу — ничего не доначисляет.
	bank, _ = env.svc.GetBank(ctx, 1, 10)
	if bank.Savings != 240 || bank.InterestPaid != nil {
		t.Errorf("повторное начисление: savings=%d paid=%v", bank.Savings, bank.InterestPaid)
	}
	if !slices.Contains(env.bank.kinds(1), "bank_interest") {
		t.Errorf("леджер процентов: %v", env.bank.kinds(1))
	}
}

// ── Кредит ──────────────────────────────────────────────────────────

func TestBankLoanTakeAndRepay(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 0

	bank, err := env.svc.BankTakeLoan(ctx, 1, 10, 50)
	if err != nil {
		t.Fatalf("BankTakeLoan: %v", err)
	}
	// Стартовый уровень: комиссия 20% → долг 60.
	if bank.Kudos != 50 || bank.Loan != 60 {
		t.Errorf("после кредита: kudos=%d loan=%d", bank.Kudos, bank.Loan)
	}
	// Второй кредит при активном долге запрещён.
	if _, err := env.svc.BankTakeLoan(ctx, 1, 10, 10); errCode(err) != "LOAN_ACTIVE" {
		t.Errorf("второй кредит: %v", err)
	}
	// Вклад при долге запрещён (арбитраж «кредит → вклад»).
	if _, err := env.svc.BankDeposit(ctx, 1, 10, 10); errCode(err) != "LOAN_ACTIVE" {
		t.Errorf("вклад при долге: %v", err)
	}

	bank, err = env.svc.BankRepayLoan(ctx, 1, 10, 20)
	if err != nil {
		t.Fatalf("BankRepayLoan: %v", err)
	}
	if bank.Kudos != 30 || bank.Loan != 40 {
		t.Errorf("после частичного погашения: kudos=%d loan=%d", bank.Kudos, bank.Loan)
	}
	// Сумма сверх долга клампится к остатку («Погасить всё»).
	pet.Kudos = 100
	bank, err = env.svc.BankRepayLoan(ctx, 1, 10, 999)
	if err != nil {
		t.Fatalf("BankRepayLoan (всё): %v", err)
	}
	if bank.Loan != 0 || bank.Kudos != 60 {
		t.Errorf("после полного погашения: kudos=%d loan=%d", bank.Kudos, bank.Loan)
	}
	if _, err := env.svc.BankRepayLoan(ctx, 1, 10, 5); errCode(err) != "NO_LOAN" {
		t.Errorf("погашение без долга: %v", err)
	}
	kinds := env.bank.kinds(1)
	if !slices.Contains(kinds, "loan_taken") || !slices.Contains(kinds, "loan_repaid") {
		t.Errorf("леджер кредита: %v", kinds)
	}
}

// ── Уровни клиента ──────────────────────────────────────────────────

func TestBankTierProgress(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 0

	bank, _ := env.svc.GetBank(ctx, 1, 10)
	if bank.Tier.Key != "start" || bank.NextTier == nil || bank.NextTier.Key != "bronze" {
		t.Fatalf("стартовый уровень: %+v", bank.Tier)
	}

	// Заработок двигает уровень; входящие переводы — нет.
	env.bank.add(&domain.LedgerEntry{UserID: 1, CompanyID: 10, Delta: 250, Kind: "unit"})
	env.bank.add(&domain.LedgerEntry{UserID: 1, CompanyID: 10, Delta: 100, Kind: "transfer_in"})
	bank, _ = env.svc.GetBank(ctx, 1, 10)
	if bank.Tier.Key != "start" || bank.Earned != 250 {
		t.Errorf("transfer_in посчитан заработком: tier=%s earned=%d", bank.Tier.Key, bank.Earned)
	}
	env.bank.add(&domain.LedgerEntry{UserID: 1, CompanyID: 10, Delta: 100, Kind: "task_closed"})
	bank, _ = env.svc.GetBank(ctx, 1, 10)
	if bank.Tier.Key != "bronze" {
		t.Errorf("уровень после 350 заработанных: %s", bank.Tier.Key)
	}
}

// ── Выписка и её пополнение механиками ──────────────────────────────

func TestLedgerRecordsEconomyOperations(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 100

	env.svc.AwardKudos(ctx, 1, 10, "unit", 5)
	if _, err := env.svc.FeedPet(ctx, 1, 10); err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	kinds := env.bank.kinds(1)
	if !slices.Contains(kinds, "unit") || !slices.Contains(kinds, "feed") {
		t.Errorf("леджер экономики: %v", kinds)
	}

	ledger, err := env.svc.GetBankLedger(ctx, 1, 0)
	if err != nil {
		t.Fatalf("GetBankLedger: %v", err)
	}
	if len(ledger.Items) < 2 {
		t.Errorf("выписка пуста: %d", len(ledger.Items))
	}
	// Свежие записи первыми (keyset вниз по id).
	if ledger.Items[0].Kind != "feed" {
		t.Errorf("порядок выписки: %v", ledger.Items[0].Kind)
	}
}

func errCode(err error) string {
	derr := domain.AsDomainError(err)
	if derr == nil {
		return ""
	}
	return derr.Code
}

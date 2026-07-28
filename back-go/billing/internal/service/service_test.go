package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/payments"
)

// Фейки портов: биллинг тестируется без БД — вся арифметика тарифов, скидок и
// выдачи покупок живёт в сервисе.

type fakeRepo struct {
	plans   map[string]*domain.Plan
	addons  map[string]*domain.Addon
	subs    map[int64]*domain.Subscription
	uaddons map[int64][]*domain.UserAddon
	orders  map[int64]*domain.Order
	pays    map[int64]*domain.Payment
	promos  map[string]*domain.Promo
	balance map[int64]*domain.AIBalance
	storage map[int64]int64
	audit   []*domain.AuditEntry
	seq     int64
}

func newRepo() *fakeRepo {
	return &fakeRepo{
		plans: map[string]*domain.Plan{
			domain.PlanJunior: {Code: domain.PlanJunior, Name: "Джун", IsActive: true},
			domain.PlanMiddle: {Code: domain.PlanMiddle, Name: "Мидл", PriceMonth: 29900, PriceYear: 238800, IsActive: true},
			domain.PlanSenior: {Code: domain.PlanSenior, Name: "Синьор", PriceMonth: 49900, PriceYear: 478800, IsActive: true},
		},
		addons: map[string]*domain.Addon{
			"storage_10":  {Code: "storage_10", Kind: domain.AddonStorage, Name: "+10 Гб", Amount: 10 * domain.GiB, PriceMonth: 19900, Recurring: true, IsActive: true},
			"tokens_1000": {Code: "tokens_1000", Kind: domain.AddonTokens, Name: "+1000 токенов", Amount: 1000, PriceMonth: 10000, IsActive: true},
		},
		subs:    map[int64]*domain.Subscription{},
		uaddons: map[int64][]*domain.UserAddon{},
		orders:  map[int64]*domain.Order{},
		pays:    map[int64]*domain.Payment{},
		promos:  map[string]*domain.Promo{},
		balance: map[int64]*domain.AIBalance{},
		storage: map[int64]int64{},
	}
}

func (r *fakeRepo) next() int64 { r.seq++; return r.seq }

// ---- CatalogRepository ----

func (r *fakeRepo) ListPlans(context.Context, bool) ([]*domain.Plan, error) {
	out := []*domain.Plan{}
	for _, code := range domain.PlanCodes {
		out = append(out, r.plans[code])
	}
	return out, nil
}
func (r *fakeRepo) GetPlan(_ context.Context, code string) (*domain.Plan, error) {
	return r.plans[code], nil
}
func (r *fakeRepo) UpdatePlan(_ context.Context, p *domain.Plan) error { r.plans[p.Code] = p; return nil }
func (r *fakeRepo) ListAddons(context.Context, bool) ([]*domain.Addon, error) {
	out := []*domain.Addon{}
	for _, a := range r.addons {
		out = append(out, a)
	}
	return out, nil
}
func (r *fakeRepo) GetAddon(_ context.Context, code string) (*domain.Addon, error) {
	return r.addons[code], nil
}
func (r *fakeRepo) UpdateAddon(_ context.Context, a *domain.Addon) error {
	r.addons[a.Code] = a
	return nil
}

// ---- SubscriptionRepository ----

func (r *fakeRepo) GetSubscription(_ context.Context, userID int64) (*domain.Subscription, error) {
	return r.subs[userID], nil
}
func (r *fakeRepo) GetSubscriptions(_ context.Context, ids []int64) (map[int64]*domain.Subscription, error) {
	out := map[int64]*domain.Subscription{}
	for _, id := range ids {
		if s := r.subs[id]; s != nil {
			out[id] = s
		}
	}
	return out, nil
}
func (r *fakeRepo) SaveSubscription(_ context.Context, s *domain.Subscription) error {
	r.subs[s.UserID] = s
	return nil
}
func (r *fakeRepo) DeleteSubscription(_ context.Context, userID int64) error {
	delete(r.subs, userID)
	return nil
}
func (r *fakeRepo) ListSubscriptions(context.Context, string, string, int, int) ([]*domain.Subscription, int, error) {
	out := []*domain.Subscription{}
	for _, s := range r.subs {
		out = append(out, s)
	}
	return out, len(out), nil
}
func (r *fakeRepo) DueRenewals(_ context.Context, now time.Time, _ int) ([]*domain.Subscription, error) {
	out := []*domain.Subscription{}
	for _, s := range r.subs {
		if s.ExpiresAt != nil && !s.ExpiresAt.After(now) {
			out = append(out, s)
		}
	}
	return out, nil
}
func (r *fakeRepo) ListUserAddons(_ context.Context, userID int64) ([]*domain.UserAddon, error) {
	return r.uaddons[userID], nil
}
func (r *fakeRepo) ListUserAddonsFor(_ context.Context, ids []int64) (map[int64][]*domain.UserAddon, error) {
	out := map[int64][]*domain.UserAddon{}
	for _, id := range ids {
		out[id] = r.uaddons[id]
	}
	return out, nil
}
func (r *fakeRepo) AddAddon(_ context.Context, a *domain.UserAddon) error {
	a.ID = r.next()
	r.uaddons[a.UserID] = append(r.uaddons[a.UserID], a)
	return nil
}
func (r *fakeRepo) CancelAddon(_ context.Context, id, userID int64) error {
	list := r.uaddons[userID]
	for i, a := range list {
		if a.ID == id {
			r.uaddons[userID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}
func (r *fakeRepo) DueAddonRenewals(context.Context, time.Time, int) ([]*domain.UserAddon, error) {
	return nil, nil
}
func (r *fakeRepo) RenewAddon(context.Context, int64, time.Time) error { return nil }

// ---- OrderRepository ----

func (r *fakeRepo) CreateOrder(_ context.Context, o *domain.Order) error {
	o.ID = r.next()
	o.CreatedAt = time.Now()
	r.orders[o.ID] = o
	return nil
}
func (r *fakeRepo) GetOrder(_ context.Context, id int64) (*domain.Order, error) { return r.orders[id], nil }
func (r *fakeRepo) ListOrders(context.Context, int64, int, int) ([]*domain.Order, int, error) {
	return nil, 0, nil
}
func (r *fakeRepo) ListAllOrders(context.Context, string, int, int) ([]*domain.Order, int, error) {
	return nil, 0, nil
}
func (r *fakeRepo) SetOrderStatus(_ context.Context, id int64, status string, paidAt *time.Time) error {
	if o := r.orders[id]; o != nil {
		o.Status = status
		o.PaidAt = paidAt
	}
	return nil
}
func (r *fakeRepo) MarkApplied(_ context.Context, id int64) (bool, error) {
	o := r.orders[id]
	if o == nil || o.AppliedAt != nil {
		return false, nil
	}
	now := time.Now()
	o.AppliedAt = &now
	return true, nil
}
func (r *fakeRepo) CreatePayment(_ context.Context, p *domain.Payment) error {
	p.ID = r.next()
	r.pays[p.OrderID] = p
	return nil
}
func (r *fakeRepo) GetPayment(context.Context, int64) (*domain.Payment, error) { return nil, nil }
func (r *fakeRepo) GetPaymentByOrder(_ context.Context, orderID int64) (*domain.Payment, error) {
	return r.pays[orderID], nil
}
func (r *fakeRepo) SetPaymentStatus(_ context.Context, id int64, status string, _ map[string]any) error {
	for _, p := range r.pays {
		if p.ID == id {
			p.Status = status
		}
	}
	return nil
}

// ---- PromoRepository ----

func (r *fakeRepo) ListPromos(context.Context) ([]*domain.Promo, error) { return nil, nil }
func (r *fakeRepo) GetPromo(context.Context, int64) (*domain.Promo, error) {
	return nil, nil
}
func (r *fakeRepo) GetPromoByCode(_ context.Context, code string) (*domain.Promo, error) {
	return r.promos[code], nil
}
func (r *fakeRepo) CreatePromo(_ context.Context, p *domain.Promo) error {
	p.ID = r.next()
	r.promos[p.Code] = p
	return nil
}
func (r *fakeRepo) UpdatePromo(context.Context, *domain.Promo) error { return nil }
func (r *fakeRepo) DeletePromo(context.Context, int64) error         { return nil }
func (r *fakeRepo) CountRedemptions(context.Context, int64, int64) (int, error) {
	return 0, nil
}
func (r *fakeRepo) Redeem(_ context.Context, promoID, _ int64, _ *int64) (bool, error) {
	for _, p := range r.promos {
		if p.ID == promoID {
			if p.MaxUses > 0 && p.UsedCount >= p.MaxUses {
				return false, nil
			}
			p.UsedCount++
			return true, nil
		}
	}
	return false, nil
}

// ---- ProductRepository (в тестах не используется) ----

func (r *fakeRepo) ListShowcase(context.Context, string, string, int64, int, int) ([]*domain.Product, int, error) {
	return nil, 0, nil
}
func (r *fakeRepo) GetProduct(context.Context, int64) (*domain.Product, error) { return nil, nil }
func (r *fakeRepo) CreateProduct(context.Context, *domain.Product) error       { return nil }
func (r *fakeRepo) UpdateProduct(context.Context, *domain.Product) error       { return nil }
func (r *fakeRepo) SetProductStatus(context.Context, int64, string, string) error {
	return nil
}
func (r *fakeRepo) DeleteProduct(context.Context, int64) error { return nil }
func (r *fakeRepo) ListByAuthor(context.Context, int64) ([]*domain.Product, error) {
	return nil, nil
}
func (r *fakeRepo) ListModeration(context.Context) ([]*domain.Product, error) { return nil, nil }
func (r *fakeRepo) PurchaseProduct(context.Context, int64, int64, *int64, int64, int64, *int64) error {
	return nil
}
func (r *fakeRepo) ListPurchases(context.Context, int64) ([]*domain.ProductPurchase, error) {
	return nil, nil
}
func (r *fakeRepo) IsOwned(context.Context, int64, int64) (bool, error) { return false, nil }
func (r *fakeRepo) GetSellerBalance(_ context.Context, userID int64) (*domain.SellerBalance, error) {
	return &domain.SellerBalance{UserID: userID}, nil
}
func (r *fakeRepo) CreatePayout(context.Context, *domain.Payout) error { return nil }
func (r *fakeRepo) ListPayouts(context.Context, int64, bool) ([]*domain.Payout, error) {
	return nil, nil
}
func (r *fakeRepo) ProcessPayout(context.Context, int64, string, string) error { return nil }

// ---- AIRepository ----

func (r *fakeRepo) GetBalance(_ context.Context, userID int64) (*domain.AIBalance, error) {
	return r.balance[userID], nil
}
func (r *fakeRepo) EnsureBalance(_ context.Context, userID int64, planTokens int64, now time.Time) (*domain.AIBalance, error) {
	b := r.balance[userID]
	if b == nil {
		b = &domain.AIBalance{UserID: userID, PeriodStart: now, PeriodEnd: now.AddDate(0, 1, 0)}
		r.balance[userID] = b
	}
	if !b.PeriodEnd.After(now) {
		b.UsedTokens = 0
		b.PeriodStart, b.PeriodEnd = now, now.AddDate(0, 1, 0)
	}
	b.PlanTokens = planTokens
	return b, nil
}
func (r *fakeRepo) Consume(_ context.Context, userID int64, tokens int64) (bool, *domain.AIBalance, error) {
	b := r.balance[userID]
	if b == nil || b.Left() < tokens {
		return false, nil, nil
	}
	free := b.PlanTokens - b.UsedTokens
	if free >= tokens {
		b.UsedTokens += tokens
	} else {
		b.UsedTokens = b.PlanTokens
		b.ExtraTokens -= tokens - max(free, 0)
	}
	return true, b, nil
}
func (r *fakeRepo) AddExtraTokens(_ context.Context, userID int64, tokens int64) error {
	b := r.balance[userID]
	if b == nil {
		b = &domain.AIBalance{UserID: userID, PeriodEnd: time.Now().AddDate(0, 1, 0)}
		r.balance[userID] = b
	}
	b.ExtraTokens += tokens
	return nil
}
func (r *fakeRepo) SetBalance(_ context.Context, userID int64, plan, used, extra int64) error {
	r.balance[userID] = &domain.AIBalance{UserID: userID, PlanTokens: plan, UsedTokens: used,
		ExtraTokens: extra, PeriodEnd: time.Now().AddDate(0, 1, 0)}
	return nil
}
func (r *fakeRepo) LogUsage(context.Context, domain.AIUsageRecord) error { return nil }
func (r *fakeRepo) UsageByFeature(context.Context, int64, time.Time) (map[string]int64, error) {
	return map[string]int64{}, nil
}
func (r *fakeRepo) TotalUsage(context.Context, time.Time) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// ---- StorageRepository ----

func (r *fakeRepo) Usage(context.Context, int64) ([]*domain.StorageEntry, error) { return nil, nil }
func (r *fakeRepo) Total(_ context.Context, userID int64) (int64, error) {
	return r.storage[userID], nil
}
func (r *fakeRepo) TotalsFor(_ context.Context, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	for _, id := range ids {
		out[id] = r.storage[id]
	}
	return out, nil
}
func (r *fakeRepo) Track(_ context.Context, userID int64, _ string, delta int64) (int64, error) {
	r.storage[userID] += delta
	if r.storage[userID] < 0 {
		r.storage[userID] = 0
	}
	return r.storage[userID], nil
}
func (r *fakeRepo) Set(_ context.Context, userID int64, _ string, bytes int64) error {
	r.storage[userID] = bytes
	return nil
}

// ---- Settings/Audit ----

func (r *fakeRepo) GetSettings(context.Context) (*domain.Settings, error) {
	return &domain.Settings{CommissionPct: 10, PaymentProvider: "manual", StoreEnabled: true}, nil
}
func (r *fakeRepo) UpdateSettings(context.Context, *domain.Settings) error { return nil }
func (r *fakeRepo) LogAction(_ context.Context, e *domain.AuditEntry) error {
	r.audit = append(r.audit, e)
	return nil
}
func (r *fakeRepo) ListAudit(context.Context, string, int, int) ([]*domain.AuditEntry, int, error) {
	return r.audit, len(r.audit), nil
}

// ---- IdentityReader ----

type fakeIdentity struct {
	owners map[int64]int64 // company_id → created_by
}

func (f *fakeIdentity) GetUser(_ context.Context, id int64) (*domain.User, error) {
	return &domain.User{ID: id, FIO: "Тестовый", Login: "test", IsActive: true}, nil
}
func (f *fakeIdentity) SearchUsers(context.Context, string, int) ([]*domain.User, error) {
	return nil, nil
}
func (f *fakeIdentity) CompanyOwner(_ context.Context, companyID int64) (int64, error) {
	return f.owners[companyID], nil
}
func (f *fakeIdentity) OwnedCompanies(context.Context, int64) ([]int64, error) { return nil, nil }
func (f *fakeIdentity) IsCompanyMember(context.Context, int64, int64) (bool, error) {
	return true, nil
}

func newService(t *testing.T) (*Service, *fakeRepo, *fakeIdentity) {
	t.Helper()
	repo := newRepo()
	identity := &fakeIdentity{owners: map[int64]int64{}}
	svc := New(Deps{
		Catalog: repo, Subs: repo, Orders: repo, Promos: repo, Products: repo,
		AI: repo, Storage: repo, Settings: repo, Audit: repo,
		Identity: identity, Provider: payments.NewManual(),
		Log: slog.New(slog.DiscardHandler),
	})
	return svc, repo, identity
}

// Без подписки действует бесплатный тариф со своими лимитами.
func TestEntitlementsDefaultsToFreePlan(t *testing.T) {
	svc, _, _ := newService(t)
	ent, err := svc.Entitlements(context.Background(), 1, 0)
	if err != nil {
		t.Fatalf("Entitlements: %v", err)
	}
	if ent.Plan != domain.PlanJunior {
		t.Fatalf("ожидался бесплатный тариф, получили %q", ent.Plan)
	}
	if ent.Limits.Companies != 1 || ent.Limits.Boards != 1 {
		t.Fatalf("лимиты «Джуна» не совпали: %+v", ent.Limits)
	}
	if ent.Limits.Portal {
		t.Fatal("портал не входит в бесплатный тариф")
	}
}

// Компания наследует тариф своего СОЗДАТЕЛЯ, а не того, кто спрашивает.
func TestCompanyInheritsOwnerPlan(t *testing.T) {
	svc, repo, identity := newService(t)
	until := time.Now().AddDate(0, 1, 0)
	repo.subs[7] = &domain.Subscription{UserID: 7, PlanCode: domain.PlanSenior, ExpiresAt: &until}
	identity.owners[42] = 7

	ent, err := svc.Entitlements(context.Background(), 99, 42)
	if err != nil {
		t.Fatalf("Entitlements: %v", err)
	}
	if ent.Plan != domain.PlanSenior || ent.OwnerID != 7 {
		t.Fatalf("ожидался тариф создателя компании, получили %q (owner %d)", ent.Plan, ent.OwnerID)
	}
	if !ent.Limits.Portal || ent.Limits.Members != 15 {
		t.Fatalf("лимиты «Синьора» не применились: %+v", ent.Limits)
	}
}

// Истёкшая подписка не действует — лимиты падают до бесплатных.
func TestExpiredSubscriptionIsIgnored(t *testing.T) {
	svc, repo, _ := newService(t)
	past := time.Now().Add(-time.Hour)
	repo.subs[1] = &domain.Subscription{UserID: 1, PlanCode: domain.PlanSenior, ExpiresAt: &past}

	ent, _ := svc.Entitlements(context.Background(), 1, 0)
	if ent.Plan != domain.PlanJunior {
		t.Fatalf("истёкшая подписка не должна действовать, получили %q", ent.Plan)
	}
}

// Докупленное место складывается с лимитом тарифа.
func TestStorageAddonExtendsLimit(t *testing.T) {
	svc, repo, _ := newService(t)
	repo.uaddons[1] = []*domain.UserAddon{{
		ID: 1, UserID: 1, AddonCode: "storage_10", Kind: domain.AddonStorage,
		Amount: 10 * domain.GiB, Qty: 1,
	}}
	ent, _ := svc.Entitlements(context.Background(), 1, 0)
	want := 5*domain.GiB + 10*domain.GiB // «Джун» + докупка
	if ent.Limits.StorageBytes != want {
		t.Fatalf("ожидалось %d байт, получили %d", want, ent.Limits.StorageBytes)
	}
}

// Место сотрудника действует только в той компании, куда его купили.
func TestMemberAddonIsCompanyScoped(t *testing.T) {
	svc, repo, identity := newService(t)
	identity.owners[10] = 1
	identity.owners[20] = 1
	company := int64(10)
	repo.uaddons[1] = []*domain.UserAddon{{
		ID: 1, UserID: 1, Kind: domain.AddonMember, Amount: 2, Qty: 1, CompanyID: &company,
	}}

	own, _ := svc.Entitlements(context.Background(), 1, 10)
	other, _ := svc.Entitlements(context.Background(), 1, 20)
	if own.Limits.Members != 5 { // 3 у «Джуна» + 2 докупленных
		t.Fatalf("в своей компании ожидалось 5 мест, получили %d", own.Limits.Members)
	}
	if other.Limits.Members != 3 {
		t.Fatalf("в чужой компании докупка не действует, ожидалось 3, получили %d", other.Limits.Members)
	}
}

// Покупка тарифа: заказ оплачен вручную → подписка выдана ровно один раз.
func TestPurchaseGrantsSubscriptionOnce(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()

	order, err := svc.Purchase(ctx, 1, PurchaseRequest{
		Kind: domain.OrderKindSubscription, ItemCode: domain.PlanMiddle, Period: domain.PeriodMonth,
	})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if order.Amount != 29900 || order.Status != domain.OrderPending {
		t.Fatalf("неожиданный заказ: %+v", order)
	}

	if _, err := svc.ConfirmOrder(ctx, 1, order.ID); err != nil {
		t.Fatalf("ConfirmOrder: %v", err)
	}
	sub := repo.subs[1]
	if sub == nil || sub.PlanCode != domain.PlanMiddle {
		t.Fatalf("подписка не выдана: %+v", sub)
	}
	first := *sub.ExpiresAt

	// Повторное подтверждение того же заказа не должно продлевать подписку.
	if _, err := svc.ConfirmOrder(ctx, 1, order.ID); !errors.Is(err, domain.ErrOrderFinished) {
		t.Fatalf("ожидался отказ ORDER_FINISHED, получили %v", err)
	}
	if !repo.subs[1].ExpiresAt.Equal(first) {
		t.Fatal("повторное подтверждение продлило подписку — заказ применён дважды")
	}
}

// Годовая оплата даёт год, а не месяц.
func TestYearPeriodGrantsYear(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()
	order, err := svc.Purchase(ctx, 2, PurchaseRequest{
		Kind: domain.OrderKindSubscription, ItemCode: domain.PlanSenior, Period: domain.PeriodYear,
	})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if order.Amount != 478800 {
		t.Fatalf("годовая цена не применилась: %d", order.Amount)
	}
	if _, err := svc.ConfirmOrder(ctx, 1, order.ID); err != nil {
		t.Fatalf("ConfirmOrder: %v", err)
	}
	got := repo.subs[2].ExpiresAt
	if got.Before(time.Now().AddDate(0, 11, 0)) {
		t.Fatalf("ожидался год подписки, получили %v", got)
	}
}

// Промокод-скидка уменьшает сумму заказа; 100%% — покупка без оплаты.
func TestPromoDiscount(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()
	repo.promos["HALF"] = &domain.Promo{
		ID: 100, Code: "HALF", Kind: domain.PromoPercent, Value: 50,
		AppliesTo: "any", PerUserLimit: 1, IsActive: true,
	}
	repo.promos["FREE"] = &domain.Promo{
		ID: 101, Code: "FREE", Kind: domain.PromoPercent, Value: 100,
		AppliesTo: "any", PerUserLimit: 1, IsActive: true,
	}

	q, err := svc.Quote(ctx, 1, PurchaseRequest{
		Kind: domain.OrderKindSubscription, ItemCode: domain.PlanMiddle, Promo: "HALF",
	})
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if q.Amount != 14950 {
		t.Fatalf("ожидалась половина цены, получили %d", q.Amount)
	}

	order, err := svc.Purchase(ctx, 3, PurchaseRequest{
		Kind: domain.OrderKindSubscription, ItemCode: domain.PlanMiddle, Promo: "FREE",
	})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if order.Status != domain.OrderPaid {
		t.Fatalf("бесплатный заказ должен примениться сразу: %s", order.Status)
	}
	if repo.subs[3] == nil {
		t.Fatal("подписка по стопроцентному промокоду не выдана")
	}
}

// Токены: списание из квоты тарифа, потом из докупленных; при нехватке отказ.
func TestConsumeAIUsesQuotaThenExtra(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()
	until := time.Now().AddDate(0, 1, 0)
	repo.subs[5] = &domain.Subscription{UserID: 5, PlanCode: domain.PlanMiddle, ExpiresAt: &until}
	if _, err := repo.EnsureBalance(ctx, 5, 1000, time.Now()); err != nil {
		t.Fatalf("EnsureBalance: %v", err)
	}
	repo.balance[5].ExtraTokens = 200

	ok, left, err := svc.ConsumeAI(ctx, domain.AIUsageRecord{UserID: 5, Feature: "assistant", BilledTokens: 900})
	if err != nil || !ok {
		t.Fatalf("первое списание не прошло: ok=%v err=%v", ok, err)
	}
	if left != 300 { // 1000-900 квоты + 200 докупленных
		t.Fatalf("ожидался остаток 300, получили %d", left)
	}

	ok, _, _ = svc.ConsumeAI(ctx, domain.AIUsageRecord{UserID: 5, Feature: "assistant", BilledTokens: 400})
	if ok {
		t.Fatal("списание сверх остатка должно отклоняться")
	}
}

// На своём ключе токены не списываются — только фиксируется расход.
func TestConsumeAIOwnKeyDoesNotSpend(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()
	if _, err := repo.EnsureBalance(ctx, 6, 1000, time.Now()); err != nil {
		t.Fatalf("EnsureBalance: %v", err)
	}
	ok, _, err := svc.ConsumeAI(ctx, domain.AIUsageRecord{
		UserID: 6, Feature: "assistant", BilledTokens: 5000, OwnKey: true,
	})
	if err != nil || !ok {
		t.Fatalf("свой ключ не должен упираться в квоту: ok=%v err=%v", ok, err)
	}
	if repo.balance[6].UsedTokens != 0 {
		t.Fatalf("на своём ключе расход квоты не списывается, получили %d", repo.balance[6].UsedTokens)
	}
}

// Проверка места: файл больше остатка не проходит.
func TestCheckStorage(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()
	repo.storage[1] = 5*domain.GiB - 1024 // «Джун» — 5 Гб

	ok, free, _, err := svc.CheckStorage(ctx, 1, 0, 4096)
	if err != nil {
		t.Fatalf("CheckStorage: %v", err)
	}
	if ok {
		t.Fatalf("файл не должен влезть, свободно %d", free)
	}
	if ok, _, _, _ := svc.CheckStorage(ctx, 1, 0, 512); !ok {
		t.Fatal("файл в пределах остатка должен проходить")
	}
}

// Истёкшая подписка без автопродления снимается планировщиком.
func TestSchedulerDropsExpiredSubscription(t *testing.T) {
	svc, repo, _ := newService(t)
	past := time.Now().Add(-time.Minute)
	repo.subs[9] = &domain.Subscription{
		UserID: 9, PlanCode: domain.PlanMiddle, Source: domain.SourceGrace,
		ExpiresAt: &past, AutoRenew: false,
	}
	svc.tick(context.Background())
	if repo.subs[9] != nil {
		t.Fatal("истёкшая подписка должна сниматься — иначе планировщик крутит её вечно")
	}
}

// Автопродление выставляет счёт на новый период.
func TestSchedulerCreatesRenewalOrder(t *testing.T) {
	svc, repo, _ := newService(t)
	past := time.Now().Add(-time.Minute)
	repo.subs[11] = &domain.Subscription{
		UserID: 11, PlanCode: domain.PlanMiddle, Period: domain.PeriodMonth,
		Source: domain.SourcePurchase, ExpiresAt: &past, AutoRenew: true,
	}
	svc.tick(context.Background())

	found := false
	for _, o := range repo.orders {
		if o.UserID == 11 && o.Kind == domain.OrderKindSubscription && o.Status == domain.OrderPending {
			found = true
		}
	}
	if !found {
		t.Fatal("автопродление должно выставлять счёт на новый период")
	}
}

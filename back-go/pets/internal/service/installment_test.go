package service

import (
	"context"
	"testing"
	"time"
)

// Открытие рассрочки, лимит счёта, платёж части и полное погашение.
func TestInstallmentPayAndLimit(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 1000

	if err := env.svc.openInstallment(ctx, 1, 10, "shop", "hat", "hat", 200); err != nil {
		t.Fatalf("openInstallment: %v", err)
	}
	d, err := env.svc.GetInstallments(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetInstallments: %v", err)
	}
	if d.Used != 200 || d.Available != 300 || len(d.Items) != 1 {
		t.Fatalf("сводка: used=%d avail=%d items=%d", d.Used, d.Available, len(d.Items))
	}
	if d.Items[0].PartAmount != 50 { // 200 / 4
		t.Errorf("размер доли: %d", d.Items[0].PartAmount)
	}
	// Лимит: ещё 400 не влезет (200 + 400 > 500).
	if err := env.svc.checkInstallmentLimit(ctx, 1, 400); errCode(err) != "INSTALLMENT_LIMIT" {
		t.Errorf("лимит рассрочки: %v", err)
	}

	id := d.Items[0].ID
	d, err = env.svc.PayInstallment(ctx, 1, 10, id, 50)
	if err != nil {
		t.Fatalf("PayInstallment: %v", err)
	}
	if d.Items[0].Paid != 50 || d.Items[0].Outstanding != 150 {
		t.Errorf("после платежа: paid=%d out=%d", d.Items[0].Paid, d.Items[0].Outstanding)
	}
	if env.pets.byUser[1].Kudos != 950 {
		t.Errorf("кошелёк после платежа: %d", env.pets.byUser[1].Kudos)
	}

	// «Погасить всё» — сумма клампится к остатку, рассрочка закрывается.
	d, err = env.svc.PayInstallment(ctx, 1, 10, id, 9999)
	if err != nil {
		t.Fatalf("PayInstallment (всё): %v", err)
	}
	if len(d.Items) != 0 || d.Used != 0 || d.Available != 500 {
		t.Errorf("после полного погашения: %+v", d)
	}
	if env.pets.byUser[1].Kudos != 800 {
		t.Errorf("кошелёк после полного: %d", env.pets.byUser[1].Kudos)
	}
}

// Просрочка рассрочки: за каждую неделю без платежа на остаток капает 20%.
func TestInstallmentWeeklyPenalty(t *testing.T) {
	env := newEnv()
	ctx := context.Background()

	pet, _ := env.pets.GetOrCreate(ctx, 1, 10)
	pet.Kudos = 1000
	if err := env.svc.openInstallment(ctx, 1, 10, "shop", "hat", "hat", 100); err != nil {
		t.Fatalf("openInstallment: %v", err)
	}
	// Срок прошёл 8 дней назад → одна полная неделя просрочки + текущая = 2 начисления.
	env.inst.rows[0].DueAt = time.Now().Add(-8 * 24 * time.Hour)

	d, err := env.svc.GetInstallments(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetInstallments: %v", err)
	}
	// 100 → +20% = 120 → +20% = 144.
	if d.Items[0].Total != 144 || !d.Items[0].Overdue || !d.Items[0].Penalized {
		t.Fatalf("пеня рассрочки: total=%d overdue=%v penalized=%v",
			d.Items[0].Total, d.Items[0].Overdue, d.Items[0].Penalized)
	}
	// Повторный read не начисляет второй раз (чекпоинт сдвинут вперёд).
	d, _ = env.svc.GetInstallments(ctx, 1, 10)
	if d.Items[0].Total != 144 {
		t.Errorf("пеня начислена дважды: total=%d", d.Items[0].Total)
	}
}

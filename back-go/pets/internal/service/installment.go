package service

import (
	"context"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/dto"
)

// Рассрочка — кредитный счёт на оплату покупок долями. Товар выдаётся сразу,
// суммарный непогашенный долг по всем рассрочкам не превышает InstallmentLimit.
// Пропущенная неделя без платежа наращивает долг (штраф LoanWeeklyPenaltyPct на
// остаток); после InstallmentRepossessWeeks недель просрочки предмет изымается.

// checkInstallmentLimit — есть ли на кредитном счёте место под новую покупку.
func (s *Service) checkInstallmentLimit(ctx context.Context, userID int64, price int) error {
	outstanding, err := s.installments.Outstanding(ctx, userID)
	if err != nil {
		return err
	}
	if outstanding+price > domain.InstallmentLimit {
		return domain.NewError("INSTALLMENT_LIMIT",
			"Лимит рассрочки — "+strconv.Itoa(domain.InstallmentLimit)+
				" кудосов; свободно "+strconv.Itoa(domain.InstallmentLimit-outstanding), 422)
	}
	return nil
}

// openInstallment — открыть рассрочку на уже выданный товар (вызывается из
// покупок в режиме «частями»). Лимит проверен раньше — до выдачи товара.
func (s *Service) openInstallment(ctx context.Context, userID, companyID int64,
	category, itemKey, itemTitle string, price int) error {

	return s.installments.Create(ctx, &domain.Installment{
		UserID: userID, CompanyID: companyID, Category: category,
		ItemKey: itemKey, ItemTitle: itemTitle, Total: price, Parts: domain.InstallmentParts,
		DueAt: time.Now().Add(domain.InstallmentWeekDays * 24 * time.Hour),
	})
}

// GetInstallments — сводка рассрочек: ленивое начисление просрочки и изъятие
// заброшенных, затем список активных с лимитом/остатком счёта.
func (s *Service) GetInstallments(ctx context.Context, userID, companyID int64) (*dto.InstallmentsDTO, error) {
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	items, err := s.installments.ListActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	kept := make([]*domain.Installment, 0, len(items))
	for _, i := range items {
		i = s.ensureInstallmentCharges(ctx, i)
		if s.maybeRepossess(ctx, i) {
			continue // предмет изъят, рассрочка закрыта
		}
		kept = append(kept, i)
	}
	used, err := s.installments.Outstanding(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.NewInstallments(kept, used), nil
}

// PayInstallment — платёж по рассрочке (сумма сверх остатка клампится). Перед
// платежом начисляется накопленная просрочка.
func (s *Service) PayInstallment(ctx context.Context, userID, companyID, id int64, amount int) (*dto.InstallmentsDTO, error) {
	if amount < 1 {
		return nil, domain.NewError("VALIDATION", "Сумма должна быть положительной", 422)
	}
	inst, err := s.installments.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if inst == nil || inst.Outstanding() <= 0 {
		return nil, domain.NewError("NO_INSTALLMENT", "Рассрочка не найдена или уже закрыта", 404)
	}
	inst = s.ensureInstallmentCharges(ctx, inst)
	pay := min(amount, inst.Outstanding())
	_, _, _, ok, err := s.installments.Pay(ctx, id, userID, pay)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов для платежа", 422)
	}
	s.appendActivity(ctx, userID, "installment_pay", map[string]any{"id": id, "amount": pay})
	if p, err := s.pets.GetPet(ctx, userID); err == nil && p != nil {
		s.emitPetUpdate(ctx, p)
	}
	return s.GetInstallments(ctx, userID, companyID)
}

// ensureInstallmentCharges — ленивое еженедельное начисление на остаток
// просроченной рассрочки (штраф LoanWeeklyPenaltyPct на остаток, компаунд).
func (s *Service) ensureInstallmentCharges(ctx context.Context, i *domain.Installment) *domain.Installment {
	if i.Outstanding() <= 0 || time.Now().Before(i.DueAt) {
		return i
	}
	if p, err := s.pets.GetPet(ctx, i.UserID); err == nil && p != nil && p.OwnerOnVacation {
		return i // хозяин в отпуске — рассрочка на паузе
	}
	week := domain.InstallmentWeekDays * 24 * time.Hour
	weeks := int(time.Since(i.DueAt)/week) + 1
	remaining := i.Outstanding()
	charge := 0
	for k := 0; k < weeks; k++ {
		add := ((remaining + charge) * domain.LoanWeeklyPenaltyPct + 99) / 100
		charge += add
	}
	newDue := i.DueAt.Add(time.Duration(weeks) * week)
	ok, err := s.installments.AddCharge(ctx, i.ID, charge, i.DueAt, newDue)
	if err != nil {
		s.log.Warn("pets.installment_charge_failed", "id", i.ID, "error", err)
		return i
	}
	if !ok {
		return i
	}
	s.appendActivity(ctx, i.UserID, "installment_penalty", map[string]any{"id": i.ID, "amount": charge})
	if fresh, err := s.installments.Get(ctx, i.ID, i.UserID); err == nil && fresh != nil {
		return fresh
	}
	return i
}

// maybeRepossess — изъять предмет и закрыть рассрочку, если она заброшена: были
// пропуски платежа (penalized) и с покупки прошло больше InstallmentRepossessWeeks
// недель, а долг всё ещё висит. Возвращает true, если изъятие произошло.
func (s *Service) maybeRepossess(ctx context.Context, i *domain.Installment) bool {
	week := domain.InstallmentWeekDays * 24 * time.Hour
	deadline := time.Duration(domain.InstallmentRepossessWeeks+1) * week
	if !i.Penalized || i.Outstanding() <= 0 || time.Since(i.CreatedAt) < deadline {
		return false
	}
	pet, err := s.pets.GetPet(ctx, i.UserID)
	if err != nil || pet == nil {
		return false
	}
	repossessItem(pet, i.Category, i.ItemKey)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		s.log.Warn("pets.repossess_save_failed", "id", i.ID, "error", err)
		return false
	}
	if err := s.installments.Delete(ctx, i.ID); err != nil {
		s.log.Warn("pets.repossess_delete_failed", "id", i.ID, "error", err)
		return false
	}
	s.appendActivity(ctx, i.UserID, "installment_repossess",
		map[string]any{"id": i.ID, "item": i.ItemKey, "category": i.Category})
	s.emitPetUpdate(ctx, pet)
	return true
}

// repossessItem — снять изъятый предмет с питомца по категории покупки.
func repossessItem(pet *domain.Pet, category, key string) {
	switch category {
	case "shop":
		pet.Accessories = removeStr(pet.Accessories, key)
		if pet.Hat != nil && *pet.Hat == key {
			pet.Hat = nil
		}
	case "house":
		pet.HouseOwned = removeStr(pet.HouseOwned, key)
		placed := pet.HousePlaced[:0]
		for _, it := range pet.HousePlaced {
			if it.Key != key {
				placed = append(placed, it)
			}
		}
		pet.HousePlaced = placed
	}
}

func removeStr(list []string, key string) []string {
	out := list[:0]
	for _, v := range list {
		if v != key {
			out = append(out, v)
		}
	}
	return out
}

package service

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/dto"
)

// Магазин питомца: витрина живёт в БД (pet_shop_items), а не в Go-константах,
// чтобы ассортимент можно было ротировать без деплоя. Четыре механики
// нестатичности одновременно: постоянные товары, ротация по датам
// (active_from/active_to), ограниченный тираж на компанию (limited_quota) и
// достижения (unlock_kind=achievement, не продаются за кудосы).

// ── Скидка дня ──────────────────────────────────────────────────────
// «Флеш-распродажа»: детерминированно от даты (МСК) и ключа товара часть
// витрины уценивается до конца дня — за скидками стоит следить. Без
// состояния и фоновых процессов: формула одинаково считается в витрине и
// при покупке, в полночь МСК набор скидок сам меняется.

const (
	saleChanceBigPct   = 7  // доля витрины с -50%
	saleChanceSmallPct = 15 // доля витрины с -30%
)

func saleDiscountPct(key string, day time.Time) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(day.Format("2006-01-02") + "|" + key))
	v := int(h.Sum32() % 100)
	switch {
	case v < saleChanceBigPct:
		return 50
	case v < saleChanceBigPct+saleChanceSmallPct:
		return 30
	default:
		return 0
	}
}

// salePrice — цена товара с учётом скидки дня (0 скидки для достижений и
// бесплатных позиций).
func salePrice(item *domain.ShopItem, day time.Time) (price, discountPct int) {
	price = item.PriceKudos
	if item.UnlockKind != "shop" || price <= 0 {
		return price, 0
	}
	if pct := saleDiscountPct(item.Key, day); pct > 0 {
		return price - price*pct/100, pct
	}
	return price, 0
}

// isSpeciesOwned — уже разблокирован ли вид (магазинный или природный).
func isSpeciesOwned(pet *domain.Pet, key string) bool {
	naturalOK := domain.NaturalSpecies[key] && pet.Stage >= 2
	return containsStr(pet.UnlockedSpecies, key) || naturalOK
}

// remainingQuota — остаток лимитированного тиража; nil — товар не лимитирован.
func (s *Service) remainingQuota(ctx context.Context, item *domain.ShopItem, companyID int64) (*int, error) {
	if item.LimitedQuota == nil {
		return nil, nil
	}
	bought, err := s.shop.CountPurchases(ctx, item.ID, companyID)
	if err != nil {
		return nil, err
	}
	left := max(0, *item.LimitedQuota-bought)
	return &left, nil
}

func newShopItemDTO(item *domain.ShopItem, remaining *int, owned bool) *dto.ShopItemDTO {
	d := &dto.ShopItemDTO{
		Key: item.Key, Kind: item.Kind, Rarity: item.Rarity,
		PriceKudos: item.PriceKudos, UnlockKind: item.UnlockKind,
		AchievementKey: item.AchievementKey, LimitedQuota: item.LimitedQuota,
		Remaining: remaining, Owned: owned,
	}
	if sale, pct := salePrice(item, todayMSK()); pct > 0 {
		d.SalePriceKudos, d.DiscountPct = &sale, &pct
	}
	if item.ActiveFrom != nil {
		s := item.ActiveFrom.UTC().Format(time.RFC3339)
		d.ActiveFrom = &s
	}
	if item.ActiveTo != nil {
		s := item.ActiveTo.UTC().Format(time.RFC3339)
		d.ActiveTo = &s
	}
	d.SoldOut = remaining != nil && *remaining <= 0
	return d
}

// GetShopState — витрина: постоянные товары + активные сейчас ротационные,
// с остатком лимитированного тиража и владением для текущего питомца.
// mystery_taken — тот же дневной ключ, которым GetMysteryItem детектит повтор.
func (s *Service) GetShopState(ctx context.Context, userID, companyID int64) (*dto.ShopDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	items, err := s.shop.ListActiveItems(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	out := make([]*dto.ShopItemDTO, 0, len(items))
	for _, item := range items {
		remaining, err := s.remainingQuota(ctx, item, companyID)
		if err != nil {
			return nil, err
		}
		owned := false
		if item.Kind == "species" {
			owned = isSpeciesOwned(pet, item.Key)
		} else {
			owned = containsStr(pet.Accessories, item.Key)
		}
		out = append(out, newShopItemDTO(item, remaining, owned))
	}
	return &dto.ShopDTO{
		Items:        out,
		MysteryTaken: s.daily.Exists(ctx, mysteryDailyKey(userID)),
	}, nil
}

// purchaseItem — общая логика покупки: проверки окна/тиража/владения/цены,
// списание кудосов (по цене со скидкой дня), применение эффекта. paid —
// фактически списанная сумма (для выписки банка).
func (s *Service) purchaseItem(ctx context.Context, userID, companyID int64,
	key, wantKind string, installment bool) (*domain.Pet, *domain.ShopItem, int, error) {

	item, err := s.shop.GetItem(ctx, key)
	if err != nil {
		return nil, nil, 0, err
	}
	if item == nil || item.Kind != wantKind {
		return nil, nil, 0, domain.NewError("NO_ITEM", "Такого товара нет", 404)
	}
	if !item.Active(time.Now()) {
		return nil, nil, 0, domain.NewError("OUT_OF_SEASON", "Этот товар сейчас не в продаже", 422)
	}
	if item.UnlockKind == "achievement" {
		return nil, nil, 0, domain.NewError("ACHIEVEMENT_ONLY", "Этот предмет — достижение, не продаётся", 422)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, nil, 0, err
	}
	owned := containsStr(pet.Accessories, key)
	if wantKind == "species" {
		owned = isSpeciesOwned(pet, key)
	}
	if owned {
		return nil, nil, 0, domain.NewError("ALREADY_OWNED", "Уже куплено", 422)
	}
	if item.LimitedQuota != nil {
		// Превентивная проверка витрины; авторитетная — атомарно в
		// RecordPurchase (одна транзакция COUNT+INSERT под локом товара).
		remaining, err := s.remainingQuota(ctx, item, companyID)
		if err != nil {
			return nil, nil, 0, err
		}
		if remaining != nil && *remaining <= 0 {
			return nil, nil, 0, domain.ErrSoldOut
		}
	}
	price, discountPct := salePrice(item, todayMSK())
	if installment {
		// Рассрочка — только не-акционные товары; акционные берут сразу или в кредит.
		if discountPct > 0 {
			return nil, nil, 0, domain.NewError("INSTALLMENT_SALE",
				"Акционные товары нельзя в рассрочку — купите сразу или возьмите кредит", 422)
		}
		if err := s.checkInstallmentLimit(ctx, userID, price); err != nil {
			return nil, nil, 0, err
		}
		return pet, item, price, nil // без списания — оплата долями
	}
	if pet.Kudos < price {
		return nil, nil, 0, domain.NewError("NO_KUDOS", "Не хватает кудосов", 422)
	}
	pet.Kudos -= price
	return pet, item, price, nil
}

// BuyItem — купить и сразу надеть аксессуар/скин. Лимитированный тираж
// резервируется ДО SavePet: если тираж распродан (RecordPurchase → SOLD_OUT),
// кудосы покупателя не списываются.
func (s *Service) BuyItem(ctx context.Context, userID, companyID int64, key string, installment bool) (*dto.PetDTO, error) {
	pet, item, paid, err := s.buyByKinds(ctx, userID, companyID, key, installment, "skin", "accessory")
	if err != nil {
		return nil, err
	}
	if item.LimitedQuota != nil {
		if err := s.shop.RecordPurchase(ctx, item.ID, companyID, userID, item.LimitedQuota); err != nil {
			return nil, err
		}
	}
	pet.Accessories = append(pet.Accessories, key)
	pet.Hat = &key
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.appendActivity(ctx, userID, "item_bought", map[string]any{"key": key})
	if installment {
		if err := s.openInstallment(ctx, userID, companyID, "shop", key, key, paid); err != nil {
			s.log.Warn("pets.open_installment_failed", "user_id", userID, "key", key, "error", err)
		}
	} else {
		s.appendLedger(ctx, userID, companyID, -paid, "shop", nil, key)
	}
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// buyByKinds — purchaseItem, допускающий несколько kind (skin/accessory —
// разные ярлыки одного и того же способа ношения).
func (s *Service) buyByKinds(ctx context.Context, userID, companyID int64,
	key string, installment bool, kinds ...string) (*domain.Pet, *domain.ShopItem, int, error) {

	item, err := s.shop.GetItem(ctx, key)
	if err != nil {
		return nil, nil, 0, err
	}
	kindOK := false
	for _, k := range kinds {
		if item != nil && item.Kind == k {
			kindOK = true
		}
	}
	if item == nil || !kindOK {
		return nil, nil, 0, domain.NewError("NO_ITEM", "Такого товара нет", 404)
	}
	return s.purchaseItem(ctx, userID, companyID, key, item.Kind, installment)
}

func (s *Service) EquipItem(ctx context.Context, userID, companyID int64, item *string) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if item != nil && !containsStr(pet.Accessories, *item) {
		return nil, domain.NewError("NOT_OWNED", "Аксессуар не куплен", 422)
	}
	pet.Hat = item
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	if item != nil {
		s.appendActivity(ctx, userID, "item_equipped", map[string]any{"key": *item})
	}
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// BuySpecies — разблокировать новый облик питомца и сразу его надеть.
// Порядок тот же, что в BuyItem: резерв тиража до SavePet.
func (s *Service) BuySpecies(ctx context.Context, userID, companyID int64, species string, installment bool) (*dto.PetDTO, error) {
	pet, item, paid, err := s.purchaseItem(ctx, userID, companyID, species, "species", installment)
	if err != nil {
		return nil, err
	}
	if item.LimitedQuota != nil {
		if err := s.shop.RecordPurchase(ctx, item.ID, companyID, userID, item.LimitedQuota); err != nil {
			return nil, err
		}
	}
	pet.UnlockedSpecies = append(pet.UnlockedSpecies, species)
	pet.Species = species
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.appendActivity(ctx, userID, "item_bought", map[string]any{"key": species})
	if installment {
		if err := s.openInstallment(ctx, userID, companyID, "shop", species, species, paid); err != nil {
			s.log.Warn("pets.open_installment_failed", "user_id", userID, "key", species, "error", err)
		}
	} else {
		s.appendLedger(ctx, userID, companyID, -paid, "shop", nil, species)
	}
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// SwitchSpecies — сменить облик на ранее разблокированный (без оплаты).
func (s *Service) SwitchSpecies(ctx context.Context, userID, companyID int64, species string) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if !isSpeciesOwned(pet, species) {
		return nil, domain.NewError("NOT_OWNED", "Этот вид ещё не разблокирован", 422)
	}
	if !containsStr(pet.UnlockedSpecies, species) {
		pet.UnlockedSpecies = append(pet.UnlockedSpecies, species)
	}
	pet.Species = species
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// ResetSpecies — снять купленный облик: питомец возвращается к природному
// виду (по паттерну работы; до первой эволюции — яйцо/малыш как есть).
// Разблокированные виды не теряются — облик можно надеть обратно.
func (s *Service) ResetSpecies(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	natural := "egg"
	if pet.Stage >= 1 {
		natural = s.detectSpecies(ctx, pet.UserID)
	}
	if pet.Species == natural {
		return dto.NewPet(pet), nil
	}
	pet.Species = natural
	if !containsStr(pet.UnlockedSpecies, natural) && natural != "egg" {
		pet.UnlockedSpecies = append(pet.UnlockedSpecies, natural)
	}
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.appendActivity(ctx, userID, "species_reset", map[string]any{"species": natural})
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// SellItem — продать купленное имущество за половину цены (SellRefundPct);
// выручка идёт на кошелёк. category: house — декор домика, иначе товар магазина
// (скин/аксессуар/вид). Надетый предмет снимается; текущий облик продать нельзя.
func (s *Service) SellItem(ctx context.Context, userID, companyID int64, category, key string) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}

	var refund int
	if category == "house" {
		price, ok := domain.HouseDecor[key]
		if !ok || price <= 0 {
			return nil, domain.NewError("NO_ITEM", "Такого декора нет", 404)
		}
		if !containsStr(pet.HouseOwned, key) {
			return nil, domain.NewError("NOT_OWNED", "Этот декор не куплен", 422)
		}
		refund = price * domain.SellRefundPct / 100
		repossessItem(pet, "house", key) // снять с владения и расстановки
	} else {
		item, err := s.shop.GetItem(ctx, key)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, domain.NewError("NO_ITEM", "Такого товара нет", 404)
		}
		refund = item.PriceKudos * domain.SellRefundPct / 100
		if item.Kind == "species" {
			if !containsStr(pet.UnlockedSpecies, key) {
				return nil, domain.NewError("NOT_OWNED", "Этот вид не куплен", 422)
			}
			if pet.Species == key {
				return nil, domain.NewError("SPECIES_WORN", "Сначала снимите этот облик", 422)
			}
			pet.UnlockedSpecies = removeStr(pet.UnlockedSpecies, key)
		} else {
			if !containsStr(pet.Accessories, key) {
				return nil, domain.NewError("NOT_OWNED", "Этот предмет не куплен", 422)
			}
			repossessItem(pet, "shop", key) // снять из гардероба и с головы
		}
	}

	pet.Kudos += refund
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.appendActivity(ctx, userID, "item_sold", map[string]any{"key": key, "category": category, "refund": refund})
	s.appendLedger(ctx, userID, companyID, refund, "sell", nil, key)
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// ──────────────────────── мистери-слот ─────────────────────────────
// Раз в день — бесплатный взвешенный по редкости сюрприз-предмет: не
// платный лутбокс (оплаты нет вообще), а приятный бонус, чтобы витрина
// не казалась статичной. Лимитированные и достижимые товары исключены из
// пула: раздавать их бесплатно вслепую значило бы обесценивать тираж/цель.

var mysteryRarityWeights = map[string]int{
	"common": 50, "rare": 30, "epic": 15, "legendary": 5,
}

func mysteryDailyKey(userID int64) string {
	return "gw2:pets:mystery:" + strconvI64(userID) + ":" + todayMSK().Format("2006-01-02")
}

// GetMysteryItem — сегодняшний бесплатный предмет; ErrAlreadyTaken, если уже
// забирали. Не списывает кудосы и не участвует в лимитированном тираже.
func (s *Service) GetMysteryItem(ctx context.Context, userID, companyID int64) (*dto.ShopItemDTO, error) {
	key := mysteryDailyKey(userID)
	if s.daily.Exists(ctx, key) {
		return nil, domain.NewError("ALREADY_TAKEN", "Сюрприз на сегодня уже получен", 429)
	}
	items, err := s.shop.ListActiveItems(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	var pool []*domain.ShopItem
	weights := make([]int, 0, len(items))
	total := 0
	for _, item := range items {
		if item.UnlockKind != "shop" || item.LimitedQuota != nil {
			continue
		}
		w := mysteryRarityWeights[item.Rarity]
		if w <= 0 {
			w = 1
		}
		pool = append(pool, item)
		weights = append(weights, w)
		total += w
	}
	if len(pool) == 0 {
		return nil, domain.NewError("NO_ITEM", "Сюрприз сегодня недоступен", 404)
	}
	pick := randIntn(total)
	var chosen *domain.ShopItem
	for i, w := range weights {
		if pick < w {
			chosen = pool[i]
			break
		}
		pick -= w
	}

	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	owned := containsStr(pet.Accessories, chosen.Key)
	if chosen.Kind == "species" {
		owned = isSpeciesOwned(pet, chosen.Key)
	}
	if !owned {
		if chosen.Kind == "species" {
			pet.UnlockedSpecies = append(pet.UnlockedSpecies, chosen.Key)
			pet.Species = chosen.Key
		} else {
			pet.Accessories = append(pet.Accessories, chosen.Key)
			pet.Hat = &chosen.Key
		}
		if err := s.pets.SavePet(ctx, pet); err != nil {
			return nil, err
		}
		s.appendActivity(ctx, userID, "item_bought", map[string]any{"key": chosen.Key, "mystery": true})
		s.emitPetUpdate(ctx, pet)
	}
	s.daily.SetCache(ctx, key, "1", 24*time.Hour)
	return newShopItemDTO(chosen, nil, true), nil
}

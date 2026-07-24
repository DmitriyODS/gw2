package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/dto"
)

// Развитие после максимальной формы: престиж-поколения (бесконечная личная
// лестница поверх шкалы стадий), сезонный трек наград (командный ритм —
// кудосы квартала открывают пороги) и домик (долгий сток кудосов).

// ─────────────────────────── престиж ────────────────────────────────

// PrestigePet — перерождение питомца максимальной стадии: поколение +1,
// стадия/XP в ноль, вид снова яйцо. Кудосы, гардероб, купленные виды и
// домик сохраняются; эксклюзивный вид поколения (PrestigeSpecies)
// разблокируется тем же атомарным UPDATE.
func (s *Service) PrestigePet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}
	if pet.SickSince != nil {
		return nil, domain.NewError("PET_SICK", "Больному питомцу не до перерождения — сначала вылечите", 422)
	}
	if pet.Stage < domain.MaxStage {
		return nil, domain.NewError("NOT_MAX_STAGE", "Перерождение доступно только «Легенде»", 422)
	}
	// В БД generation DEFAULT 1, но нормализуем на всякий (нулевые снимки).
	unlock := domain.PrestigeSpecies[max(1, pet.Generation)+1]
	generation, ok, err := s.pets.Prestige(ctx, userID, unlock)
	if err != nil {
		return nil, err
	}
	if !ok { // гонка двух кликов / конкурентная болезнь
		return nil, domain.NewError("NOT_MAX_STAGE", "Перерождение сейчас недоступно", 422)
	}
	pet.Generation = generation
	pet.Stage = 0
	pet.XP = 0
	pet.Species = "egg"
	if unlock != "" && !containsStr(pet.UnlockedSpecies, unlock) {
		pet.UnlockedSpecies = append(pet.UnlockedSpecies, unlock)
	}
	s.appendActivity(ctx, userID, "prestige",
		map[string]any{"generation": generation, "unlocked": unlock})
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// ─────────────────────── сезонный трек ──────────────────────────────

// seasonKey — календарный квартал по МСК: «2026-Q3».
func seasonKey(now time.Time) string {
	msk := now.In(domain.MSK)
	return fmt.Sprintf("%d-Q%d", msk.Year(), (int(msk.Month())-1)/3+1)
}

// seasonEnd — первый момент следующего квартала (МСК).
func seasonEnd(now time.Time) time.Time {
	msk := now.In(domain.MSK)
	q := (int(msk.Month())-1)/3 + 1
	return time.Date(msk.Year(), time.Month(q*3+1), 1, 0, 0, 0, 0, domain.MSK)
}

// addSeasonalKudos — инкремент сезонного счётчика рядом с недельным;
// не бросает (пути начисления не должны падать из-за трека).
func (s *Service) addSeasonalKudos(ctx context.Context, userID int64, amount int) {
	if amount <= 0 {
		return
	}
	if err := s.pets.AddSeasonalKudos(ctx, userID, seasonKey(time.Now()), amount); err != nil {
		s.log.Warn("pets.seasonal_kudos_failed", "user_id", userID, "error", err)
	}
}

// GetSeason — состояние сезонного трека владельца: сколько кудосов
// заработано за квартал и какие пороги открыты/забраны.
func (s *Service) GetSeason(ctx context.Context, userID, companyID int64) (*dto.SeasonDTO, error) {
	if _, err := s.pets.GetOrCreate(ctx, userID, companyID); err != nil {
		return nil, err
	}
	now := time.Now()
	season := seasonKey(now)
	earned, err := s.pets.SeasonalKudos(ctx, userID, season)
	if err != nil {
		return nil, err
	}
	claimed, err := s.pets.SeasonClaims(ctx, userID, season)
	if err != nil {
		return nil, err
	}
	return dto.NewSeason(season, seasonEnd(now), earned, claimed), nil
}

// ClaimSeasonReward — забрать награду достигнутого порога. Двойной клик /
// гонка режутся PK pet_season_claims; применение награды — атомарные
// узкие апдейты (Append*/AdjustBalances), уже имеющийся предмет не дублируется.
func (s *Service) ClaimSeasonReward(ctx context.Context, userID, companyID int64, threshold int) (*dto.SeasonDTO, error) {
	var reward *domain.SeasonReward
	for i := range domain.SeasonTrack {
		if domain.SeasonTrack[i].Threshold == threshold {
			reward = &domain.SeasonTrack[i]
			break
		}
	}
	if reward == nil {
		return nil, domain.NewError("NO_REWARD", "Такого порога в треке нет", 404)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	season := seasonKey(now)
	earned, err := s.pets.SeasonalKudos(ctx, userID, season)
	if err != nil {
		return nil, err
	}
	if earned < threshold {
		return nil, domain.NewError("NOT_REACHED", "Порог ещё не достигнут", 422)
	}
	ok, err := s.pets.ClaimSeasonReward(ctx, userID, season, threshold)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.NewError("ALREADY_CLAIMED", "Награда уже забрана", 422)
	}
	switch reward.Kind {
	case "accessory":
		if _, err := s.pets.AppendAccessory(ctx, userID, reward.Key); err != nil {
			return nil, err
		}
	case "decor":
		if _, err := s.pets.AppendHouseDecor(ctx, userID, reward.Key); err != nil {
			return nil, err
		}
	case "kudos":
		kudos, xp, err := s.pets.AdjustBalances(ctx, userID, reward.Amount, 0)
		if err != nil {
			return nil, err
		}
		pet.Kudos, pet.XP = kudos, xp
		s.appendLedger(ctx, userID, companyID, reward.Amount, "season", nil, "")
	}
	s.appendActivity(ctx, userID, "season_reward",
		map[string]any{"threshold": threshold, "kind": reward.Kind, "key": reward.Key, "amount": reward.Amount})
	// Свежий снимок — награда могла изменить гардероб/домик/баланс.
	if fresh, err := s.pets.GetPet(ctx, userID); err == nil && fresh != nil {
		pet = fresh
	}
	s.emitPetUpdate(ctx, pet)
	claimed, err := s.pets.SeasonClaims(ctx, userID, season)
	if err != nil {
		return nil, err
	}
	return dto.NewSeason(season, seasonEnd(now), earned, claimed), nil
}

// ─────────────────────────── домик ──────────────────────────────────

// GetHouse — каталог декора с владением/расстановкой текущего питомца.
func (s *Service) GetHouse(ctx context.Context, userID, companyID int64) (*dto.HouseDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewHouse(pet), nil
}

// BuyHouseDecor — покупка декора за кудосы (кудос-сток). Предметы с ценой 0 —
// награды сезонного трека, не продаются.
func (s *Service) BuyHouseDecor(ctx context.Context, userID, companyID int64, key string, installment bool) (*dto.HouseDTO, error) {
	price, exists := domain.HouseDecor[key]
	if !exists {
		return nil, domain.NewError("NO_ITEM", "Такого декора нет", 404)
	}
	if price <= 0 {
		return nil, domain.NewError("SEASON_ONLY", "Этот декор — награда сезонного трека, не продаётся", 422)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}
	if containsStr(pet.HouseOwned, key) {
		return nil, domain.NewError("ALREADY_OWNED", "Уже куплено", 422)
	}

	if installment {
		// Декор домика всегда не-акционный — рассрочка допустима. Товар выдаём
		// полным сохранением (лимит проверяем до выдачи).
		if err := s.checkInstallmentLimit(ctx, userID, price); err != nil {
			return nil, err
		}
		pet.HouseOwned = append(pet.HouseOwned, key)
		if err := s.pets.SavePet(ctx, pet); err != nil {
			return nil, err
		}
		if err := s.openInstallment(ctx, userID, companyID, "house", key, key, price); err != nil {
			s.log.Warn("pets.open_installment_failed", "user_id", userID, "key", key, "error", err)
		}
		s.appendActivity(ctx, userID, "house_bought", map[string]any{"key": key, "price": price})
		s.emitPetUpdate(ctx, pet)
		return dto.NewHouse(pet), nil
	}

	if pet.Kudos < price {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов", 422)
	}
	ok, err := s.pets.BuyHouseDecor(ctx, userID, key, price)
	if err != nil {
		return nil, err
	}
	if !ok { // гонка: баланс уже потрачен либо декор куплен конкурентно
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов", 422)
	}
	// Свежий снимок после атомарного UPDATE — не мутируем локальную копию
	// (конкурентные начисления уже могли изменить баланс).
	if fresh, err := s.pets.GetPet(ctx, userID); err == nil && fresh != nil {
		pet = fresh
	}
	s.appendActivity(ctx, userID, "house_bought", map[string]any{"key": key, "price": price})
	s.appendLedger(ctx, userID, companyID, -price, "house", nil, key)
	s.emitPetUpdate(ctx, pet)
	return dto.NewHouse(pet), nil
}

// ArrangeHouse — свободная расстановка декора в сцене: только купленное,
// без дублей, не больше HousePlacedMax предметов; координаты — проценты
// сцены, зажимаются в границы.
func (s *Service) ArrangeHouse(ctx context.Context, userID, companyID int64, placed []domain.HouseItem) (*dto.HouseDTO, error) {
	if len(placed) > domain.HousePlacedMax {
		return nil, domain.NewError("TOO_MANY", "В домике помещается не больше "+
			fmt.Sprint(domain.HousePlacedMax)+" предметов", 422)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	clamped := make([]domain.HouseItem, 0, len(placed))
	for _, item := range placed {
		if !containsStr(pet.HouseOwned, item.Key) {
			return nil, domain.NewError("NOT_OWNED", "Декор не куплен", 422)
		}
		if seen[item.Key] {
			return nil, domain.NewError("DUPLICATE", "Один предмет — один раз", 422)
		}
		seen[item.Key] = true
		item.X = clampPct(item.X)
		item.Y = clampPct(item.Y)
		clamped = append(clamped, item)
	}
	if err := s.pets.SaveHousePlaced(ctx, userID, clamped); err != nil {
		return nil, err
	}
	pet.HousePlaced = clamped
	s.emitPetUpdate(ctx, pet)
	return dto.NewHouse(pet), nil
}

func clampPct(v float64) float64 {
	return min(100, max(0, v))
}

// SetHousePetPos — владелец двигает самого грувика по сцене комнаты.
func (s *Service) SetHousePetPos(ctx context.Context, userID, companyID int64, x, y float64) (*dto.HouseDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	x, y = clampPct(x), clampPct(y)
	if err := s.pets.SaveHousePetPos(ctx, userID, x, y); err != nil {
		return nil, err
	}
	pet.HousePetX, pet.HousePetY = &x, &y
	s.emitPetUpdate(ctx, pet)
	return dto.NewHouse(pet), nil
}

// SetHouseTheme — выбор градиентной темы комнаты (бесплатно; тему видят
// коллеги в домике и мини-игре поглаживания).
func (s *Service) SetHouseTheme(ctx context.Context, userID, companyID int64, theme string) (*dto.HouseDTO, error) {
	if !containsStr(domain.HouseThemes, theme) {
		return nil, domain.NewError("NO_THEME", "Такой темы комнаты нет", 422)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.pets.SaveHouseTheme(ctx, userID, theme); err != nil {
		return nil, err
	}
	pet.HouseTheme = theme
	s.emitPetUpdate(ctx, pet)
	return dto.NewHouse(pet), nil
}

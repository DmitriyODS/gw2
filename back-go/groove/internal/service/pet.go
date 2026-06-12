package service

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/dto"
)

// emitPetUpdate — синхронизация питомца между вкладками владельца.
func (s *Service) emitPetUpdate(ctx context.Context, pet *domain.Pet) {
	s.pub.Publish(ctx, "pet:update", []string{userRoom(pet.UserID)}, dto.NewPet(pet))
}

// ───────────────────────── начисление грувов ───────────────────────

// AwardBeans — начислить грувы с учётом дневного капа источника.
// Никогда не возвращает ошибку наружу (зовётся из хуков).
func (s *Service) AwardBeans(ctx context.Context, userID, companyID int64,
	source string, amount int) int {

	cap, ok := domain.DailyCaps[source]
	if !ok {
		cap = domain.DefaultDailyCap
	}
	granted := s.daily.TakeBudget(ctx, userID, source, amount, cap)
	if granted <= 0 {
		return 0
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		s.log.Warn("groove.award_failed", "user_id", userID, "source", source, "error", err)
		return 0
	}
	pet.Beans += granted
	if err := s.pets.SavePet(ctx, pet); err != nil {
		s.log.Warn("groove.award_failed", "user_id", userID, "source", source, "error", err)
		return 0
	}
	s.emitPetUpdate(ctx, pet)
	return granted
}

// ────────────────────────── питомец владельца ──────────────────────

func (s *Service) GetMyPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if pet.Personality == nil {
		personality := s.detectPersonality(ctx, userID)
		pet.Personality = &personality
	}
	s.ensureTodayQuest(pet)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	data := dto.NewPet(pet)
	s.fillFeedCounters(ctx, data, pet)
	return data, nil
}

func (s *Service) fillFeedCounters(ctx context.Context, data *dto.PetDTO, pet *domain.Pet) {
	var left, maxFeeds int
	if pet.SickSince != nil {
		left = s.daily.Left(ctx, pet.UserID, "sick_feeds", domain.SickFeedDailyMax)
		maxFeeds = domain.SickFeedDailyMax
	} else {
		left = s.daily.Left(ctx, pet.UserID, "feeds", domain.FeedDailyMax)
		maxFeeds = domain.FeedDailyMax
	}
	data.FeedsLeft, data.FeedsMax = &left, &maxFeeds
}

// detectSpecies — вид по паттерну работы за 60 дней.
func (s *Service) detectSpecies(ctx context.Context, userID int64) string {
	since := time.Now().UTC().AddDate(0, 0, -60)
	units, err := s.pets.FinishedUnitsForUser(ctx, userID, since, 100)
	if err != nil || len(units) == 0 {
		return "fox"
	}
	var totalMinutes float64
	var startHours []int
	for _, u := range units {
		totalMinutes += u.End.Sub(u.Start).Minutes()
		startHours = append(startHours, u.Start.In(domain.MSK).Hour())
	}
	avg := totalMinutes / float64(len(units))
	sort.Ints(startHours)
	medianHour := startHours[len(startHours)/2]
	switch {
	case avg >= 100:
		return "marathoner"
	case avg <= 35 && len(units) >= 10:
		return "sprinter"
	case medianHour < 11:
		return "lark"
	case medianHour >= 17:
		return "owl"
	}
	return "fox"
}

// detectPersonality — характер по юнитам за 21 день: ритм, время, длина сессий.
func (s *Service) detectPersonality(ctx context.Context, userID int64) string {
	since := time.Now().UTC().AddDate(0, 0, -21)
	units, err := s.pets.FinishedUnitsForUser(ctx, userID, since, 200)
	if err != nil || len(units) <= 2 {
		return "lazy"
	}
	perWeek := float64(len(units)) / 3.0
	var totalMinutes float64
	var startHours []int
	for _, u := range units {
		totalMinutes += u.End.Sub(u.Start).Minutes()
		startHours = append(startHours, u.Start.In(domain.MSK).Hour())
	}
	avg := totalMinutes / float64(len(units))
	sort.Ints(startHours)
	medianHour := startHours[len(startHours)/2]
	switch {
	case perWeek <= 3:
		return "lazy"
	case medianHour >= 19:
		return "night"
	case medianHour < 10:
		return "early"
	case perWeek >= 12 && avg <= 60:
		return "energizer"
	case avg >= 110:
		return "zen"
	}
	return "steady"
}

// ───────────────────────────── болезнь ─────────────────────────────

// applyRecovery — прибавить recovery-очки больному питомцу (без сохранения).
// true — выздоровел.
func applyRecovery(pet *domain.Pet, amount int) bool {
	if pet.SickSince == nil {
		return false
	}
	pet.Recovery = min(domain.RecoveryTarget, pet.Recovery+amount)
	if pet.Recovery >= domain.RecoveryTarget {
		pet.SickSince = nil
		pet.Recovery = 0
		return true
	}
	return false
}

// AddRecovery — лечение работой/заботой. Никогда не бросает (хуки).
func (s *Service) AddRecovery(ctx context.Context, userID, companyID int64, amount int) {
	pet, err := s.pets.GetPet(ctx, userID)
	if err != nil || pet == nil || pet.SickSince == nil {
		return
	}
	recovered := applyRecovery(pet, amount)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		s.log.Warn("groove.recovery_failed", "user_id", userID, "error", err)
		return
	}
	if recovered {
		_, _ = s.recordEvent(ctx, companyID, &userID, "pet_recovered",
			map[string]any{"pet_name": pet.Name}, true)
	}
	s.emitPetUpdate(ctx, pet)
}

// CheckSicknessForCompany — пометить больными питомцев тех, кто давно не
// работал. Простой считается в РАБОЧИХ днях компании: выходные не приближают
// болезнь, а в сам выходной Грувик не заболевает вовсе.
func (s *Service) CheckSicknessForCompany(ctx context.Context, companyID int64) (int, error) {
	weekend := s.weekendDays(ctx, companyID)
	today := todayMSK()
	if isWeekend(today, weekend) {
		return 0, nil
	}
	pets, err := s.pets.ListCompanyPets(ctx, companyID)
	if err != nil {
		return 0, err
	}
	var candidates []*domain.Pet
	for _, p := range pets {
		if p.Stage >= 1 && p.SickSince == nil {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return 0, nil
	}
	ids := make([]int64, len(candidates))
	for i, p := range candidates {
		ids[i] = p.UserID
	}
	lastEnds, err := s.pets.LastUnitEndByUsers(ctx, ids)
	if err != nil {
		return 0, err
	}
	sickCount := 0
	for _, pet := range candidates {
		last, ok := lastEnds[pet.UserID]
		// Ни одного юнита в принципе — не наказываем (свежий пользователь).
		if !ok {
			continue
		}
		lastDay := last.In(domain.MSK)
		lastDate := time.Date(lastDay.Year(), lastDay.Month(), lastDay.Day(), 0, 0, 0, 0, time.UTC)
		if workingDaysBetween(lastDate, today, weekend) < domain.SickAfterDays {
			continue
		}
		now := time.Now().UTC()
		pet.SickSince = &now
		pet.Recovery = 0
		if err := s.pets.SavePet(ctx, pet); err != nil {
			s.log.Warn("groove.sick_save_failed", "user_id", pet.UserID, "error", err)
			continue
		}
		sickCount++
		_, _ = s.recordEvent(ctx, companyID, &pet.UserID, "pet_sick",
			map[string]any{"pet_name": pet.Name}, true)
		s.emitPetUpdate(ctx, pet)
	}
	return sickCount, nil
}

// RefreshPersonalitiesForCompany — дневной пересчёт характеров.
func (s *Service) RefreshPersonalitiesForCompany(ctx context.Context, companyID int64) error {
	pets, err := s.pets.ListCompanyPets(ctx, companyID)
	if err != nil {
		return err
	}
	for _, pet := range pets {
		personality := s.detectPersonality(ctx, pet.UserID)
		if pet.Personality == nil || *pet.Personality != personality {
			pet.Personality = &personality
			if err := s.pets.SavePet(ctx, pet); err != nil {
				return err
			}
		}
	}
	return nil
}

var sickPhrases = []string{
	"Апчхи… Спасибо за бульон. Кажется, мне уже чуточку лучше.",
	"Тёплый бульончик… Ещё бы пару закрытых задач — и я на ногах!",
	"Болею… Поработай немного — твоя энергия меня лечит.",
	"Кх-кх… Говорят, лучшее лекарство — завершённый юнит хозяина.",
}

// ───────────────────────────── кормление ───────────────────────────

func (s *Service) FeedPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	// Больного кормим лечебным бульоном: дёшево, без XP, +1 к выздоровлению.
	if pet.SickSince != nil {
		if pet.Beans < domain.SickFeedCost {
			return nil, domain.NewError("NO_BEANS", "Не хватает грувов даже на бульон", 422)
		}
		if s.daily.TakeBudget(ctx, userID, "sick_feeds", 1, domain.SickFeedDailyMax) <= 0 {
			return nil, domain.NewError("FED_ENOUGH", "Бульон — не больше двух мисок в день", 429)
		}
		pet.Beans -= domain.SickFeedCost
		recovered := applyRecovery(pet, 1)
		if err := s.pets.SavePet(ctx, pet); err != nil {
			return nil, err
		}
		if recovered {
			_, _ = s.recordEvent(ctx, companyID, &userID, "pet_recovered",
				map[string]any{"pet_name": pet.Name}, true)
		}
		s.emitPetUpdate(ctx, pet)
		data := dto.NewPet(pet)
		// Выздоровел — счётчики сразу по «здоровой» шкале кормлений.
		s.fillFeedCounters(ctx, data, pet)
		phrase := sickPhrases[randIntn(len(sickPhrases))]
		if recovered {
			phrase = "Ура, я снова здоров! Спасибо, что выходил меня!"
		}
		evolved := false
		data.Phrase, data.Evolved, data.Recovered = &phrase, &evolved, &recovered
		return data, nil
	}

	if pet.Beans < domain.FeedCost {
		return nil, domain.NewError("NO_BEANS", "Не хватает грувов", 422)
	}
	if s.daily.TakeBudget(ctx, userID, "feeds", 1, domain.FeedDailyMax) <= 0 {
		return nil, domain.NewError("FED_ENOUGH", "Грувик сыт — приходите завтра", 429)
	}

	pet.Beans -= domain.FeedCost
	pet.XP += domain.FeedXP

	today := todayMSK()
	streakEvent := 0
	if pet.LastFedDate == nil || !pet.LastFedDate.Equal(today) {
		if pet.LastFedDate != nil && pet.LastFedDate.Equal(today.AddDate(0, 0, -1)) {
			pet.FeedStreak++
		} else {
			pet.FeedStreak = 1
		}
		fed := today
		pet.LastFedDate = &fed
		if domain.StreakMilestones[pet.FeedStreak] {
			streakEvent = pet.FeedStreak
		}
	}

	evolvedTo := 0
	for pet.Stage < domain.MaxStage && pet.XP >= domain.StageXP[pet.Stage+1] {
		pet.Stage++
		evolvedTo = pet.Stage
	}
	if evolvedTo > 0 {
		pet.Species = s.detectSpecies(ctx, userID)
		personality := s.detectPersonality(ctx, userID)
		pet.Personality = &personality
		if !containsStr(pet.UnlockedSpecies, pet.Species) {
			pet.UnlockedSpecies = append(pet.UnlockedSpecies, pet.Species)
		}
	}

	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}

	if streakEvent > 0 {
		_, _ = s.recordEvent(ctx, companyID, &userID, "streak",
			map[string]any{"days": streakEvent, "pet_name": pet.Name}, true)
	}
	if evolvedTo > 0 {
		_, _ = s.recordEvent(ctx, companyID, &userID, "pet_evolved",
			map[string]any{"stage": evolvedTo, "species": pet.Species,
				"pet_name": pet.Name}, true)
	}

	s.emitPetUpdate(ctx, pet)
	// Кормление двигает дневной квест feed_pet, если такой выпал.
	s.BumpQuest(ctx, userID, "feed_pet", 1)
	data := dto.NewPet(pet)
	left := s.daily.Left(ctx, userID, "feeds", domain.FeedDailyMax)
	maxFeeds := domain.FeedDailyMax
	data.FeedsLeft, data.FeedsMax = &left, &maxFeeds
	phrase := s.GetFeedPhrase(ctx, companyID)
	evolved := evolvedTo > 0
	data.Phrase, data.Evolved = &phrase, &evolved
	return data, nil
}

func (s *Service) RenamePet(ctx context.Context, userID, companyID int64, name string) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	pet.Name = strings.TrimSpace(name)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// ───────────────────────────── магазин ─────────────────────────────

func currentSeason() domain.Season {
	return domain.SeasonByMonth[todayMSK().Month()]
}

func (s *Service) GetShopState() map[string]any {
	season := currentSeason()
	prices := map[string]int{}
	for k, v := range domain.ShopPrices {
		prices[k] = v
	}
	prices[season.Item] = domain.SeasonalItems[season.Item]
	return map[string]any{
		"prices":         prices,
		"seasonal_item":  season.Item,
		"season_title":   season.Title,
		"species_prices": domain.SpeciesShop,
	}
}

func (s *Service) BuyItem(ctx context.Context, userID, companyID int64, item string) (*dto.PetDTO, error) {
	price, ok := domain.ShopPrices[item]
	if !ok {
		if seasonalPrice, isSeasonal := domain.SeasonalItems[item]; isSeasonal {
			if item != currentSeason().Item {
				return nil, domain.NewError("OUT_OF_SEASON",
					"Этот аксессуар вернётся в свой сезон", 422)
			}
			price, ok = seasonalPrice, true
		}
	}
	if !ok {
		return nil, domain.NewError("NO_ITEM", "Такого товара нет", 404)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if containsStr(pet.Accessories, item) {
		return nil, domain.NewError("ALREADY_OWNED", "Уже куплено", 422)
	}
	if pet.Beans < price {
		return nil, domain.NewError("NO_BEANS", "Не хватает грувов", 422)
	}
	pet.Beans -= price
	pet.Accessories = append(pet.Accessories, item)
	pet.Hat = &item
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
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
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// BuySpecies — разблокировать новый облик Грувика и сразу его надеть.
func (s *Service) BuySpecies(ctx context.Context, userID, companyID int64, species string) (*dto.PetDTO, error) {
	price, ok := domain.SpeciesShop[species]
	if !ok {
		return nil, domain.NewError("NO_ITEM", "Такого вида в магазине нет", 404)
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if containsStr(pet.UnlockedSpecies, species) {
		return nil, domain.NewError("ALREADY_OWNED", "Этот вид уже разблокирован", 422)
	}
	if pet.Beans < price {
		return nil, domain.NewError("NO_BEANS", "Не хватает грувов", 422)
	}
	pet.Beans -= price
	pet.UnlockedSpecies = append(pet.UnlockedSpecies, species)
	pet.Species = species
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
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
	// Природный (определённый эволюцией) вид доступен всегда — он
	// автоматически считается «своим» даже если не лежит в unlocked.
	naturalOK := domain.NaturalSpecies[species] && pet.Stage >= 2
	if !containsStr(pet.UnlockedSpecies, species) && !naturalOK {
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

// ─────────────────────────── квест дня ─────────────────────────────

// pickQuestTemplate — детерминированный выбор по (user_id, day): один и тот
// же день — тот же квест (стабильность при перезапросе).
func pickQuestTemplate(userID int64, day time.Time) domain.QuestTemplate {
	idx := (int(userID)*1009 + pythonOrdinal(day)) % len(domain.QuestTemplates)
	return domain.QuestTemplates[idx]
}

// ensureTodayQuest — назначить свежий квест, если предыдущий устарел
// (без сохранения — сохраняет вызывающий).
func (s *Service) ensureTodayQuest(pet *domain.Pet) {
	today := todayMSK()
	if pet.QuestDate != nil && pet.QuestDate.Equal(today) && pet.QuestKind != nil {
		return
	}
	tpl := pickQuestTemplate(pet.UserID, today)
	day := today
	pet.QuestDate = &day
	pet.QuestKind = &tpl.Kind
	target := tpl.Target
	pet.QuestTarget = &target
	pet.QuestProgress = 0
	pet.QuestClaimed = false
}

// BumpQuest — прибавить прогресс к дневному квесту, если совпадает по типу.
// Никогда не бросает (зовётся из хуков юнитов/задач).
func (s *Service) BumpQuest(ctx context.Context, userID int64, kind string, amount int) {
	if amount <= 0 {
		return
	}
	pet, err := s.pets.GetPet(ctx, userID)
	if err != nil || pet == nil {
		return
	}
	s.ensureTodayQuest(pet)
	if pet.QuestKind == nil || *pet.QuestKind != kind || pet.QuestClaimed {
		if err := s.pets.SavePet(ctx, pet); err != nil {
			s.log.Warn("groove.quest_bump_failed", "user_id", userID, "error", err)
		}
		return
	}
	target := 0
	if pet.QuestTarget != nil {
		target = *pet.QuestTarget
	}
	pet.QuestProgress = min(target, pet.QuestProgress+amount)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		s.log.Warn("groove.quest_bump_failed", "user_id", userID, "kind", kind, "error", err)
		return
	}
	s.emitPetUpdate(ctx, pet)
}

func (s *Service) ClaimQuest(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	s.ensureTodayQuest(pet)
	target := 0
	if pet.QuestTarget != nil {
		target = *pet.QuestTarget
	}
	if pet.QuestClaimed {
		return nil, domain.NewError("ALREADY_CLAIMED", "Награда уже забрана сегодня", 422)
	}
	if pet.QuestProgress < target {
		return nil, domain.NewError("NOT_DONE", "Квест ещё не выполнен", 422)
	}
	pet.QuestClaimed = true
	pet.Beans += domain.QuestRewardBeans
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	title := "Квест дня"
	if pet.QuestKind != nil {
		for _, tpl := range domain.QuestTemplates {
			if tpl.Kind == *pet.QuestKind {
				title = tpl.Title
				break
			}
		}
	}
	_, _ = s.recordEvent(ctx, companyID, &userID, "quest_done", map[string]any{
		"pet_name": pet.Name, "title": title, "reward": domain.QuestRewardBeans,
	}, true)
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// ─────────────────────────── зоопарк ───────────────────────────────

func (s *Service) GetZoo(ctx context.Context, companyID, viewerID int64) ([]*dto.PetDTO, error) {
	pets, err := s.pets.ListCompanyPets(ctx, companyID)
	if err != nil {
		return nil, err
	}
	today := todayMSK()
	ids := make([]int64, len(pets))
	for i, p := range pets {
		ids[i] = p.UserID
	}
	strokes, err := s.pets.StrokesToday(ctx, ids, today)
	if err != nil {
		return nil, err
	}
	my, err := s.pets.MyStrokesToday(ctx, viewerID, today)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.PetDTO, 0, len(pets))
	for _, p := range pets {
		data := dto.NewPet(p)
		count := strokes[p.UserID]
		stroked := my[p.UserID]
		data.StrokesToday, data.StrokedByMe = &count, &stroked
		result = append(result, data)
	}
	return result, nil
}

func (s *Service) StrokePet(ctx context.Context, viewerID, targetUserID,
	companyID int64) (map[string]any, error) {

	if viewerID == targetUserID {
		return nil, domain.NewError("SELF_STROKE",
			"Своего Грувика гладьте сколько угодно — грувы за это не положены", 422)
	}
	target, err := s.users.GetUser(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if target == nil || target.IsHidden || target.CompanyID == nil || *target.CompanyID != companyID {
		return nil, domain.NewError("USER_NOT_FOUND", "Сотрудник не найден", 404)
	}
	pet, err := s.pets.GetOrCreate(ctx, targetUserID, companyID)
	if err != nil {
		return nil, err
	}
	added, err := s.pets.AddStroke(ctx, targetUserID, viewerID, todayMSK())
	if err != nil {
		return nil, err
	}
	if !added {
		return nil, domain.NewError("ALREADY_STROKED", "Сегодня вы уже погладили этого Грувика", 422)
	}
	s.AwardBeans(ctx, targetUserID, companyID, "stroke_in", 1)
	s.AwardBeans(ctx, viewerID, companyID, "stroke_out", 1)
	// Забота лечит: поглаживание больного Грувика даёт очко выздоровления.
	s.AddRecovery(ctx, targetUserID, companyID, 1)

	fromFIO := "Коллега"
	if viewer, err := s.users.GetUser(ctx, viewerID); err == nil && viewer != nil {
		fromFIO = viewer.FIO
	}
	s.pub.Publish(ctx, "groove:stroke", []string{userRoom(targetUserID)}, map[string]any{
		"from_fio": fromFIO,
		"pet_name": pet.Name,
	})
	strokes, err := s.pets.StrokesToday(ctx, []int64{targetUserID}, todayMSK())
	if err != nil {
		return nil, err
	}
	return map[string]any{"strokes_today": strokes[targetUserID]}, nil
}

// ────────────────────────────── рейды ──────────────────────────────

func (s *Service) ensureRaid(ctx context.Context, companyID int64) (*domain.Raid, error) {
	weekStart := weekStartMSK()
	raid, err := s.pets.GetRaid(ctx, companyID, weekStart)
	if err != nil {
		return nil, err
	}
	if raid != nil {
		return raid, nil
	}
	prevStart := weekStart.AddDate(0, 0, -7)
	prevClosed, err := s.pets.CountClosedBetween(ctx, companyID,
		mskMidnight(prevStart), mskMidnight(weekStart))
	if err != nil {
		return nil, err
	}
	target := max(10, int(math.Ceil(float64(prevClosed)*1.2/5.0))*5)
	_, week := weekStart.ISOWeek()
	boss := domain.Bosses[week%len(domain.Bosses)]
	raid, err = s.pets.CreateRaid(ctx, companyID, weekStart, boss, target,
		domain.RaidRewardItem)
	if err != nil {
		return nil, err
	}
	_, _ = s.recordEvent(ctx, companyID, nil, "raid_started", map[string]any{
		"boss": boss, "target": target,
		"week_start": weekStart.Format("2006-01-02"),
	}, false)
	return raid, nil
}

func (s *Service) raidProgress(ctx context.Context, companyID int64, raid *domain.Raid) (int, error) {
	return s.pets.CountClosedBetween(ctx, companyID,
		mskMidnight(raid.WeekStart), time.Now().UTC().Add(time.Second))
}

func (s *Service) GetRaidState(ctx context.Context, companyID int64) (*dto.RaidDTO, error) {
	raid, err := s.ensureRaid(ctx, companyID)
	if err != nil {
		return nil, err
	}
	progress, err := s.raidProgress(ctx, companyID, raid)
	if err != nil {
		return nil, err
	}
	if raid.DefeatedAt != nil {
		progress = min(progress, raid.Target)
	}
	weekEnd := raid.WeekStart.AddDate(0, 0, 7)
	return &dto.RaidDTO{
		ID:        raid.ID,
		Boss:      raid.Boss,
		Target:    raid.Target,
		Progress:  progress,
		Reward:    raid.Reward,
		Defeated:  raid.DefeatedAt != nil,
		WeekStart: raid.WeekStart.Format("2006-01-02"),
		DaysLeft:  max(0, int(weekEnd.Sub(todayMSK()).Hours()/24)),
	}, nil
}

// OnTaskClosedRaid — прогресс рейда после закрытия задачи. Никогда не бросает.
func (s *Service) OnTaskClosedRaid(ctx context.Context, companyID int64) {
	raid, err := s.ensureRaid(ctx, companyID)
	if err != nil {
		s.log.Warn("groove.raid_failed", "company_id", companyID, "error", err)
		return
	}
	progress, err := s.raidProgress(ctx, companyID, raid)
	if err != nil {
		s.log.Warn("groove.raid_failed", "company_id", companyID, "error", err)
		return
	}
	defeatedNow := false
	if raid.DefeatedAt == nil && progress >= raid.Target {
		now := time.Now().UTC()
		if err := s.pets.SetRaidDefeated(ctx, raid.ID, now); err != nil {
			s.log.Warn("groove.raid_failed", "company_id", companyID, "error", err)
			return
		}
		raid.DefeatedAt = &now
		if err := s.pets.GrantRaidRewards(ctx, companyID, domain.RaidWinBeans,
			raid.Reward); err != nil {
			s.log.Warn("groove.raid_failed", "company_id", companyID, "error", err)
		}
		defeatedNow = true
		_, _ = s.recordEvent(ctx, companyID, nil, "raid_won", map[string]any{
			"boss": raid.Boss, "target": raid.Target,
			"reward": raid.Reward, "beans": domain.RaidWinBeans,
		}, true)
	}
	s.pub.Publish(ctx, "raid:update", []string{"all"}, map[string]any{
		"company_id":   companyID,
		"progress":     progress,
		"target":       raid.Target,
		"boss":         raid.Boss,
		"defeated":     raid.DefeatedAt != nil,
		"defeated_now": defeatedNow,
	})
}

// ───────────────────── ТВ-витрина Groove ───────────────────────────

func (s *Service) GetGrooveTV(ctx context.Context, companyID int64) (map[string]any, error) {
	pets, err := s.pets.ListCompanyPets(ctx, companyID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(pets))
	for i, p := range pets {
		ids[i] = p.UserID
	}
	strokes, err := s.pets.StrokesToday(ctx, ids, todayMSK())
	if err != nil {
		return nil, err
	}
	top := make([]*dto.PetDTO, 0, 8)
	for _, p := range pets[:min(len(pets), 8)] {
		data := dto.NewPet(p)
		count := strokes[p.UserID]
		data.StrokesToday = &count
		top = append(top, data)
	}
	raid, err := s.GetRaidState(ctx, companyID)
	if err != nil {
		return nil, err
	}
	sick, beans, totalStrokes := 0, 0, 0
	for _, p := range pets {
		if p.SickSince != nil {
			sick++
		}
		beans += p.Beans
	}
	for _, n := range strokes {
		totalStrokes += n
	}
	return map[string]any{
		"pets": top,
		"raid": raid,
		"totals": map[string]any{
			"pets":          len(pets),
			"sick":          sick,
			"beans":         beans,
			"strokes_today": totalStrokes,
		},
	}, nil
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

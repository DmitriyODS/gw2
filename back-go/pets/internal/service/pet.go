package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/dto"
)

// emitPetUpdate — синхронизация питомца между вкладками владельца.
func (s *Service) emitPetUpdate(ctx context.Context, pet *domain.Pet) {
	s.pub.Publish(ctx, "pet:update", []string{userRoom(pet.UserID)}, dto.NewPet(pet))
}

// ───────────────────────── начисление кудосов ──────────────────────

// AwardKudos — начислить кудосы с учётом дневного капа источника; кудосы,
// заработанные с начала недели, идут и в счётчик признания рейтинга
// (pet_kudos_weekly). Баланс инкрементируется атомарно (AdjustBalances):
// хук может прийти параллельно покупке/кормлению, и full-row SavePet здесь
// перетирал бы их устаревшим снимком. Никогда не возвращает ошибку наружу
// (зовётся из хуков).
func (s *Service) AwardKudos(ctx context.Context, userID, companyID int64,
	source string, amount int) int {

	cap, ok := domain.DailyCaps[source]
	if !ok {
		cap = domain.DefaultDailyCap
	}
	granted := s.daily.TakeBudget(ctx, userID, source, amount, cap)
	if granted <= 0 {
		return 0
	}
	// GetOrCreate — гарантия существования питомца + снапшот для события.
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		s.log.Warn("pets.award_failed", "user_id", userID, "source", source, "error", err)
		return 0
	}
	kudos, xp, err := s.pets.AdjustBalances(ctx, userID, granted, 0)
	if err != nil {
		s.log.Warn("pets.award_failed", "user_id", userID, "source", source, "error", err)
		return 0
	}
	pet.Kudos, pet.XP = kudos, xp
	isoYear, isoWeek := time.Now().In(domain.MSK).ISOWeek()
	if err := s.pets.AddWeeklyKudos(ctx, userID, isoYear, isoWeek, granted); err != nil {
		s.log.Warn("pets.weekly_kudos_failed", "user_id", userID, "error", err)
	}
	s.emitPetUpdate(ctx, pet)
	return granted
}

// ──────────────────── прямой XP за работу ──────────────────────────

// applyEvolution — поднять стадию по накопленному XP (без сохранения).
// >0 — питомец эволюционировал до этой стадии (вид и характер пересчитаны).
func (s *Service) applyEvolution(ctx context.Context, pet *domain.Pet) int {
	evolvedTo := 0
	for pet.Stage < domain.MaxStage && pet.XP >= domain.StageXP[pet.Stage+1] {
		pet.Stage++
		evolvedTo = pet.Stage
	}
	if evolvedTo > 0 {
		pet.Species = s.detectSpecies(ctx, pet.UserID)
		personality := s.detectPersonality(ctx, pet.UserID)
		pet.Personality = &personality
		if !containsStr(pet.UnlockedSpecies, pet.Species) {
			pet.UnlockedSpecies = append(pet.UnlockedSpecies, pet.Species)
		}
	}
	return evolvedTo
}

// celebrateEvolution — единая фиксация состоявшейся эволюции: запись в
// приватную историю + системный пост-поздравление в корпоративном портале
// (system_kind='pet_evolved'). Клиент портала fire-and-forget (горутина с
// таймаутом внутри клиента), дедуп повторов — на стороне portalsvc;
// nil-клиент — портал не настроен, публикуем только историю.
func (s *Service) celebrateEvolution(ctx context.Context, pet *domain.Pet, evolvedTo int) {
	s.appendActivity(ctx, pet.UserID, "evolved",
		map[string]any{"stage": evolvedTo, "species": pet.Species})
	if s.portal == nil {
		return
	}
	stage := ""
	if evolvedTo > 0 && evolvedTo < len(domain.StageTitles) {
		stage = domain.StageTitles[evolvedTo]
	}
	s.portal.CreateSystemPost(pet.CompanyID, pet.UserID, "pet_evolved",
		"🎉 Эволюция грувика",
		fmt.Sprintf("«%s» дорос до стадии «%s»!", pet.Name, stage))
}

// fedToday — питомца сегодня хотя бы раз кормили («сытость» до конца дня).
func fedToday(pet *domain.Pet) bool {
	return pet.LastFedDate != nil && pet.LastFedDate.Equal(todayMSK())
}

// AwardXP — прямой XP за работу с дневным капом источника (source —
// "xp_unit"/"xp_task"/"xp_walk"). Сытость умножает начисление на FedXPBoost;
// больному питомцу XP заморожен. Баланс инкрементируется атомарно
// (AdjustBalances — та же причина, что в AwardKudos); эволюция, если
// случилась, сохраняется отдельным узким SaveEvolution — full-row SavePet из
// хука перетирал бы конкурентные изменения балансов/квеста. Никогда не
// возвращает ошибку наружу (зовётся из хуков).
func (s *Service) AwardXP(ctx context.Context, userID, companyID int64,
	source string, amount, cap int) int {

	if amount <= 0 {
		return 0
	}
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		s.log.Warn("pets.award_xp_failed", "user_id", userID, "source", source, "error", err)
		return 0
	}
	if pet.SickSince != nil {
		return 0 // болезнь замораживает XP — и прямой тоже
	}
	granted := s.daily.TakeBudget(ctx, userID, source, amount, cap)
	if granted <= 0 {
		return 0
	}
	if fedToday(pet) {
		granted = int(float64(granted) * domain.FedXPBoost)
	}
	kudos, xp, err := s.pets.AdjustBalances(ctx, userID, 0, granted)
	if err != nil {
		s.log.Warn("pets.award_xp_failed", "user_id", userID, "source", source, "error", err)
		return 0
	}
	pet.Kudos, pet.XP = kudos, xp
	if evolvedTo := s.applyEvolution(ctx, pet); evolvedTo > 0 {
		if err := s.pets.SaveEvolution(ctx, pet); err != nil {
			s.log.Warn("pets.evolution_save_failed", "user_id", userID, "error", err)
		} else {
			s.celebrateEvolution(ctx, pet, evolvedTo)
		}
	}
	s.emitPetUpdate(ctx, pet)
	return granted
}

// ─────────────────────────── приключение ───────────────────────────
// Appointment-механика: владелец бесплатно отправляет питомца в приключение
// на случайные 2–4 часа; возврат фиксируется ЛЕНИВО (любой GET владельца
// после срока, без фонового цикла) и приносит вариативную награду. Пока
// питомец «в пути», платные действия недоступны (PET_AWAY).

// StartAdventure — отправить питомца в приключение: бесплатно, доступно
// здоровому и не находящемуся в пути; кап AdventureDailyMax стартов в день
// (источник 'adventure' в Redis daily, проверка при старте).
func (s *Service) StartAdventure(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	s.maybeReturnAdventure(ctx, pet) // истёкшее приключение не блокирует новое
	if pet.AdventureUntil != nil {
		return nil, domain.ErrPetAway
	}
	if pet.SickSince != nil {
		return nil, domain.NewError("PET_SICK", "Больному питомцу не до приключений — сначала вылечите", 422)
	}
	if s.daily.TakeBudget(ctx, userID, "adventure", 1, domain.AdventureDailyMax) <= 0 {
		return nil, domain.ErrAdventureLimit
	}
	minutes := domain.AdventureMinMinutes +
		randIntn(domain.AdventureMaxMinutes-domain.AdventureMinMinutes+1)
	until := time.Now().UTC().Add(time.Duration(minutes) * time.Minute)
	place := domain.AdventurePlaces[randIntn(len(domain.AdventurePlaces))]
	ok, err := s.pets.StartAdventure(ctx, userID, until, place)
	if err != nil {
		return nil, err
	}
	if !ok { // гонка двух стартов / конкурентная болезнь
		return nil, domain.ErrPetAway
	}
	pet.AdventureUntil, pet.AdventurePlace = &until, &place
	s.appendActivity(ctx, userID, "adventure_started", map[string]any{"place": place})
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// maybeReturnAdventure — ленивый возврат: если срок истёк, атомарно фиксирует
// возврат (FinishAdventure с RETURNING — двойной GET не начислит дважды) и
// начисляет награду АТОМАРНЫМ AdjustBalances (дневные капы источников не
// применяются — кап у этой механики на старты). Возвращает награду, если
// возврат зафиксировал именно этот вызов. Никогда не бросает (read-путь).
func (s *Service) maybeReturnAdventure(ctx context.Context, pet *domain.Pet) *dto.AdventureRewardDTO {
	now := time.Now().UTC()
	if pet.AdventureUntil == nil || now.Before(*pet.AdventureUntil) {
		return nil
	}
	place, returned, err := s.pets.FinishAdventure(ctx, pet.UserID, now)
	if err != nil {
		s.log.Warn("pets.adventure_return_failed", "user_id", pet.UserID, "error", err)
		return nil
	}
	pet.AdventureUntil, pet.AdventurePlace = nil, nil
	if !returned {
		return nil // конкурентный GET уже зафиксировал возврат
	}
	kudos := domain.AdventureKudosMin + randIntn(domain.AdventureKudosMax-domain.AdventureKudosMin+1)
	xp := domain.AdventureXPMin + randIntn(domain.AdventureXPMax-domain.AdventureXPMin+1)
	newKudos, newXP, err := s.pets.AdjustBalances(ctx, pet.UserID, kudos, xp)
	if err != nil {
		s.log.Warn("pets.adventure_award_failed", "user_id", pet.UserID, "error", err)
		return nil
	}
	pet.Kudos, pet.XP = newKudos, newXP
	isoYear, isoWeek := time.Now().In(domain.MSK).ISOWeek()
	if err := s.pets.AddWeeklyKudos(ctx, pet.UserID, isoYear, isoWeek, kudos); err != nil {
		s.log.Warn("pets.weekly_kudos_failed", "user_id", pet.UserID, "error", err)
	}
	if evolvedTo := s.applyEvolution(ctx, pet); evolvedTo > 0 {
		if err := s.pets.SaveEvolution(ctx, pet); err != nil {
			s.log.Warn("pets.evolution_save_failed", "user_id", pet.UserID, "error", err)
		} else {
			s.celebrateEvolution(ctx, pet, evolvedTo)
		}
	}
	s.appendActivity(ctx, pet.UserID, "adventure_returned",
		map[string]any{"kudos": kudos, "xp": xp, "place": place})
	s.emitPetUpdate(ctx, pet)
	return &dto.AdventureRewardDTO{Kudos: kudos, XP: xp, Place: place}
}

// ensureNotAway — гейт платных действий владельца: сперва ленивый возврат
// (истёкшее приключение действие не блокирует), иначе PET_AWAY.
func (s *Service) ensureNotAway(ctx context.Context, pet *domain.Pet) error {
	s.maybeReturnAdventure(ctx, pet)
	if pet.AdventureUntil != nil {
		return domain.ErrPetAway
	}
	return nil
}

// ────────────────────────── питомец владельца ──────────────────────

func (s *Service) GetMyPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	// Ленивый возврат из приключения — только на GET владельца.
	reward := s.maybeReturnAdventure(ctx, pet)
	changed := false
	if pet.Personality == nil {
		personality := s.detectPersonality(ctx, userID)
		pet.Personality = &personality
		changed = true
	}
	if s.ensureTodayQuest(pet) {
		changed = true
	}
	// SavePet только если что-то реально поменялось — иначе GET на каждый
	// опрос клиента (несколько раз в секунду) гоняет read-modify-write и
	// рискует затереть устаревшим снимком конкурентное начисление хука.
	if changed {
		if err := s.pets.SavePet(ctx, pet); err != nil {
			return nil, err
		}
	}
	data := dto.NewPet(pet)
	data.AdventureReward = reward
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

// AddRecovery — лечение работой. Никогда не бросает (хуки).
func (s *Service) AddRecovery(ctx context.Context, userID, companyID int64, amount int) {
	pet, err := s.pets.GetPet(ctx, userID)
	if err != nil || pet == nil || pet.SickSince == nil {
		return
	}
	recovered := applyRecovery(pet, amount)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		s.log.Warn("pets.recovery_failed", "user_id", userID, "error", err)
		return
	}
	if recovered {
		s.appendActivity(ctx, userID, "recovered", nil)
	}
	s.emitPetUpdate(ctx, pet)
}

// CheckSicknessForCompany — пометить больными питомцев тех, кто давно не
// работал. Простой считается в РАБОЧИХ днях компании: выходные не приближают
// болезнь, а в сам выходной питомец не заболевает вовсе.
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
			s.log.Warn("pets.sick_save_failed", "user_id", pet.UserID, "error", err)
			continue
		}
		sickCount++
		s.appendActivity(ctx, pet.UserID, "sickness_started", nil)
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
	"Бульон принят. Медицинский факт: закрытые задачи лечат быстрее.",
	"Спасибо за бульон. До выздоровления — по протоколу: работа и забота.",
	"Болею. Бульон помогает, юнит от 15 минут — эффективнее.",
	"Апчхи. Бульон зачтён в курс лечения, продолжаем наблюдение.",
}

// Фолбэк-реплики кормления — фиксированный пул без ИИ-персонализации.
var feedPhrases = []string{
	"Принято. 3 кудоса конвертированы в 12 XP — курс сегодня неплохой.",
	"Съедено. Рост зафиксирован в протоколе.",
	"Спасибо. Работа кормит нас обоих — в буквальном смысле.",
	"Обед по расписанию. Осталось только поработать.",
	"Плюс 12 XP. Медленно, но статистически значимо.",
	"Кудосы усвоены. Сытый я усваиваю XP в полтора раза быстрее — учти.",
	"Зачёт. Ещё пара таких дней — и стадия сменится сама.",
	"Питательно. Возвращаемся к задачам.",
}

// ───────────────────────────── кормление ───────────────────────────

func (s *Service) FeedPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}

	// Больного кормим лечебным бульоном: дёшево, без XP, +1 к выздоровлению.
	if pet.SickSince != nil {
		if pet.Kudos < domain.SickFeedCost {
			return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов даже на бульон", 422)
		}
		if s.daily.TakeBudget(ctx, userID, "sick_feeds", 1, domain.SickFeedDailyMax) <= 0 {
			return nil, domain.NewError("FED_ENOUGH", "Бульон — не больше двух мисок в день", 429)
		}
		pet.Kudos -= domain.SickFeedCost
		recovered := applyRecovery(pet, 1)
		if err := s.pets.SavePet(ctx, pet); err != nil {
			return nil, err
		}
		s.appendActivity(ctx, userID, "fed", map[string]any{"sick": true})
		if recovered {
			s.appendActivity(ctx, userID, "recovered", nil)
		}
		s.emitPetUpdate(ctx, pet)
		data := dto.NewPet(pet)
		// Выздоровел — счётчики сразу по «здоровой» шкале кормлений.
		s.fillFeedCounters(ctx, data, pet)
		phrase := sickPhrases[randIntn(len(sickPhrases))]
		if recovered {
			phrase = "Выздоровел. Официально: лечение работой и бульоном подтверждено."
		}
		evolved := false
		data.Phrase, data.Evolved, data.Recovered = &phrase, &evolved, &recovered
		return data, nil
	}

	if pet.Kudos < domain.FeedCost {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов", 422)
	}
	if s.daily.TakeBudget(ctx, userID, "feeds", 1, domain.FeedDailyMax) <= 0 {
		return nil, domain.NewError("FED_ENOUGH", "Питомец сыт — приходите завтра", 429)
	}

	pet.Kudos -= domain.FeedCost
	pet.XP += domain.FeedXP

	today := todayMSK()
	if pet.LastFedDate == nil || !pet.LastFedDate.Equal(today) {
		if pet.LastFedDate != nil && pet.LastFedDate.Equal(today.AddDate(0, 0, -1)) {
			pet.FeedStreak++
		} else {
			pet.FeedStreak = 1
		}
		fed := today
		pet.LastFedDate = &fed
	}

	evolvedTo := s.applyEvolution(ctx, pet)

	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}

	s.appendActivity(ctx, userID, "fed", map[string]any{"streak": pet.FeedStreak})
	if evolvedTo > 0 {
		s.celebrateEvolution(ctx, pet, evolvedTo)
	}

	s.emitPetUpdate(ctx, pet)
	// Кормление двигает дневной квест feed_pet, если такой выпал.
	s.BumpQuest(ctx, userID, "feed_pet", 1)
	data := dto.NewPet(pet)
	left := s.daily.Left(ctx, userID, "feeds", domain.FeedDailyMax)
	maxFeeds := domain.FeedDailyMax
	data.FeedsLeft, data.FeedsMax = &left, &maxFeeds
	phrase := feedPhrases[randIntn(len(feedPhrases))]
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

// ───────────────────────────── прогулка ────────────────────────────

// WalkPet — платная мини-игра «прогулка»: списывает WalkCost кудосов
// (дневной кап WalkDailyMax), даёт небольшой XP/настроение и, если питомец
// болен, ускоряет выздоровление — как и работа.
func (s *Service) WalkPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}
	if pet.Kudos < domain.WalkCost {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов на прогулку", 422)
	}
	if s.daily.TakeBudget(ctx, userID, "walks", 1, domain.WalkDailyMax) <= 0 {
		return nil, domain.NewError("WALKED_ENOUGH", "Прогулок на сегодня достаточно", 429)
	}
	pet.Kudos -= domain.WalkCost
	recovered := false
	if pet.SickSince != nil {
		// Больному питомцу прогулка лечит, а не растит XP (тот и так заморожен).
		recovered = applyRecovery(pet, domain.WalkRecovery)
		if err := s.pets.SavePet(ctx, pet); err != nil {
			return nil, err
		}
	} else {
		pet.XP += domain.WalkXP
		evolvedTo := s.applyEvolution(ctx, pet)
		if err := s.pets.SavePet(ctx, pet); err != nil {
			return nil, err
		}
		if evolvedTo > 0 {
			s.celebrateEvolution(ctx, pet, evolvedTo)
		}
	}
	s.appendActivity(ctx, userID, "walked", nil)
	if recovered {
		s.appendActivity(ctx, userID, "recovered", nil)
	}
	s.emitPetUpdate(ctx, pet)
	data := dto.NewPet(pet)
	data.Recovered = &recovered
	return data, nil
}

// ───────────────────────────── лечение ─────────────────────────────

// HealPet — активное лечение мини-игрой за кудосы: только для больного
// питомца, с дневным лимитом попыток (не должно обесценивать лечение
// работой — оно остаётся основным и бесплатным путём).
func (s *Service) HealPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}
	if pet.SickSince == nil {
		return nil, domain.NewError("NOT_SICK", "Питомец здоров — лечить нечего", 422)
	}
	if pet.Kudos < domain.HealCost {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов на лечение", 422)
	}
	if s.daily.TakeBudget(ctx, userID, "heals", 1, domain.HealDailyMax) <= 0 {
		return nil, domain.NewError("HEALED_ENOUGH", "Лечение на сегодня исчерпано", 429)
	}
	pet.Kudos -= domain.HealCost
	recovered := applyRecovery(pet, domain.HealRecoveryPoints)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.appendActivity(ctx, userID, "healed", nil)
	if recovered {
		s.appendActivity(ctx, userID, "recovered", nil)
	}
	s.emitPetUpdate(ctx, pet)
	data := dto.NewPet(pet)
	data.Recovered = &recovered
	return data, nil
}

// ─────────────────── поглаживание чужого питомца ───────────────────

// StrokePet — символическая «плата за внимание» коллеге: списывает
// небольшую сумму кудосов у гладящего, дневной лимит на ОДНОГО чужого
// питомца (pet_strokes), получателю — небольшой XP/настроение без кудосов.
func (s *Service) StrokePet(ctx context.Context, strokerID, petOwnerID, companyID int64) (*dto.PetDTO, error) {
	if strokerID == petOwnerID {
		return nil, domain.NewError("SELF_STROKE", "Своего питомца гладить незачем — он и так знает", 422)
	}
	ownerMember, err := s.users.IsCompanyMember(ctx, petOwnerID, companyID)
	if err != nil {
		return nil, err
	}
	strokerMember, err := s.users.IsCompanyMember(ctx, strokerID, companyID)
	if err != nil {
		return nil, err
	}
	if !ownerMember || !strokerMember {
		return nil, domain.NewError("USER_NOT_FOUND", "Сотрудник не найден", 404)
	}

	today := todayMSK()
	used, err := s.pets.StrokesToday(ctx, petOwnerID, strokerID, today)
	if err != nil {
		return nil, err
	}
	if used >= domain.StrokeDailyMaxPerPet {
		return nil, domain.NewError("STROKED_ENOUGH", "Этого питомца вы уже сегодня погладили", 429)
	}

	stroker, err := s.pets.GetOrCreate(ctx, strokerID, companyID)
	if err != nil {
		return nil, err
	}
	if stroker.Kudos < domain.StrokeCost {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов на поглаживание", 422)
	}
	pet, err := s.pets.GetOrCreate(ctx, petOwnerID, companyID)
	if err != nil {
		return nil, err
	}
	// Чужого питомца в приключении не погладить; возврат фиксирует ТОЛЬКО
	// владелец (свой GET) — здесь просто проверка «срок ещё не истёк».
	if pet.Away(time.Now().UTC()) {
		return nil, domain.ErrPetAway
	}

	stroker.Kudos -= domain.StrokeCost
	if err := s.pets.SavePet(ctx, stroker); err != nil {
		return nil, err
	}
	pet.XP += domain.StrokeMoodXP
	evolvedTo := s.applyEvolution(ctx, pet)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	if err := s.pets.RecordStroke(ctx, petOwnerID, strokerID, today); err != nil {
		return nil, err
	}

	s.appendActivity(ctx, petOwnerID, "stroked_by", map[string]any{"stroker_id": strokerID})
	if evolvedTo > 0 {
		s.celebrateEvolution(ctx, pet, evolvedTo)
	}
	s.emitPetUpdate(ctx, stroker)
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

// ─────────────────────────── зоопарк ───────────────────────────────

// GetZoo — витрина питомцев компании. viewerID нужен для strokes_today:
// «наглажен до завтра» должен переживать перезагрузку страницы.
func (s *Service) GetZoo(ctx context.Context, companyID, viewerID int64) ([]*dto.PetDTO, error) {
	pets, err := s.pets.ListCompanyPets(ctx, companyID)
	if err != nil {
		return nil, err
	}
	// Fail-open: без счётчика витрина всё равно полезна, а лимит честно
	// защитит сам StrokePet.
	strokes, err := s.pets.StrokesTodayByStroker(ctx, viewerID, todayMSK())
	if err != nil {
		strokes = nil
	}
	result := make([]*dto.PetDTO, 0, len(pets))
	for _, p := range pets {
		d := dto.NewPet(p)
		if p.UserID != viewerID {
			n := strokes[p.UserID]
			d.StrokesToday = &n
		}
		result = append(result, d)
	}
	return result, nil
}

// ─────────────────── рейтинг питомцев компании ─────────────────────

const ratingTopLimit = 10

// GetRating — «Топ недели»: питомцы компании по кудосам, заработанным с
// начала текущей ISO-недели (pet_kudos_weekly); отдельно — строка зрителя,
// даже если он не попал в топ. Ничьи по кудосам ранжируются стадией/XP
// (порядок ListCompanyPets, стабильная сортировка).
func (s *Service) GetRating(ctx context.Context, companyID, viewerID int64) (map[string]any, error) {
	pets, err := s.pets.ListCompanyPets(ctx, companyID)
	if err != nil {
		return nil, err
	}
	isoYear, isoWeek := weekStartMSK().ISOWeek()
	// Fail-open: счётчик признания не должен ронять карточку рейтинга.
	kudosWeek, err := s.pets.WeeklyKudosCounts(ctx, companyID, isoYear, isoWeek)
	if err != nil {
		s.log.Warn("pets.rating_kudos_failed", "company_id", companyID, "error", err)
		kudosWeek = map[int64]int{}
	}
	pets = append([]*domain.Pet(nil), pets...)
	sort.SliceStable(pets, func(i, j int) bool {
		return kudosWeek[pets[i].UserID] > kudosWeek[pets[j].UserID]
	})
	entry := func(p *domain.Pet, position int) map[string]any {
		var nextXP any
		if p.Stage < domain.MaxStage {
			nextXP = domain.StageXP[p.Stage+1]
		}
		return map[string]any{
			"position":      position,
			"pet_name":      p.Name,
			"species":       p.Species,
			"stage":         p.Stage,
			"xp":            p.XP,
			"next_stage_xp": nextXP,
			"hat":           p.Hat,
			"sick":          p.SickSince != nil,
			"kudos_week":    kudosWeek[p.UserID],
			"user":          p.User,
		}
	}
	items := make([]map[string]any, 0, min(len(pets), ratingTopLimit))
	var me map[string]any
	for i, p := range pets {
		e := entry(p, i+1)
		if i < ratingTopLimit {
			items = append(items, e)
		}
		if p.UserID == viewerID {
			me = e
		}
	}
	return map[string]any{
		"items": items,
		"me":    me,
		"total": len(pets),
	}, nil
}

// ─────────────────────── «Сейчас в эфире» ──────────────────────────

func (s *Service) GetLive(ctx context.Context, companyID int64) (*dto.LiveDTO, error) {
	units, err := s.work.ListActiveUnits(ctx, companyID)
	if err != nil {
		return nil, err
	}
	items := make([]*dto.LiveItemDTO, 0, len(units))
	for _, u := range units {
		items = append(items, dto.NewLiveItem(u))
	}
	return &dto.LiveDTO{Items: items}, nil
}

// ─────────────────────── история активности ────────────────────────

const activityPageLimit = 50

// GetActivityLog — приватная история активности своего питомца (замена
// публичной ленты); видна только владельцу.
func (s *Service) GetActivityLog(ctx context.Context, userID int64) ([]*dto.ActivityLogDTO, error) {
	entries, err := s.activity.ListForPet(ctx, userID, activityPageLimit)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.ActivityLogDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, dto.NewActivityLog(e))
	}
	return out, nil
}

// ─────────────────────────── квест дня ─────────────────────────────

// pickQuestTemplate — детерминированный выбор по (user_id, day): один и тот
// же день — тот же квест (стабильность при перезапросе).
func pickQuestTemplate(userID int64, day time.Time) domain.QuestTemplate {
	idx := (int(userID)*1009 + pythonOrdinal(day)) % len(domain.QuestTemplates)
	return domain.QuestTemplates[idx]
}

// ensureTodayQuest — назначить дневной квест, если для сегодня его ещё нет
// (без сохранения — сохраняет вызывающий). Возвращает true, если состояние
// питомца реально изменилось (нужен SavePet) — вызывающие read-эндпоинты
// (GetMyPet/BumpQuest) не должны писать в БД впустую на каждый запрос:
// лишний SavePet без реального изменения — это гонка lost-update с
// конкурентными начислениями (хуки юнитов/задач могут писать в тот же
// момент), см. TestPetsUnitStoppedAwardsKudosAndXP.
func (s *Service) ensureTodayQuest(pet *domain.Pet) bool {
	today := todayMSK()
	if pet.QuestDate != nil && pet.QuestDate.Equal(today) && pet.QuestKind != nil {
		return false
	}
	tpl := pickQuestTemplate(pet.UserID, today)
	day := today
	pet.QuestDate = &day
	pet.QuestKind = &tpl.Kind
	target := tpl.Target
	pet.QuestTarget = &target
	pet.QuestProgress = 0
	pet.QuestClaimed = false
	return true
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
	questChanged := s.ensureTodayQuest(pet)
	if pet.QuestKind == nil || *pet.QuestKind != kind || pet.QuestClaimed {
		// Квест не совпал по типу (или уже забран) — писать нечего, кроме
		// случая реального переезда на новый день (questChanged): та же
		// причина, что и в GetMyPet — впустую сохранённый снимок гоняет
		// lost-update с конкурентными начислениями хуков.
		if questChanged {
			if err := s.pets.SavePet(ctx, pet); err != nil {
				s.log.Warn("pets.quest_bump_failed", "user_id", userID, "error", err)
			}
		}
		return
	}
	target := 0
	if pet.QuestTarget != nil {
		target = *pet.QuestTarget
	}
	pet.QuestProgress = min(target, pet.QuestProgress+amount)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		s.log.Warn("pets.quest_bump_failed", "user_id", userID, "kind", kind, "error", err)
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
	pet.Kudos += domain.QuestRewardKudos
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.emitPetUpdate(ctx, pet)
	return dto.NewPet(pet), nil
}

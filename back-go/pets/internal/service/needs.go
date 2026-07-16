package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/dto"
)

// Потребности грувика: ленивое убывание шкал, вытекающие болезни и побег
// заброшенного питомца. Всё считается на чтении (по needs_at), без фонового
// цикла — фон лишь подчищает тех, кто давно не заходил (loops.go).

// refreshNeeds — привести питомца к текущему моменту (без побега): применить
// убывание шкал и, если какая-то опустела, уложить в её болезнь. Сохраняет
// узким SaveNeeds и только при реальном изменении — read-путь не должен
// впустую гонять запись (та же причина, что у GetMyPet/BumpQuest: лишний
// UPDATE рискует затереть конкурентное начисление хука). Никогда не бросает.
// true — состояние изменилось (клиентам нужен свежий снапшот).
func (s *Service) refreshNeeds(ctx context.Context, pet *domain.Pet) bool {
	changed := pet.ApplyNeedsDecay(time.Now().UTC())
	fell := ""
	if !pet.Sick() {
		if fell = pet.PendingAilment(); fell != "" {
			pet.Fall(fell, time.Now().UTC())
			changed = true
		}
	}
	if !changed {
		return false
	}
	if err := s.pets.SaveNeeds(ctx, pet); err != nil {
		s.log.Warn("pets.needs_save_failed", "user_id", pet.UserID, "error", err)
		return false
	}
	if fell != "" {
		s.onFellSick(ctx, pet, fell)
	}
	return true
}

// onFellSick — фиксация заболевания: приватная история + пуш владельцу
// (pushsvc слушает pet:sick — запущенная болезнь кончается побегом, и хозяин
// должен узнать о ней, не открывая приложение).
func (s *Service) onFellSick(ctx context.Context, pet *domain.Pet, ailment string) {
	s.appendActivity(ctx, pet.UserID, "sickness_started", map[string]any{"ailment": ailment})
	s.pub.Publish(ctx, "pet:sick", []string{userRoom(pet.UserID)}, map[string]any{
		"user_id": pet.UserID, "name": pet.Name,
		"ailment": ailment, "ailment_title": domain.AilmentTitle(ailment),
	})
}

// maybeRunAway — побег заброшенного питомца: болеет дольше RunawaySickDays —
// уходит, оставляя новое яйцо. Фиксация атомарна (guard в WHERE), поэтому
// ленивый GET владельца и фоновый цикл не сделают этого дважды. Возвращает
// снапшот побега, если зафиксировал именно этот вызов; pet при этом
// перечитывается — сброс делает SQL, не локальная копия.
func (s *Service) maybeRunAway(ctx context.Context, pet *domain.Pet) *dto.RunawayDTO {
	if !pet.Sick() {
		return nil
	}
	deadline := time.Now().UTC().AddDate(0, 0, -domain.RunawaySickDays)
	if pet.SickSince.After(deadline) {
		return nil
	}
	ailment := pet.AilmentKey()
	gone, err := s.pets.RunAway(ctx, pet.UserID, deadline)
	if err != nil {
		s.log.Warn("pets.runaway_failed", "user_id", pet.UserID, "error", err)
		return nil
	}
	if !gone {
		return nil // конкурентный вызов уже зафиксировал побег (или вылечили)
	}
	name := pet.Name
	if fresh, err := s.pets.GetPet(ctx, pet.UserID); err == nil && fresh != nil {
		*pet = *fresh
	} else {
		pet.Cure()
	}
	s.appendActivity(ctx, pet.UserID, "ran_away",
		map[string]any{"name": name, "ailment": ailment, "days": domain.RunawaySickDays})
	// Комната владельца: пуш офлайн-хозяину (pushsvc) + живой тост во вкладках.
	s.pub.Publish(ctx, "pet:runaway", []string{userRoom(pet.UserID)}, map[string]any{
		"user_id": pet.UserID, "name": name,
		"ailment": ailment, "ailment_title": domain.AilmentTitle(ailment),
		"days": domain.RunawaySickDays,
	})
	s.emitPetUpdate(ctx, pet)
	return &dto.RunawayDTO{Name: name, Ailment: ailment, Days: domain.RunawaySickDays}
}

// syncPet — полный ленивый апдейт состояния на read-пути владельца: возврат
// из приключения → убывание потребностей и болезни → побег. Порядок важен:
// вернувшийся из похода питомец должен успеть проголодаться, а сбежавший —
// не получить награду за поход задним числом.
func (s *Service) syncPet(ctx context.Context, pet *domain.Pet) (*dto.AdventureRewardDTO, *dto.RunawayDTO) {
	reward := s.maybeReturnAdventure(ctx, pet)
	s.refreshNeeds(ctx, pet)
	runaway := s.maybeRunAway(ctx, pet)
	if runaway != nil {
		reward = nil // питомец ушёл — хвастаться добычей уже некому
	}
	return reward, runaway
}

// adjustNeeds — атомарный сдвиг шкал по эффекту действия + обновление
// снапшота свежими значениями (не досчитываем дельту сами — конкурентный
// сдвиг учёлся бы дважды). Никогда не бросает: потребности не должны ронять
// действие, ради которого пришёл пользователь.
func (s *Service) adjustNeeds(ctx context.Context, pet *domain.Pet, action string) {
	needs, err := s.pets.AdjustNeeds(ctx, pet.UserID, domain.NeedGains[action])
	if err != nil {
		s.log.Warn("pets.needs_adjust_failed", "user_id", pet.UserID, "action", action, "error", err)
		return
	}
	pet.Needs = needs
}

// applyAction — общий эффект действия владельца на состояние питомца (без
// сохранения): двигает потребности и лечит, если действие — верный рецепт от
// текущей болезни. Возвращает признак выздоровления.
func (s *Service) applyAction(pet *domain.Pet, action string) bool {
	pet.ApplyNeedGains(action)
	if !pet.Sick() {
		return false
	}
	return applyRecovery(pet, domain.CureFor(pet.AilmentKey(), action))
}

// ─────────────────────────────── сон ────────────────────────────────

// SleepPet — бесплатный сон: восполняет энергию (единственная потребность,
// которую нельзя закрывать за кудосы — иначе питомец без денег обречён) и
// лечит простуду. Ограничен SleepDailyMax раз в день.
func (s *Service) SleepPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}
	s.refreshNeeds(ctx, pet)
	if s.daily.TakeBudget(ctx, userID, "sleeps", 1, domain.SleepDailyMax) <= 0 {
		return nil, domain.NewError("SLEPT_ENOUGH", "Питомец выспался — больше сегодня не уснёт", 429)
	}
	recovered := s.applyAction(pet, domain.ActionSleep)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.appendActivity(ctx, userID, "slept", nil)
	if recovered {
		s.appendActivity(ctx, userID, "recovered", nil)
	}
	s.emitPetUpdate(ctx, pet)
	data := dto.NewPet(pet)
	data.Recovered = &recovered
	return data, nil
}

// ────────────────────────────── купание ─────────────────────────────

// BathPet — платное купание: восполняет чистоту и одним разом лечит
// «грязнулю» (верный рецепт — сильный эффект).
func (s *Service) BathPet(ctx context.Context, userID, companyID int64) (*dto.PetDTO, error) {
	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAway(ctx, pet); err != nil {
		return nil, err
	}
	s.refreshNeeds(ctx, pet)
	if pet.Kudos < domain.BathCost {
		return nil, domain.NewError("NO_KUDOS", "Не хватает кудосов на купание", 422)
	}
	if s.daily.TakeBudget(ctx, userID, "baths", 1, domain.BathDailyMax) <= 0 {
		return nil, domain.NewError("WASHED_ENOUGH", "Чище уже некуда — купаний на сегодня хватит", 429)
	}
	pet.Kudos -= domain.BathCost
	recovered := s.applyAction(pet, domain.ActionBath)
	if err := s.pets.SavePet(ctx, pet); err != nil {
		return nil, err
	}
	s.appendActivity(ctx, userID, "bathed", nil)
	s.appendLedger(ctx, userID, companyID, -domain.BathCost, "bath", nil, "")
	if recovered {
		s.appendActivity(ctx, userID, "recovered", nil)
	}
	s.emitPetUpdate(ctx, pet)
	data := dto.NewPet(pet)
	data.Recovered = &recovered
	return data, nil
}

package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/dto"
)

// HandleWebhook — применить событие LiveKit. Источник истины о том, кто
// реально в комнате; финализирует историю и публикует сокет-события
// клиентам (через gatewaysvc). Работает и после рестарта сервиса:
// call_id восстанавливается из имени комнаты, запись — из БД.
func (s *Service) HandleWebhook(ctx context.Context, event dto.WebhookEvent) error {
	callID := domain.CallIDFromRoom(event.Room)
	if callID == 0 {
		return nil
	}
	s.log.Info("livekit.webhook", "event", event.Event, "room", event.Room,
		"identity", event.Identity, "call_id", callID)

	switch event.Event {
	case "participant_joined":
		return s.applyParticipantJoined(ctx, callID, event.Identity)
	case "participant_left":
		return s.applyParticipantLeft(ctx, callID, event.Identity)
	case "room_finished":
		return s.applyRoomFinished(ctx, callID)
	}
	return nil
}

// ensureRingState — восстановить потерянный ринг-state живого звонка
// (рестарт сервиса при упавшем ReconcileStartup): без него participant_joined
// никогда не переведёт звонок в active, а participant_left через
// ShouldEnd()==true для незнакомого callID финализирует звонок, в котором
// ещё есть люди.
func (s *Service) ensureRingState(ctx context.Context, call *domain.Call) error {
	if _, ok := s.ring.Snapshot(call.ID); ok || call.Finished() {
		return nil
	}
	var identities []string
	if call.RoomName != "" {
		identities, _ = s.media.ListParticipantIdentities(ctx, call.RoomName)
	}
	if err := s.restoreRingState(ctx, call, identities); err != nil {
		return err
	}
	s.log.Info("calls.ring_state_restored", "call_id", call.ID, "occupants", len(identities))
	return nil
}

// applyParticipantJoined — кто-то реально подключился к комнате.
func (s *Service) applyParticipantJoined(ctx context.Context, callID int64, identity string) error {
	call, err := s.repo.GetCall(ctx, callID)
	if err != nil {
		return err
	}
	if call == nil || call.Finished() {
		return nil
	}
	if err := s.ensureRingState(ctx, call); err != nil {
		return err
	}

	if userID := domain.UserIDFromIdentity(identity); userID > 0 {
		if ring, ok := s.ring.Snapshot(callID); ok && !domain.Has(ring.Invited, userID) {
			s.ring.AddInvitee(callID, userID)
		}
		s.ring.MarkJoined(callID, userID)
		part, err := s.repo.GetParticipant(ctx, callID, userID)
		if err != nil {
			return err
		}
		if part != nil {
			if part.JoinedAt == nil {
				ts := now()
				part.JoinedAt = &ts
			}
			part.LeftAt = nil
			if err := s.repo.UpdateParticipant(ctx, part); err != nil {
				return err
			}
		}
	} else {
		s.ring.AddGuest(callID, identity)
	}

	// «Разговор начался» = в комнате двое. Сам инициатор, сидящий в комнате
	// один во время дозвона, статус не меняет.
	if call.Status == domain.StatusRinging && s.ring.OccupantsCount(callID) >= 2 {
		call.Status = domain.StatusActive
		if err := s.repo.UpdateCall(ctx, call); err != nil {
			return err
		}
		s.pub.PillUpdated(ctx, callID)
	}
	return nil
}

// applyParticipantLeft — кто-то отключился от комнаты. Сразу ничего не
// вычищаем и не финализируем: перезагрузка страницы, переход по
// ссылке-приглашению и перехват identity второй вкладкой выглядят точно так
// же, как настоящий выход. Даём грейс-период вернуться, после — сверяем
// состав с фактическим списком LiveKit и закрываем звонок, только если он
// действительно опустел. Явный выход кнопкой (LeaveCall/EndCall) финализирует
// мгновенно, как и раньше.
func (s *Service) applyParticipantLeft(ctx context.Context, callID int64, _ string) error {
	call, err := s.repo.GetCall(ctx, callID)
	if err != nil {
		return err
	}
	if call == nil || call.Finished() {
		return nil
	}
	if err := s.ensureRingState(ctx, call); err != nil {
		return err
	}
	s.scheduleLeftSweep(callID)
	return nil
}

// scheduleLeftSweep — отложенная сверка состава звонка; не более одной
// запланированной на звонок. При leftGrace <= 0 — синхронно (тесты).
func (s *Service) scheduleLeftSweep(callID int64) {
	if s.leftGrace <= 0 {
		if err := s.sweepDeparted(context.Background(), callID); err != nil {
			s.log.Error("calls.left_sweep_failed", "call_id", callID, "error", err)
		}
		return
	}
	s.sweepMu.Lock()
	if _, scheduled := s.sweepSet[callID]; scheduled {
		s.sweepMu.Unlock()
		return
	}
	s.sweepSet[callID] = struct{}{}
	s.sweepMu.Unlock()

	time.AfterFunc(s.leftGrace, func() {
		s.sweepMu.Lock()
		delete(s.sweepSet, callID)
		s.sweepMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.sweepDeparted(ctx, callID); err != nil {
			s.log.Error("calls.left_sweep_failed", "call_id", callID, "error", err)
		}
	})
}

// sweepDeparted — сверка после грейс-периода: участники, бывавшие в комнате
// (joined_at есть) и не вернувшиеся, помечаются ушедшими; пропавшие гости
// вычищаются; опустевший звонок финализируется с событиями клиентам.
func (s *Service) sweepDeparted(ctx context.Context, callID int64) error {
	call, err := s.repo.GetCall(ctx, callID)
	if err != nil {
		return err
	}
	if call == nil || call.Finished() {
		return nil
	}
	if err := s.ensureRingState(ctx, call); err != nil {
		return err
	}
	identities, ok := s.media.ListParticipantIdentities(ctx, call.RoomName)
	if !ok {
		return nil // LiveKit недоступен — опустевшую комнату добьёт room_finished
	}

	presentUsers := make(map[int64]struct{})
	presentGuests := make(map[string]struct{})
	for _, identity := range identities {
		if uid := domain.UserIDFromIdentity(identity); uid > 0 {
			presentUsers[uid] = struct{}{}
		} else {
			presentGuests[identity] = struct{}{}
		}
	}

	// Снимок ДО вычистки: ушедший тоже должен получить call_ended
	// (его другие вкладки/устройства).
	ring, _ := s.ring.Snapshot(callID)

	parts, err := s.repo.ListParticipants(ctx, callID)
	if err != nil {
		return err
	}
	ts := now()
	for _, part := range parts {
		if part.JoinedAt == nil || part.LeftAt != nil {
			continue // ещё дозванивается / уже ушёл
		}
		if _, here := presentUsers[part.UserID]; here {
			continue
		}
		s.ring.RemoveUserFromCall(callID, part.UserID)
		part.LeftAt = &ts
		if err := s.repo.UpdateParticipant(ctx, part); err != nil {
			return err
		}
	}
	if ring != nil {
		for _, g := range ring.Guests {
			if _, here := presentGuests[g]; !here {
				s.ring.RemoveGuest(callID, g)
			}
		}
	}

	if !s.ring.ShouldEnd(callID) {
		return nil
	}
	if err := s.finalize(ctx, call); err != nil {
		return err
	}
	s.ring.EndCall(callID)
	s.pub.CallEnded(ctx, callID, call.Status, s.endedNotifyIDs(ctx, call, ring))
	s.pub.PillUpdated(ctx, callID)
	return nil
}

// applyRoomFinished — комната LiveKit закрылась (все вышли или DeleteRoom).
func (s *Service) applyRoomFinished(ctx context.Context, callID int64) error {
	call, err := s.repo.GetCall(ctx, callID)
	if err != nil {
		return err
	}
	ring, _ := s.ring.EndCall(callID)
	if call == nil {
		return nil
	}

	if !call.Finished() {
		// Никогда не было второго участника — «не дозвонился».
		if call.Status == domain.StatusRinging {
			call.Status = domain.StatusMissed
		} else {
			call.Status = domain.StatusEnded
		}
		if call.EndedAt == nil {
			ts := now()
			call.EndedAt = &ts
		}
		if err := s.repo.UpdateCall(ctx, call); err != nil {
			return err
		}
	}
	if ring != nil {
		ts := now()
		for _, uid := range ring.Joined {
			part, err := s.repo.GetParticipant(ctx, callID, uid)
			if err != nil {
				return err
			}
			if part != nil && part.LeftAt == nil {
				part.LeftAt = &ts
				if err := s.repo.UpdateParticipant(ctx, part); err != nil {
					return err
				}
			}
		}
	}
	s.pub.CallEnded(ctx, callID, call.Status, s.endedNotifyIDs(ctx, call, ring))
	s.pub.PillUpdated(ctx, callID)
	return nil
}

// ReconcileStartup — при старте сервиса сверить звонки ringing/active из БД
// с LiveKit. Комнаты переживают рестарт: если комната жива и в ней люди —
// восстанавливаем ринг-state по фактическому составу; иначе финализируем
// запись (ringing → missed, active → ended), чтобы плашки в чате не звали
// в несуществующий звонок.
func (s *Service) ReconcileStartup(ctx context.Context) error {
	stuck, err := s.repo.ListUnfinishedCalls(ctx)
	if err != nil {
		return err
	}
	finalized, restored := 0, 0
	for _, call := range stuck {
		var identities []string
		ok := false
		if call.RoomName != "" {
			identities, ok = s.media.ListParticipantIdentities(ctx, call.RoomName)
		}
		if ok && len(identities) > 0 {
			if err := s.restoreRingState(ctx, call, identities); err != nil {
				return err
			}
			restored++
			continue
		}
		finalized++
		ended := now()
		if call.Status == domain.StatusRinging {
			call.Status = domain.StatusMissed
			ended = call.StartedAt
		} else {
			call.Status = domain.StatusEnded
		}
		call.EndedAt = &ended
		if err := s.repo.UpdateCall(ctx, call); err != nil {
			return err
		}
		if err := s.repo.CloseOpenParticipants(ctx, call.ID, ended); err != nil {
			return err
		}
	}
	if len(stuck) > 0 {
		s.log.Info("calls.startup_cleanup", "finalized", finalized, "restored", restored)
	}
	return nil
}

// restoreRingState — восстановить state живого звонка: кто в комнате — по
// факту от LiveKit, приглашённые — по записям в БД.
func (s *Service) restoreRingState(ctx context.Context, call *domain.Call, identities []string) error {
	var members []int64
	var guests []string
	for _, identity := range identities {
		if uid := domain.UserIDFromIdentity(identity); uid > 0 {
			members = append(members, uid)
		} else {
			guests = append(guests, identity)
		}
	}
	parts, err := s.repo.ListParticipants(ctx, call.ID)
	if err != nil {
		return err
	}
	var pending []int64
	for _, p := range parts {
		if p.LeftAt == nil && !p.Declined {
			pending = append(pending, p.UserID)
		}
	}
	invited := unionExcept(members, pending, 0)
	s.ring.RestoreCall(call.ID, call.InitiatorID, call.Kind, call.Media,
		invited, members, guests)
	return nil
}

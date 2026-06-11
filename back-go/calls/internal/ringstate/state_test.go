package ringstate

import (
	"testing"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

// Порт юнит-тестов back/tests/test_call_state.py: семантика ринг-state
// должна остаться байт-в-байт той же, что была в Python.

func TestCreateCallMarksEveryoneBusy(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20, 30}, domain.KindGroup, domain.MediaVideo)

	for _, uid := range []int64{10, 20, 30} {
		if !s.IsUserBusy(uid) {
			t.Errorf("пользователь %d должен быть занят", uid)
		}
	}
	if s.IsUserBusy(40) {
		t.Error("посторонний не должен быть занят")
	}
	if id, _ := s.UserActiveCall(20); id != 1 {
		t.Errorf("активный звонок приглашённого = %d, ожидался 1", id)
	}
	// Инициатор сразу joined.
	snap, _ := s.Snapshot(1)
	if !domain.Has(snap.Joined, 10) || domain.Has(snap.Joined, 20) {
		t.Errorf("joined некорректен: %v", snap.Joined)
	}
}

func TestMarkJoinedRequiresInvite(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20}, domain.KindP2P, domain.MediaAudio)

	s.MarkJoined(1, 99) // не приглашён
	snap, _ := s.Snapshot(1)
	if domain.Has(snap.Joined, 99) {
		t.Error("непрошенный не должен попасть в joined")
	}

	s.MarkJoined(1, 20)
	snap, _ = s.Snapshot(1)
	if !domain.Has(snap.Joined, 20) {
		t.Error("приглашённый должен попасть в joined")
	}
}

func TestMarkJoinedIdempotent(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20}, domain.KindP2P, domain.MediaVideo)
	s.MarkJoined(1, 20)
	s.MarkJoined(1, 20) // rejoin после F5 — не дублирует
	snap, _ := s.Snapshot(1)
	if len(snap.Joined) != 2 {
		t.Errorf("joined = %v, ожидались ровно двое", snap.Joined)
	}
}

func TestDeclineFreesUser(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20}, domain.KindP2P, domain.MediaVideo)
	s.MarkDeclined(1, 20)

	if s.IsUserBusy(20) {
		t.Error("отклонивший должен освободиться")
	}
	snap, _ := s.Snapshot(1)
	if !domain.Has(snap.Declined, 20) {
		t.Error("отклонивший должен числиться в declined")
	}
}

// Регресс v2.6.2: выход обязан убирать и из invited, иначе ушедший вечно
// «pending» и p2p-звонок не завершается, когда собеседник кладёт трубку.
func TestRemoveUserAlsoDropsInvite(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20}, domain.KindP2P, domain.MediaVideo)
	s.MarkJoined(1, 20)
	s.RemoveUserFromCall(1, 20)

	snap, _ := s.Snapshot(1)
	if domain.Has(snap.Invited, 20) || domain.Has(snap.Joined, 20) {
		t.Errorf("ушедший остался в state: invited=%v joined=%v", snap.Invited, snap.Joined)
	}
	if !s.ShouldEnd(1) {
		t.Error("оставшийся один инициатор — звонок должен завершиться")
	}
}

func TestShouldEndWaitsPendingInvitees(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20, 30}, domain.KindGroup, domain.MediaVideo)

	if s.ShouldEnd(1) {
		t.Error("инициатор один, но есть pending-приглашённые — ждём")
	}
	s.MarkDeclined(1, 20)
	s.MarkDeclined(1, 30)
	if !s.ShouldEnd(1) {
		t.Error("все отклонили — завершаем")
	}
}

func TestShouldEndTwoJoined(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20}, domain.KindP2P, domain.MediaVideo)
	s.MarkJoined(1, 20)
	if s.ShouldEnd(1) {
		t.Error("двое в комнате — звонок живёт")
	}
}

func TestGuestsCountInOccupantsAndShouldEnd(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20}, domain.KindP2P, domain.MediaVideo)
	s.MarkDeclined(1, 20)
	s.AddGuest(1, "g-abc123")

	if got := s.OccupantsCount(1); got != 2 {
		t.Errorf("occupants = %d, ожидалось 2 (инициатор + гость)", got)
	}
	if s.ShouldEnd(1) {
		t.Error("инициатор + гость — звонок живёт")
	}
	s.RemoveGuest(1, "g-abc123")
	if !s.ShouldEnd(1) {
		t.Error("гость ушёл, отказ получен — завершаем")
	}
}

func TestEndCallFreesEveryone(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20, 30}, domain.KindGroup, domain.MediaVideo)
	snap, ok := s.EndCall(1)
	if !ok || snap == nil {
		t.Fatal("EndCall должен вернуть прежнее состояние")
	}
	for _, uid := range []int64{10, 20, 30} {
		if s.IsUserBusy(uid) {
			t.Errorf("после end пользователь %d должен освободиться", uid)
		}
	}
	if _, ok := s.Snapshot(1); ok {
		t.Error("state звонка должен исчезнуть")
	}
	if !s.ShouldEnd(1) {
		t.Error("несуществующий звонок считается завершённым")
	}
}

func TestRestoreCall(t *testing.T) {
	s := New()
	s.RestoreCall(7, 10, domain.KindGroup, domain.MediaVideo,
		[]int64{10, 20, 30}, []int64{10, 20}, []string{"g-feed01"})

	if !s.IsUserBusy(30) {
		t.Error("pending-приглашённый занят после restore")
	}
	if got := s.OccupantsCount(7); got != 3 {
		t.Errorf("occupants = %d, ожидалось 3", got)
	}
	if s.ShouldEnd(7) {
		t.Error("восстановленный живой звонок не должен завершаться")
	}
	snap, _ := s.Snapshot(7)
	if snap.InitiatorID != 10 || snap.Kind != domain.KindGroup {
		t.Errorf("метаданные restore потеряны: %+v", snap)
	}
}

func TestAddInviteeMarksBusy(t *testing.T) {
	s := New()
	s.CreateCall(1, 10, []int64{20}, domain.KindP2P, domain.MediaVideo)
	s.AddInvitee(1, 30)

	if !s.IsUserBusy(30) {
		t.Error("новоприглашённый должен считаться занятым")
	}
	snap, _ := s.Snapshot(1)
	if !domain.Has(snap.Invited, 30) {
		t.Error("новоприглашённый должен быть в invited")
	}
	// На несуществующий звонок — no-op.
	s.AddInvitee(99, 40)
	if s.IsUserBusy(40) {
		t.Error("invite в несуществующий звонок не должен занимать пользователя")
	}
}

// Package ringstate — состояние ринг-фазы активных звонков в памяти процесса.
//
// Порт sockets/call_state.py. Медиа-потоки ходят через LiveKit; здесь только
// «кому отправлен invite, кто принял/отклонил» и быстрые проверки занятости.
// Список «кто реально в комнате» подтверждают вебхуки LiveKit — они же
// финализируют историю в БД, даже если этот state потерялся при рестарте.
//
// Сервис звонков развёрнут в одном экземпляре, поэтому in-memory достаточно;
// при горизонтальном масштабировании — выносить в Redis.
package ringstate

import (
	"sync"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

type callState struct {
	initiatorID int64
	kind        string
	media       string
	invited     map[int64]struct{} // кому отправлен invite (включая инициатора)
	joined      map[int64]struct{} // кто из пользователей платформы в комнате
	declined    map[int64]struct{} // кто явно отклонил
	guests      map[string]struct{} // identity внешних гостей по ссылке
}

// State — потокобезопасная реализация domain.RingState.
type State struct {
	mu       sync.Mutex
	calls    map[int64]*callState
	userCall map[int64]int64 // активный звонок пользователя
}

var _ domain.RingState = (*State)(nil)

func New() *State {
	return &State{
		calls:    make(map[int64]*callState),
		userCall: make(map[int64]int64),
	}
}

func (s *State) UserActiveCall(userID int64) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.userCall[userID]
	return id, ok
}

// IsUserBusy — занят, если уже в звонке или ему висит активный invite.
func (s *State) IsUserBusy(userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.userCall[userID]
	return ok
}

func (s *State) Snapshot(callID int64) (*domain.RingSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return nil, false
	}
	return snapshotOf(c), true
}

func (s *State) OccupantsCount(callID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return 0
	}
	return len(c.joined) + len(c.guests)
}

// CreateCall — регистрация нового звонка. Инициатор сразу считается joined
// (он подключается к комнате LiveKit немедленно), приглашённые висят в
// invited до accept/decline.
func (s *State) CreateCall(callID, initiatorID int64, inviteeIDs []int64, kind, media string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &callState{
		initiatorID: initiatorID,
		kind:        kind,
		media:       media,
		invited:     map[int64]struct{}{initiatorID: {}},
		joined:      map[int64]struct{}{initiatorID: {}},
		declined:    make(map[int64]struct{}),
		guests:      make(map[string]struct{}),
	}
	s.userCall[initiatorID] = callID
	for _, uid := range inviteeIDs {
		c.invited[uid] = struct{}{}
		s.userCall[uid] = callID
	}
	s.calls[callID] = c
}

// RestoreCall — восстановление state живого звонка после рестарта сервиса
// (по записи в БД и фактическому составу комнаты LiveKit).
func (s *State) RestoreCall(callID, initiatorID int64, kind, media string,
	invited, joined []int64, guests []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &callState{
		initiatorID: initiatorID,
		kind:        kind,
		media:       media,
		invited:     make(map[int64]struct{}, len(invited)),
		joined:      make(map[int64]struct{}, len(joined)),
		declined:    make(map[int64]struct{}),
		guests:      make(map[string]struct{}, len(guests)),
	}
	for _, uid := range invited {
		c.invited[uid] = struct{}{}
		s.userCall[uid] = callID
	}
	for _, uid := range joined {
		c.joined[uid] = struct{}{}
	}
	for _, g := range guests {
		c.guests[g] = struct{}{}
	}
	s.calls[callID] = c
}

// AddInvitee — новый приглашённый в идущий звонок; висит в invited до accept.
func (s *State) AddInvitee(callID, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return
	}
	c.invited[userID] = struct{}{}
	s.userCall[userID] = callID
}

func (s *State) SetKind(callID int64, kind string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.calls[callID]; ok {
		c.kind = kind
	}
}

func (s *State) MarkJoined(callID, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return
	}
	if _, invited := c.invited[userID]; !invited {
		return
	}
	c.joined[userID] = struct{}{}
	s.userCall[userID] = callID
}

func (s *State) MarkDeclined(callID, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return
	}
	c.declined[userID] = struct{}{}
	delete(c.joined, userID)
	if s.userCall[userID] == callID {
		delete(s.userCall, userID)
	}
}

func (s *State) AddGuest(callID int64, identity string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.calls[callID]; ok {
		c.guests[identity] = struct{}{}
	}
}

func (s *State) RemoveGuest(callID int64, identity string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.calls[callID]; ok {
		delete(c.guests, identity)
	}
}

// RemoveUserFromCall — выход из звонка. Убираем и из invited: иначе ушедший
// навсегда считается «ожидающим» и ShouldEnd никогда не вернёт true (p2p не
// завершится, когда собеседник положил трубку).
func (s *State) RemoveUserFromCall(callID, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return
	}
	delete(c.joined, userID)
	delete(c.invited, userID)
	if s.userCall[userID] == callID {
		delete(s.userCall, userID)
	}
}

// EndCall — полностью снять звонок: убрать всех из user→call и вернуть
// прежнее состояние.
func (s *State) EndCall(callID int64) (*domain.RingSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return nil, false
	}
	delete(s.calls, callID)
	for uid := range c.invited {
		if s.userCall[uid] == callID {
			delete(s.userCall, uid)
		}
	}
	return snapshotOf(c), true
}

// ShouldEnd — звонок пора закрывать: в комнате никого или остался один,
// и больше никто не приглашён (некого ждать).
func (s *State) ShouldEnd(callID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[callID]
	if !ok {
		return true
	}
	occupants := len(c.joined) + len(c.guests)
	if occupants > 1 {
		return false
	}
	for uid := range c.invited {
		if _, joined := c.joined[uid]; joined {
			continue
		}
		if _, declined := c.declined[uid]; declined {
			continue
		}
		return false // есть pending-приглашённый — ждём его
	}
	return true
}

func snapshotOf(c *callState) *domain.RingSnapshot {
	snap := &domain.RingSnapshot{
		InitiatorID: c.initiatorID,
		Kind:        c.kind,
		Media:       c.media,
	}
	for uid := range c.invited {
		snap.Invited = append(snap.Invited, uid)
	}
	for uid := range c.joined {
		snap.Joined = append(snap.Joined, uid)
	}
	for uid := range c.declined {
		snap.Declined = append(snap.Declined, uid)
	}
	for g := range c.guests {
		snap.Guests = append(snap.Guests, g)
	}
	return snap
}

package domain

import (
	"context"
	"time"
)

// CallRepository — персистентность звонков (PostgreSQL, общая БД платформы).
type CallRepository interface {
	// CreateCall — звонок + участники одной транзакцией; заполняет call.ID,
	// RoomName и ShareCode уже проставлены вызывающим.
	CreateCall(ctx context.Context, call *Call, participants []*Participant) error
	GetCall(ctx context.Context, id int64) (*Call, error)
	GetCallByShareCode(ctx context.Context, code string) (*Call, error)
	UpdateCall(ctx context.Context, call *Call) error
	// DeleteCall — физическое удаление (откат не состоявшегося звонка).
	DeleteCall(ctx context.Context, id int64) error

	GetParticipant(ctx context.Context, callID, userID int64) (*Participant, error)
	// ListParticipants — с ФИО/аватаром (join users), в порядке записи.
	ListParticipants(ctx context.Context, callID int64) ([]*Participant, error)
	CreateParticipant(ctx context.Context, p *Participant) error
	UpdateParticipant(ctx context.Context, p *Participant) error
	// CloseOpenParticipants — left_at для всех, кто его ещё не имеет.
	CloseOpenParticipants(ctx context.Context, callID int64, leftAt time.Time) error

	// ListUnfinishedCalls — ringing/active (реконсиляция при старте сервиса).
	ListUnfinishedCalls(ctx context.Context) ([]*Call, error)
	ListHistoryForUser(ctx context.Context, userID int64, limit int) ([]*Call, error)
}

// UserReader — read-only доступ к пользователям платформы.
type UserReader interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	// ListVisibleUsers — только существующие и не скрытые.
	ListVisibleUsers(ctx context.Context, ids []int64) ([]*User, error)
	// CompanyActive — активна ли выбранная (активная) компания сессии из
	// токена. nil (Администратор системы) → true.
	CompanyActive(ctx context.Context, companyID *int64) (bool, error)
	// Memberships — множество компаний для каждого пользователя из членств
	// user_companies (многокомпанийность). Нужно лишь чтобы проставить звонку
	// общую компанию, если она есть; общей компании может и не быть.
	Memberships(ctx context.Context, ids []int64) (map[int64]map[int64]bool, error)
}

// RingSnapshot — копия состояния ринг-фазы звонка на момент запроса.
type RingSnapshot struct {
	InitiatorID int64
	Kind        string
	Media       string
	Invited     []int64
	Joined      []int64
	Declined    []int64
	Guests      []string
}

// Has — входит ли пользователь в список.
func Has(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// RingState — состояние ринг-фазы активных звонков (in-memory).
type RingState interface {
	UserActiveCall(userID int64) (int64, bool)
	IsUserBusy(userID int64) bool
	Snapshot(callID int64) (*RingSnapshot, bool)
	OccupantsCount(callID int64) int

	CreateCall(callID, initiatorID int64, inviteeIDs []int64, kind, media string)
	RestoreCall(callID, initiatorID int64, kind, media string,
		invited, joined []int64, guests []string)
	AddInvitee(callID, userID int64)
	SetKind(callID int64, kind string)
	MarkJoined(callID, userID int64)
	MarkDeclined(callID, userID int64)
	AddGuest(callID int64, identity string)
	RemoveGuest(callID int64, identity string)
	RemoveUserFromCall(callID, userID int64)
	EndCall(callID int64) (*RingSnapshot, bool)
	ShouldEnd(callID int64) bool
}

// MediaServer — медиа-сервер звонков (LiveKit): токены и управление комнатами.
type MediaServer interface {
	AccessToken(identity, name, room string, metadata map[string]any) (string, error)
	// CreateRoom/DeleteRoom — best effort: недоступность LiveKit API не
	// фатальна (комната автосоздастся при первом подключении).
	CreateRoom(ctx context.Context, name string, maxParticipants int)
	DeleteRoom(ctx context.Context, name string)
	// ListParticipantIdentities — nil + false, если LiveKit недоступен.
	ListParticipantIdentities(ctx context.Context, room string) ([]string, bool)
	ClientURL() string
}

// EventPublisher — сокет-события клиентам через Redis gw2:calls:events
// (общий envelope {event, rooms, payload}; доставляет gatewaysvc).
// CallEnded нужен для изменений, которые инициирует сам сервис (вебхуки
// LiveKit): результат ринг-команд gateway эмитит по ответу gRPC сам.
// Плашка звонка в чате (kind='call') — оркестрация callsvc: PillCreated
// создаёт её через msgsvc и рассылает message:new, PillUpdated перечитывает
// снапшот и рассылает message:updated. Оба — fire-and-forget: плашка
// вторична и звонок не роняет.
type EventPublisher interface {
	CallEnded(ctx context.Context, callID int64, status string, notifyUserIDs []int64)
	PillCreated(ctx context.Context, conversationID, senderID, callID int64)
	PillUpdated(ctx context.Context, callID int64)
}

// MessengerClient — gRPC msgsvc: парный диалог для p2p-звонка (создаётся ДО
// записи звонка, чтобы FK conversation_id уже существовал).
type MessengerClient interface {
	EnsureDialog(ctx context.Context, userAID, userBID int64) (int64, error)
}

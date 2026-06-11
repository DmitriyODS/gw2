// Package domain — сущности и порты домена «звонки».
//
// Слой не знает ни о транспортах, ни о конкретных хранилищах: только
// бизнес-типы, доменные ошибки и интерфейсы (ports), которые реализуют
// внешние адаптеры (postgres, livekit, redis, in-memory ring state).
package domain

import (
	"fmt"
	"time"
)

const (
	KindP2P   = "p2p"
	KindGroup = "group"

	StatusRinging = "ringing"
	StatusActive  = "active"
	StatusEnded   = "ended"
	StatusMissed  = "missed"

	MediaAudio = "audio"
	MediaVideo = "video"

	RoleInitiator = "initiator"
	RoleInvitee   = "invitee"

	// MaxParticipants — жёсткий потолок конференции: инициатор + 8.
	// Гости по ссылке считаются в этом же лимите.
	MaxParticipants = 9
)

// Call — запись звонка (история в БД + текущий статус).
type Call struct {
	ID             int64
	InitiatorID    int64
	CompanyID      int64
	Kind           string
	Status         string
	Media          string
	StartedAt      time.Time
	EndedAt        *time.Time
	ConversationID *int64
	RoomName       string
	ShareCode      string
}

// Finished — звонок уже финализирован (вернуться/присоединиться нельзя).
func (c *Call) Finished() bool {
	return c.Status == StatusEnded || c.Status == StatusMissed
}

// RoomNameFor — имя комнаты LiveKit для звонка.
func RoomNameFor(callID int64) string { return fmt.Sprintf("call-%d", callID) }

// CallIDFromRoom — обратный разбор имени комнаты; 0 = не наша комната.
func CallIDFromRoom(room string) int64 {
	var id int64
	if _, err := fmt.Sscanf(room, "call-%d", &id); err != nil {
		return 0
	}
	return id
}

// IdentityForUser — identity участника платформы в LiveKit.
func IdentityForUser(userID int64) string { return fmt.Sprintf("u%d", userID) }

// UserIDFromIdentity — u{id} → id; гостевые identity (g-…) → 0.
func UserIDFromIdentity(identity string) int64 {
	var id int64
	if _, err := fmt.Sscanf(identity, "u%d", &id); err != nil {
		return 0
	}
	return id
}

// Participant — участник звонка (включая денормализованные поля пользователя
// для снапшотов: ФИО и аватар тянутся join'ом из users).
type Participant struct {
	ID         int64
	CallID     int64
	UserID     int64
	Role       string
	InvitedAt  time.Time
	JoinedAt   *time.Time
	LeftAt     *time.Time
	Declined   bool
	FIO        string
	AvatarPath *string
}

// User — проекция пользователя платформы (только нужное звонкам).
type User struct {
	ID            int64
	FIO           string
	AvatarPath    *string
	CompanyID     *int64
	IsHidden      bool
	CompanyActive bool
}

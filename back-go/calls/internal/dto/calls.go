// Package dto — структуры запросов/ответов сервисного слоя.
//
// JSON-теги повторяют форму marshmallow CallSchema из прежнего Flask-бэка:
// REST-ответы Fiber и сокет-payload'ы Flask должны быть байт-в-байт
// совместимы со старым фронтом.
package dto

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

// ParticipantDTO — участник звонка в снапшоте (CallParticipantBriefSchema).
type ParticipantDTO struct {
	UserID     int64   `json:"user_id"`
	FIO        string  `json:"fio"`
	AvatarPath *string `json:"avatar_path"`
	Role       string  `json:"role"`
	JoinedAt   *string `json:"joined_at"`
	LeftAt     *string `json:"left_at"`
	Declined   bool    `json:"declined"`
}

// CallDTO — снапшот звонка (CallSchema).
type CallDTO struct {
	ID             int64            `json:"id"`
	Kind           string           `json:"kind"`
	Status         string           `json:"status"`
	Media          string           `json:"media"`
	StartedAt      string           `json:"started_at"`
	EndedAt        *string          `json:"ended_at"`
	InitiatorID    int64            `json:"initiator_id"`
	InitiatorFIO   *string          `json:"initiator_fio"`
	ConversationID *int64           `json:"conversation_id"`
	ShareCode      *string          `json:"share_code"`
	DurationSec    *int64           `json:"duration_sec"`
	Participants   []ParticipantDTO `json:"participants"`
}

// LivekitDTO — данные подключения клиента к комнате LiveKit.
type LivekitDTO struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

// ── Запросы/ответы ринг-фазы (gRPC из Flask) ─────────────────────

type StartCallRequest struct {
	InitiatorID    int64
	InviteeIDs     []int64
	Media          string
	ConversationID int64 // 0 — нет парного диалога
}

type StartCallResponse struct {
	Call    *CallDTO
	Livekit LivekitDTO
}

type InviteRequest struct {
	CallID     int64
	InviterID  int64
	InviteeIDs []int64
}

type InviteResponse struct {
	Call          *CallDTO
	NewInviteeIDs []int64
	NotifyUserIDs []int64 // кто сейчас в звонке — им уйдёт call:invited
}

type AcceptRequest struct {
	CallID int64
	UserID int64
}

type AcceptResponse struct {
	Call    *CallDTO
	Livekit LivekitDTO
}

type HangupRequest struct { // decline / leave / end — одинаковая форма
	CallID int64
	UserID int64
}

type HangupResponse struct {
	Call          *CallDTO
	Ended         bool
	NotifyUserIDs []int64
}

// ── REST (Fiber) ─────────────────────────────────────────────────

type TokenResponse struct {
	Call    *CallDTO   `json:"call"`
	Livekit LivekitDTO `json:"livekit"`
}

type JoinInfoResponse struct {
	Status          string  `json:"status"`
	Media           string  `json:"media"`
	Kind            string  `json:"kind"`
	StartedAt       *string `json:"started_at"`
	InitiatorFIO    *string `json:"initiator_fio"`
	Occupants       int     `json:"occupants"`
	MaxParticipants int     `json:"max_participants"`
	Live            bool    `json:"live"`
}

type JoinByCodeRequest struct {
	Code      string
	UserID    int64 // 0 — внешний гость
	GuestName string
}

type JoinByCodeResponse struct {
	Call     *CallDTO   `json:"call"`
	Livekit  LivekitDTO `json:"livekit"`
	Identity string     `json:"identity"`
	Guest    bool       `json:"guest"`
}

type ActiveCallResponse struct {
	Call *CallDTO `json:"call"`
}

// ── Вебхук LiveKit ───────────────────────────────────────────────

type WebhookEvent struct {
	Event    string // participant_joined | participant_left | room_finished
	Room     string
	Identity string
}

// ── Сборка снапшота ──────────────────────────────────────────────

// FormatTime — единый формат времени наружу (RFC3339, как isoformat()).
func FormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatTime(*t)
	return &s
}

// NewCallDTO — снапшот из доменных сущностей.
func NewCallDTO(call *domain.Call, initiatorFIO string, parts []*domain.Participant) *CallDTO {
	out := &CallDTO{
		ID:           call.ID,
		Kind:         call.Kind,
		Status:       call.Status,
		Media:        call.Media,
		StartedAt:    FormatTime(call.StartedAt),
		EndedAt:      formatTimePtr(call.EndedAt),
		InitiatorID:  call.InitiatorID,
		Participants: make([]ParticipantDTO, 0, len(parts)),
	}
	if initiatorFIO != "" {
		out.InitiatorFIO = &initiatorFIO
	}
	if call.ConversationID != nil {
		out.ConversationID = call.ConversationID
	}
	if call.ShareCode != "" {
		out.ShareCode = &call.ShareCode
	}
	if call.EndedAt != nil {
		d := int64(call.EndedAt.Sub(call.StartedAt).Seconds())
		out.DurationSec = &d
	}
	for _, p := range parts {
		out.Participants = append(out.Participants, ParticipantDTO{
			UserID:     p.UserID,
			FIO:        p.FIO,
			AvatarPath: p.AvatarPath,
			Role:       p.Role,
			JoinedAt:   formatTimePtr(p.JoinedAt),
			LeftAt:     formatTimePtr(p.LeftAt),
			Declined:   p.Declined,
		})
	}
	return out
}

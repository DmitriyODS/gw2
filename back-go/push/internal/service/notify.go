package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/push/internal/domain"
)

// Dispatch — превратить событие микросервиса в пуши. Неинтересные события
// игнорируются. rooms — адресные комнаты из envelope (user_{id} / all).
func (s *Service) Dispatch(ctx context.Context, event string, payload json.RawMessage, rooms []string) {
	if !s.sender.Enabled() {
		return
	}
	switch event {
	case "message:new":
		s.onMessage(ctx, payload, rooms)
	case "task:created":
		s.onTask(ctx, payload)
	case "call:incoming":
		s.onCall(ctx, payload, rooms)
	}
}

func (s *Service) onMessage(ctx context.Context, payload json.RawMessage, rooms []string) {
	var e struct {
		ConversationID int64  `json:"conversation_id"`
		FromUserID     *int64 `json:"from_user_id"`
		Message        *struct {
			SenderID    *int64  `json:"sender_id"`
			Text        *string `json:"text"`
			Kind        string  `json:"kind"`
			Attachments []any   `json:"attachments"`
			Task        *struct {
				Name string `json:"name"`
			} `json:"task"`
		} `json:"message"`
	}
	if err := json.Unmarshal(payload, &e); err != nil || e.Message == nil {
		return
	}
	sender := e.FromUserID
	if sender == nil {
		sender = e.Message.SenderID
	}
	recipients := excluding(usersFromRooms(rooms), sender)
	if len(recipients) == 0 {
		return
	}

	title := "Новое сообщение"
	if sender != nil {
		if names, _ := s.users.Names(ctx, []int64{*sender}); names[*sender] != "" {
			title = names[*sender]
		}
	}
	body := messagePreview(e.Message.Text, e.Message.Kind, e.Message.Task != nil, len(e.Message.Attachments) > 0)

	s.deliver(ctx, recipients, domain.Notification{
		Title:   title,
		Body:    body,
		Channel: domain.ChannelMessages,
		Data: map[string]string{
			"type":            "message",
			"conversation_id": strconv.FormatInt(e.ConversationID, 10),
		},
	})
}

func (s *Service) onTask(ctx context.Context, payload json.RawMessage) {
	var e struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		AuthorID          int64  `json:"author_id"`
		ResponsibleUserID *int64 `json:"responsible_user_id"`
	}
	if err := json.Unmarshal(payload, &e); err != nil {
		return
	}
	// Пуш — ответственному (если он назначен и это не сам автор).
	if e.ResponsibleUserID == nil || *e.ResponsibleUserID == e.AuthorID {
		return
	}
	s.deliver(ctx, []int64{*e.ResponsibleUserID}, domain.Notification{
		Title:   "Новая задача",
		Body:    e.Name,
		Channel: domain.ChannelTasks,
		Data: map[string]string{
			"type":    "task",
			"task_id": strconv.FormatInt(e.ID, 10),
		},
	})
}

func (s *Service) onCall(ctx context.Context, payload json.RawMessage, rooms []string) {
	var e struct {
		ID           int64  `json:"id"`
		Media        string `json:"media"`
		InitiatorID  int64  `json:"initiator_id"`
		InitiatorFio string `json:"initiator_fio"`
	}
	if err := json.Unmarshal(payload, &e); err != nil {
		return
	}
	recipients := excluding(usersFromRooms(rooms), &e.InitiatorID)
	if len(recipients) == 0 {
		return
	}
	title := "Входящий звонок"
	if e.Media == "video" {
		title = "Входящий видеозвонок"
	}
	caller := e.InitiatorFio
	if caller == "" {
		caller = "Звонок"
	}
	s.deliver(ctx, recipients, domain.Notification{
		Title:        title,
		Body:         caller,
		Channel:      domain.ChannelCalls,
		HighPriority: true,
		Data: map[string]string{
			"type":      "call",
			"call_id":   strconv.FormatInt(e.ID, 10),
			"media":     e.Media,
			"caller":    caller,
			"caller_id": strconv.FormatInt(e.InitiatorID, 10),
		},
	})
}

func messagePreview(text *string, kind string, hasTask, hasAttachment bool) string {
	if text != nil && strings.TrimSpace(*text) != "" {
		return *text
	}
	switch {
	case kind == "call":
		return "Звонок"
	case kind == "post":
		return "Пересланный пост"
	case hasTask:
		return "Прикреплена задача"
	case hasAttachment:
		return "Вложение"
	default:
		return "Сообщение"
	}
}

// usersFromRooms — id из комнат вида "user_{id}" (комнату "all" игнорируем —
// в неё шлются company-wide события, не адресные пуши).
func usersFromRooms(rooms []string) []int64 {
	var out []int64
	for _, r := range rooms {
		if id, ok := strings.CutPrefix(r, "user_"); ok {
			if n, err := strconv.ParseInt(id, 10, 64); err == nil {
				out = append(out, n)
			}
		}
	}
	return out
}

func excluding(ids []int64, skip *int64) []int64 {
	if skip == nil {
		return ids
	}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != *skip {
			out = append(out, id)
		}
	}
	return out
}

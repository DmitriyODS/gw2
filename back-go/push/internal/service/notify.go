package service

import (
	"context"
	"encoding/json"
	"regexp"
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
	case "kudos:received":
		s.onKudos(ctx, payload, rooms)
	case "post:new":
		s.onPost(ctx, payload)
	}
}

// onPost — новый пост портала: адресован всей компании (комната all), поэтому
// получателей берём из членства компании; автору пуш не шлём.
func (s *Service) onPost(ctx context.Context, payload json.RawMessage) {
	var e struct {
		ID        int64  `json:"id"`
		CompanyID int64  `json:"company_id"`
		AuthorID  int64  `json:"author_id"`
		Title     string `json:"title"`
		Body      string `json:"body"`
	}
	if err := json.Unmarshal(payload, &e); err != nil || e.ID == 0 || e.CompanyID == 0 {
		return
	}
	members, err := s.users.MembersOf(ctx, e.CompanyID)
	if err != nil {
		s.log.Warn("push.post_members_failed", "company_id", e.CompanyID, "error", err)
		return
	}
	recipients := excluding(members, &e.AuthorID)
	if len(recipients) == 0 {
		return
	}
	title := "Новый пост на портале"
	author := ""
	if names, _ := s.users.Names(ctx, []int64{e.AuthorID}); names[e.AuthorID] != "" {
		author = names[e.AuthorID]
	}
	body := strings.TrimSpace(e.Title)
	if body == "" {
		body = postPreview(e.Body)
	}
	if author != "" {
		title = author + " — новый пост"
	}
	s.deliver(ctx, recipients, domain.Notification{
		Title:   title,
		Body:    body,
		Channel: domain.ChannelPortal,
		Data: map[string]string{
			"type":    "post",
			"post_id": strconv.FormatInt(e.ID, 10),
		},
	})
}

var (
	mdImage   = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	mdLink    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdFence   = regexp.MustCompile("(?m)^[ \t]*`{3,}.*$")
	mdRule    = regexp.MustCompile(`(?m)^[ \t]*(?:-{3,}|\*{3,}|_{3,})[ \t]*$`)
	mdHeading = regexp.MustCompile(`(?m)^[ \t]*#{1,6}[ \t]*`)
	mdQuote   = regexp.MustCompile(`(?m)^[ \t]*(?:>[ \t]*)+`)
	mdBullet  = regexp.MustCompile(`(?m)^[ \t]*(?:[-*+][ \t]+\[[ xX]\][ \t]*|[-*+][ \t]+|\d+\.[ \t]+)`)
	mdSpaces  = regexp.MustCompile(`\s+`)
)

// stripMarkdown — очистка markdown-разметки для текста уведомления (зеркало
// stripMarkdown portalsvc/front): маркеры убираются, содержимое остаётся,
// ссылки и картинки сводятся к тексту/alt, всё схлопывается в одну строку.
func stripMarkdown(s string) string {
	s = mdFence.ReplaceAllString(s, " ")
	s = mdRule.ReplaceAllString(s, " ")
	s = mdImage.ReplaceAllString(s, "$1")
	s = mdLink.ReplaceAllString(s, "$1")
	s = mdHeading.ReplaceAllString(s, "")
	s = mdQuote.ReplaceAllString(s, "")
	s = mdBullet.ReplaceAllString(s, "")
	s = strings.NewReplacer("**", "", "~~", "", "*", "", "`", "", "|", " ").Replace(s)
	return strings.TrimSpace(mdSpaces.ReplaceAllString(s, " "))
}

// postPreview — короткий текст без Markdown-обвязки для тела уведомления.
func postPreview(body string) string {
	clean := stripMarkdown(body)
	r := []rune(clean)
	if len(r) > 120 {
		return string(r[:120]) + "…"
	}
	if clean == "" {
		return "Открыть портал"
	}
	return clean
}

// onKudos — входящий перевод кудо-банка: адресный пуш получателю
// (rooms = [user_{id}], онлайн-гейт общий в deliver).
func (s *Service) onKudos(ctx context.Context, payload json.RawMessage, rooms []string) {
	var e struct {
		Amount  int    `json:"amount"`
		Comment string `json:"comment"`
		From    *struct {
			ID  int64  `json:"id"`
			FIO string `json:"fio"`
		} `json:"from"`
	}
	if err := json.Unmarshal(payload, &e); err != nil || e.Amount <= 0 {
		return
	}
	recipients := usersFromRooms(rooms)
	if len(recipients) == 0 {
		return
	}
	body := "Вам перевели кудосы"
	if e.From != nil && e.From.FIO != "" {
		body = "От " + e.From.FIO
	}
	if strings.TrimSpace(e.Comment) != "" {
		body += " — «" + e.Comment + "»"
	}
	s.deliver(ctx, recipients, domain.Notification{
		Title:   "+" + strconv.Itoa(e.Amount) + " кудосов 🎉",
		Body:    body,
		Channel: domain.ChannelKudos,
		Data:    map[string]string{"type": "kudos"},
	})
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
	// Разметка в уведомлении не рендерится — показываем чистый текст.
	if text != nil {
		if clean := stripMarkdown(*text); clean != "" {
			return clean
		}
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

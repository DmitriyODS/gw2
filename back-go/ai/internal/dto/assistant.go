package dto

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// AssistantMessage — форма REST /api/ai/assistant/history. Sources —
// провенанс («Данные: …»), MyFeedback — голос пользователя (up|down); оба
// заполняются только у сообщений роли assistant.
type AssistantMessage struct {
	ID         int64     `json:"id"`
	Role       string    `json:"role"`
	Text       string    `json:"text"`
	Sources    *string   `json:"sources,omitempty"`
	MyFeedback *string   `json:"my_feedback,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewAssistantMessages(items []domain.AssistantMessage) []AssistantMessage {
	out := make([]AssistantMessage, 0, len(items))
	for _, m := range items {
		out = append(out, AssistantMessage{
			ID: m.ID, Role: m.Role, Text: m.Text,
			Sources: m.Sources, MyFeedback: m.MyFeedback, CreatedAt: m.CreatedAt,
		})
	}
	return out
}

// AssistantReply — ответ POST /api/ai/assistant/messages: сохранённое
// сообщение ассистента (id — для обратной связи 👍/👎 на фронте).
type AssistantReply struct {
	ID        int64     `json:"id"`
	Text      string    `json:"text"`
	Sources   *string   `json:"sources,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Package http — REST /api/ai/assistant/* и /api/ai/my-settings: деловой
// ИИ-ассистент. Доступен любому авторизованному пользователю: ключ и диалог у
// ассистента ЛИЧНЫЕ (user_ai_settings), поэтому активная компания больше не
// обязательна — без неё работает всё, кроме инструментов компанийной
// статистики.
package http

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/service"
)

const (
	assistantHistoryDefaultLimit = 20
	assistantHistoryMaxLimit     = 100
)

// assistantScope — владелец диалога и его активная компания (nil — компании
// нет: супер-админ либо пользователь без компаний). Маршруты закрыты
// RequireAuth, поэтому пользователь в Locals всегда есть.
func assistantScope(c *fiber.Ctx) (userID int64, companyID *int64) {
	user := currentUser(c)
	if user == nil {
		return 0, nil
	}
	return user.ID, user.CompanyID
}

func (h *handlers) sendAssistantMessage(c *fiber.Ctx) error {
	userID, companyID := assistantScope(c)
	var body struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || strings.TrimSpace(body.Text) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "details": fiber.Map{"text": []string{"Missing data for required field."}},
		})
	}
	resp, err := h.eps.SendAssistantMessage(c.Context(), endpoint.SendAssistantMessageRequest{
		UserID: userID, CompanyID: companyID, Text: body.Text,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	reply := resp.(*service.AssistantReply)
	return c.JSON(dto.AssistantReply{
		ID: reply.ID, Text: reply.Text, Sources: reply.Sources, CreatedAt: reply.CreatedAt,
	})
}

// sendAssistantFeedback — POST /api/ai/assistant/feedback: голос 👍/👎 по
// ответу ассистента, идемпотентный upsert (повторный голос заменяет).
func (h *handlers) sendAssistantFeedback(c *fiber.Ctx) error {
	userID, _ := assistantScope(c)
	var body struct {
		MessageID int64   `json:"message_id"`
		Verdict   string  `json:"verdict"`
		Reason    *string `json:"reason"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.MessageID <= 0 || body.Verdict == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "details": fiber.Map{"message_id": []string{"Missing data for required field."}},
		})
	}
	if _, err := h.eps.SendAssistantFeedback(c.Context(), endpoint.SendAssistantFeedbackRequest{
		UserID:    userID,
		MessageID: body.MessageID, Verdict: body.Verdict, Reason: body.Reason,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) getAssistantHistory(c *fiber.Ctx) error {
	userID, _ := assistantScope(c)
	// Кламп [1..100]: limit=0/отрицательный/огромный не должен ни ронять
	// выборку, ни выгружать всю историю разом.
	limit := min(max(c.QueryInt("limit", assistantHistoryDefaultLimit), 1), assistantHistoryMaxLimit)
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
			t := time.UnixMilli(ms).UTC()
			before = &t
		} else if t, err := time.Parse(time.RFC3339, raw); err == nil {
			before = &t
		}
	}
	resp, err := h.eps.GetAssistantHistory(c.Context(), endpoint.GetAssistantHistoryRequest{
		UserID: userID, Limit: limit, Before: before,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(dto.NewAssistantMessages(resp.([]domain.AssistantMessage)))
}

// ── Личные ИИ-настройки (/api/ai/my-settings) ─────────────────────

// getMySettings — свой ключ ассистента: маска, модель и тумблер. Прав тут не
// проверяем — человек всегда смотрит СВОИ настройки (в отличие от компанийных
// ai-settings, которые правит только администратор компании).
func (h *handlers) getMySettings(c *fiber.Ctx) error {
	userID, _ := assistantScope(c)
	resp, err := h.eps.GetMySettings(c.Context(), endpoint.MySettingsRequest{UserID: userID})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) updateMySettings(c *fiber.Ctx) error {
	userID, _ := assistantScope(c)
	var body struct {
		Enabled   *bool   `json:"enabled"`
		APIKey    *string `json:"api_key"`
		ClearKey  bool    `json:"clear_key"`
		ModelChat *string `json:"model_chat"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "details": fiber.Map{"_schema": []string{"Invalid input type."}},
		})
	}
	resp, err := h.eps.UpdateMySettings(c.Context(), endpoint.UpdateMySettingsRequest{
		UserID: userID,
		Update: dto.MyAiSettingsUpdate{
			Enabled:   body.Enabled,
			APIKey:    body.APIKey,
			ClearKey:  body.ClearKey,
			ModelChat: body.ModelChat,
		},
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// testMySettings — реальная проверка личного ключа одним tiny-chat.
func (h *handlers) testMySettings(c *fiber.Ctx) error {
	userID, _ := assistantScope(c)
	resp, err := h.eps.TestMySettings(c.Context(), endpoint.MySettingsRequest{UserID: userID})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

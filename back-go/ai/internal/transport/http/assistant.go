// Package http — REST /api/ai/assistant/*: деловой ИИ-ассистент. Доступен
// любому авторизованному пользователю АКТИВНОЙ компании (та же company-scope
// логика, что у tv-fact) — ассистент не привязан к роли, это Q&A-инструмент.
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

// assistantScope — (userID, companyID) из токена; companyID==nil (нет
// активной компании — супер-админ либо пользователь без компаний) → 400.
func assistantScope(c *fiber.Ctx) (userID, companyID int64, err error) {
	user := currentUser(c)
	if user == nil || user.CompanyID == nil {
		return 0, 0, fiber.NewError(fiber.StatusBadRequest, "Требуется активная компания")
	}
	return user.ID, *user.CompanyID, nil
}

func (h *handlers) sendAssistantMessage(c *fiber.Ctx) error {
	userID, companyID, err := assistantScope(c)
	if err != nil {
		return scopeBadRequest(c, err.Error())
	}
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
	userID, companyID, err := assistantScope(c)
	if err != nil {
		return scopeBadRequest(c, err.Error())
	}
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
		UserID: userID, CompanyID: companyID,
		MessageID: body.MessageID, Verdict: body.Verdict, Reason: body.Reason,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) getAssistantHistory(c *fiber.Ctx) error {
	userID, companyID, err := assistantScope(c)
	if err != nil {
		return scopeBadRequest(c, err.Error())
	}
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
		UserID: userID, CompanyID: companyID, Limit: limit, Before: before,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(dto.NewAssistantMessages(resp.([]domain.AssistantMessage)))
}

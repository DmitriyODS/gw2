// Package http — REST /api/ai/text-tools: ИИ-инструменты текста заметок.
// Работают на ключе КОМПАНИИ (в отличие от ассистента, у которого ключ личный),
// поэтому активная компания обязательна; компания без AI → 409 AI_DISABLED
// из сервиса.
package http

import (
	"encoding/json"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/endpoint"
)

// textToolsScope — активная компания сессии; ok=false → 400 (без компании
// компанийного ключа взять неоткуда).
func textToolsScope(c *fiber.Ctx) (int64, bool) {
	user := currentUser(c)
	if user == nil || user.CompanyID == nil {
		return 0, false
	}
	return *user.CompanyID, true
}

func (h *handlers) transformText(c *fiber.Ctx) error {
	var body struct {
		Action string `json:"action"`
		Style  string `json:"style"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil ||
		strings.TrimSpace(body.Action) == "" || strings.TrimSpace(body.Text) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "details": fiber.Map{"text": []string{"Missing data for required field."}},
		})
	}
	resp, err := h.eps.TransformText(c.Context(), endpoint.TransformTextRequest{
		UserID: pasetoauth.UserID(c), Action: body.Action, Style: body.Style, Text: body.Text,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"text": resp.(string)})
}

// proofread — корректура орфографии/пунктуации всей заметки: массив текстовых
// сегментов → исправленный массив той же длины (клиент подменяет узлы по индексу).
func (h *handlers) proofread(c *fiber.Ctx) error {
	var body struct {
		Segments []string `json:"segments"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || len(body.Segments) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "details": fiber.Map{"segments": []string{"Missing data for required field."}},
		})
	}
	resp, err := h.eps.Proofread(c.Context(), endpoint.ProofreadRequest{
		UserID: pasetoauth.UserID(c), Segments: body.Segments,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"segments": resp.([]string)})
}

package http

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type handlers struct {
	eps endpoint.Endpoints
	log *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...} с её
// HTTP-статусом; прочее — 500, как Flask-обработчик ошибок.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

// validationError — форма marshmallow ValidationError: message — словарь
// {поле: [тексты]}, как возвращал Flask.
func validationError(c *fiber.Ctx, field, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":   "VALIDATION_ERROR",
		"message": fiber.Map{field: []string{msg}},
	})
}

// parseBody — request.get_json(silent=True) or {}: невалидный JSON не
// ошибка, просто пустое тело (required-поля отвалятся на валидации).
func parseBody(c *fiber.Ctx, out any) {
	_ = json.Unmarshal(c.Body(), out)
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func pathID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("id")
	return int64(id)
}

func scope(c *fiber.Ctx) endpoint.Scope {
	companyID, _ := c.Locals(localCompanyID).(int64)
	level := 0
	if info := pasetoauth.Current(c); info != nil {
		level = info.RoleLevel
	}
	return endpoint.Scope{
		UserID:    pasetoauth.UserID(c),
		CompanyID: companyID,
		UserLevel: level,
	}
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// ───────────────────────────── лента ───────────────────────────────

func (h *handlers) getFeed(c *fiber.Ctx) error {
	beforeID, _ := parseInt64(c.Query("before_id", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "0"))
	resp, err := h.eps.GetFeed(c.Context(), endpoint.GetFeedRequest{
		Scope: scope(c), BeforeID: beforeID, Limit: limit,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) toggleReaction(c *fiber.Ctx) error {
	var body struct {
		Emoji *string `json:"emoji"`
	}
	parseBody(c, &body)
	if body.Emoji == nil || *body.Emoji == "" {
		return validationError(c, "emoji", "Обязательное поле")
	}
	resp, err := h.eps.ToggleReaction(c.Context(), endpoint.ToggleReactionRequest{
		EventRequest: endpoint.EventRequest{Scope: scope(c), EventID: pathID(c)},
		Emoji:        *body.Emoji,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) listComments(c *fiber.Ctx) error {
	resp, err := h.eps.ListComments(c.Context(), endpoint.EventRequest{
		Scope: scope(c), EventID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) addComment(c *fiber.Ctx) error {
	var body struct {
		Text      *string `json:"text"`
		ReplyToID *int64  `json:"reply_to_id"`
	}
	parseBody(c, &body)
	if body.Text == nil {
		return validationError(c, "text", "Обязательное поле")
	}
	if n := runeLen(*body.Text); n < 1 || n > 2000 {
		return validationError(c, "text", "Длина должна быть от 1 до 2000 символов")
	}
	resp, err := h.eps.AddComment(c.Context(), endpoint.AddCommentRequest{
		EventRequest: endpoint.EventRequest{Scope: scope(c), EventID: pathID(c)},
		Text:         *body.Text,
		ReplyToID:    body.ReplyToID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) deleteComment(c *fiber.Ctx) error {
	_, err := h.eps.DeleteComment(c.Context(), endpoint.DeleteCommentRequest{
		Scope: scope(c), CommentID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Комментарий удалён"})
}

// ─────────────────────── кудосы, live, заряды ──────────────────────

func (h *handlers) sendKudos(c *fiber.Ctx) error {
	var body struct {
		ToUserID *int64  `json:"to_user_id"`
		Text     *string `json:"text"`
	}
	parseBody(c, &body)
	if body.ToUserID == nil {
		return validationError(c, "to_user_id", "Обязательное поле")
	}
	if body.Text == nil {
		return validationError(c, "text", "Обязательное поле")
	}
	if n := runeLen(*body.Text); n < 1 || n > 500 {
		return validationError(c, "text", "Длина должна быть от 1 до 500 символов")
	}
	_, err := h.eps.SendKudos(c.Context(), endpoint.KudosRequest{
		Scope: scope(c), ToUserID: *body.ToUserID, Text: *body.Text,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Кудос отправлен"})
}

func (h *handlers) getLive(c *fiber.Ctx) error {
	resp, err := h.eps.GetLive(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sendZap(c *fiber.Ctx) error {
	var body struct {
		ToUserID *int64 `json:"to_user_id"`
	}
	parseBody(c, &body)
	if body.ToUserID == nil {
		return validationError(c, "to_user_id", "Обязательное поле")
	}
	resp, err := h.eps.SendZap(c.Context(), endpoint.ZapRequest{
		Scope: scope(c), ToUserID: *body.ToUserID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ───────────────────────────── питомец ─────────────────────────────

func (h *handlers) getMyPet(c *fiber.Ctx) error {
	resp, err := h.eps.GetMyPet(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) feedPet(c *fiber.Ctx) error {
	resp, err := h.eps.FeedPet(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) renamePet(c *fiber.Ctx) error {
	var body struct {
		Name *string `json:"name"`
	}
	parseBody(c, &body)
	if body.Name == nil {
		return validationError(c, "name", "Обязательное поле")
	}
	if n := runeLen(*body.Name); n < 1 || n > 50 {
		return validationError(c, "name", "Длина должна быть от 1 до 50 символов")
	}
	resp, err := h.eps.RenamePet(c.Context(), endpoint.NameRequest{
		Scope: scope(c), Name: *body.Name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) equipItem(c *fiber.Ctx) error {
	var body struct {
		Item *string `json:"item"`
	}
	parseBody(c, &body)
	if body.Item != nil && runeLen(*body.Item) > 32 {
		return validationError(c, "item", "Длина не должна превышать 32 символа")
	}
	resp, err := h.eps.EquipItem(c.Context(), endpoint.EquipRequest{
		Scope: scope(c), Item: body.Item,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getShop(c *fiber.Ctx) error {
	resp, err := h.eps.GetShop(c.Context(), nil)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func itemRequest(c *fiber.Ctx, field string) (*endpoint.ItemRequest, error) {
	var body map[string]any
	parseBody(c, &body)
	raw, _ := body[field].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, validationError(c, field, "Обязательное поле")
	}
	maxLen := 32
	if field == "species" {
		maxLen = 24
	}
	if runeLen(raw) > maxLen {
		return nil, validationError(c, field,
			"Длина не должна превышать "+strconv.Itoa(maxLen)+" символа")
	}
	return &endpoint.ItemRequest{Scope: scope(c), Item: raw}, nil
}

func (h *handlers) buyItem(c *fiber.Ctx) error {
	req, verr := itemRequest(c, "item")
	if req == nil {
		return verr
	}
	resp, err := h.eps.BuyItem(c.Context(), *req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) buySpecies(c *fiber.Ctx) error {
	req, verr := itemRequest(c, "species")
	if req == nil {
		return verr
	}
	resp, err := h.eps.BuySpecies(c.Context(), *req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) switchSpecies(c *fiber.Ctx) error {
	req, verr := itemRequest(c, "species")
	if req == nil {
		return verr
	}
	resp, err := h.eps.SwitchSpecies(c.Context(), *req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) claimQuest(c *fiber.Ctx) error {
	resp, err := h.eps.ClaimQuest(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ─────────────────────────── зоопарк и рейд ────────────────────────

func (h *handlers) getZoo(c *fiber.Ctx) error {
	resp, err := h.eps.GetZoo(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) strokePet(c *fiber.Ctx) error {
	targetID, _ := c.ParamsInt("user_id")
	resp, err := h.eps.StrokePet(c.Context(), endpoint.StrokeRequest{
		Scope: scope(c), TargetUserID: int64(targetID),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getRaid(c *fiber.Ctx) error {
	resp, err := h.eps.GetRaid(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ──────────────────── wrapped, брифинг, ТВ ─────────────────────────

func (h *handlers) getWrapped(c *fiber.Ctx) error {
	resp, err := h.eps.GetWrapped(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) shareWrapped(c *fiber.Ctx) error {
	_, err := h.eps.ShareWrapped(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Итог недели опубликован"})
}

func (h *handlers) morning(c *fiber.Ctx) error {
	resp, err := h.eps.Morning(c.Context(), endpoint.MorningRequest{
		Scope: scope(c), Part: strings.TrimSpace(c.Query("part", "morning")),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) grooveTV(c *fiber.Ctx) error {
	resp, err := h.eps.GrooveTV(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

package http

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pets/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type handlers struct {
	eps endpoint.Endpoints
	log *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...} с её
// HTTP-статусом; прочее — 500.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

// validationError — форма marshmallow ValidationError: message — словарь
// {поле: [тексты]}.
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
	resp, err := h.eps.GetShop(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getMystery(c *fiber.Ctx) error {
	resp, err := h.eps.GetMystery(c.Context(), scope(c))
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

func (h *handlers) resetSpecies(c *fiber.Ctx) error {
	resp, err := h.eps.ResetSpecies(c.Context(), scope(c))
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

func (h *handlers) startAdventure(c *fiber.Ctx) error {
	resp, err := h.eps.StartAdventure(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ─────────────── престиж / сезонный трек / домик ────────────────────

func (h *handlers) prestigePet(c *fiber.Ctx) error {
	resp, err := h.eps.PrestigePet(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getSeason(c *fiber.Ctx) error {
	resp, err := h.eps.GetSeason(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) claimSeasonReward(c *fiber.Ctx) error {
	var body struct {
		Threshold *int `json:"threshold"`
	}
	parseBody(c, &body)
	if body.Threshold == nil || *body.Threshold <= 0 {
		return validationError(c, "threshold", "Обязательное поле")
	}
	resp, err := h.eps.ClaimSeasonReward(c.Context(), endpoint.SeasonClaimRequest{
		Scope: scope(c), Threshold: *body.Threshold,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getHouse(c *fiber.Ctx) error {
	resp, err := h.eps.GetHouse(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) buyHouseDecor(c *fiber.Ctx) error {
	req, verr := itemRequest(c, "item")
	if req == nil {
		return verr
	}
	resp, err := h.eps.BuyHouseDecor(c.Context(), *req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) arrangeHouse(c *fiber.Ctx) error {
	var body struct {
		Placed []domain.HouseItem `json:"placed"`
	}
	parseBody(c, &body)
	if body.Placed == nil {
		return validationError(c, "placed", "Обязательное поле")
	}
	for _, item := range body.Placed {
		if item.Key == "" || runeLen(item.Key) > 32 {
			return validationError(c, "placed", "Неверный ключ декора")
		}
	}
	resp, err := h.eps.ArrangeHouse(c.Context(), endpoint.ArrangeRequest{
		Scope: scope(c), Placed: body.Placed,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ────────────────── прогулка / лечение / поглаживание ──────────────

func (h *handlers) walkPet(c *fiber.Ctx) error {
	resp, err := h.eps.WalkPet(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) healPet(c *fiber.Ctx) error {
	resp, err := h.eps.HealPet(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) strokePet(c *fiber.Ctx) error {
	ownerID, err := c.ParamsInt("userId")
	if err != nil {
		return validationError(c, "userId", "Неверный идентификатор пользователя")
	}
	resp, err := h.eps.StrokePet(c.Context(), endpoint.StrokeRequest{
		Scope: scope(c), PetOwnerID: int64(ownerID),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ─────────────────────────── зоопарк и рейтинг ─────────────────────

func (h *handlers) getZoo(c *fiber.Ctx) error {
	resp, err := h.eps.GetZoo(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// deleteZooPet — удаление питомца сотрудника; право (администратор компании)
// проверяет сервис по уровню роли из токена.
func (h *handlers) deleteZooPet(c *fiber.Ctx) error {
	targetID, err := c.ParamsInt("userId")
	if err != nil {
		return validationError(c, "userId", "Неверный идентификатор пользователя")
	}
	if _, err := h.eps.DeleteZooPet(c.Context(), endpoint.ZooDeleteRequest{
		Scope: scope(c), TargetUserID: int64(targetID),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) getRating(c *fiber.Ctx) error {
	resp, err := h.eps.GetRating(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getLive(c *fiber.Ctx) error {
	resp, err := h.eps.GetLive(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getActivityLog(c *fiber.Ctx) error {
	resp, err := h.eps.GetActivityLog(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": resp})
}

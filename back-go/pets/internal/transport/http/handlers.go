package http

import (
	"context"
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

func (h *handlers) setHousePetPos(c *fiber.Ctx) error {
	var body struct {
		X *float64 `json:"x"`
		Y *float64 `json:"y"`
	}
	parseBody(c, &body)
	if body.X == nil || body.Y == nil {
		return validationError(c, "x", "Обязательные поля x и y")
	}
	resp, err := h.eps.SetHousePetPos(c.Context(), endpoint.PetPosRequest{
		Scope: scope(c), X: *body.X, Y: *body.Y,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) setHouseTheme(c *fiber.Ctx) error {
	req, verr := itemRequest(c, "theme")
	if req == nil {
		return verr
	}
	resp, err := h.eps.SetHouseTheme(c.Context(), *req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ──────── прогулка / лечение / сон / купание / поглаживание ────────

func (h *handlers) sleepPet(c *fiber.Ctx) error {
	resp, err := h.eps.SleepPet(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) bathPet(c *fiber.Ctx) error {
	resp, err := h.eps.BathPet(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

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

// ─────────────────────────── кудо-банк ─────────────────────────────

func (h *handlers) getBank(c *fiber.Ctx) error {
	resp, err := h.eps.GetBank(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getBankLedger(c *fiber.Ctx) error {
	beforeID, _ := parseInt64(c.Query("before_id", "0"))
	resp, err := h.eps.GetBankLedger(c.Context(), endpoint.LedgerRequest{
		Scope: scope(c), BeforeID: beforeID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) transferKudos(c *fiber.Ctx) error {
	var body struct {
		ToUserID *int64  `json:"to_user_id"`
		Amount   *int    `json:"amount"`
		Comment  *string `json:"comment"`
	}
	parseBody(c, &body)
	if body.ToUserID == nil || *body.ToUserID <= 0 {
		return validationError(c, "to_user_id", "Обязательное поле")
	}
	if body.Amount == nil || *body.Amount <= 0 {
		return validationError(c, "amount", "Сумма должна быть положительной")
	}
	comment := ""
	if body.Comment != nil {
		comment = strings.TrimSpace(*body.Comment)
	}
	if runeLen(comment) > domain.TransferCommentMax {
		return validationError(c, "comment",
			"Длина не должна превышать "+strconv.Itoa(domain.TransferCommentMax)+" символов")
	}
	resp, err := h.eps.TransferKudos(c.Context(), endpoint.TransferRequest{
		Scope: scope(c), ToUserID: *body.ToUserID, Amount: *body.Amount, Comment: comment,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// bankAmount — общий разбор тела {amount} банковских операций.
func (h *handlers) bankAmount(c *fiber.Ctx, ep func(context.Context, any) (any, error)) error {
	var body struct {
		Amount *int `json:"amount"`
	}
	parseBody(c, &body)
	if body.Amount == nil || *body.Amount <= 0 {
		return validationError(c, "amount", "Сумма должна быть положительной")
	}
	resp, err := ep(c.Context(), endpoint.BankAmountRequest{Scope: scope(c), Amount: *body.Amount})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) bankDeposit(c *fiber.Ctx) error  { return h.bankAmount(c, h.eps.BankDeposit) }
func (h *handlers) bankWithdraw(c *fiber.Ctx) error { return h.bankAmount(c, h.eps.BankWithdraw) }
func (h *handlers) bankTakeLoan(c *fiber.Ctx) error { return h.bankAmount(c, h.eps.BankTakeLoan) }
func (h *handlers) bankRepayLoan(c *fiber.Ctx) error {
	return h.bankAmount(c, h.eps.BankRepayLoan)
}

func (h *handlers) getBankStats(c *fiber.Ctx) error {
	resp, err := h.eps.GetBankStats(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── копилки-цели ────────────────────────────────────────────────────

func (h *handlers) createGoal(c *fiber.Ctx) error {
	var body struct {
		Title  *string `json:"title"`
		Emoji  *string `json:"emoji"`
		Target *int    `json:"target"`
	}
	parseBody(c, &body)
	if body.Title == nil || strings.TrimSpace(*body.Title) == "" {
		return validationError(c, "title", "Обязательное поле")
	}
	if body.Target == nil || *body.Target <= 0 {
		return validationError(c, "target", "Цель должна быть положительной")
	}
	emoji := ""
	if body.Emoji != nil {
		emoji = *body.Emoji
	}
	resp, err := h.eps.CreateGoal(c.Context(), endpoint.GoalCreateRequest{
		Scope: scope(c), Title: *body.Title, Emoji: emoji, Target: *body.Target,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// goalAmount — общий разбор {amount} операций над копилкой (:id из пути).
func (h *handlers) goalAmount(c *fiber.Ctx, ep func(context.Context, any) (any, error)) error {
	goalID, _ := c.ParamsInt("id")
	var body struct {
		Amount *int `json:"amount"`
	}
	parseBody(c, &body)
	if body.Amount == nil || *body.Amount <= 0 {
		return validationError(c, "amount", "Сумма должна быть положительной")
	}
	resp, err := ep(c.Context(), endpoint.GoalAmountRequest{
		Scope: scope(c), GoalID: int64(goalID), Amount: *body.Amount,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) goalDeposit(c *fiber.Ctx) error  { return h.goalAmount(c, h.eps.GoalDeposit) }
func (h *handlers) goalWithdraw(c *fiber.Ctx) error { return h.goalAmount(c, h.eps.GoalWithdraw) }

func (h *handlers) deleteGoal(c *fiber.Ctx) error {
	goalID, _ := c.ParamsInt("id")
	resp, err := h.eps.DeleteGoal(c.Context(), endpoint.GoalRequest{
		Scope: scope(c), GoalID: int64(goalID),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── благотворительные сборы ─────────────────────────────────────────

func (h *handlers) createFund(c *fiber.Ctx) error {
	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Emoji       *string `json:"emoji"`
		Target      *int    `json:"target"`
	}
	parseBody(c, &body)
	if body.Title == nil || strings.TrimSpace(*body.Title) == "" {
		return validationError(c, "title", "Обязательное поле")
	}
	if body.Target == nil || *body.Target <= 0 {
		return validationError(c, "target", "Цель должна быть положительной")
	}
	description, emoji := "", ""
	if body.Description != nil {
		description = *body.Description
	}
	if body.Emoji != nil {
		emoji = *body.Emoji
	}
	resp, err := h.eps.CreateFund(c.Context(), endpoint.FundCreateRequest{
		Scope: scope(c), Title: *body.Title, Description: description,
		Emoji: emoji, Target: *body.Target,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) donateFund(c *fiber.Ctx) error {
	fundID, _ := c.ParamsInt("id")
	var body struct {
		Amount *int `json:"amount"`
	}
	parseBody(c, &body)
	if body.Amount == nil || *body.Amount <= 0 {
		return validationError(c, "amount", "Сумма должна быть положительной")
	}
	resp, err := h.eps.DonateFund(c.Context(), endpoint.FundAmountRequest{
		Scope: scope(c), FundID: int64(fundID), Amount: *body.Amount,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) closeFund(c *fiber.Ctx) error {
	fundID, _ := c.ParamsInt("id")
	resp, err := h.eps.CloseFund(c.Context(), endpoint.FundRequest{
		Scope: scope(c), FundID: int64(fundID),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── приключение: досрочный возврат ──────────────────────────────────

func (h *handlers) recallAdventure(c *fiber.Ctx) error {
	resp, err := h.eps.RecallAdventure(c.Context(), scope(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

package http

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

type handlers struct {
	eps endpoint.Endpoints
	log *slog.Logger
}

// respondError — бизнес-ошибка в форме {"error": code, "message": ...} с её
// HTTP-статусом; пустой message опускается (jsonify({"error": "NOT_FOUND"})
// во Flask); прочее — 500, как Flask-обработчик ошибок.
func (h *handlers) respondError(c *fiber.Ctx, err error) error {
	return apierror.Respond(c, err, h.log)
}

func (h *handlers) settingsRequest(c *fiber.Ctx) endpoint.SettingsRequest {
	companyID, _ := c.ParamsInt("companyId")
	return endpoint.SettingsRequest{Actor: currentUser(c), CompanyID: int64(companyID)}
}

func (h *handlers) getSettings(c *fiber.Ctx) error {
	resp, err := h.eps.GetSettings(c.Context(), h.settingsRequest(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) updateSettings(c *fiber.Ctx) error {
	req := h.settingsRequest(c)
	upd, details := parseSettingsUpdate(c.Body())
	if len(details) > 0 {
		// Порядок как во Flask: сначала резолв компании и доступ (404/403),
		// валидация тела — после.
		if _, err := h.eps.GetSettings(c.Context(), req); err != nil {
			return h.respondError(c, err)
		}
		// Форма marshmallow ValidationError из Flask:
		// {"error": "VALIDATION", "details": {поле: [тексты]}}.
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "details": details,
		})
	}
	resp, err := h.eps.UpdateSettings(c.Context(), endpoint.UpdateSettingsRequest{
		Actor: req.Actor, CompanyID: req.CompanyID, Update: upd,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) testSettings(c *fiber.Ctx) error {
	resp, err := h.eps.TestSettings(c.Context(), h.settingsRequest(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) indexingStatus(c *fiber.Ctx) error {
	resp, err := h.eps.IndexingStatus(c.Context(), h.settingsRequest(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) reindexTasks(c *fiber.Ctx) error {
	resp, err := h.eps.StartReindex(c.Context(), h.settingsRequest(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusAccepted).JSON(resp)
}

// tvFact — GET /api/ai/tv-fact: текущий факт дня для ТВ-табло; AI выключен /
// факт не сгенерён → null с 200 OK (фронт молча падает на фолбэк-слайд).
// Company-scope как @require_company_scope во Flask: обычный пользователь —
// всегда своя компания, Администратор системы — ?company_id=.
func (h *handlers) tvFact(c *fiber.Ctx) error {
	user := currentUser(c)
	var companyID int64
	if user != nil && user.CompanyID != nil {
		companyID = *user.CompanyID
	} else {
		raw := c.Query("company_id")
		if raw == "" {
			return scopeBadRequest(c, "Требуется указать company_id")
		}
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return scopeBadRequest(c, "Неверный company_id")
		}
		companyID = v
	}
	resp, err := h.eps.GetTVFact(c.Context(), companyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// scopeBadRequest — форма flask abort(400, description=...).
func scopeBadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "BAD_REQUEST", "message": message,
	})
}

// ── Валидация PUT-тела (формы AiSettingsUpdateSchema/marshmallow) ─

// parseSettingsUpdate — разбор и валидация PUT-тела. details непустой —
// 400 VALIDATION с сообщениями в формате marshmallow {поле: [тексты]}.
// Невалидный/пустой JSON трактуем как {} (request.get_json() or {}).
func parseSettingsUpdate(body []byte) (dto.AiSettingsUpdate, map[string][]string) {
	var upd dto.AiSettingsUpdate
	details := map[string][]string{}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)

	for field, value := range raw {
		switch field {
		case "enabled":
			if b, ok := asBool(value); ok {
				upd.Enabled = &b
			} else {
				details[field] = append(details[field], "Not a valid boolean.")
			}
		case "clear_key":
			if b, ok := asBool(value); ok {
				upd.ClearKey = b
			} else {
				details[field] = append(details[field], "Not a valid boolean.")
			}
		case "api_key":
			// allow_none: null = «не менять».
			if string(value) == "null" {
				continue
			}
			s, ok := asString(value)
			switch {
			case !ok:
				details[field] = append(details[field], "Not a valid string.")
			case utf8.RuneCountInString(s) > 512:
				details[field] = append(details[field], "Longer than maximum length 512.")
			default:
				upd.APIKey = &s
			}
		case "model_chat", "model_embedding":
			s, ok := asString(value)
			switch {
			case !ok:
				details[field] = append(details[field], "Not a valid string.")
			case utf8.RuneCountInString(s) < 1 || utf8.RuneCountInString(s) > 64:
				details[field] = append(details[field], "Length must be between 1 and 64.")
			case field == "model_chat":
				upd.ModelChat = &s
			default:
				upd.ModelEmbedding = &s
			}
		default:
			// marshmallow по умолчанию RAISE на неизвестных полях.
			details[field] = append(details[field], "Unknown field.")
		}
	}
	if len(details) > 0 {
		return dto.AiSettingsUpdate{}, details
	}
	return upd, nil
}

func asString(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// asBool — как marshmallow Boolean: bool, truthy/falsy-строки, числа 0/1.
func asBool(raw json.RawMessage) (bool, bool) {
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch strings.ToLower(s) {
		case "t", "true", "1", "on", "y", "yes":
			return true, true
		case "f", "false", "0", "off", "n", "no":
			return false, true
		}
		return false, false
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		switch n {
		case 1:
			return true, true
		case 0:
			return false, true
		}
	}
	return false, false
}

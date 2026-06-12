package http

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
)

// Хендлеры компаний и ролей. Формы запросов/ответов и тексты ошибок
// валидации — байт-в-байт с прежними marshmallow-схемами Flask
// (schemas/company.py): {"error": "VALIDATION_ERROR", "message": {поле: [...]}}.

func (h *handlers) listRoles(c *fiber.Ctx) error {
	resp, err := h.eps.ListRoles(c.Context(), nil)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) listCompanies(c *fiber.Ctx) error {
	resp, err := h.eps.ListCompanies(c.Context(), nil)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getCompany(c *fiber.Ctx) error {
	resp, err := h.eps.GetCompany(c.Context(), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createCompany(c *fiber.Ctx) error {
	req, details := parseCompanyCreate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.CreateCompany(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateCompany(c *fiber.Ctx) error {
	req, details := parseCompanyUpdate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.UpdateCompany(c.Context(), endpoint.UpdateCompanyEpRequest{
		CompanyID: pathID(c), Body: req,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) toggleCompanyActive(c *fiber.Ctx) error {
	isActive, details := parseToggleActive(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.ToggleCompanyActive(c.Context(), endpoint.ToggleCompanyEpRequest{
		CompanyID: pathID(c), IsActive: isActive,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteCompany(c *fiber.Ctx) error {
	if _, err := h.eps.DeleteCompany(c.Context(), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Компания удалена"})
}

func (h *handlers) getWeekendSettings(c *fiber.Ctx) error {
	resp, err := h.eps.GetWeekendSettings(c.Context(), endpoint.CompanyScopeEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) updateWeekendSettings(c *fiber.Ctx) error {
	days, details := parseWeekendDays(c.Body())
	if details != nil {
		// Порядок как во Flask: сначала 404/403 (компания и доступ),
		// валидация тела — после.
		if _, err := h.eps.GetWeekendSettings(c.Context(), endpoint.CompanyScopeEpRequest{
			Actor: currentUser(c), CompanyID: pathID(c),
		}); err != nil {
			return h.respondError(c, err)
		}
		return validationError(c, details)
	}
	resp, err := h.eps.UpdateWeekendSettings(c.Context(), endpoint.WeekendEpRequest{
		Actor: currentUser(c), CompanyID: pathID(c), Days: days,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── Валидация тел (формы marshmallow-схем companies) ─────────────

func validationError(c *fiber.Ctx, details map[string]any) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "VALIDATION_ERROR", "message": details,
	})
}

func addFieldError(details map[string]any, field, message string) {
	msgs, _ := details[field].([]string)
	details[field] = append(msgs, message)
}

func asJSONString(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// asJSONBool — как marshmallow Boolean: bool, truthy/falsy-строки, числа 0/1.
func asJSONBool(raw json.RawMessage) (bool, bool) {
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

// asJSONInt — как marshmallow Integer: число без дробной части или
// строка-число; bool — невалиден.
func asJSONInt(raw json.RawMessage) (int64, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "true" || trimmed == "false" {
		return 0, false
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		if n != float64(int64(n)) {
			return 0, false
		}
		return int64(n), true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

// parseWeekendList — weekend_days: List(Int 0..6, max 7) — формы ошибок
// marshmallow: по индексам элементов либо общие для всего поля.
func parseWeekendList(raw json.RawMessage) ([]int, any) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, []string{"Not a valid list."}
	}
	itemErrors := map[string][]string{}
	out := make([]int, 0, len(items))
	for i, item := range items {
		v, ok := asJSONInt(item)
		switch {
		case !ok:
			itemErrors[strconv.Itoa(i)] = []string{"Not a valid integer."}
		case v < 0 || v > 6:
			itemErrors[strconv.Itoa(i)] = []string{
				"Must be greater than or equal to 0 and less than or equal to 6."}
		default:
			out = append(out, int(v))
		}
	}
	if len(itemErrors) > 0 {
		return nil, itemErrors
	}
	if len(out) > 7 {
		return nil, []string{"Longer than maximum length 7."}
	}
	return out, nil
}

func parseWeekendDays(body []byte) ([]int, map[string]any) {
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)

	details := map[string]any{}
	wd, ok := raw["weekend_days"]
	if !ok {
		details["weekend_days"] = []string{"Missing data for required field."}
	}
	for field := range raw {
		if field != "weekend_days" {
			addFieldError(details, field, "Unknown field.")
		}
	}
	var days []int
	if ok {
		parsed, errs := parseWeekendList(wd)
		if errs != nil {
			details["weekend_days"] = errs
		} else {
			days = parsed
		}
	}
	if len(details) > 0 {
		return nil, details
	}
	return days, nil
}

func parseToggleActive(body []byte) (bool, map[string]any) {
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)

	details := map[string]any{}
	var isActive bool
	if v, ok := raw["is_active"]; ok {
		if b, ok := asJSONBool(v); ok {
			isActive = b
		} else {
			details["is_active"] = []string{"Not a valid boolean."}
		}
	} else {
		details["is_active"] = []string{"Missing data for required field."}
	}
	for field := range raw {
		if field != "is_active" {
			addFieldError(details, field, "Unknown field.")
		}
	}
	if len(details) > 0 {
		return false, details
	}
	return isActive, nil
}

// parseCompanySettings — Nested CompanySettingsSchema: при partial=false
// отсутствующие ключи получают load_default'ы схемы.
func parseCompanySettings(raw json.RawMessage, partial bool) (map[string]any, map[string]any) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, map[string]any{"_schema": []string{"Invalid input type."}}
	}

	errs := map[string]any{}
	out := map[string]any{}
	for field, value := range fields {
		switch field {
		case "uses_yougile", "uses_stages", "uses_calls":
			if b, ok := asJSONBool(value); ok {
				out[field] = b
			} else {
				errs[field] = []string{"Not a valid boolean."}
			}
		case "weekend_days":
			days, dErrs := parseWeekendList(value)
			if dErrs != nil {
				errs[field] = dErrs
			} else {
				out[field] = days
			}
		default:
			errs[field] = []string{"Unknown field."}
		}
	}
	if !partial {
		if _, ok := fields["uses_yougile"]; !ok {
			out["uses_yougile"] = false
		}
		if _, ok := fields["uses_stages"]; !ok {
			out["uses_stages"] = false
		}
		if _, ok := fields["uses_calls"]; !ok {
			out["uses_calls"] = true
		}
		if _, ok := fields["weekend_days"]; !ok {
			out["weekend_days"] = []int{5, 6}
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return out, nil
}

func parseCompanyCreate(body []byte) (dto.CompanyCreate, map[string]any) {
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)

	details := map[string]any{}
	req := dto.CompanyCreate{IsActive: true}

	if value, ok := raw["name"]; ok {
		s, ok := asJSONString(value)
		switch {
		case !ok:
			details["name"] = []string{"Not a valid string."}
		case utf8.RuneCountInString(s) < 1:
			details["name"] = []string{"Shorter than minimum length 1."}
		case utf8.RuneCountInString(s) > 255:
			details["name"] = []string{"Longer than maximum length 255."}
		default:
			req.Name = s
		}
	} else {
		details["name"] = []string{"Missing data for required field."}
	}

	for field, value := range raw {
		switch field {
		case "name":
			// разобран выше
		case "description":
			if string(value) == "null" {
				req.Description = nil
			} else if s, ok := asJSONString(value); ok {
				req.Description = &s
			} else {
				details[field] = []string{"Not a valid string."}
			}
		case "director_id":
			if string(value) == "null" {
				req.DirectorID = nil
			} else if v, ok := asJSONInt(value); ok {
				req.DirectorID = &v
			} else {
				details[field] = []string{"Not a valid integer."}
			}
		case "is_active":
			if b, ok := asJSONBool(value); ok {
				req.IsActive = b
			} else {
				details[field] = []string{"Not a valid boolean."}
			}
		case "settings":
			settings, errs := parseCompanySettings(value, false)
			if errs != nil {
				details[field] = errs
			} else {
				req.Settings = settings
			}
		default:
			addFieldError(details, field, "Unknown field.")
		}
	}
	if req.Settings == nil && details["settings"] == nil {
		// load_default=dict + load-дефолты вложенной схемы.
		req.Settings = map[string]any{
			"uses_yougile": false, "uses_stages": false, "uses_calls": true,
			"weekend_days": []int{5, 6},
		}
	}
	if len(details) > 0 {
		return dto.CompanyCreate{}, details
	}
	return req, nil
}

func parseCompanyUpdate(body []byte) (dto.CompanyUpdate, map[string]any) {
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)

	details := map[string]any{}
	var req dto.CompanyUpdate

	for field, value := range raw {
		switch field {
		case "name":
			// без allow_none: null — невалидная строка.
			s, ok := asJSONString(value)
			switch {
			case !ok:
				details[field] = []string{"Not a valid string."}
			case utf8.RuneCountInString(s) < 1:
				details[field] = []string{"Shorter than minimum length 1."}
			case utf8.RuneCountInString(s) > 255:
				details[field] = []string{"Longer than maximum length 255."}
			default:
				req.Name = &s
			}
		case "description":
			req.DescriptionSet = true
			if string(value) == "null" {
				req.Description = nil
			} else if s, ok := asJSONString(value); ok {
				req.Description = &s
			} else {
				req.DescriptionSet = false
				details[field] = []string{"Not a valid string."}
			}
		case "director_id":
			req.DirectorSet = true
			if string(value) == "null" {
				req.DirectorID = nil
			} else if v, ok := asJSONInt(value); ok {
				req.DirectorID = &v
			} else {
				req.DirectorSet = false
				details[field] = []string{"Not a valid integer."}
			}
		case "is_active":
			if b, ok := asJSONBool(value); ok {
				req.IsActive = &b
			} else {
				details[field] = []string{"Not a valid boolean."}
			}
		case "settings":
			settings, errs := parseCompanySettings(value, true)
			if errs != nil {
				details[field] = errs
			} else {
				req.Settings = settings
				req.SettingsSet = true
			}
		default:
			addFieldError(details, field, "Unknown field.")
		}
	}
	if len(details) > 0 {
		return dto.CompanyUpdate{}, details
	}
	return req, nil
}

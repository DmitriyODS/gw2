package http

import (
	"encoding/json"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/marshform"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// Формы ошибок валидации — {"error": "VALIDATION_ERROR", "message": {...}}
// с дефолтными текстами marshmallow (как ValidationError.messages во Flask).

func validationError(c *fiber.Ctx, details map[string]any) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "VALIDATION_ERROR", "message": details,
	})
}

func validationMsg(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "VALIDATION_ERROR", "message": message,
	})
}

func rawBody(body []byte) map[string]json.RawMessage {
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	return raw
}

// parseISODateTime — datetime.fromisoformat: ISO с зоной или naive
// (naive трактуется как UTC, как replace(tzinfo=utc) во Flask-статистике).
func parseISODateTime(s string) (time.Time, bool) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseDateOnly — fields.Date: строго YYYY-MM-DD.
func parseDateOnly(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ── Задачи ───────────────────────────────────────────────────────

func validateTaskName(value json.RawMessage, details map[string]any) *string {
	s, ok := marshform.AsString(value)
	switch {
	case !ok:
		details["name"] = []string{marshform.MsgNotString}
	case utf8.RuneCountInString(s) < 1:
		details["name"] = []string{marshform.LengthMin(1)}
	case utf8.RuneCountInString(s) > 500:
		details["name"] = []string{marshform.LengthMax(500)}
	default:
		return &s
	}
	return nil
}

func parseTaskCreate(body []byte) (dto.TaskCreate, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var req dto.TaskCreate

	if value, ok := raw["name"]; ok {
		if name := validateTaskName(value, details); name != nil {
			req.Name = *name
		}
	} else {
		details["name"] = []string{marshform.MsgRequired}
	}
	if value, ok := raw["department_id"]; ok {
		if v, ok := marshform.AsInt(value); ok {
			req.DepartmentID = v
		} else {
			details["department_id"] = []string{marshform.MsgNotInteger}
		}
	} else {
		details["department_id"] = []string{marshform.MsgRequired}
	}

	for field, value := range raw {
		switch field {
		case "name", "department_id":
			// разобраны выше
		case "link_yougile":
			if marshform.IsNull(value) {
				continue
			}
			s, ok := marshform.AsString(value)
			switch {
			case !ok:
				details[field] = []string{marshform.MsgNotString}
			case utf8.RuneCountInString(s) > 2000:
				details[field] = []string{marshform.LengthMax(2000)}
			default:
				req.LinkYougile = &s
			}
		case "received_at", "deadline":
			if marshform.IsNull(value) {
				continue
			}
			s, ok := marshform.AsString(value)
			if !ok {
				details[field] = []string{marshform.MsgNotDate}
				continue
			}
			t, ok := parseDateOnly(s)
			if !ok {
				details[field] = []string{marshform.MsgNotDate}
				continue
			}
			if field == "received_at" {
				req.ReceivedAt = &t
			} else {
				req.Deadline = &t
			}
		case "responsible_user_id", "stage_id":
			if marshform.IsNull(value) {
				continue
			}
			v, ok := marshform.AsInt(value)
			if !ok {
				details[field] = []string{marshform.MsgNotInteger}
				continue
			}
			if field == "responsible_user_id" {
				req.ResponsibleUserID = &v
			} else {
				req.StageID = &v
			}
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return dto.TaskCreate{}, details
	}
	return req, nil
}

func parseTaskUpdate(body []byte) (dto.TaskUpdate, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var req dto.TaskUpdate

	for field, value := range raw {
		switch field {
		case "name":
			req.Name = validateTaskName(value, details)
		case "link_yougile":
			req.LinkYougileSet = true
			if marshform.IsNull(value) {
				continue
			}
			s, ok := marshform.AsString(value)
			switch {
			case !ok:
				req.LinkYougileSet = false
				details[field] = []string{marshform.MsgNotString}
			case utf8.RuneCountInString(s) > 2000:
				req.LinkYougileSet = false
				details[field] = []string{marshform.LengthMax(2000)}
			default:
				req.LinkYougile = &s
			}
		case "department_id":
			// без allow_none: null — невалидное целое.
			if v, ok := marshform.AsInt(value); ok {
				req.DepartmentID = &v
			} else {
				details[field] = []string{marshform.MsgNotInteger}
			}
		case "received_at", "deadline":
			set := field == "received_at"
			if marshform.IsNull(value) {
				if set {
					req.ReceivedAtSet = true
				} else {
					req.DeadlineSet = true
				}
				continue
			}
			s, ok := marshform.AsString(value)
			if !ok {
				details[field] = []string{marshform.MsgNotDate}
				continue
			}
			t, tok := parseDateOnly(s)
			if !tok {
				details[field] = []string{marshform.MsgNotDate}
				continue
			}
			if set {
				req.ReceivedAt, req.ReceivedAtSet = &t, true
			} else {
				req.Deadline, req.DeadlineSet = &t, true
			}
		case "responsible_user_id", "stage_id":
			resp := field == "responsible_user_id"
			if marshform.IsNull(value) {
				if resp {
					req.ResponsibleSet = true
				} else {
					req.StageSet = true
				}
				continue
			}
			v, ok := marshform.AsInt(value)
			if !ok {
				details[field] = []string{marshform.MsgNotInteger}
				continue
			}
			if resp {
				req.ResponsibleUserID, req.ResponsibleSet = &v, true
			} else {
				req.StageID, req.StageSet = &v, true
			}
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return dto.TaskUpdate{}, details
	}
	return req, nil
}

// parseColorBody — TaskColorSchema: color allow_none OneOf(TASK_COLORS).
func parseColorBody(body []byte) (*string, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var color *string
	for field, value := range raw {
		if field != "color" {
			details[field] = []string{marshform.MsgUnknownField}
			continue
		}
		if marshform.IsNull(value) {
			continue
		}
		s, ok := marshform.AsString(value)
		switch {
		case !ok:
			details[field] = []string{marshform.MsgNotString}
		case !domain.ValidTaskColor(s):
			details[field] = []string{marshform.OneOf(domain.TaskColors)}
		default:
			color = &s
		}
	}
	if len(details) > 0 {
		return nil, details
	}
	return color, nil
}

// parseNullableIntBody — TaskResponsibleSchema/TaskStageSchema:
// поле required + allow_none.
func parseNullableIntBody(body []byte, field string) (*int64, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var out *int64
	if value, ok := raw[field]; ok {
		if !marshform.IsNull(value) {
			if v, vok := marshform.AsInt(value); vok {
				out = &v
			} else {
				details[field] = []string{marshform.MsgNotInteger}
			}
		}
	} else {
		details[field] = []string{marshform.MsgRequired}
	}
	for f := range raw {
		if f != field {
			details[f] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return nil, details
	}
	return out, nil
}

// ── Юниты ────────────────────────────────────────────────────────

func parseUnitCreate(body []byte) (name string, unitTypeID int64, details map[string]any) {
	raw := rawBody(body)
	details = map[string]any{}
	if value, ok := raw["name"]; ok {
		s, sok := marshform.AsString(value)
		switch {
		case !sok:
			details["name"] = []string{marshform.MsgNotString}
		case utf8.RuneCountInString(s) < 1:
			details["name"] = []string{marshform.LengthMin(1)}
		case utf8.RuneCountInString(s) > 500:
			details["name"] = []string{marshform.LengthMax(500)}
		default:
			name = s
		}
	} else {
		details["name"] = []string{marshform.MsgRequired}
	}
	if value, ok := raw["unit_type_id"]; ok {
		if v, vok := marshform.AsInt(value); vok {
			unitTypeID = v
		} else {
			details["unit_type_id"] = []string{marshform.MsgNotInteger}
		}
	} else {
		details["unit_type_id"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "name" && field != "unit_type_id" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return "", 0, details
	}
	return name, unitTypeID, nil
}

func parseUnitUpdate(body []byte) (dto.UnitUpdate, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var req dto.UnitUpdate

	for field, value := range raw {
		switch field {
		case "name":
			s, ok := marshform.AsString(value)
			switch {
			case !ok:
				details[field] = []string{marshform.MsgNotString}
			case utf8.RuneCountInString(s) < 1:
				details[field] = []string{marshform.LengthMin(1)}
			case utf8.RuneCountInString(s) > 500:
				details[field] = []string{marshform.LengthMax(500)}
			default:
				req.Name = &s
			}
		case "unit_type_id":
			if v, ok := marshform.AsInt(value); ok {
				req.UnitTypeID = &v
			} else {
				details[field] = []string{marshform.MsgNotInteger}
			}
		case "datetime_start":
			s, ok := marshform.AsString(value)
			if !ok {
				details[field] = []string{marshform.MsgNotDateTime}
				continue
			}
			t, tok := parseISODateTime(s)
			if !tok {
				details[field] = []string{marshform.MsgNotDateTime}
				continue
			}
			req.DatetimeStart = &t
		case "datetime_end":
			req.DatetimeEndSet = true
			if marshform.IsNull(value) {
				continue
			}
			s, ok := marshform.AsString(value)
			if !ok {
				req.DatetimeEndSet = false
				details[field] = []string{marshform.MsgNotDateTime}
				continue
			}
			t, tok := parseISODateTime(s)
			if !tok {
				req.DatetimeEndSet = false
				details[field] = []string{marshform.MsgNotDateTime}
				continue
			}
			req.DatetimeEnd = &t
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return dto.UnitUpdate{}, details
	}
	return req, nil
}

// ── Комментарии, справочники, этапы ──────────────────────────────

// parseTextBody — CommentCreate/UpdateSchema: text required 1..10000.
func parseTextBody(body []byte) (string, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var text string
	if value, ok := raw["text"]; ok {
		s, sok := marshform.AsString(value)
		switch {
		case !sok:
			details["text"] = []string{marshform.MsgNotString}
		case utf8.RuneCountInString(s) < 1:
			details["text"] = []string{marshform.LengthMin(1)}
		case utf8.RuneCountInString(s) > 10000:
			details["text"] = []string{marshform.LengthMax(10000)}
		default:
			text = s
		}
	} else {
		details["text"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "text" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return "", details
	}
	return text, nil
}

// parseNameBody — Department/UnitType схемы: name required 1..255.
func parseNameBody(body []byte) (string, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var name string
	if value, ok := raw["name"]; ok {
		s, sok := marshform.AsString(value)
		switch {
		case !sok:
			details["name"] = []string{marshform.MsgNotString}
		case utf8.RuneCountInString(s) < 1:
			details["name"] = []string{marshform.LengthMin(1)}
		case utf8.RuneCountInString(s) > 255:
			details["name"] = []string{marshform.LengthMax(255)}
		default:
			name = s
		}
	} else {
		details["name"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "name" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return "", details
	}
	return name, nil
}

// parseStageCreate — StageCreateSchema: name required, color OneOf
// (load_default "blue").
func parseStageCreate(body []byte) (name, color string, details map[string]any) {
	raw := rawBody(body)
	details = map[string]any{}
	color = "blue"
	if value, ok := raw["name"]; ok {
		s, sok := marshform.AsString(value)
		switch {
		case !sok:
			details["name"] = []string{marshform.MsgNotString}
		case utf8.RuneCountInString(s) < 1:
			details["name"] = []string{marshform.LengthMin(1)}
		case utf8.RuneCountInString(s) > 255:
			details["name"] = []string{marshform.LengthMax(255)}
		default:
			name = s
		}
	} else {
		details["name"] = []string{marshform.MsgRequired}
	}
	for field, value := range raw {
		switch field {
		case "name":
		case "color":
			s, ok := marshform.AsString(value)
			switch {
			case !ok:
				details[field] = []string{marshform.MsgNotString}
			case !domain.ValidTaskColor(s):
				details[field] = []string{marshform.OneOf(domain.TaskColors)}
			default:
				color = s
			}
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return "", "", details
	}
	return name, color, nil
}

func parseStageUpdate(body []byte) (name, color *string, details map[string]any) {
	raw := rawBody(body)
	details = map[string]any{}
	for field, value := range raw {
		switch field {
		case "name":
			s, ok := marshform.AsString(value)
			switch {
			case !ok:
				details[field] = []string{marshform.MsgNotString}
			case utf8.RuneCountInString(s) < 1:
				details[field] = []string{marshform.LengthMin(1)}
			case utf8.RuneCountInString(s) > 255:
				details[field] = []string{marshform.LengthMax(255)}
			default:
				name = &s
			}
		case "color":
			s, ok := marshform.AsString(value)
			switch {
			case !ok:
				details[field] = []string{marshform.MsgNotString}
			case !domain.ValidTaskColor(s):
				details[field] = []string{marshform.OneOf(domain.TaskColors)}
			default:
				color = &s
			}
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return nil, nil, details
	}
	return name, color, nil
}

// parseReorder — StageReorderSchema: ids List(Int) required min 1.
func parseReorder(body []byte) ([]int64, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var ids []int64
	if value, ok := raw["ids"]; ok {
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			details["ids"] = []string{marshform.MsgNotList}
		} else {
			itemErrors := map[string][]string{}
			for i, item := range items {
				v, vok := marshform.AsInt(item)
				if !vok {
					itemErrors[strconv.Itoa(i)] = []string{marshform.MsgNotInteger}
					continue
				}
				ids = append(ids, v)
			}
			switch {
			case len(itemErrors) > 0:
				details["ids"] = itemErrors
			case len(items) < 1:
				details["ids"] = []string{marshform.LengthMin(1)}
			}
		}
	} else {
		details["ids"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "ids" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return nil, details
	}
	return ids, nil
}

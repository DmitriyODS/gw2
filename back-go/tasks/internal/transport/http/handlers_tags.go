package http

import (
	"encoding/json"
	"strconv"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/marshform"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

// Теги задач: справочник компании (CRUD — менеджер, как отделы/этапы) и
// назначение набора тегов задаче (сотрудник). Роуты живут ПОД /api/tasks
// (/api/tasks/tags, /api/tasks/:id/tags) — отдельный префикс потребовал бы
// правок nginx/vite-роутинга.

func (h *handlers) listTags(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	resp, err := h.eps.ListTags(c.Context(), companyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createTag(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, color, details := parseTagCreate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.CreateTag(c.Context(), endpoint.TagCreateRequest{
		CompanyID: companyID, Name: name, Color: color,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateTag(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, color, details := parseTagUpdate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.UpdateTag(c.Context(), endpoint.TagUpdateRequest{
		CompanyID: companyID, TagID: pathID(c), Name: name, Color: color,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteTag(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	if _, err := h.eps.DeleteTag(c.Context(), endpoint.CompanyItemRequest{
		CompanyID: companyID, ItemID: pathID(c),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Тег удалён"})
}

func (h *handlers) setTaskTags(c *fiber.Ctx) error {
	user := currentUser(c)
	tagIDs, details := parseTagIDs(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.SetTaskTags(c.Context(), endpoint.SetTaskTagsRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID, TagIDs: tagIDs,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── Парсеры (формы marshmallow, как parseStage*) ──────────────────

const tagNameMax = 64

func parseTagName(value json.RawMessage, details map[string]any) (string, bool) {
	s, ok := marshform.AsString(value)
	switch {
	case !ok:
		details["name"] = []string{marshform.MsgNotString}
	case utf8.RuneCountInString(s) < 1:
		details["name"] = []string{marshform.LengthMin(1)}
	case utf8.RuneCountInString(s) > tagNameMax:
		details["name"] = []string{marshform.LengthMax(tagNameMax)}
	default:
		return s, true
	}
	return "", false
}

func parseTagCreate(body []byte) (name, color string, details map[string]any) {
	raw := rawBody(body)
	details = map[string]any{}
	color = "blue"
	if value, ok := raw["name"]; ok {
		name, _ = parseTagName(value, details)
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

func parseTagUpdate(body []byte) (name, color *string, details map[string]any) {
	raw := rawBody(body)
	details = map[string]any{}
	for field, value := range raw {
		switch field {
		case "name":
			if s, ok := parseTagName(value, details); ok {
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

// parseTagIDs — тело PUT /api/tasks/:id/tags: {tag_ids: [..]}; пустой список
// валиден (снять все теги).
func parseTagIDs(body []byte) ([]int64, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	ids := []int64{}
	if value, ok := raw["tag_ids"]; ok {
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			details["tag_ids"] = []string{marshform.MsgNotList}
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
			if len(itemErrors) > 0 {
				details["tag_ids"] = itemErrors
			}
		}
	} else {
		details["tag_ids"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "tag_ids" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return nil, details
	}
	return ids, nil
}

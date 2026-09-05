package http

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/service"
)

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

// parseDate — дата дня (YYYY-MM-DD); "" и мусор дают nil.
func parseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse(domain.DateLayout, s)
	if err != nil {
		return nil
	}
	return &t
}

// ── Расписания ───────────────────────────────────────────────────

func (h *handlers) listSchedules(c *fiber.Ctx) error {
	list, err := h.svc.ListSchedules(c.Context(), currentUserID(c), c.Query("tab") == "shared")
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"schedules": list})
}

func (h *handlers) getSchedule(c *fiber.Ctx) error {
	view, err := h.svc.GetSchedule(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(view)
}

func (h *handlers) createSchedule(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		CycleWeeks  int    `json:"cycle_weeks"`
		CycleAnchor string `json:"cycle_anchor"`
		Timezone    string `json:"timezone"`
	}
	parseBody(c, &body)
	if strings.TrimSpace(body.Name) == "" {
		return validationError(c, "Укажите название расписания")
	}
	in := service.ScheduleInput{
		Name: body.Name, CycleWeeks: body.CycleWeeks, Timezone: body.Timezone,
	}
	if anchor := parseDate(body.CycleAnchor); anchor != nil {
		in.CycleAnchor = *anchor
	}
	view, err := h.svc.CreateSchedule(c.Context(), currentUserID(c), in)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(view)
}

func (h *handlers) updateSchedule(c *fiber.Ctx) error {
	var body struct {
		Name        *string  `json:"name"`
		CycleWeeks  *int     `json:"cycle_weeks"`
		CycleAnchor *string  `json:"cycle_anchor"`
		WeekLabels  []string `json:"week_labels"`
		Timezone    *string  `json:"timezone"`
		GapMin      *int     `json:"gap_min"`
	}
	parseBody(c, &body)
	up := service.ScheduleUpdate{
		Name: body.Name, CycleWeeks: body.CycleWeeks, WeekLabels: body.WeekLabels,
		Timezone: body.Timezone, GapMin: body.GapMin,
	}
	if body.CycleAnchor != nil {
		up.CycleAnchor = parseDate(*body.CycleAnchor)
	}
	view, adjusted, err := h.svc.UpdateSchedule(c.Context(), currentUserID(c), pathID(c), up)
	if err != nil {
		return h.respondError(c, err)
	}
	// adjusted — сколько занятий потеряло номера недель за укороченным циклом.
	// Клиент говорит об этом человеку: тихая правка чужих занятий недопустима.
	return c.JSON(fiber.Map{
		"schedule": view.Schedule, "items": view.Items, "can_edit": view.CanEdit,
		"adjusted_items": adjusted,
	})
}

func (h *handlers) deleteSchedule(c *fiber.Ctx) error {
	if err := h.svc.DeleteSchedule(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Структура ────────────────────────────────────────────────────

func (h *handlers) createCategory(c *fiber.Ctx) error {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	parseBody(c, &body)
	cat, err := h.svc.CreateCategory(c.Context(), currentUserID(c), pathID(c),
		service.CategoryInput{Name: body.Name, Color: body.Color})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(cat)
}

func (h *handlers) updateCategory(c *fiber.Ctx) error {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	parseBody(c, &body)
	cat, err := h.svc.UpdateCategory(c.Context(), currentUserID(c), pathID(c),
		paramID(c, "cid"), service.CategoryInput{Name: body.Name, Color: body.Color})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(cat)
}

func (h *handlers) deleteCategory(c *fiber.Ctx) error {
	if err := h.svc.DeleteCategory(c.Context(), currentUserID(c), pathID(c), paramID(c, "cid")); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// fieldBody — поле карточки занятия. Известный id сохраняется: по нему лежат
// значения в занятиях, и пересоздание поля обнулило бы заполненные карточки.
type fieldBody struct {
	ID          int64          `json:"id"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	ColSpan     int            `json:"col_span"`
	RowSpan     int            `json:"row_span"`
	ShowOnBlock bool           `json:"show_on_block"`
	ShowInCard  bool           `json:"show_in_card"`
}

func (h *handlers) replaceFields(c *fiber.Ctx) error {
	var body struct {
		Fields []fieldBody `json:"fields"`
	}
	parseBody(c, &body)
	fields := make([]*domain.Field, 0, len(body.Fields))
	for _, f := range body.Fields {
		fields = append(fields, &domain.Field{
			ID: f.ID, Label: f.Label, Type: f.Type, Config: f.Config,
			ColSpan: f.ColSpan, RowSpan: f.RowSpan,
			ShowOnBlock: f.ShowOnBlock, ShowInCard: f.ShowInCard,
		})
	}
	view, err := h.svc.ReplaceFields(c.Context(), currentUserID(c), pathID(c), fields)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(view)
}

// ── Занятия ──────────────────────────────────────────────────────

type itemBody struct {
	Weekday     int            `json:"weekday"`
	StartMin    int            `json:"start_min"`
	EndMin      int            `json:"end_min"`
	Title       string         `json:"title"`
	Short       string         `json:"short"`
	CategoryID  *int64         `json:"category_id"`
	Weeks       []int          `json:"weeks"`
	RepeatEvery *int           `json:"repeat_every"`
	RepeatFrom  *string        `json:"repeat_from"`
	RepeatUntil *string        `json:"repeat_until"`
	Data        map[string]any `json:"data"`
}

func (b itemBody) toInput() service.ItemInput {
	in := service.ItemInput{
		Weekday: b.Weekday, StartMin: b.StartMin, EndMin: b.EndMin,
		Title: b.Title, Short: b.Short, CategoryID: b.CategoryID,
		Weeks: b.Weeks, RepeatEvery: b.RepeatEvery, Data: b.Data,
	}
	if in.Weeks == nil {
		in.Weeks = []int{}
	}
	if in.Data == nil {
		in.Data = map[string]any{}
	}
	if b.RepeatFrom != nil {
		in.RepeatFrom = parseDate(*b.RepeatFrom)
	}
	if b.RepeatUntil != nil {
		in.RepeatUntil = parseDate(*b.RepeatUntil)
	}
	return in
}

func (h *handlers) createItem(c *fiber.Ctx) error {
	var body itemBody
	parseBody(c, &body)
	item, err := h.svc.CreateItem(c.Context(), currentUserID(c), pathID(c), body.toInput())
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *handlers) updateItem(c *fiber.Ctx) error {
	var body itemBody
	parseBody(c, &body)
	item, err := h.svc.UpdateItem(c.Context(), currentUserID(c), pathID(c),
		paramID(c, "itemId"), body.toInput())
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(item)
}

func (h *handlers) deleteItem(c *fiber.Ctx) error {
	if err := h.svc.DeleteItem(c.Context(), currentUserID(c), pathID(c), paramID(c, "itemId")); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Плитка, поиск, каталоги ──────────────────────────────────────

// agenda — что идёт сейчас и что дальше. День и текущую минуту присылает
// клиент: зон у расписаний много, а «сейчас» у человека одно.
func (h *handlers) agenda(c *fiber.Ctx) error {
	date := parseDate(c.Query("date"))
	if date == nil {
		return validationError(c, "Укажите день (date)")
	}
	resp, err := h.svc.Agenda(c.Context(), currentUserID(c), *date, c.QueryInt("minute"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) search(c *fiber.Ctx) error {
	items, err := h.svc.Search(c.Context(), currentUserID(c), c.Query("q"), c.QueryInt("limit"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) directory(c *fiber.Ctx) error {
	users, err := h.svc.Directory(c.Context(), currentUserID(c), c.Query("q"), c.QueryInt("limit"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"users": users})
}

func (h *handlers) companies(c *fiber.Ctx) error {
	list, err := h.svc.Companies(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"companies": list})
}

// ── Шаринг ───────────────────────────────────────────────────────

func (h *handlers) sharedSchedule(c *fiber.Ctx) error {
	view, err := h.svc.SharedView(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(view)
}

func (h *handlers) listShares(c *fiber.Ctx) error {
	shares, err := h.svc.ListShares(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"shares": shares})
}

func (h *handlers) createShare(c *fiber.Ctx) error {
	share, err := h.svc.CreateShare(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(share)
}

func (h *handlers) revokeShare(c *fiber.Ctx) error {
	if err := h.svc.RevokeShare(c.Context(), currentUserID(c), pathID(c), paramID(c, "shareId")); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"revoked": true})
}

func (h *handlers) listUserShares(c *fiber.Ctx) error {
	shares, err := h.svc.ListUserShares(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"access": shares})
}

// targetBody — адресат доступа: человек ЛИБО компания.
type targetBody struct {
	UserID    *int64 `json:"user_id"`
	CompanyID *int64 `json:"company_id"`
}

func (h *handlers) shareWith(c *fiber.Ctx) error {
	var body targetBody
	parseBody(c, &body)
	share, err := h.svc.ShareWith(c.Context(), currentUserID(c), pathID(c), body.UserID, body.CompanyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(share)
}

func (h *handlers) unshare(c *fiber.Ctx) error {
	var body targetBody
	parseBody(c, &body)
	if err := h.svc.Unshare(c.Context(), currentUserID(c), pathID(c), body.UserID, body.CompanyID); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"removed": true})
}

// ── Перенос ──────────────────────────────────────────────────────

// export — расписание файлом: таблицей (xlsx) или для переноса (json).
func (h *handlers) export(c *fiber.Ctx) error {
	uid, id := currentUserID(c), pathID(c)
	if c.Query("format") == "json" {
		raw, name, err := h.svc.Export(c.Context(), uid, id)
		if err != nil {
			return h.respondError(c, err)
		}
		return sendFile(c, raw, name+".json", "application/json")
	}
	raw, name, err := h.svc.ExportXLSX(c.Context(), uid, id)
	if err != nil {
		return h.respondError(c, err)
	}
	return sendFile(c, raw, name+".xlsx", xlsxMime)
}

func (h *handlers) importInto(c *fiber.Ctx) error {
	view, err := h.svc.Import(c.Context(), currentUserID(c), pathID(c), c.Body())
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(view)
}

func (h *handlers) importNew(c *fiber.Ctx) error {
	view, err := h.svc.ImportNew(c.Context(), currentUserID(c), c.Body(), c.Query("name"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(view)
}

// sendFile — вложение с именем файла. Имя уезжает и в ASCII-виде, и в UTF-8
// (filename*): кириллицу в обычном filename понимают не все клиенты.
func sendFile(c *fiber.Ctx, body []byte, name, mime string) error {
	c.Set(fiber.HeaderContentType, mime)
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="schedule"; filename*=UTF-8''`+url.PathEscape(name))
	return c.Send(body)
}

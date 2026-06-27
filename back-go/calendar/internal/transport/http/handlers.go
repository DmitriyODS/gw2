package http

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calendar/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/calendar/internal/service"
)

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// csvInts — разбор query-параметра вида "1,2,3" в срез id (мусор отбрасывается).
func csvInts(s string) []int64 {
	if s == "" {
		return nil
	}
	out := []int64{}
	for _, part := range strings.Split(s, ",") {
		if n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

// parseTime — ISO-8601/RFC3339 в *time.Time ("" → nil).
func parseTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}

// entryParams — диапазон дат + поиск из query-строки.
func entryParams(c *fiber.Ctx) service.EntryListParams {
	return service.EntryListParams{
		Search: c.Query("search"),
		From:   parseTime(c.Query("from")),
		To:     parseTime(c.Query("to")),
	}
}

// ── Календари ────────────────────────────────────────────────────

func (h *handlers) listCalendars(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ListCalendars(c.Context(), endpoint.CompanyReq{CompanyID: companyID})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"calendars": resp})
}

func (h *handlers) getCalendar(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.GetCalendar(c.Context(), endpoint.CalendarReq{CompanyID: companyID, ID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createCalendar(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		Name string `json:"name"`
	}
	parseBody(c, &body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return validationError(c, "Укажите название календаря")
	}
	if len([]rune(name)) > 120 {
		return validationError(c, "Название слишком длинное (макс. 120)")
	}
	resp, err := h.eps.CreateCalendar(c.Context(), endpoint.CreateCalendarReq{
		CompanyID: companyID, UserID: currentUser(c).ID, Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateCalendar(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		Name string `json:"name"`
	}
	parseBody(c, &body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return validationError(c, "Укажите название календаря")
	}
	if len([]rune(name)) > 120 {
		return validationError(c, "Название слишком длинное (макс. 120)")
	}
	resp, err := h.eps.UpdateCalendar(c.Context(), endpoint.UpdateCalendarReq{
		CompanyID: companyID, ID: pathID(c), Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteCalendar(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	if _, err := h.eps.DeleteCalendar(c.Context(), endpoint.CalendarReq{CompanyID: companyID, ID: pathID(c)}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) replaceFields(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		Fields []fieldInput `json:"fields"`
	}
	parseBody(c, &body)
	fields, msg := parseFields(body.Fields)
	if msg != "" {
		return validationError(c, msg)
	}
	resp, err := h.eps.ReplaceFields(c.Context(), endpoint.ReplaceFieldsReq{
		CompanyID: companyID, ID: pathID(c), Fields: fields,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── Записи ───────────────────────────────────────────────────────

func (h *handlers) listEntries(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ListEntries(c.Context(), endpoint.ListEntriesReq{
		CompanyID: companyID, CalendarID: pathID(c), Params: entryParams(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getEntry(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.GetEntry(c.Context(), endpoint.EntryReq{
		CompanyID: companyID, CalendarID: pathID(c), EntryID: entryID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// entryBody — тело записи: дата/время + произвольные значения полей.
type entryBody struct {
	EventAt string         `json:"event_at"`
	Data    map[string]any `json:"data"`
}

func (h *handlers) createEntry(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body entryBody
	parseBody(c, &body)
	at := parseTime(body.EventAt)
	if at == nil {
		return validationError(c, "Укажите дату и время записи")
	}
	if body.Data == nil {
		body.Data = map[string]any{}
	}
	resp, err := h.eps.CreateEntry(c.Context(), endpoint.WriteEntryReq{
		CompanyID: companyID, CalendarID: pathID(c), UserID: currentUser(c).ID,
		EventAt: *at, Data: body.Data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateEntry(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body entryBody
	parseBody(c, &body)
	at := parseTime(body.EventAt)
	if at == nil {
		return validationError(c, "Укажите дату и время записи")
	}
	if body.Data == nil {
		body.Data = map[string]any{}
	}
	resp, err := h.eps.UpdateEntry(c.Context(), endpoint.WriteEntryReq{
		CompanyID: companyID, CalendarID: pathID(c), EntryID: entryID(c),
		EventAt: *at, Data: body.Data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteEntry(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	if _, err := h.eps.DeleteEntry(c.Context(), endpoint.EntryReq{
		CompanyID: companyID, CalendarID: pathID(c), EntryID: entryID(c),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) bulkDeleteEntries(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		IDs []int64 `json:"ids"`
	}
	parseBody(c, &body)
	resp, err := h.eps.DeleteEntries(c.Context(), endpoint.DeleteEntriesReq{
		CompanyID: companyID, CalendarID: pathID(c), IDs: body.IDs,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": resp})
}

func (h *handlers) exportEntries(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ExportEntries(c.Context(), endpoint.ExportReq{
		CompanyID: companyID, CalendarID: pathID(c),
		FieldIDs: csvInts(c.Query("fields")), Params: entryParams(c), IDs: csvInts(c.Query("ids")),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return sendXLSX(c, resp.(endpoint.ExportResp))
}

// ── Загрузка файла ───────────────────────────────────────────────

func (h *handlers) upload(c *fiber.Ctx) error {
	if _, ok := companyScope(c); !ok {
		return nil
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NO_FILE", "message": "Файл не передан"})
	}
	if fileHeader.Size > uploadMaxBytes {
		return validationError(c, "Файл слишком большой (макс. 25 МБ)")
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, uploadMaxBytes+1))
	if err != nil {
		return h.respondError(c, err)
	}
	if int64(len(data)) > uploadMaxBytes {
		return validationError(c, "Файл слишком большой (макс. 25 МБ)")
	}
	resp, err := h.eps.Upload(c.Context(), endpoint.UploadReq{
		FileName: fileHeader.Filename,
		Mime:     fileHeader.Header.Get(fiber.HeaderContentType),
		Data:     data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// ── Публичные ссылки: управление (участник компании) ─────────────

func (h *handlers) listShares(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ListShares(c.Context(), endpoint.ShareReq{CompanyID: companyID, CalendarID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"shares": resp})
}

func (h *handlers) createShare(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.CreateShare(c.Context(), endpoint.ShareReq{
		CompanyID: companyID, CalendarID: pathID(c), UserID: currentUser(c).ID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) revokeShare(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	shareID, _ := c.ParamsInt("shareId")
	if _, err := h.eps.RevokeShare(c.Context(), endpoint.ShareReq{
		CompanyID: companyID, CalendarID: pathID(c), ShareID: int64(shareID),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Публичный доступ по коду (без авторизации) ───────────────────

func (h *handlers) sharedCalendar(c *fiber.Ctx) error {
	resp, err := h.eps.SharedCalendar(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedEntries(c *fiber.Ctx) error {
	resp, err := h.eps.SharedEntries(c.Context(), endpoint.SharedEntriesReq{
		Code: c.Params("code"), Params: entryParams(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedExport(c *fiber.Ctx) error {
	resp, err := h.eps.SharedExport(c.Context(), endpoint.SharedExportReq{
		Code: c.Params("code"), FieldIDs: csvInts(c.Query("fields")),
		Params: entryParams(c), IDs: csvInts(c.Query("ids")),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return sendXLSX(c, resp.(endpoint.ExportResp))
}

func sendXLSX(c *fiber.Ctx, out endpoint.ExportResp) error {
	c.Set(fiber.HeaderContentType, xlsxMime)
	// Имя файла из названия календаря: ascii-fallback + UTF-8 (RFC 5987).
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="calendar.xlsx"; filename*=UTF-8''`+url.PathEscape(out.Name)+`.xlsx`)
	return c.Send(out.Data)
}

// ── Парсинг и валидация полей календаря ──────────────────────────

type fieldInput struct {
	ID             int64          `json:"id"`
	Label          string         `json:"label"`
	Type           string         `json:"type"`
	Config         map[string]any `json:"config"`
	ColSpan        int            `json:"col_span"`
	RowSpan        int            `json:"row_span"`
	ShowInTable    bool           `json:"show_in_table"`
	ShowInCard     bool           `json:"show_in_card"`
	VisibleFieldID *int64         `json:"visible_field_id"`
	VisibleValue   *string        `json:"visible_value"`
}

// parseFields — провалидировать вход и сконвертировать в доменные поля. Второй
// результат — текст ошибки валидации ("" — успех).
func parseFields(in []fieldInput) ([]domain.Field, string) {
	out := make([]domain.Field, 0, len(in))
	for _, fi := range in {
		label := strings.TrimSpace(fi.Label)
		if label == "" {
			return nil, "У каждого поля должно быть название"
		}
		if len([]rune(label)) > 120 {
			return nil, "Название поля слишком длинное (макс. 120)"
		}
		if !domain.FieldTypes[fi.Type] {
			return nil, "Неизвестный тип поля: " + fi.Type
		}
		cfg := fi.Config
		if cfg == nil {
			cfg = map[string]any{}
		}
		out = append(out, domain.Field{
			ID: fi.ID, Label: label, Type: fi.Type, Config: cfg,
			ColSpan: fi.ColSpan, RowSpan: fi.RowSpan, ShowInTable: fi.ShowInTable, ShowInCard: fi.ShowInCard,
			VisibleFieldID: fi.VisibleFieldID, VisibleValue: fi.VisibleValue,
		})
	}
	return out, ""
}

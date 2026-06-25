package http

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/diary/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/diary/internal/service"
)

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

// parseTime — дата дня (YYYY-MM-DD) или RFC3339 в *time.Time ("" → nil).
func parseTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.Parse(domain.DateLayout, s); err == nil {
		return &t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}

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

func listParams(c *fiber.Ctx) service.ListParams {
	archived := c.Query("archived") == "1" || c.Query("archived") == "true"
	return service.ListParams{
		Archived: archived,
		Search:   c.Query("search"),
		From:     parseTime(c.Query("from")),
		To:       parseTime(c.Query("to")),
	}
}

// ── Ежедневники ──────────────────────────────────────────────────

func (h *handlers) listDiaries(c *fiber.Ctx) error {
	uid := currentUserID(c)
	var resp any
	var err error
	if c.Query("tab") == "shared" {
		resp, err = h.eps.ListShared(c.Context(), endpoint.UserReq{UserID: uid})
	} else {
		resp, err = h.eps.ListOwned(c.Context(), endpoint.UserReq{UserID: uid})
	}
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"diaries": resp})
}

func (h *handlers) getDiary(c *fiber.Ctx) error {
	resp, err := h.eps.GetDiary(c.Context(), endpoint.DiaryReq{UserID: currentUserID(c), ID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createDiary(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	parseBody(c, &body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return validationError(c, "Укажите название ежедневника")
	}
	if len([]rune(name)) > 120 {
		return validationError(c, "Название слишком длинное (макс. 120)")
	}
	resp, err := h.eps.CreateDiary(c.Context(), endpoint.CreateDiaryReq{UserID: currentUserID(c), Name: name})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateDiary(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	parseBody(c, &body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return validationError(c, "Укажите название ежедневника")
	}
	if len([]rune(name)) > 120 {
		return validationError(c, "Название слишком длинное (макс. 120)")
	}
	resp, err := h.eps.UpdateDiary(c.Context(), endpoint.UpdateDiaryReq{
		UserID: currentUserID(c), ID: pathID(c), Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteDiary(c *fiber.Ctx) error {
	if _, err := h.eps.DeleteDiary(c.Context(), endpoint.DiaryReq{UserID: currentUserID(c), ID: pathID(c)}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Записи ───────────────────────────────────────────────────────

func (h *handlers) listEntries(c *fiber.Ctx) error {
	resp, err := h.eps.ListEntries(c.Context(), endpoint.ListEntriesReq{
		UserID: currentUserID(c), DiaryID: pathID(c), Params: listParams(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getEntry(c *fiber.Ctx) error {
	resp, err := h.eps.GetEntry(c.Context(), endpoint.EntryReq{
		UserID: currentUserID(c), DiaryID: pathID(c), EntryID: recordID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// entryBody — тело записи: день + опциональное время + название/описание.
type entryBody struct {
	EntryDate   string `json:"entry_date"`
	StartMin    *int   `json:"start_min"`
	EndMin      *int   `json:"end_min"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (b entryBody) toInput(c *fiber.Ctx) (service.EntryInput, bool) {
	at := parseTime(b.EntryDate)
	if at == nil {
		_ = validationError(c, "Укажите дату записи")
		return service.EntryInput{}, false
	}
	title := strings.TrimSpace(b.Title)
	if title == "" {
		_ = validationError(c, "Укажите название записи")
		return service.EntryInput{}, false
	}
	if len([]rune(title)) > 200 {
		_ = validationError(c, "Название слишком длинное (макс. 200)")
		return service.EntryInput{}, false
	}
	return service.EntryInput{
		Date: *at, StartMin: clampMin(b.StartMin), EndMin: clampMin(b.EndMin),
		Title: title, Description: strings.TrimSpace(b.Description),
	}, true
}

// clampMin — минуты от полуночи в допустимый диапазон 0..1439 (nil — без времени).
func clampMin(v *int) *int {
	if v == nil {
		return nil
	}
	m := *v
	if m < 0 {
		m = 0
	}
	if m > 1439 {
		m = 1439
	}
	return &m
}

func (h *handlers) createEntry(c *fiber.Ctx) error {
	var body entryBody
	parseBody(c, &body)
	in, ok := body.toInput(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.CreateEntry(c.Context(), endpoint.WriteEntryReq{
		UserID: currentUserID(c), DiaryID: pathID(c), In: in,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateEntry(c *fiber.Ctx) error {
	var body entryBody
	parseBody(c, &body)
	in, ok := body.toInput(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.UpdateEntry(c.Context(), endpoint.WriteEntryReq{
		UserID: currentUserID(c), DiaryID: pathID(c), EntryID: recordID(c), In: in,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) setDone(c *fiber.Ctx) error {
	var body struct {
		Done bool `json:"done"`
	}
	parseBody(c, &body)
	resp, err := h.eps.SetDone(c.Context(), endpoint.DoneReq{
		UserID: currentUserID(c), DiaryID: pathID(c), EntryID: recordID(c), Done: body.Done,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) setLink(c *fiber.Ctx) error {
	var body struct {
		TaskID *int64 `json:"task_id"`
	}
	parseBody(c, &body)
	resp, err := h.eps.SetLink(c.Context(), endpoint.LinkReq{
		UserID: currentUserID(c), DiaryID: pathID(c), EntryID: recordID(c), TaskID: body.TaskID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteEntry(c *fiber.Ctx) error {
	if _, err := h.eps.DeleteEntry(c.Context(), endpoint.EntryReq{
		UserID: currentUserID(c), DiaryID: pathID(c), EntryID: recordID(c),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) bulkDeleteEntries(c *fiber.Ctx) error {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	parseBody(c, &body)
	resp, err := h.eps.DeleteEntries(c.Context(), endpoint.DeleteEntriesReq{
		UserID: currentUserID(c), DiaryID: pathID(c), IDs: body.IDs,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": resp})
}

func (h *handlers) exportEntries(c *fiber.Ctx) error {
	resp, err := h.eps.ExportEntries(c.Context(), endpoint.ExportReq{
		UserID: currentUserID(c), DiaryID: pathID(c),
		Params: listParams(c), IDs: csvInts(c.Query("ids")),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return sendXLSX(c, resp.(endpoint.ExportResp))
}

// ── Публичные ссылки (владелец) ──────────────────────────────────

func (h *handlers) listShares(c *fiber.Ctx) error {
	resp, err := h.eps.ListShares(c.Context(), endpoint.ShareReq{UserID: currentUserID(c), DiaryID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"shares": resp})
}

func (h *handlers) createShare(c *fiber.Ctx) error {
	resp, err := h.eps.CreateShare(c.Context(), endpoint.ShareReq{UserID: currentUserID(c), DiaryID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) revokeShare(c *fiber.Ctx) error {
	shareID, _ := c.ParamsInt("shareId")
	if _, err := h.eps.RevokeShare(c.Context(), endpoint.ShareReq{
		UserID: currentUserID(c), DiaryID: pathID(c), ShareID: int64(shareID),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Адресный доступ (владелец) ───────────────────────────────────

func (h *handlers) listMembers(c *fiber.Ctx) error {
	resp, err := h.eps.ListMembers(c.Context(), endpoint.ShareReq{UserID: currentUserID(c), DiaryID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"members": resp})
}

func (h *handlers) addMember(c *fiber.Ctx) error {
	var body struct {
		UserID int64 `json:"user_id"`
	}
	parseBody(c, &body)
	if body.UserID <= 0 {
		return validationError(c, "Укажите пользователя")
	}
	resp, err := h.eps.AddMember(c.Context(), endpoint.MemberReq{
		UserID: currentUserID(c), DiaryID: pathID(c), MemberID: body.UserID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) removeMember(c *fiber.Ctx) error {
	memberID, _ := c.ParamsInt("userId")
	if _, err := h.eps.RemoveMember(c.Context(), endpoint.MemberReq{
		UserID: currentUserID(c), DiaryID: pathID(c), MemberID: int64(memberID),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Публичный доступ по коду (без авторизации) ───────────────────

func (h *handlers) sharedDiary(c *fiber.Ctx) error {
	resp, err := h.eps.SharedDiary(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedEntries(c *fiber.Ctx) error {
	resp, err := h.eps.SharedEntries(c.Context(), endpoint.SharedEntriesReq{
		Code: c.Params("code"), Params: listParams(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedExport(c *fiber.Ctx) error {
	resp, err := h.eps.SharedExport(c.Context(), endpoint.SharedExportReq{
		Code: c.Params("code"), Params: listParams(c), IDs: csvInts(c.Query("ids")),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return sendXLSX(c, resp.(endpoint.ExportResp))
}

func sendXLSX(c *fiber.Ctx, out endpoint.ExportResp) error {
	c.Set(fiber.HeaderContentType, xlsxMime)
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="diary.xlsx"; filename*=UTF-8''`+url.PathEscape(out.Name)+`.xlsx`)
	return c.Send(out.Data)
}

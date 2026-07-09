package http

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/endpoint"
)

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

// noteBody — частичная правка заметки: отсутствующие поля не меняются.
// Color и Archived правятся только владельцем (PATCH), по edit-ссылке игнорируются.
type noteBody struct {
	Title    *string         `json:"title"`
	Color    *string         `json:"color"`
	Archived *bool           `json:"archived"`
	Doc      json.RawMessage `json:"doc"`
}

// validate — общая валидация правки (владелец и edit-ссылка).
func (b *noteBody) validate(c *fiber.Ctx) bool {
	if b.Title != nil {
		t := strings.TrimSpace(*b.Title)
		if len([]rune(t)) > 300 {
			_ = validationError(c, "Заголовок слишком длинный (макс. 300)")
			return false
		}
		b.Title = &t
	}
	if b.Doc != nil && !json.Valid(b.Doc) {
		_ = validationError(c, "Некорректный документ")
		return false
	}
	return true
}

// ── Заметки ──────────────────────────────────────────────────────

func (h *handlers) listNotes(c *fiber.Ctx) error {
	groupID, _ := strconv.ParseInt(c.Query("group_id"), 10, 64)
	resp, err := h.eps.ListNotes(c.Context(), endpoint.ListNotesReq{
		UserID: currentUserID(c), GroupID: groupID, Search: c.Query("search"),
		Archived: c.Query("archived") == "1",
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"notes": resp})
}

func (h *handlers) getNote(c *fiber.Ctx) error {
	resp, err := h.eps.GetNote(c.Context(), endpoint.NoteReq{UserID: currentUserID(c), ID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createNote(c *fiber.Ctx) error {
	var body struct {
		Title string `json:"title"`
	}
	parseBody(c, &body)
	title := strings.TrimSpace(body.Title)
	if len([]rune(title)) > 300 {
		return validationError(c, "Заголовок слишком длинный (макс. 300)")
	}
	resp, err := h.eps.CreateNote(c.Context(), endpoint.CreateNoteReq{UserID: currentUserID(c), Title: title})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateNote(c *fiber.Ctx) error {
	var body noteBody
	parseBody(c, &body)
	if !body.validate(c) {
		return nil
	}
	resp, err := h.eps.UpdateNote(c.Context(), endpoint.UpdateNoteReq{
		UserID: currentUserID(c), ID: pathID(c), Title: body.Title, Color: body.Color,
		Archived: body.Archived, Doc: body.Doc,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteNote(c *fiber.Ctx) error {
	if _, err := h.eps.DeleteNote(c.Context(), endpoint.NoteReq{UserID: currentUserID(c), ID: pathID(c)}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) setGroups(c *fiber.Ctx) error {
	var body struct {
		GroupIDs []int64 `json:"group_ids"`
	}
	parseBody(c, &body)
	resp, err := h.eps.SetGroups(c.Context(), endpoint.SetGroupsReq{
		UserID: currentUserID(c), ID: pathID(c), GroupIDs: body.GroupIDs,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// ── Группы ───────────────────────────────────────────────────────

func (h *handlers) listGroups(c *fiber.Ctx) error {
	resp, err := h.eps.ListGroups(c.Context(), endpoint.NoteReq{UserID: currentUserID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"groups": resp})
}

func groupName(c *fiber.Ctx) (string, bool) {
	var body struct {
		Name string `json:"name"`
	}
	parseBody(c, &body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		_ = validationError(c, "Укажите название группы")
		return "", false
	}
	if len([]rune(name)) > 100 {
		_ = validationError(c, "Название слишком длинное (макс. 100)")
		return "", false
	}
	return name, true
}

func (h *handlers) createGroup(c *fiber.Ctx) error {
	name, ok := groupName(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.CreateGroup(c.Context(), endpoint.GroupReq{UserID: currentUserID(c), Name: name})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateGroup(c *fiber.Ctx) error {
	name, ok := groupName(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.UpdateGroup(c.Context(), endpoint.GroupReq{
		UserID: currentUserID(c), ID: pathID(c), Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteGroup(c *fiber.Ctx) error {
	if _, err := h.eps.DeleteGroup(c.Context(), endpoint.GroupReq{UserID: currentUserID(c), ID: pathID(c)}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Публичные ссылки (владелец) ──────────────────────────────────

func (h *handlers) listShares(c *fiber.Ctx) error {
	resp, err := h.eps.ListShares(c.Context(), endpoint.ShareReq{UserID: currentUserID(c), NoteID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"shares": resp})
}

func (h *handlers) createShare(c *fiber.Ctx) error {
	var body struct {
		Access string `json:"access"`
	}
	parseBody(c, &body)
	resp, err := h.eps.CreateShare(c.Context(), endpoint.ShareReq{
		UserID: currentUserID(c), NoteID: pathID(c), Access: body.Access,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) revokeShare(c *fiber.Ctx) error {
	shareID, _ := c.ParamsInt("shareId")
	if _, err := h.eps.RevokeShare(c.Context(), endpoint.ShareReq{
		UserID: currentUserID(c), NoteID: pathID(c), ShareID: int64(shareID),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Картинки редактора, экспорт/импорт ───────────────────────────

func (h *handlers) upload(c *fiber.Ctx) error {
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
		UserID: currentUserID(c), NoteID: pathID(c),
		FileName: fileHeader.Filename, Data: data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) exportNote(c *fiber.Ctx) error {
	resp, err := h.eps.Export(c.Context(), endpoint.NoteReq{UserID: currentUserID(c), ID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	out := resp.(endpoint.ExportResp)
	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="note.txt"; filename*=UTF-8''`+url.PathEscape(out.Name)+`.txt`)
	return c.Send(out.Data)
}

func (h *handlers) importNote(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NO_FILE", "message": "Файл не передан"})
	}
	if fileHeader.Size > 1024*1024 {
		return validationError(c, "Файл слишком большой (макс. 1 МБ)")
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if err != nil {
		return h.respondError(c, err)
	}
	resp, err := h.eps.Import(c.Context(), endpoint.ImportReq{UserID: currentUserID(c), Text: string(data)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// ── Публичный доступ по коду (без авторизации) ───────────────────

func (h *handlers) sharedNote(c *fiber.Ctx) error {
	resp, err := h.eps.SharedNote(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedUpdate(c *fiber.Ctx) error {
	var body noteBody
	parseBody(c, &body)
	if !body.validate(c) {
		return nil
	}
	resp, err := h.eps.SharedUpdate(c.Context(), endpoint.SharedUpdateReq{
		Code: c.Params("code"), Title: body.Title, Doc: body.Doc,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

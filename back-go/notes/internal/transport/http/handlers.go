package http

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/docx"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/service"
)

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

// parseTagIDs — CSV «1,2,3» → []int64 (пустые/битые пропускаются).
func parseTagIDs(s string) []int64 {
	if s == "" {
		return nil
	}
	out := []int64{}
	for _, part := range strings.Split(s, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && id > 0 {
			out = append(out, id)
		}
	}
	return out
}

// ── Заметки ──────────────────────────────────────────────────────

func (h *handlers) listNotes(c *fiber.Ctx) error {
	uid := currentUserID(c)
	// ?shared=1 — чужие заметки, открытые мне (адресно/через папку).
	if c.Query("shared") == "1" {
		resp, err := h.svc.ListSharedNotes(c.Context(), uid, c.Query("search"))
		if err != nil {
			return h.respondError(c, err)
		}
		return c.JSON(fiber.Map{"notes": resp})
	}
	p := service.ListNotesParams{
		TagIDs: parseTagIDs(c.Query("tag_ids")), Search: c.Query("search"),
		Archived: c.Query("archived") == "1",
	}
	// folder_id: отсутствует — все заметки; "root" — корень; число — папка.
	switch fq := c.Query("folder_id"); fq {
	case "":
	case "root":
		p.FolderSet = true
	default:
		if id, err := strconv.ParseInt(fq, 10, 64); err == nil {
			p.FolderSet = true
			p.FolderID = &id
		}
	}
	resp, err := h.svc.ListNotes(c.Context(), uid, currentCompanyID(c), p)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"notes": resp})
}

func (h *handlers) getNote(c *fiber.Ctx) error {
	resp, err := h.svc.GetNote(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createNote(c *fiber.Ctx) error {
	var body struct {
		Title    string `json:"title"`
		FolderID *int64 `json:"folder_id"`
	}
	parseBody(c, &body)
	title := strings.TrimSpace(body.Title)
	if len([]rune(title)) > 300 {
		return validationError(c, "Заголовок слишком длинный (макс. 300)")
	}
	resp, err := h.svc.CreateNote(c.Context(), currentUserID(c), title, body.FolderID)
	if err != nil {
		return h.respondError(c, err)
	}
	h.svc.ReindexNoteAsync(resp.ID, currentCompanyID(c))
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// noteBody — частичная правка заметки: отсутствующие поля не меняются.
type noteBody struct {
	Title    *string         `json:"title"`
	Color    *string         `json:"color"`
	Archived *bool           `json:"archived"`
	Pinned   *bool           `json:"pinned"`
	Doc      json.RawMessage `json:"doc"`
}

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

func (h *handlers) updateNote(c *fiber.Ctx) error {
	var body noteBody
	parseBody(c, &body)
	if !body.validate(c) {
		return nil
	}
	resp, err := h.svc.UpdateNote(c.Context(), currentUserID(c), pathID(c), domain.NoteUpdate{
		Title: body.Title, Color: body.Color, Archived: body.Archived, Pinned: body.Pinned, Doc: body.Doc,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	if body.Title != nil || body.Doc != nil { // изменился текст — переиндексируем
		h.svc.ReindexNoteAsync(resp.ID, currentCompanyID(c))
	}
	return c.JSON(resp)
}

func (h *handlers) deleteNote(c *fiber.Ctx) error {
	if err := h.svc.DeleteNote(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) moveNote(c *fiber.Ctx) error {
	var body struct {
		FolderID *int64 `json:"folder_id"`
	}
	parseBody(c, &body)
	resp, err := h.svc.MoveNote(c.Context(), currentUserID(c), pathID(c), body.FolderID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) copyNote(c *fiber.Ctx) error {
	resp, err := h.svc.CopyNote(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	h.svc.ReindexNoteAsync(resp.ID, currentCompanyID(c))
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) setTags(c *fiber.Ctx) error {
	var body struct {
		TagIDs []int64 `json:"tag_ids"`
	}
	parseBody(c, &body)
	resp, err := h.svc.SetTags(c.Context(), currentUserID(c), pathID(c), body.TagIDs)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// myCompanies — компании пользователя для выбора аудитории шаринга.
func (h *handlers) myCompanies(c *fiber.Ctx) error {
	list, err := h.svc.MyCompanies(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	out := make([]fiber.Map, len(list))
	for i, co := range list {
		out[i] = fiber.Map{"id": co.ID, "name": co.Name}
	}
	return c.JSON(fiber.Map{"companies": out})
}

// ── Папки ────────────────────────────────────────────────────────

func (h *handlers) listFolders(c *fiber.Ctx) error {
	resp, err := h.svc.ListFolders(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) folderChildren(c *fiber.Ctx) error {
	resp, err := h.svc.FolderChildren(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createFolder(c *fiber.Ctx) error {
	var body struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		ParentID *int64 `json:"parent_id"`
	}
	parseBody(c, &body)
	resp, err := h.svc.CreateFolder(c.Context(), currentUserID(c), body.Name, body.Color, body.ParentID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateFolder(c *fiber.Ctx) error {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	parseBody(c, &body)
	resp, err := h.svc.UpdateFolder(c.Context(), currentUserID(c), pathID(c), body.Name, body.Color)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) moveFolder(c *fiber.Ctx) error {
	var body struct {
		ParentID *int64 `json:"parent_id"`
	}
	parseBody(c, &body)
	resp, err := h.svc.MoveFolder(c.Context(), currentUserID(c), pathID(c), body.ParentID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) copyFolder(c *fiber.Ctx) error {
	resp, err := h.svc.CopyFolder(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) deleteFolder(c *fiber.Ctx) error {
	if err := h.svc.DeleteFolder(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Теги ─────────────────────────────────────────────────────────

func (h *handlers) listTags(c *fiber.Ctx) error {
	resp, err := h.svc.ListTags(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"tags": resp})
}

// tagBody — валидация имени тега (обязательно, ≤ 60).
func tagBody(c *fiber.Ctx) (name, color string, ok bool) {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	parseBody(c, &body)
	name = strings.TrimSpace(body.Name)
	if name == "" {
		_ = validationError(c, "Укажите название тега")
		return "", "", false
	}
	if len([]rune(name)) > 60 {
		_ = validationError(c, "Название слишком длинное (макс. 60)")
		return "", "", false
	}
	return name, body.Color, true
}

func (h *handlers) createTag(c *fiber.Ctx) error {
	name, color, ok := tagBody(c)
	if !ok {
		return nil
	}
	resp, err := h.svc.CreateTag(c.Context(), currentUserID(c), name, color)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateTag(c *fiber.Ctx) error {
	name, color, ok := tagBody(c)
	if !ok {
		return nil
	}
	resp, err := h.svc.UpdateTag(c.Context(), currentUserID(c), pathID(c), name, color)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteTag(c *fiber.Ctx) error {
	if err := h.svc.DeleteTag(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Публичные ссылки (владелец) ──────────────────────────────────

func (h *handlers) listShares(c *fiber.Ctx) error {
	resp, err := h.svc.ListShares(c.Context(), currentUserID(c), pathID(c))
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
	resp, err := h.svc.CreateShare(c.Context(), currentUserID(c), pathID(c), body.Access)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) revokeShare(c *fiber.Ctx) error {
	shareID, _ := c.ParamsInt("shareId")
	if err := h.svc.RevokeShare(c.Context(), currentUserID(c), pathID(c), int64(shareID)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Адресный шаринг заметок и папок ──────────────────────────────

// shareBody — тело шаринга: аудитория + право.
type shareBody struct {
	Target    string `json:"target"` // user | company
	UserID    int64  `json:"user_id"`
	CompanyID int64  `json:"company_id"`
	CanEdit   bool   `json:"can_edit"`
}

func (b shareBody) targetID() int64 {
	if b.Target == domain.TargetCompany {
		return b.CompanyID
	}
	return b.UserID
}

func (h *handlers) listNoteMembers(c *fiber.Ctx) error {
	resp, err := h.svc.ListNoteMembers(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"members": resp})
}

func (h *handlers) shareNote(c *fiber.Ctx) error {
	var body shareBody
	parseBody(c, &body)
	resp, err := h.svc.ShareNote(c.Context(), currentUserID(c), pathID(c), body.Target, body.targetID(), body.CanEdit)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) unshareNoteUser(c *fiber.Ctx) error {
	uid, _ := c.ParamsInt("userId")
	if err := h.svc.UnshareNote(c.Context(), currentUserID(c), pathID(c), domain.TargetUser, int64(uid)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) unshareNoteCompany(c *fiber.Ctx) error {
	cid, _ := c.ParamsInt("companyId")
	if err := h.svc.UnshareNote(c.Context(), currentUserID(c), pathID(c), domain.TargetCompany, int64(cid)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) listFolderMembers(c *fiber.Ctx) error {
	resp, err := h.svc.ListFolderMembers(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"members": resp})
}

func (h *handlers) shareFolder(c *fiber.Ctx) error {
	var body shareBody
	parseBody(c, &body)
	resp, err := h.svc.ShareFolder(c.Context(), currentUserID(c), pathID(c), body.Target, body.targetID(), body.CanEdit)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) unshareFolderUser(c *fiber.Ctx) error {
	uid, _ := c.ParamsInt("userId")
	if err := h.svc.UnshareFolder(c.Context(), currentUserID(c), pathID(c), domain.TargetUser, int64(uid)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) unshareFolderCompany(c *fiber.Ctx) error {
	cid, _ := c.ParamsInt("companyId")
	if err := h.svc.UnshareFolder(c.Context(), currentUserID(c), pathID(c), domain.TargetCompany, int64(cid)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── collab-броадкаст ─────────────────────────────────────────────

func (h *handlers) collab(c *fiber.Ctx) error {
	var body struct {
		Kind   string               `json:"kind"`
		Cursor *domain.CollabCursor `json:"cursor"`
		Doc    json.RawMessage      `json:"doc"`
		Title  *string              `json:"title"`
	}
	parseBody(c, &body)
	if body.Doc != nil && !json.Valid(body.Doc) {
		return validationError(c, "Некорректный документ")
	}
	if body.Title != nil && len(*body.Title) > 1000 {
		return validationError(c, "Слишком длинное название")
	}
	if err := h.svc.Collab(c.Context(), currentUserID(c), pathID(c), body.Kind, body.Cursor, body.Doc, body.Title); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
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
	path, err := h.svc.Upload(c.Context(), currentUserID(c), pathID(c), fileHeader.Filename, data)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"path": path})
}

// contentType — MIME по расширению выгрузки.
func contentType(ext string) string {
	switch ext {
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "zip":
		return "application/zip"
	default:
		return "text/plain; charset=utf-8"
	}
}

func sendFile(c *fiber.Ctx, f *service.ExportFile) error {
	c.Set(fiber.HeaderContentType, contentType(f.Ext))
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="note.`+f.Ext+`"; filename*=UTF-8''`+url.PathEscape(f.Name)+`.`+f.Ext)
	return c.Send(f.Data)
}

func (h *handlers) exportNote(c *fiber.Ctx) error {
	f, err := h.svc.Export(c.Context(), currentUserID(c), pathID(c), c.Query("format"))
	if err != nil {
		return h.respondError(c, err)
	}
	return sendFile(c, f)
}

// exportAll — zip особой группировки (?scope=all|archive|shared&format=txt|docx).
func (h *handlers) exportAll(c *fiber.Ctx) error {
	f, err := h.svc.ExportScope(c.Context(), currentUserID(c), c.Query("scope"), c.Query("format"))
	if err != nil {
		return h.respondError(c, err)
	}
	return sendFile(c, f)
}

func (h *handlers) exportFolder(c *fiber.Ctx) error {
	f, err := h.svc.ExportFolder(c.Context(), currentUserID(c), pathID(c), c.Query("format"))
	if err != nil {
		return h.respondError(c, err)
	}
	return sendFile(c, f)
}

func (h *handlers) importNote(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NO_FILE", "message": "Файл не передан"})
	}
	if fileHeader.Size > 25*1024*1024 {
		return validationError(c, "Файл слишком большой (макс. 25 МБ)")
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 25*1024*1024+1))
	if err != nil {
		return h.respondError(c, err)
	}
	text := string(data)
	if strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".docx") {
		parsed, perr := docx.Parse(data)
		if perr != nil {
			return validationError(c, "Не удалось прочитать .docx")
		}
		// Если у файла нет заголовка внутри — берём имя файла первой строкой.
		title := strings.TrimSuffix(fileHeader.Filename, ".docx")
		text = title + "\n" + parsed
	}
	var folderID *int64
	if fq := c.FormValue("folder_id"); fq != "" && fq != "root" {
		if id, e := strconv.ParseInt(fq, 10, 64); e == nil {
			folderID = &id
		}
	}
	resp, err := h.svc.Import(c.Context(), currentUserID(c), text, folderID)
	if err != nil {
		return h.respondError(c, err)
	}
	h.svc.ReindexNoteAsync(resp.ID, currentCompanyID(c))
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// ── Публичный доступ по коду (без авторизации) ───────────────────

func (h *handlers) sharedNote(c *fiber.Ctx) error {
	resp, err := h.svc.GetSharedNote(c.Context(), c.Params("code"))
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
	resp, err := h.svc.UpdateSharedNote(c.Context(), c.Params("code"), body.Title, body.Doc)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

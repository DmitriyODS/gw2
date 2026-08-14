package http

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/board/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
)

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

// ── Доски ──────────────────────────────────────────────────────

func (h *handlers) listBoards(c *fiber.Ctx) error {
	uid := currentUserID(c)
	// ?shared=1 — чужие доски, открытые мне (адресно/через папку).
	if c.Query("shared") == "1" {
		resp, err := h.svc.ListSharedBoards(c.Context(), uid, c.Query("search"))
		if err != nil {
			return h.respondError(c, err)
		}
		return c.JSON(fiber.Map{"boards": resp})
	}
	p := service.ListBoardsParams{
		Search: c.Query("search"), Archived: c.Query("archived") == "1",
	}
	// folder_id: отсутствует — все доски; "root" — корень; число — папка.
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
	resp, err := h.svc.ListBoards(c.Context(), uid, p)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"boards": resp})
}

func (h *handlers) getBoard(c *fiber.Ctx) error {
	resp, err := h.svc.GetBoard(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createBoard(c *fiber.Ctx) error {
	var body struct {
		Title    string `json:"title"`
		FolderID *int64 `json:"folder_id"`
	}
	parseBody(c, &body)
	title := strings.TrimSpace(body.Title)
	if len([]rune(title)) > 300 {
		return validationError(c, "Заголовок слишком длинный (макс. 300)")
	}
	resp, err := h.svc.CreateBoard(c.Context(), currentUserID(c), title, body.FolderID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// boardBody — частичная правка доски: отсутствующие поля не меняются.
type boardBody struct {
	Title    *string         `json:"title"`
	Color    *string         `json:"color"`
	Archived *bool           `json:"archived"`
	Pinned   *bool           `json:"pinned"`
	Scene    json.RawMessage `json:"scene"`
}

func (b *boardBody) validate(c *fiber.Ctx) bool {
	if b.Title != nil {
		t := strings.TrimSpace(*b.Title)
		if len([]rune(t)) > 300 {
			_ = validationError(c, "Заголовок слишком длинный (макс. 300)")
			return false
		}
		b.Title = &t
	}
	if b.Scene != nil && !json.Valid(b.Scene) {
		_ = validationError(c, "Некорректная сцена доски")
		return false
	}
	return true
}

func (h *handlers) updateBoard(c *fiber.Ctx) error {
	var body boardBody
	parseBody(c, &body)
	if !body.validate(c) {
		return nil
	}
	resp, err := h.svc.UpdateBoard(c.Context(), currentUserID(c), pathID(c), domain.BoardUpdate{
		Title: body.Title, Color: body.Color, Archived: body.Archived, Pinned: body.Pinned, Scene: body.Scene,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteBoard(c *fiber.Ctx) error {
	if err := h.svc.DeleteBoard(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) moveBoard(c *fiber.Ctx) error {
	var body struct {
		FolderID *int64 `json:"folder_id"`
	}
	parseBody(c, &body)
	resp, err := h.svc.MoveBoard(c.Context(), currentUserID(c), pathID(c), body.FolderID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) copyBoard(c *fiber.Ctx) error {
	resp, err := h.svc.CopyBoard(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
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

// ── Адресный шаринг досок и папок ──────────────────────────────

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

func (h *handlers) listBoardMembers(c *fiber.Ctx) error {
	resp, err := h.svc.ListBoardMembers(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"members": resp})
}

func (h *handlers) shareBoard(c *fiber.Ctx) error {
	var body shareBody
	parseBody(c, &body)
	resp, err := h.svc.ShareBoard(c.Context(), currentUserID(c), pathID(c), body.Target, body.targetID(), body.CanEdit)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) unshareBoardUser(c *fiber.Ctx) error {
	uid, _ := c.ParamsInt("userId")
	if err := h.svc.UnshareBoard(c.Context(), currentUserID(c), pathID(c), domain.TargetUser, int64(uid)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) unshareBoardCompany(c *fiber.Ctx) error {
	cid, _ := c.ParamsInt("companyId")
	if err := h.svc.UnshareBoard(c.Context(), currentUserID(c), pathID(c), domain.TargetCompany, int64(cid)); err != nil {
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
		Scene  json.RawMessage      `json:"scene"`
		Ops    json.RawMessage      `json:"ops"`
		Title  *string              `json:"title"`
	}
	parseBody(c, &body)
	if body.Scene != nil && !json.Valid(body.Scene) {
		return validationError(c, "Некорректная сцена доски")
	}
	if body.Ops != nil && !json.Valid(body.Ops) {
		return validationError(c, "Некорректные правки холста")
	}
	if body.Title != nil && len(*body.Title) > 1000 {
		return validationError(c, "Слишком длинное название")
	}
	if err := h.svc.Collab(c.Context(), currentUserID(c), pathID(c), body.Kind, body.Cursor, body.Scene, body.Ops, body.Title); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Картинки редактора, экспорт/импорт ───────────────────────────

/* Приём картинки холста частями: право проверяем ДО первого байта, собираем
   потоком. Место — из квоты владельца доски. */

func (h *handlers) beginUpload(c *fiber.Ctx, in chunkupload.InitRequest, s *chunkupload.Session) error {
	userID := currentUserID(c)
	if err := h.svc.CheckUpload(c.Context(), userID, pathID(c)); err != nil {
		return err
	}
	if in.Size > uploadMaxBytes {
		return domain.NewError("FILE_TOO_BIG", "Файл слишком большой (макс. 25 МБ)", 413)
	}
	s.QuotaUserID = userID
	// Доску запоминаем: на сборке путь уже другой (там код загрузки).
	s.Scope = strconv.FormatInt(pathID(c), 10)
	return nil
}

func (h *handlers) finishUpload(c *fiber.Ctx, s chunkupload.Session, r io.Reader) (any, error) {
	boardID, _ := strconv.ParseInt(s.Scope, 10, 64)
	path, err := h.svc.UploadStream(c.Context(), currentUserID(c), boardID, s.FileName, r, s.TotalSize)
	if err != nil {
		return nil, err
	}
	// Форма ответа — та же, что у одиночной загрузки: она не должна зависеть
	// от размера файла, иначе потребитель ломается ровно на больших.
	return fiber.Map{"path": path}, nil
}

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
	case service.FormatSVG:
		return "image/svg+xml"
	case "zip":
		return "application/zip"
	default:
		return "application/json; charset=utf-8"
	}
}

func sendFile(c *fiber.Ctx, f *service.ExportFile) error {
	c.Set(fiber.HeaderContentType, contentType(f.Ext))
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="board.`+f.Ext+`"; filename*=UTF-8''`+url.PathEscape(f.Name)+`.`+f.Ext)
	return c.Send(f.Data)
}

func (h *handlers) exportBoard(c *fiber.Ctx) error {
	f, err := h.svc.Export(c.Context(), currentUserID(c), pathID(c), c.Query("format"))
	if err != nil {
		return h.respondError(c, err)
	}
	return sendFile(c, f)
}

// exportAll — zip особой группировки (?scope=all|archive|shared&format=svg|json).
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

func (h *handlers) importBoard(c *fiber.Ctx) error {
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
	name := fileHeader.Filename
	isScene := strings.HasSuffix(strings.ToLower(name), ".json")
	title := strings.TrimSuffix(strings.TrimSuffix(name, ".json"), ".txt")

	var folderID *int64
	if fq := c.FormValue("folder_id"); fq != "" && fq != "root" {
		if id, e := strconv.ParseInt(fq, 10, 64); e == nil {
			folderID = &id
		}
	}
	resp, err := h.svc.Import(c.Context(), currentUserID(c), title, data, isScene, folderID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// setPreview — миниатюра холста для плитки списка (png-снимок делает клиент).
func (h *handlers) setPreview(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NO_FILE", "message": "Файл не передан"})
	}
	if fileHeader.Size > previewMaxBytes {
		return validationError(c, "Превью слишком большое")
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, previewMaxBytes+1))
	if err != nil {
		return h.respondError(c, err)
	}
	resp, err := h.svc.SetPreview(c.Context(), currentUserID(c), pathID(c), data)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"preview_url": resp.PreviewURL})
}

// ── Публичный доступ по коду (без авторизации) ───────────────────

func (h *handlers) sharedBoard(c *fiber.Ctx) error {
	resp, err := h.svc.GetSharedBoard(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedUpdate(c *fiber.Ctx) error {
	var body boardBody
	parseBody(c, &body)
	if !body.validate(c) {
		return nil
	}
	resp, err := h.svc.UpdateSharedBoard(c.Context(), c.Params("code"), body.Title, body.Scene)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

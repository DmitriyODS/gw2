package http

import (
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/drive/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
)

// ── Обзор ────────────────────────────────────────────────────────────────────

func (h *handlers) browse(c *fiber.Ctx) error {
	f := domain.ListFilter{
		Search:  strings.TrimSpace(c.Query("search")),
		Trash:   c.Query("view") == "trash",
		Starred: c.Query("view") == "starred",
		Recent:  c.Query("view") == "recent",
	}
	if id := int64(c.QueryInt("folder_id", 0)); id > 0 {
		f.FolderID = &id
	}
	out, err := h.svc.Browse(c.Context(), currentUserID(c), f)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) sharedWithMe(c *fiber.Ctx) error {
	out, err := h.svc.SharedWithMe(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) searchUsers(c *fiber.Ctx) error {
	users, err := h.svc.SearchUsers(c.Context(), strings.TrimSpace(c.Query("q")), 20)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": users})
}

// ── Папки ────────────────────────────────────────────────────────────────────

func (h *handlers) createFolder(c *fiber.Ctx) error {
	var body struct {
		Name     string `json:"name"`
		ParentID *int64 `json:"parent_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	out, err := h.svc.CreateFolder(c.Context(), currentUserID(c), body.Name, optionalID(body.ParentID))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *handlers) updateFolder(c *fiber.Ctx) error {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	out, err := h.svc.RenameFolder(c.Context(), currentUserID(c), pathID(c), body.Name, body.Color)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) moveFolder(c *fiber.Ctx) error {
	var body struct {
		ParentID *int64 `json:"parent_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	out, err := h.svc.MoveFolder(c.Context(), currentUserID(c), pathID(c), optionalID(body.ParentID))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) trashFolder(c *fiber.Ctx) error {
	if err := h.svc.TrashFolder(c.Context(), currentUserID(c), pathID(c), true); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) restoreFolder(c *fiber.Ctx) error {
	if err := h.svc.TrashFolder(c.Context(), currentUserID(c), pathID(c), false); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) purgeFolder(c *fiber.Ctx) error {
	if err := h.svc.PurgeFolder(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// ── Файлы ────────────────────────────────────────────────────────────────────

func (h *handlers) upload(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	src, err := fh.Open()
	if err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return h.respondError(c, domain.ErrValidation)
	}

	var folderID *int64
	if id := int64(c.QueryInt("folder_id", 0)); id > 0 {
		folderID = &id
	}
	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(fh.Filename)))
	}

	out, err := h.svc.Upload(c.Context(), currentUserID(c), fh.Filename, data, mimeType, folderID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

/* Загрузка по частям — для больших файлов: клиент заводит загрузку, шлёт
   куски по порядку и просит собрать. Тело куска идёт СЫРЫМИ байтами, без
   multipart: обёртка формы на каждый кусок — лишние проценты трафика и лишний
   разбор. */

/* Приём файла частями. Общая механика — в pkg/chunkupload; разделу остаётся
   сказать, куда кладём (и можно ли) и как собрать файл. */

func (h *handlers) beginUpload(c *fiber.Ctx, in chunkupload.InitRequest, s *chunkupload.Session) error {
	folderID := parseFolderScope(in.Scope)
	// Право на запись в папку и владельца квоты выясняем ДО первого байта:
	// незачем принимать полгигабайта, чтобы отказать на сборке.
	ownerID, _, err := h.svc.UploadTarget(c.Context(), currentUserID(c), in.FileName, folderID)
	if err != nil {
		return err
	}
	if in.Size > domain.MaxFileSize {
		return domain.ErrFileTooBig
	}
	s.QuotaUserID = ownerID
	return nil
}

func (h *handlers) finishUpload(c *fiber.Ctx, s chunkupload.Session, r io.Reader) (any, error) {
	return h.svc.UploadStream(c.Context(), currentUserID(c), s.FileName, r,
		s.Mime, s.TotalSize, parseFolderScope(s.Scope))
}

// parseFolderScope — папка назначения из контекста сессии ("" — корень диска).
func parseFolderScope(scope string) *int64 {
	id, err := strconv.ParseInt(scope, 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

func (h *handlers) getFile(c *fiber.Ctx) error {
	out, err := h.svc.Get(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) download(c *fiber.Ctx) error {
	file, data, err := h.svc.Download(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return sendFile(c, file, data, c.Query("inline") == "1")
}

func (h *handlers) updateFile(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	out, err := h.svc.Rename(c.Context(), currentUserID(c), pathID(c), body.Name)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) moveFile(c *fiber.Ctx) error {
	var body struct {
		FolderID *int64 `json:"folder_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	out, err := h.svc.Move(c.Context(), currentUserID(c), pathID(c), optionalID(body.FolderID))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) starFile(c *fiber.Ctx) error {
	var body struct {
		Starred bool `json:"starred"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	out, err := h.svc.Star(c.Context(), currentUserID(c), pathID(c), body.Starred)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) trashFile(c *fiber.Ctx) error {
	if err := h.svc.Trash(c.Context(), currentUserID(c), pathID(c), true); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) restoreFile(c *fiber.Ctx) error {
	if err := h.svc.Trash(c.Context(), currentUserID(c), pathID(c), false); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) purgeFile(c *fiber.Ctx) error {
	if err := h.svc.Purge(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) emptyTrash(c *fiber.Ctx) error {
	n, err := h.svc.EmptyTrash(c.Context(), currentUserID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": n})
}

// ── Доступ ───────────────────────────────────────────────────────────────────

func (h *handlers) fileAccessList(c *fiber.Ctx) error   { return h.accessList(c, fileTarget(c)) }
func (h *handlers) folderAccessList(c *fiber.Ctx) error { return h.accessList(c, folderTarget(c)) }

func (h *handlers) accessList(c *fiber.Ctx, t service.Target) error {
	links, people, err := h.svc.ListShares(c.Context(), currentUserID(c), t)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"links": links, "members": people})
}

func (h *handlers) shareFile(c *fiber.Ctx) error   { return h.share(c, fileTarget(c)) }
func (h *handlers) shareFolder(c *fiber.Ctx) error { return h.share(c, folderTarget(c)) }

func (h *handlers) share(c *fiber.Ctx, t service.Target) error {
	var body struct {
		UserID    *int64 `json:"user_id"`
		CompanyID *int64 `json:"company_id"`
		CanEdit   bool   `json:"can_edit"`
	}
	if err := c.BodyParser(&body); err != nil {
		return h.respondError(c, domain.ErrValidation)
	}
	out, err := h.svc.ShareTo(c.Context(), currentUserID(c), t,
		optionalID(body.UserID), optionalID(body.CompanyID), body.CanEdit)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *handlers) createFileLink(c *fiber.Ctx) error   { return h.createLink(c, fileTarget(c)) }
func (h *handlers) createFolderLink(c *fiber.Ctx) error { return h.createLink(c, folderTarget(c)) }

func (h *handlers) createLink(c *fiber.Ctx, t service.Target) error {
	out, err := h.svc.CreateShare(c.Context(), currentUserID(c), t)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *handlers) revokeAccess(c *fiber.Ctx) error {
	if err := h.svc.RevokeShare(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *handlers) deleteLink(c *fiber.Ctx) error {
	if err := h.svc.DeleteShare(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// ── Публичные ссылки ─────────────────────────────────────────────────────────

func (h *handlers) sharedTarget(c *fiber.Ctx) error {
	file, folder, err := h.svc.ByCode(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"file": file, "folder": folder})
}

func (h *handlers) sharedList(c *fiber.Ctx) error {
	out, err := h.svc.SharedListing(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(out)
}

func (h *handlers) sharedDownload(c *fiber.Ctx) error {
	file, data, err := h.svc.SharedDownload(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return sendFile(c, file, data, c.Query("inline") == "1")
}

// sendFile — отдать содержимое. inline нужен просмотру в разделе (картинка,
// PDF, видео открываются прямо в окне), остальное скачивается файлом.
func sendFile(c *fiber.Ctx, file *domain.File, data []byte, inline bool) error {
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	if file.Mime != "" {
		c.Set(fiber.HeaderContentType, file.Mime)
	}
	// filename* — имя в UTF-8: кириллица в обычном filename ломается.
	c.Set(fiber.HeaderContentDisposition,
		disposition+`; filename*=UTF-8''`+url.PathEscape(file.Name))
	return c.Send(data)
}

func fileTarget(c *fiber.Ctx) service.Target {
	id := pathID(c)
	return service.Target{FileID: &id}
}

func folderTarget(c *fiber.Ctx) service.Target {
	id := pathID(c)
	return service.Target{FolderID: &id}
}

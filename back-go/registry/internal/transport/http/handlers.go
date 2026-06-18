package http

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/service"
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

// ── Реестры ──────────────────────────────────────────────────────

func (h *handlers) listRegistries(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ListRegistries(c.Context(), endpoint.CompanyReq{CompanyID: companyID})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"registries": resp})
}

func (h *handlers) getRegistry(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.GetRegistry(c.Context(), endpoint.RegistryReq{CompanyID: companyID, ID: pathID(c)})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createRegistry(c *fiber.Ctx) error {
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
		return validationError(c, "Укажите название реестра")
	}
	if len([]rune(name)) > 120 {
		return validationError(c, "Название слишком длинное (макс. 120)")
	}
	resp, err := h.eps.CreateRegistry(c.Context(), endpoint.CreateRegistryReq{
		CompanyID: companyID, UserID: currentUser(c).ID, Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateRegistry(c *fiber.Ctx) error {
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
		return validationError(c, "Укажите название реестра")
	}
	if len([]rune(name)) > 120 {
		return validationError(c, "Название слишком длинное (макс. 120)")
	}
	resp, err := h.eps.UpdateRegistry(c.Context(), endpoint.UpdateRegistryReq{
		CompanyID: companyID, ID: pathID(c), Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteRegistry(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	if _, err := h.eps.DeleteRegistry(c.Context(), endpoint.RegistryReq{CompanyID: companyID, ID: pathID(c)}); err != nil {
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

func (h *handlers) listRecords(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ListRecords(c.Context(), endpoint.ListRecordsReq{
		CompanyID:  companyID,
		RegistryID: pathID(c),
		Params: service.RecordListParams{
			Search:  c.Query("search"),
			Sort:    c.Query("sort"),
			Order:   c.Query("order"),
			Page:    c.QueryInt("page", 1),
			PerPage: c.QueryInt("per_page", 30),
		},
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) getRecord(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.GetRecord(c.Context(), endpoint.RecordReq{
		CompanyID: companyID, RegistryID: pathID(c), RecordID: recordID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createRecord(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	parseBody(c, &body)
	if body.Data == nil {
		body.Data = map[string]any{}
	}
	resp, err := h.eps.CreateRecord(c.Context(), endpoint.WriteRecordReq{
		CompanyID: companyID, RegistryID: pathID(c), UserID: currentUser(c).ID, Data: body.Data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateRecord(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	parseBody(c, &body)
	if body.Data == nil {
		body.Data = map[string]any{}
	}
	resp, err := h.eps.UpdateRecord(c.Context(), endpoint.WriteRecordReq{
		CompanyID: companyID, RegistryID: pathID(c), RecordID: recordID(c), Data: body.Data,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteRecord(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	if _, err := h.eps.DeleteRecord(c.Context(), endpoint.RecordReq{
		CompanyID: companyID, RegistryID: pathID(c), RecordID: recordID(c),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) exportRecords(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	resp, err := h.eps.ExportRecords(c.Context(), endpoint.ExportReq{
		CompanyID:  companyID,
		RegistryID: pathID(c),
		FieldIDs:   csvInts(c.Query("fields")),
		Search:     c.Query("search"),
		IDs:        csvInts(c.Query("ids")),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	out := resp.(endpoint.ExportResp)
	c.Set(fiber.HeaderContentType, xlsxMime)
	// Имя файла из названия реестра: ascii-fallback + UTF-8 (RFC 5987).
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="registry.xlsx"; filename*=UTF-8''`+url.PathEscape(out.Name)+`.xlsx`)
	return c.Send(out.Data)
}

func (h *handlers) bulkDeleteRecords(c *fiber.Ctx) error {
	companyID, ok := companyScope(c)
	if !ok {
		return nil
	}
	var body struct {
		IDs []int64 `json:"ids"`
	}
	parseBody(c, &body)
	resp, err := h.eps.DeleteRecords(c.Context(), endpoint.DeleteRecordsReq{
		CompanyID: companyID, RegistryID: pathID(c), IDs: body.IDs,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": resp})
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
	resp, err := h.eps.ListShares(c.Context(), endpoint.ShareReq{CompanyID: companyID, RegistryID: pathID(c)})
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
		CompanyID: companyID, RegistryID: pathID(c), UserID: currentUser(c).ID,
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
		CompanyID: companyID, RegistryID: pathID(c), ShareID: int64(shareID),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// ── Публичный доступ по коду (без авторизации) ───────────────────

func (h *handlers) sharedRegistry(c *fiber.Ctx) error {
	resp, err := h.eps.SharedRegistry(c.Context(), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedRecords(c *fiber.Ctx) error {
	resp, err := h.eps.SharedRecords(c.Context(), endpoint.SharedRecordsReq{
		Code: c.Params("code"),
		Params: service.RecordListParams{
			Search:  c.Query("search"),
			Sort:    c.Query("sort"),
			Order:   c.Query("order"),
			Page:    c.QueryInt("page", 1),
			PerPage: c.QueryInt("per_page", 30),
		},
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) sharedExport(c *fiber.Ctx) error {
	resp, err := h.eps.SharedExport(c.Context(), endpoint.SharedExportReq{
		Code:     c.Params("code"),
		FieldIDs: csvInts(c.Query("fields")),
		Search:   c.Query("search"),
		IDs:      csvInts(c.Query("ids")),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	out := resp.(endpoint.ExportResp)
	c.Set(fiber.HeaderContentType, xlsxMime)
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="registry.xlsx"; filename*=UTF-8''`+url.PathEscape(out.Name)+`.xlsx`)
	return c.Send(out.Data)
}

// ── Парсинг и валидация полей реестра ────────────────────────────

type fieldInput struct {
	ID          int64          `json:"id"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	ColSpan     int            `json:"col_span"`
	RowSpan     int            `json:"row_span"`
	ShowInTable bool           `json:"show_in_table"`
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
			ColSpan: fi.ColSpan, RowSpan: fi.RowSpan, ShowInTable: fi.ShowInTable,
		})
	}
	return out, ""
}

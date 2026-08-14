package http

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/service"
)

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// singleRequestMax — потолок файла, принимаемого ОДНИМ запросом. Всё крупнее
// клиент обязан слать частями (chunkupload), поэтому здесь запас невелик.
const singleRequestMax = chunkupload.Threshold

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

/*
columnFilters — фильтры колонок из query.

	Формат: filter=<field_id>:<op>[:<value>[|<value>…]], параметр повторяется по
	разу на колонку. Значения разделены «|», а не запятой: в тексте фильтра
	запятая встречается сплошь и рядом.
*/
func columnFilters(c *fiber.Ctx) []domain.ColumnFilter {
	raw := c.Request().URI().QueryArgs().PeekMulti("filter")
	out := make([]domain.ColumnFilter, 0, len(raw))
	for _, item := range raw {
		parts := strings.SplitN(string(item), ":", 3)
		if len(parts) < 2 {
			continue
		}
		fieldID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		f := domain.ColumnFilter{FieldID: fieldID, Op: parts[1]}
		if len(parts) == 3 && parts[2] != "" {
			f.Values = strings.Split(parts[2], "|")
		}
		out = append(out, f)
	}
	return out
}

// listParams — общий разбор query списка записей (раздел и публичная ссылка).
func listParams(c *fiber.Ctx) service.RecordListParams {
	return service.RecordListParams{
		Search:  c.Query("search"),
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
		Section: c.Query("section"),
		Columns: columnFilters(c),
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 30),
	}
}

// selectionParams — набор записей под массовую операцию из query (выгрузка).
func selectionParams(c *fiber.Ctx) service.BulkParams {
	return service.BulkParams{
		IDs:     csvInts(c.Query("ids")),
		All:     c.Query("all") == "true",
		Search:  c.Query("search"),
		Section: c.Query("section"),
		Columns: columnFilters(c),
		Exclude: csvInts(c.Query("exclude")),
	}
}

func exportParams(c *fiber.Ctx) service.ExportParams {
	return service.ExportParams{
		FieldIDs:  csvInts(c.Query("fields")),
		Selection: selectionParams(c),
	}
}

func (h *handlers) sendXLSX(c *fiber.Ctx, data []byte, name string) error {
	c.Set(fiber.HeaderContentType, xlsxMime)
	// Имя файла из названия реестра: ascii-fallback + UTF-8 (RFC 5987).
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="registry.xlsx"; filename*=UTF-8''`+url.PathEscape(name)+`.xlsx`)
	return c.Send(data)
}

// ── Реестры ──────────────────────────────────────────────────────

// searchRecords — строка поиска Hola: записи всех доступных реестров одним
// запросом (свои, расшаренные лично и расшаренные компаниям).
func (h *handlers) searchRecords(c *fiber.Ctx) error {
	items, err := h.svc.SearchRecords(c.Context(), userID(c), c.Query("q"), c.QueryInt("limit"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) listRegistries(c *fiber.Ctx) error {
	regs, err := h.svc.ListRegistries(c.Context(), userID(c), c.Query("scope"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"registries": regs})
}

func (h *handlers) getRegistry(c *fiber.Ctx) error {
	reg, err := h.svc.GetRegistry(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(reg)
}

func (h *handlers) createRegistry(c *fiber.Ctx) error {
	var body struct {
		Name       string `json:"name"`
		Accounting bool   `json:"accounting"`
	}
	parseBody(c, &body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return validationError(c, "Укажите название реестра")
	}
	if len([]rune(name)) > 120 {
		return validationError(c, "Название слишком длинное (макс. 120)")
	}
	reg, err := h.svc.CreateRegistry(c.Context(), userID(c), activeCompany(c), name, body.Accounting)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(reg)
}

// registryPatch — разбор тела правки реестра. Второй результат — текст ошибки
// валидации ("" — успех). Общий для своего раздела и для внешней ссылки.
func registryPatch(c *fiber.Ctx) (service.RegistryPatch, string) {
	var body struct {
		Name string `json:"name"`
		// Указатель на указатель различает «ключа нет» (настройку не трогаем) и
		// явный null (выключить) — иначе переименование сбрасывало бы подразделы.
		SectionFieldID **int64 `json:"section_field_id"`
		Accounting     *bool   `json:"accounting"`
	}
	parseBody(c, &body)
	name := strings.TrimSpace(body.Name)
	if len([]rune(name)) > 120 {
		return service.RegistryPatch{}, "Название слишком длинное (макс. 120)"
	}
	p := service.RegistryPatch{Name: name}
	if body.SectionFieldID != nil {
		p.SectionFieldID, p.SectionFieldSet = *body.SectionFieldID, true
	}
	if body.Accounting != nil {
		p.Accounting, p.AccountingSet = *body.Accounting, true
	}
	return p, ""
}

func (h *handlers) updateRegistry(c *fiber.Ctx) error {
	p, msg := registryPatch(c)
	if msg != "" {
		return validationError(c, msg)
	}
	reg, err := h.svc.UpdateRegistry(c.Context(), userID(c), pathID(c), p)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(reg)
}

func (h *handlers) deleteRegistry(c *fiber.Ctx) error {
	if err := h.svc.DeleteRegistry(c.Context(), userID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) replaceFields(c *fiber.Ctx) error {
	var body struct {
		Fields []fieldInput `json:"fields"`
	}
	parseBody(c, &body)
	fields, msg := parseFields(body.Fields)
	if msg != "" {
		return validationError(c, msg)
	}
	reg, err := h.svc.ReplaceFields(c.Context(), userID(c), pathID(c), fields)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(reg)
}

// ── Записи ───────────────────────────────────────────────────────

func (h *handlers) listRecords(c *fiber.Ctx) error {
	list, err := h.svc.ListRecords(c.Context(), userID(c), pathID(c), listParams(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(list)
}

func (h *handlers) getRecord(c *fiber.Ctx) error {
	rec, err := h.svc.GetRecord(c.Context(), userID(c), pathID(c), recordID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(rec)
}

func (h *handlers) createRecord(c *fiber.Ctx) error {
	data := recordData(c)
	rec, err := h.svc.CreateRecord(c.Context(), userID(c), pathID(c), data)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(rec)
}

func (h *handlers) updateRecord(c *fiber.Ctx) error {
	rec, err := h.svc.UpdateRecord(c.Context(), userID(c), pathID(c), recordID(c), recordData(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(rec)
}

func (h *handlers) deleteRecord(c *fiber.Ctx) error {
	if err := h.svc.DeleteRecord(c.Context(), userID(c), pathID(c), recordID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) exportRecords(c *fiber.Ctx) error {
	data, name, err := h.svc.ExportRecords(c.Context(), userID(c), pathID(c), exportParams(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return h.sendXLSX(c, data, name)
}

func (h *handlers) bulkDeleteRecords(c *fiber.Ctx) error {
	// all=true — «выбрано всё по текущему фильтру»: список id тогда не нужен,
	// приходят только снятые галочки (exclude).
	var body struct {
		IDs     []int64               `json:"ids"`
		All     bool                  `json:"all"`
		Search  string                `json:"search"`
		Section string                `json:"section"`
		Filters []columnFilterPayload `json:"filters"`
		Exclude []int64               `json:"exclude"`
	}
	parseBody(c, &body)
	n, err := h.svc.DeleteRecords(c.Context(), userID(c), pathID(c), service.BulkParams{
		IDs: body.IDs, All: body.All, Search: body.Search, Section: body.Section,
		Columns: toColumnFilters(body.Filters), Exclude: body.Exclude,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": n})
}

// recordData — значения полей из тела запроса (nil трактуем как пустую карту:
// «очистить всё» — законная правка).
func recordData(c *fiber.Ctx) map[string]any {
	var body struct {
		Data map[string]any `json:"data"`
	}
	parseBody(c, &body)
	if body.Data == nil {
		return map[string]any{}
	}
	return body.Data
}

type columnFilterPayload struct {
	FieldID int64    `json:"field_id"`
	Op      string   `json:"op"`
	Values  []string `json:"values"`
}

func toColumnFilters(in []columnFilterPayload) []domain.ColumnFilter {
	out := make([]domain.ColumnFilter, 0, len(in))
	for _, f := range in {
		out = append(out, domain.ColumnFilter{FieldID: f.FieldID, Op: f.Op, Values: f.Values})
	}
	return out
}

// ── Учётный реестр ───────────────────────────────────────────────

func (h *handlers) issueHistory(c *fiber.Ctx) error {
	items, err := h.svc.IssueHistory(c.Context(), userID(c), pathID(c), recordID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"issues": items})
}

// issueParams — разбор тела выдачи. Второй результат — текст ошибки ("" —
// успех). Общий для своего раздела и для внешней ссылки.
func issueParams(c *fiber.Ctx) (service.IssueParams, string) {
	var body struct {
		IssuedTo     string  `json:"issued_to"`
		HolderName   string  `json:"holder_name"`
		HolderPhone  string  `json:"holder_phone"`
		HolderUserID *int64  `json:"holder_user_id"`
		DueAt        *string `json:"due_at"`
		Comment      string  `json:"comment"`
	}
	parseBody(c, &body)
	/* Ответственный и его телефон обязательны: выдача под запись «неизвестно
	   кому» отчётности не даёт, а искать вещь потом не по чему. Проверяем и
	   здесь, а не только на форме, — REST открыт и мимо неё. */
	if strings.TrimSpace(body.IssuedTo) == "" {
		return service.IssueParams{}, "Укажите, кому выдаём"
	}
	if strings.TrimSpace(body.HolderName) == "" {
		return service.IssueParams{}, "Укажите ФИО ответственного"
	}
	if !records.ValidPhone(body.HolderPhone) {
		return service.IssueParams{}, "Укажите телефон ответственного"
	}
	dueAt, err := parseTime(body.DueAt)
	if err != nil {
		return service.IssueParams{}, "Непонятная дата возврата"
	}
	return service.IssueParams{
		IssuedTo: body.IssuedTo, HolderName: body.HolderName, HolderPhone: body.HolderPhone,
		HolderUserID: body.HolderUserID, DueAt: dueAt, Comment: body.Comment,
	}, ""
}

// extendParams — срок и комментарий продления.
func extendParams(c *fiber.Ctx) (*time.Time, string, string) {
	var body struct {
		DueAt   *string `json:"due_at"`
		Comment string  `json:"comment"`
	}
	parseBody(c, &body)
	dueAt, err := parseTime(body.DueAt)
	if err != nil {
		return nil, "", "Непонятная дата возврата"
	}
	return dueAt, body.Comment, ""
}

func commentParam(c *fiber.Ctx) string {
	var body struct {
		Comment string `json:"comment"`
	}
	parseBody(c, &body)
	return body.Comment
}

func (h *handlers) issue(c *fiber.Ctx) error {
	p, msg := issueParams(c)
	if msg != "" {
		return validationError(c, msg)
	}
	issue, err := h.svc.Issue(c.Context(), userID(c), pathID(c), recordID(c), p)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(issue)
}

func (h *handlers) extendIssue(c *fiber.Ctx) error {
	dueAt, comment, msg := extendParams(c)
	if msg != "" {
		return validationError(c, msg)
	}
	issue, err := h.svc.Extend(c.Context(), userID(c), pathID(c), recordID(c), dueAt, comment)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(issue)
}

func (h *handlers) returnIssue(c *fiber.Ctx) error {
	issue, err := h.svc.Return(c.Context(), userID(c), pathID(c), recordID(c), commentParam(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(issue)
}

// ── Учётный реестр по внешней ссылке ──

func (h *handlers) sharedIssueHistory(c *fiber.Ctx) error {
	items, err := h.svc.SharedIssueHistory(c.Context(), c.Params("code"), h.visitor(c), recordID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"issues": items})
}

func (h *handlers) sharedIssue(c *fiber.Ctx) error {
	p, msg := issueParams(c)
	if msg != "" {
		return validationError(c, msg)
	}
	issue, err := h.svc.SharedIssue(c.Context(), c.Params("code"), h.visitor(c), recordID(c), p)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(issue)
}

func (h *handlers) sharedExtendIssue(c *fiber.Ctx) error {
	dueAt, comment, msg := extendParams(c)
	if msg != "" {
		return validationError(c, msg)
	}
	issue, err := h.svc.SharedExtend(c.Context(), c.Params("code"), h.visitor(c), recordID(c), dueAt, comment)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(issue)
}

func (h *handlers) sharedReturnIssue(c *fiber.Ctx) error {
	issue, err := h.svc.SharedReturn(c.Context(), c.Params("code"), h.visitor(c), recordID(c), commentParam(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(issue)
}

// parseTime — необязательная отметка времени из тела (ISO 8601). Пустая строка
// и отсутствие ключа означают «без срока».
func parseTime(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*s))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ── Загрузка файла ───────────────────────────────────────────────

// uploadPayload — прочитать файл формы с проверкой размера. ok=false — ответ
// клиенту уже записан.
func (h *handlers) uploadPayload(c *fiber.Ctx) (string, string, []byte, bool) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NO_FILE", "message": "Файл не передан"})
		return "", "", nil, false
	}
	tooBig := func() {
		validationError(c, "Такой файл нужно загружать частями")
	}
	if fileHeader.Size > singleRequestMax {
		tooBig()
		return "", "", nil, false
	}
	f, err := fileHeader.Open()
	if err != nil {
		h.respondError(c, err)
		return "", "", nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, singleRequestMax+1))
	if err != nil {
		h.respondError(c, err)
		return "", "", nil, false
	}
	if int64(len(data)) > singleRequestMax {
		tooBig()
		return "", "", nil, false
	}
	return fileHeader.Filename, fileHeader.Header.Get(fiber.HeaderContentType), data, true
}

func (h *handlers) upload(c *fiber.Ctx) error {
	name, mime, data, ok := h.uploadPayload(c)
	if !ok {
		return nil
	}
	registryID, err := strconv.ParseInt(c.Query("registry_id"), 10, 64)
	if err != nil {
		return validationError(c, "Не указан реестр")
	}
	file, err := h.svc.Upload(c.Context(), userID(c), registryID, name, mime, data)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(file)
}

// ── Чанковая загрузка ────────────────────────────────────────────

func (h *handlers) beginUpload(c *fiber.Ctx) error {
	var body struct {
		RegistryID int64  `json:"registry_id"`
		FileName   string `json:"file_name"`
		Mime       string `json:"mime"`
		Size       int64  `json:"size"`
	}
	parseBody(c, &body)
	if strings.TrimSpace(body.FileName) == "" {
		return validationError(c, "Не указано имя файла")
	}
	sess, err := h.svc.BeginUpload(c.Context(), userID(c), body.RegistryID,
		body.FileName, body.Mime, body.Size)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(sess)
}

// writeChunk — часть загрузки приходит СЫРЫМ телом, а не формой: обёртка
// multipart на каждый кусок — лишние проценты трафика и лишняя сборка в памяти.
func (h *handlers) writeChunk(c *fiber.Ctx) error {
	index := c.QueryInt("index", -1)
	if index < 0 {
		return validationError(c, "Не указан номер части")
	}
	sess, err := h.svc.WriteChunk(c.Context(), userID(c), c.Params("code"), index, c.Body())
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(sess)
}

func (h *handlers) finishUpload(c *fiber.Ctx) error {
	file, err := h.svc.FinishUpload(c.Context(), userID(c), c.Params("code"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(file)
}

func (h *handlers) cancelUpload(c *fiber.Ctx) error {
	if err := h.svc.CancelUpload(c.Context(), userID(c), c.Params("code")); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"cancelled": true})
}

// ── Шаринг: внешние ссылки ───────────────────────────────────────

func (h *handlers) listShares(c *fiber.Ctx) error {
	shares, err := h.svc.ListShares(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"shares": shares})
}

func shareParams(c *fiber.Ctx) service.ShareParams {
	var body struct {
		Name        string `json:"name"`
		Access      string `json:"access"`
		RequireAuth bool   `json:"require_auth"`
	}
	parseBody(c, &body)
	return service.ShareParams{Name: body.Name, Access: body.Access, RequireAuth: body.RequireAuth}
}

func (h *handlers) createShare(c *fiber.Ctx) error {
	share, err := h.svc.CreateShare(c.Context(), userID(c), pathID(c), shareParams(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(share)
}

func (h *handlers) updateShare(c *fiber.Ctx) error {
	if err := h.svc.UpdateShare(c.Context(), userID(c), pathID(c), shareID(c), shareParams(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"updated": true})
}

func (h *handlers) revokeShare(c *fiber.Ctx) error {
	if err := h.svc.RevokeShare(c.Context(), userID(c), pathID(c), shareID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) shareVisits(c *fiber.Ctx) error {
	visits, err := h.svc.ShareVisits(c.Context(), userID(c), pathID(c), shareID(c), c.QueryInt("limit"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"visits": visits})
}

// ── Шаринг: люди и компании ──────────────────────────────────────

func (h *handlers) listUserShares(c *fiber.Ctx) error {
	shares, err := h.svc.ListUserShares(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"access": shares})
}

func (h *handlers) shareWith(c *fiber.Ctx) error {
	var body struct {
		Targets []struct {
			UserID    *int64 `json:"user_id"`
			CompanyID *int64 `json:"company_id"`
			Access    string `json:"access"`
		} `json:"targets"`
	}
	parseBody(c, &body)
	targets := make([]service.ShareTarget, 0, len(body.Targets))
	for _, t := range body.Targets {
		targets = append(targets, service.ShareTarget{
			UserID: t.UserID, CompanyID: t.CompanyID, Access: t.Access,
		})
	}
	shares, err := h.svc.ShareWith(c.Context(), userID(c), pathID(c), targets)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"access": shares})
}

func (h *handlers) unshare(c *fiber.Ctx) error {
	var target, company *int64
	if v := c.QueryInt("user_id"); v > 0 {
		id := int64(v)
		target = &id
	}
	if v := c.QueryInt("company_id"); v > 0 {
		id := int64(v)
		company = &id
	}
	if target == nil && company == nil {
		return validationError(c, "Не указано, у кого отозвать доступ")
	}
	if err := h.svc.Unshare(c.Context(), userID(c), pathID(c), target, company); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) directory(c *fiber.Ctx) error {
	users, err := h.svc.Directory(c.Context(), userID(c), c.Query("q"), c.QueryInt("limit"))
	if err != nil {
		return h.respondError(c, err)
	}
	// Наружу — только то, что нужно списку выбора: имя и аватар.
	items := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		items = append(items, fiber.Map{"id": u.ID, "fio": u.FIO, "avatar_path": u.AvatarPath})
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) companies(c *fiber.Ctx) error {
	items, err := h.svc.Companies(c.Context(), userID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

// ── Публичный доступ по коду ─────────────────────────────────────

func (h *handlers) sharedRegistry(c *fiber.Ctx) error {
	view, err := h.svc.SharedRegistry(c.Context(), c.Params("code"), h.visitor(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(view)
}

func (h *handlers) sharedRecords(c *fiber.Ctx) error {
	list, err := h.svc.SharedRecords(c.Context(), c.Params("code"), h.visitor(c), listParams(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(list)
}

func (h *handlers) sharedExport(c *fiber.Ctx) error {
	data, name, err := h.svc.SharedExport(c.Context(), c.Params("code"), h.visitor(c), exportParams(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return h.sendXLSX(c, data, name)
}

func (h *handlers) sharedCreateRecord(c *fiber.Ctx) error {
	rec, err := h.svc.SharedCreateRecord(c.Context(), c.Params("code"), h.visitor(c), recordData(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(rec)
}

func (h *handlers) sharedUpdateRecord(c *fiber.Ctx) error {
	rec, err := h.svc.SharedUpdateRecord(c.Context(), c.Params("code"), h.visitor(c),
		recordID(c), recordData(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(rec)
}

func (h *handlers) sharedDeleteRecord(c *fiber.Ctx) error {
	if err := h.svc.SharedDeleteRecord(c.Context(), c.Params("code"), h.visitor(c), recordID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) sharedUpdateRegistry(c *fiber.Ctx) error {
	p, msg := registryPatch(c)
	if msg != "" {
		return validationError(c, msg)
	}
	reg, err := h.svc.SharedUpdateRegistry(c.Context(), c.Params("code"), h.visitor(c), p)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(reg)
}

func (h *handlers) sharedReplaceFields(c *fiber.Ctx) error {
	var body struct {
		Fields []fieldInput `json:"fields"`
	}
	parseBody(c, &body)
	fields, msg := parseFields(body.Fields)
	if msg != "" {
		return validationError(c, msg)
	}
	reg, err := h.svc.SharedReplaceFields(c.Context(), c.Params("code"), h.visitor(c), fields)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(reg)
}

func (h *handlers) sharedUpload(c *fiber.Ctx) error {
	name, mime, data, ok := h.uploadPayload(c)
	if !ok {
		return nil
	}
	file, err := h.svc.SharedUpload(c.Context(), c.Params("code"), h.visitor(c), name, mime, data)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(file)
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

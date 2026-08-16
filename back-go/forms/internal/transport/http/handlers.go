package http

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/forms/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
)

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// singleRequestMax — потолок файла, принимаемого ОДНИМ запросом. Всё крупнее
// клиент обязан слать частями (chunkupload).
const singleRequestMax = chunkupload.Threshold

func parseBody(c *fiber.Ctx, out any) { _ = json.Unmarshal(c.Body(), out) }

func validationError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "VALIDATION", "message": msg})
}

// parseTime — необязательная отметка времени из тела (ISO 8601). Пустая строка
// означает «убрать срок».
func parseTime(raw json.RawMessage) (*time.Time, error) {
	var s *string
	if err := json.Unmarshal(raw, &s); err != nil || s == nil {
		return nil, err
	}
	value := strings.TrimSpace(*s)
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ── Формы ────────────────────────────────────────────────────────

func (h *handlers) listForms(c *fiber.Ctx) error {
	forms, err := h.svc.ListForms(c.Context(), userID(c), c.Query("scope"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"forms": forms})
}

func (h *handlers) getForm(c *fiber.Ctx) error {
	form, err := h.svc.GetForm(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(form)
}

func (h *handlers) createForm(c *fiber.Ctx) error {
	var body struct {
		Title string `json:"title"`
		Quiz  bool   `json:"quiz"`
	}
	parseBody(c, &body)
	title := strings.TrimSpace(body.Title)
	if title == "" {
		return validationError(c, "Укажите название формы")
	}
	if len([]rune(title)) > 200 {
		return validationError(c, "Название слишком длинное (макс. 200)")
	}
	form, err := h.svc.CreateForm(c.Context(), userID(c), activeCompany(c), title, body.Quiz)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(form)
}

/*
formPatch — разбор тела правки формы.

	Тело читается картой сырых значений: «ключа нет» означает «не трогать», и
	отличить это от «выключить» иначе нельзя — переименование сбрасывало бы
	настройки приёма ответов.
*/
func formPatch(c *fiber.Ctx) (service.FormPatch, string) {
	raw := map[string]json.RawMessage{}
	_ = json.Unmarshal(c.Body(), &raw)

	var p service.FormPatch
	str := func(key string) *string {
		v, ok := raw[key]
		if !ok {
			return nil
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return nil
		}
		return &s
	}
	flag := func(key string) *bool {
		v, ok := raw[key]
		if !ok {
			return nil
		}
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return nil
		}
		return &b
	}

	if title := str("title"); title != nil {
		if len([]rune(strings.TrimSpace(*title))) > 200 {
			return p, "Название слишком длинное (макс. 200)"
		}
		p.Title = title
	}
	p.Description = str("description")
	p.Status = str("status")
	p.Confirmation = str("confirmation")
	p.QuizRelease = str("quiz_release")
	p.AllowAnonymous = flag("allow_anonymous")
	p.OneResponse = flag("one_response")
	p.AllowEdit = flag("allow_edit")
	p.CollectEmail = flag("collect_email")
	p.CollectName = flag("collect_name")
	p.ShowProgress = flag("show_progress")
	p.ShuffleQuestions = flag("shuffle_questions")
	p.ShowSummary = flag("show_summary")
	p.Quiz = flag("quiz")
	p.QuizShowAnswers = flag("quiz_show_answers")

	if v, ok := raw["max_responses"]; ok {
		var n int
		if err := json.Unmarshal(v, &n); err == nil && n >= 0 {
			p.MaxResponses = &n
		}
	}
	// Сроки приёма: пришедший ключ с пустым значением означает «убрать срок»,
	// поэтому у них указатель двойной.
	if v, ok := raw["opens_at"]; ok {
		t, err := parseTime(v)
		if err != nil {
			return p, "Непонятная дата начала приёма ответов"
		}
		p.OpensAt = &t
	}
	if v, ok := raw["closes_at"]; ok {
		t, err := parseTime(v)
		if err != nil {
			return p, "Непонятная дата окончания приёма ответов"
		}
		p.ClosesAt = &t
	}
	return p, ""
}

func (h *handlers) updateForm(c *fiber.Ctx) error {
	p, msg := formPatch(c)
	if msg != "" {
		return validationError(c, msg)
	}
	form, err := h.svc.UpdateForm(c.Context(), userID(c), pathID(c), p)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(form)
}

func (h *handlers) deleteForm(c *fiber.Ctx) error {
	if err := h.svc.DeleteForm(c.Context(), userID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) duplicateForm(c *fiber.Ctx) error {
	form, err := h.svc.DuplicateForm(c.Context(), userID(c), pathID(c), activeCompany(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(form)
}

// ── Структура ────────────────────────────────────────────────────

// sectionInput / questionInput — тело сохранения структуры. Ветвление приходит
// ПОЗИЦИЯМИ разделов (next_index у раздела, "#<позиция>" в targets вопроса): у
// только что добавленного раздела id ещё нет.
type sectionInput struct {
	ID          int64           `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	NextAction  string          `json:"next_action"`
	NextIndex   *int            `json:"next_index"`
	// Условие показа раздела: вопрос-источник и ожидаемые ответы. Источником
	// бывает только УЖЕ сохранённый вопрос — у нового id ещё нет.
	VisibleQuestionID *int64          `json:"visible_question_id"`
	VisibleValues     []string        `json:"visible_values"`
	Questions         []questionInput `json:"questions"`
}

type questionInput struct {
	ID          int64          `json:"id"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Config      map[string]any `json:"config"`
	Points      int            `json:"points"`
	AnswerKey   map[string]any `json:"answer_key"`
}

// parseStructure — привести тело к доменным разделам. Второй результат — текст
// ошибки валидации ("" — успех).
func parseStructure(in []sectionInput) ([]domain.Section, string) {
	out := make([]domain.Section, 0, len(in))
	for _, s := range in {
		if len([]rune(s.Title)) > 200 {
			return nil, "Название раздела слишком длинное (макс. 200)"
		}
		section := domain.Section{
			ID: s.ID, Title: strings.TrimSpace(s.Title),
			Description: strings.TrimSpace(s.Description),
			NextAction:  s.NextAction, NextIndex: -1,
			VisibleQuestionID: s.VisibleQuestionID, VisibleValues: s.VisibleValues,
		}
		if s.NextIndex != nil {
			section.NextIndex = *s.NextIndex
		}
		for _, q := range s.Questions {
			if !domain.QuestionTypes[q.Type] {
				return nil, "Неизвестный тип вопроса: " + q.Type
			}
			if len([]rune(q.Title)) > 500 {
				return nil, "Текст вопроса слишком длинный (макс. 500)"
			}
			section.Questions = append(section.Questions, domain.Question{
				ID: q.ID, Type: q.Type, Title: strings.TrimSpace(q.Title),
				Description: strings.TrimSpace(q.Description), Required: q.Required,
				Config: q.Config, Points: q.Points, AnswerKey: q.AnswerKey,
			})
		}
		out = append(out, section)
	}
	return out, ""
}

func (h *handlers) replaceStructure(c *fiber.Ctx) error {
	var body struct {
		Sections []sectionInput `json:"sections"`
	}
	parseBody(c, &body)
	sections, msg := parseStructure(body.Sections)
	if msg != "" {
		return validationError(c, msg)
	}
	form, err := h.svc.ReplaceStructure(c.Context(), userID(c), pathID(c), sections)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(form)
}

// ── Заполнение и ответы ──────────────────────────────────────────

func (h *handlers) fill(c *fiber.Ctx) error {
	view, err := h.svc.Fill(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(view)
}

// submitInput — тело отправки ответа.
func submitInput(c *fiber.Ctx) service.SubmitInput {
	var body struct {
		Answers map[string]any `json:"answers"`
		Email   string         `json:"email"`
		Name    string         `json:"name"`
	}
	parseBody(c, &body)
	if body.Answers == nil {
		body.Answers = map[string]any{}
	}
	return service.SubmitInput{Answers: body.Answers, Email: body.Email, Name: body.Name}
}

func (h *handlers) submit(c *fiber.Ctx) error {
	result, err := h.svc.Submit(c.Context(), userID(c), pathID(c), submitInput(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *handlers) updateMine(c *fiber.Ctx) error {
	result, err := h.svc.UpdateMine(c.Context(), userID(c), pathID(c), submitInput(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(result)
}

func (h *handlers) listResponses(c *fiber.Ctx) error {
	list, err := h.svc.ListResponses(c.Context(), userID(c), pathID(c), domain.ResponseListFilter{
		Search:  c.Query("search"),
		Sort:    c.Query("sort"),
		Desc:    !strings.EqualFold(c.Query("order"), "asc"),
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 30),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(list)
}

func (h *handlers) getResponse(c *fiber.Ctx) error {
	resp, err := h.svc.GetResponse(c.Context(), userID(c), pathID(c), responseID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteResponse(c *fiber.Ctx) error {
	if err := h.svc.DeleteResponse(c.Context(), userID(c), pathID(c), responseID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) bulkDeleteResponses(c *fiber.Ctx) error {
	var body struct {
		IDs []int64 `json:"ids"`
		All bool    `json:"all"`
	}
	parseBody(c, &body)
	n, err := h.svc.DeleteResponses(c.Context(), userID(c), pathID(c), body.IDs, body.All)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": n})
}

func (h *handlers) publishGrades(c *fiber.Ctx) error {
	var body struct {
		ResponseID int64 `json:"response_id"`
	}
	parseBody(c, &body)
	if err := h.svc.PublishGrades(c.Context(), userID(c), pathID(c), body.ResponseID); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"published": true})
}

func (h *handlers) summary(c *fiber.Ctx) error {
	sum, err := h.svc.Summary(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(sum)
}

func (h *handlers) progress(c *fiber.Ctx) error {
	progress, err := h.svc.Progress(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(progress)
}

func (h *handlers) exportResponses(c *fiber.Ctx) error {
	data, title, err := h.svc.ExportResponses(c.Context(), userID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	c.Set(fiber.HeaderContentType, xlsxMime)
	// Имя файла из названия формы: ascii-fallback + UTF-8 (RFC 5987).
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename="form.xlsx"; filename*=UTF-8''`+url.PathEscape(title)+`.xlsx`)
	return c.Send(data)
}

func (h *handlers) searchForms(c *fiber.Ctx) error {
	items, err := h.svc.SearchForms(c.Context(), userID(c), c.Query("q"), c.QueryInt("limit"))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

// ── Публичный доступ по коду ─────────────────────────────────────

func (h *handlers) sharedForm(c *fiber.Ctx) error {
	view, err := h.svc.SharedForm(c.Context(), c.Params("code"), h.visitor(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(view)
}

func (h *handlers) sharedSubmit(c *fiber.Ctx) error {
	result, err := h.svc.SharedSubmit(c.Context(), c.Params("code"), h.visitor(c), submitInput(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(result)
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

// ── Загрузка файлов ──────────────────────────────────────────────

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
	formID, err := strconv.ParseInt(c.Query("form_id"), 10, 64)
	if err != nil {
		return validationError(c, "Не указана форма")
	}
	questionID, _ := strconv.ParseInt(c.Query("question_id"), 10, 64)
	file, err := h.svc.Upload(c.Context(), userID(c), formID, questionID, name, mime, data)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(file)
}

func (h *handlers) beginUpload(c *fiber.Ctx) error {
	var body struct {
		FormID     int64  `json:"form_id"`
		QuestionID int64  `json:"question_id"`
		FileName   string `json:"file_name"`
		Mime       string `json:"mime"`
		Size       int64  `json:"size"`
	}
	parseBody(c, &body)
	if strings.TrimSpace(body.FileName) == "" {
		return validationError(c, "Не указано имя файла")
	}
	sess, err := h.svc.BeginUpload(c.Context(), userID(c), body.FormID, body.QuestionID,
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
		RequireAuth bool   `json:"require_auth"`
	}
	parseBody(c, &body)
	return service.ShareParams{Name: body.Name, RequireAuth: body.RequireAuth}
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

// ── Адресный доступ и назначения ─────────────────────────────────

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
			UserID    *int64          `json:"user_id"`
			CompanyID *int64          `json:"company_id"`
			Access    string          `json:"access"`
			DueAt     json.RawMessage `json:"due_at"`
		} `json:"targets"`
	}
	parseBody(c, &body)

	targets := make([]service.ShareTarget, 0, len(body.Targets))
	for _, t := range body.Targets {
		target := service.ShareTarget{UserID: t.UserID, CompanyID: t.CompanyID, Access: t.Access}
		if len(t.DueAt) > 0 {
			due, err := parseTime(t.DueAt)
			if err != nil {
				return validationError(c, "Непонятный срок ответа")
			}
			target.DueAt = due
		}
		targets = append(targets, target)
	}
	shares, err := h.svc.ShareWith(c.Context(), userID(c), pathID(c), targets)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"access": shares})
}

func (h *handlers) unshare(c *fiber.Ctx) error {
	var userTarget, companyTarget *int64
	if id, err := strconv.ParseInt(c.Query("user_id"), 10, 64); err == nil {
		userTarget = &id
	}
	if id, err := strconv.ParseInt(c.Query("company_id"), 10, 64); err == nil {
		companyTarget = &id
	}
	if err := h.svc.Unshare(c.Context(), userID(c), pathID(c), userTarget, companyTarget); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *handlers) directory(c *fiber.Ctx) error {
	users, err := h.svc.Directory(c.Context(), userID(c), c.Query("q"), c.QueryInt("limit"))
	if err != nil {
		return h.respondError(c, err)
	}
	items := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		items = append(items, fiber.Map{"id": u.ID, "fio": u.FIO, "avatar_path": u.AvatarPath})
	}
	return c.JSON(fiber.Map{"users": items})
}

func (h *handlers) companies(c *fiber.Ctx) error {
	companies, err := h.svc.Companies(c.Context(), userID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"companies": companies})
}

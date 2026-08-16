package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

// maxPerPage — потолок страницы ответов (выгрузка целиком идёт через xlsx).
const maxPerPage = 200

// FillView — форма глазами отвечающего: структура без правильных ответов,
// состояние приёма и его собственный прошлый ответ, если правка разрешена.
type FillView struct {
	Form *domain.Form `json:"form"`
	// CanRespond — принимает ли форма ответ ИМЕННО ОТ ЭТОГО человека сейчас.
	CanRespond bool   `json:"can_respond"`
	Reason     string `json:"reason,omitempty"` // почему нельзя (человеку)
	// Mine — уже отправленный ответ (для повторного показа и правки).
	Mine *domain.Response `json:"mine,omitempty"`
	// AnswerKeys — правильные ответы: приходят только вместе с открытой
	// оценкой теста, до неё их не видит никто, кроме автора формы.
	AnswerKeys map[string]any `json:"answer_keys,omitempty"`
	/* Booking — занятые места вопросов «Запись»: {ключ вопроса: {вариант:
	   занято}}. Остаток показывает форма («осталось 3 из 10»), а сверяет его
	   всё равно сервер при отправке. */
	Booking map[string]map[string]int `json:"booking,omitempty"`
}

// Fill — форма для заполнения участником (уровень respond и выше).
func (s *Service) Fill(ctx context.Context, userID, formID int64) (*FillView, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, formID, domain.AccessRespond)
	if err != nil {
		return nil, err
	}
	return s.fillView(ctx, form, &userID)
}

// fillView — общее ядро показа формы отвечающему (участнику и гостю по ссылке).
func (s *Service) fillView(ctx context.Context, form *domain.Form, userID *int64) (*FillView, error) {
	sections, err := s.repo.ListSections(ctx, form.ID)
	if err != nil {
		return nil, err
	}
	form.Sections = stripAnswerKeys(sections)

	view := &FillView{Form: form, CanRespond: true}
	// Остатки мест нужны ещё до ответа: занятые варианты форма показывает
	// недоступными, а не отказывает уже после отправки.
	booking, err := s.bookingCounts(ctx, form.ID, sections, 0)
	if err != nil {
		return nil, err
	}
	view.Booking = booking
	if err := s.acceptable(ctx, form); err != nil {
		view.CanRespond = false
		if de := domain.AsDomainError(err); de != nil {
			view.Reason = de.Message
		}
	}
	if userID == nil || *userID == 0 {
		if !form.AllowAnonymous {
			view.CanRespond, view.Reason = false, domain.ErrAuthRequired.Message
		}
		return view, nil
	}

	mine, err := s.repo.ResponseOfUser(ctx, form.ID, *userID)
	if err != nil {
		return nil, err
	}
	if mine == nil {
		return view, nil
	}
	view.Mine = mine
	form.MyResponded = true
	/* Повторную отправку закрывает ТОЛЬКО «один ответ от человека»: без неё
	   форму заполняют сколько угодно раз, и каждая отправка — отдельный ответ.
	   «Разрешить менять свой ответ» — про правку уже отправленного, а не про
	   право ответить снова. */
	if form.OneResponse {
		view.CanRespond = false
		view.Reason = "Вы уже отвечали на эту форму"
	}
	if form.Quiz && mine.Graded && form.QuizShowAnswers {
		view.AnswerKeys = answerKeys(sections)
	}
	return view, nil
}

// bookingCounts — занятость мест по всем вопросам «Запись» формы (пусто, если
// таких вопросов нет). exceptResponse — не считать свой прежний ответ (правка).
func (s *Service) bookingCounts(ctx context.Context, formID int64, sections []domain.Section, exceptResponse int64) (map[string]map[string]int, error) {
	keys := []string{}
	for _, q := range domain.AllQuestions(sections) {
		if domain.Bookable(q.Type) {
			keys = append(keys, domain.QuestionID(q.ID))
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return s.repo.BookingCounts(ctx, formID, keys, exceptResponse)
}

/*
bookings — какие места занимает этот ответ.

	Потолок берётся из самого вопроса, а остаток проверяет репозиторий под
	локом формы: между показом «осталось одно» и отправкой это место мог занять
	другой человек.
*/
func bookings(questions []domain.Question, answers map[string]any) []domain.Booking {
	out := []domain.Booking{}
	for _, q := range questions {
		if !domain.Bookable(q.Type) {
			continue
		}
		option := domain.Text(answers[domain.QuestionID(q.ID)])
		if option == "" {
			continue
		}
		out = append(out, domain.Booking{
			QuestionKey: domain.QuestionID(q.ID),
			Option:      option,
			Capacity:    q.Capacity(option),
		})
	}
	return out
}

// answerKeys — правильные ответы и пояснения по вопросам (открываются вместе с
// оценкой теста).
func answerKeys(sections []domain.Section) map[string]any {
	out := map[string]any{}
	for _, q := range domain.AllQuestions(sections) {
		if len(q.AnswerKey) > 0 {
			out[domain.QuestionID(q.ID)] = q.AnswerKey
		}
	}
	return out
}

/*
acceptable — принимает ли форма ответы прямо сейчас.

	Проверка одна на все пути (раздел, внешняя ссылка) и живёт на сервере: и
	черновик, и окно приёма, и потолок ответов клиент показывает, но решает не он.
*/
func (s *Service) acceptable(ctx context.Context, form *domain.Form) error {
	if form.Status != domain.StatusOpen {
		if form.Status == domain.StatusClosed {
			return domain.ErrClosed
		}
		return domain.ErrNotOpen
	}
	now := time.Now()
	if form.OpensAt != nil && now.Before(*form.OpensAt) {
		return domain.ErrNotStarted
	}
	if form.ClosesAt != nil && now.After(*form.ClosesAt) {
		return domain.ErrClosed
	}
	if form.MaxResponses > 0 {
		count, err := s.repo.CountResponses(ctx, form.ID)
		if err != nil {
			return err
		}
		if count >= form.MaxResponses {
			return domain.ErrLimitReached
		}
	}
	return nil
}

// SubmitInput — отправка ответа. UserID пуст у гостя: аккаунта у него нет, и
// авторства у такого ответа тоже.
type SubmitInput struct {
	UserID    *int64
	Name      string
	Email     string
	Answers   map[string]any
	ShareID   *int64
	IP        string
	UserAgent string
}

// SubmitResult — что показать сразу после отправки: сам ответ, оценку теста и
// (если она открыта) правильные ответы.
type SubmitResult struct {
	Response   *domain.Response `json:"response"`
	Score      int              `json:"score"`
	MaxScore   int              `json:"max_score"`
	Graded     bool             `json:"graded"`
	AnswerKeys map[string]any   `json:"answer_keys,omitempty"`
}

// Submit — приём ответа участником (уровень respond и выше).
func (s *Service) Submit(ctx context.Context, userID, formID int64, in SubmitInput) (*SubmitResult, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, formID, domain.AccessRespond)
	if err != nil {
		return nil, err
	}
	in.UserID = &userID
	return s.submitTo(ctx, form, in)
}

// submitTo — ядро приёма ответа (доступ уже проверен либо это внешняя ссылка).
func (s *Service) submitTo(ctx context.Context, form *domain.Form, in SubmitInput) (*SubmitResult, error) {
	if err := s.acceptable(ctx, form); err != nil {
		return nil, err
	}
	guest := in.UserID == nil || *in.UserID == 0
	if guest && !form.AllowAnonymous {
		return nil, domain.ErrAuthRequired
	}

	sections, err := s.repo.ListSections(ctx, form.ID)
	if err != nil {
		return nil, err
	}
	questions := domain.AllQuestions(sections)
	if len(questions) == 0 {
		return nil, domain.ErrNoQuestions
	}

	// «Один ответ от человека» проверяем и здесь: база отобьёт гонку двух
	// вкладок частичным уникальным индексом, а человеку нужен внятный отказ.
	// Без этой настройки повторная отправка законна — это ещё один ответ.
	if !guest && form.OneResponse {
		mine, err := s.repo.ResponseOfUser(ctx, form.ID, *in.UserID)
		if err != nil {
			return nil, err
		}
		if mine != nil {
			return nil, domain.ErrAlreadyAnswered
		}
	}

	email := strings.TrimSpace(in.Email)
	if form.CollectEmail && email == "" {
		return nil, domain.ErrEmailRequired
	}
	answers, err := coerceAnswers(questions, in.Answers)
	if err != nil {
		return nil, err
	}
	// Значения скрытых условием вопросов не сохраняем: человек их не видел, а
	// в ответе они остались бы от прежних попыток ветвления.
	answers = onlyVisible(sections, answers)
	if missing := domain.MissingRequired(sections, answers); missing != "" {
		return nil, domain.ErrRequired(missing)
	}

	resp := &domain.Response{
		FormID: form.ID, UserID: in.UserID, Email: email,
		Name: strings.TrimSpace(in.Name), Answers: answers, ShareID: in.ShareID,
		IP: in.IP, UserAgent: in.UserAgent,
	}
	if form.Quiz {
		resp.Score, resp.MaxScore = grade(questions, answers)
		resp.Graded = form.QuizRelease == domain.QuizImmediately
	}
	if err := s.repo.CreateResponse(ctx, resp, domain.SearchText(questions, answers),
		bookings(questions, answers)); err != nil {
		return nil, err
	}

	s.publish(ctx, form.ID, "response:new", map[string]any{
		"form_id": form.ID, "form_title": form.Title, "response_id": resp.ID,
		"user_id": resp.UserID, "name": resp.Name,
	})
	return s.submitResult(form, sections, resp), nil
}

func (s *Service) submitResult(form *domain.Form, sections []domain.Section, resp *domain.Response) *SubmitResult {
	out := &SubmitResult{
		Response: resp, Score: resp.Score, MaxScore: resp.MaxScore, Graded: resp.Graded,
	}
	if form.Quiz && resp.Graded && form.QuizShowAnswers {
		out.AnswerKeys = answerKeys(sections)
	}
	return out
}

// UpdateMine — правка собственного ответа (когда форма это разрешает).
func (s *Service) UpdateMine(ctx context.Context, userID, formID int64, in SubmitInput) (*SubmitResult, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, formID, domain.AccessRespond)
	if err != nil {
		return nil, err
	}
	if !form.AllowEdit {
		return nil, domain.ErrEditNotAllowed
	}
	mine, err := s.repo.ResponseOfUser(ctx, formID, userID)
	if err != nil {
		return nil, err
	}
	if mine == nil {
		return nil, domain.ErrResponseNotFound
	}
	if err := s.acceptable(ctx, form); err != nil {
		return nil, err
	}

	sections, err := s.repo.ListSections(ctx, formID)
	if err != nil {
		return nil, err
	}
	questions := domain.AllQuestions(sections)
	answers, err := coerceAnswers(questions, in.Answers)
	if err != nil {
		return nil, err
	}
	answers = onlyVisible(sections, answers)
	if missing := domain.MissingRequired(sections, answers); missing != "" {
		return nil, domain.ErrRequired(missing)
	}
	if form.CollectEmail && strings.TrimSpace(in.Email) == "" && mine.Email == "" {
		return nil, domain.ErrEmailRequired
	}

	old := mine.Answers
	mine.Answers = answers
	if email := strings.TrimSpace(in.Email); email != "" {
		mine.Email = email
	}
	if form.Quiz {
		mine.Score, mine.MaxScore = grade(questions, answers)
		mine.Graded = form.QuizRelease == domain.QuizImmediately
	}
	if err := s.repo.UpdateResponse(ctx, mine, domain.SearchText(questions, answers),
		bookings(questions, answers)); err != nil {
		return nil, err
	}
	// Файлы, которых в новом ответе не осталось, из хранилища убираем: иначе
	// замена вложения тихо копила бы мусор в квоте владельца формы.
	s.removeOrphanFiles(ctx, form, old, answers)

	s.publish(ctx, formID, "response:updated", map[string]any{
		"form_id": formID, "response_id": mine.ID,
	})
	return s.submitResult(form, sections, mine), nil
}

// ── Ответы глазами автора ────────────────────────────────────────

// ResponseList — страница собранных ответов.
type ResponseList struct {
	Items   []*domain.Response `json:"items"`
	Total   int                `json:"total"`
	Page    int                `json:"page"`
	PerPage int                `json:"per_page"`
}

// ListResponses — собранные ответы (уровень view и выше).
func (s *Service) ListResponses(ctx context.Context, userID, formID int64, f domain.ResponseListFilter) (*ResponseList, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessView); err != nil {
		return nil, err
	}
	f.FormID = formID
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage <= 0 {
		f.PerPage = 30
	}
	if f.PerPage > maxPerPage {
		f.PerPage = maxPerPage
	}
	if f.Sort != "score" {
		f.Sort = "created_at"
	}
	items, total, err := s.repo.ListResponses(ctx, f)
	if err != nil {
		return nil, err
	}
	return &ResponseList{Items: items, Total: total, Page: f.Page, PerPage: f.PerPage}, nil
}

func (s *Service) GetResponse(ctx context.Context, userID, formID, responseID int64) (*domain.Response, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessView); err != nil {
		return nil, err
	}
	resp, err := s.repo.GetResponse(ctx, responseID)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.FormID != formID {
		return nil, domain.ErrResponseNotFound
	}
	return resp, nil
}

// DeleteResponse — удалить один ответ (уровень edit: это правка собранного).
func (s *Service) DeleteResponse(ctx context.Context, userID, formID, responseID int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	form, err := s.require(ctx, a, formID, domain.AccessEdit)
	if err != nil {
		return err
	}
	resp, err := s.repo.GetResponse(ctx, responseID)
	if err != nil {
		return err
	}
	if resp == nil || resp.FormID != formID {
		return domain.ErrResponseNotFound
	}
	if err := s.repo.DeleteResponse(ctx, formID, responseID); err != nil {
		return err
	}
	s.removeAnswerFiles(ctx, form, resp)
	s.publish(ctx, formID, "response:deleted", map[string]any{
		"form_id": formID, "response_id": responseID,
	})
	return nil
}

// DeleteResponses — массовое удаление: перечисленные ответы либо все сразу
// («очистить ответы» перед новым запуском опроса).
func (s *Service) DeleteResponses(ctx context.Context, userID, formID int64, ids []int64, all bool) (int, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return 0, err
	}
	form, err := s.require(ctx, a, formID, domain.AccessEdit)
	if err != nil {
		return 0, err
	}
	if !all && len(ids) == 0 {
		return 0, nil
	}
	deleted, err := s.repo.DeleteResponses(ctx, formID, ids, all)
	if err != nil {
		return 0, err
	}
	if len(deleted) == 0 {
		return 0, nil
	}
	s.removeAnswerFiles(ctx, form, deleted...)
	removedIDs := make([]int64, 0, len(deleted))
	for _, r := range deleted {
		removedIDs = append(removedIDs, r.ID)
	}
	s.publish(ctx, formID, "response:bulk-deleted", map[string]any{
		"form_id": formID, "ids": removedIDs,
	})
	return len(deleted), nil
}

// PublishGrades — открыть оценки теста отвечающим (режим ручной проверки).
// responseID == 0 — открыть сразу все.
func (s *Service) PublishGrades(ctx context.Context, userID, formID, responseID int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return err
	}
	if err := s.repo.PublishGrades(ctx, formID, responseID); err != nil {
		return err
	}
	s.publish(ctx, formID, "form:grades", map[string]any{"form_id": formID})
	return nil
}

// ── Хелперы ──────────────────────────────────────────────────────

// coerceAnswers — оставить только значения существующих вопросов и проверить
// их по типу. Неизвестные ключи отбрасываются.
func coerceAnswers(questions []domain.Question, in map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, q := range questions {
		key := domain.QuestionID(q.ID)
		v, ok := in[key]
		if !ok {
			continue
		}
		clean, err := q.CoerceAnswer(v)
		if err != nil {
			return nil, err
		}
		if clean != nil {
			out[key] = clean
		}
	}
	return out, nil
}

// onlyVisible — оставить ответы вопросов, которые человек реально видел:
// пройденный маршрут разделов и внутри них — прошедшие условие показа.
func onlyVisible(sections []domain.Section, answers map[string]any) map[string]any {
	allowed := map[string]bool{}
	for _, q := range domain.AnsweredQuestions(sections, answers) {
		allowed[domain.QuestionID(q.ID)] = true
	}
	out := make(map[string]any, len(answers))
	for key, v := range answers {
		if allowed[key] {
			out[key] = v
		}
	}
	return out
}

// grade — набранный и максимальный балл теста.
func grade(questions []domain.Question, answers map[string]any) (score, max int) {
	for _, q := range questions {
		score += domain.Grade(q, answers[domain.QuestionID(q.ID)])
	}
	return score, domain.MaxScore(questions)
}

// filePaths — ключи хранилища из значений ответа (сам файл и миниатюра).
func filePaths(answers map[string]any) []string {
	files := domain.AnswerFiles(answers)
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Path)
		if f.Thumb != "" {
			out = append(out, f.Thumb)
		}
	}
	return out
}

// removeAnswerFiles — убрать из хранилища файлы удаляемых ответов.
func (s *Service) removeAnswerFiles(ctx context.Context, form *domain.Form, responses ...*domain.Response) {
	var paths []string
	for _, r := range responses {
		if r != nil {
			paths = append(paths, filePaths(r.Answers)...)
		}
	}
	if len(paths) > 0 {
		quotaUser, companyID := quotaScope(form)
		s.files.RemoveFor(ctx, quotaUser, companyID, paths)
	}
}

// removeOrphanFiles — файлы прежней версии ответа, которых нет в новой.
func (s *Service) removeOrphanFiles(ctx context.Context, form *domain.Form, old, next map[string]any) {
	kept := map[string]bool{}
	for _, p := range filePaths(next) {
		kept[p] = true
	}
	var gone []string
	for _, p := range filePaths(old) {
		if !kept[p] {
			gone = append(gone, p)
		}
	}
	if len(gone) > 0 {
		quotaUser, companyID := quotaScope(form)
		s.files.RemoveFor(ctx, quotaUser, companyID, gone)
	}
}

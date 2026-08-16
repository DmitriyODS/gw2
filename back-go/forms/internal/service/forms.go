package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

// ListForms — формы выбранной области. Структура сюда не входит: списку хватает
// счётчиков, а разделы с вопросами нужны только открытой форме.
func (s *Service) ListForms(ctx context.Context, userID int64, scope string) ([]*domain.Form, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListForms(ctx, a.UserID, a.Companies, domain.NormalizeScope(scope))
}

// GetForm — одна доступная форма со структурой. Ключи правильных ответов
// оставляем только тем, кто вправе видеть ответы: отвечающему они ни к чему.
func (s *Service) GetForm(ctx context.Context, userID, id int64) (*domain.Form, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, id, domain.AccessRespond)
	if err != nil {
		return nil, err
	}
	sections, err := s.repo.ListSections(ctx, id)
	if err != nil {
		return nil, err
	}
	if !domain.AccessAtLeast(form.MyAccess, domain.AccessView) {
		sections = stripAnswerKeys(sections)
	}
	form.Sections = sections
	return form, nil
}

// stripAnswerKeys — убрать правильные ответы теста из структуры. Баллы
// остаются: сколько стоит вопрос, отвечающий видеть вправе.
func stripAnswerKeys(sections []domain.Section) []domain.Section {
	out := make([]domain.Section, len(sections))
	for i, section := range sections {
		questions := make([]domain.Question, len(section.Questions))
		for j, q := range section.Questions {
			q.AnswerKey = nil
			questions[j] = q
		}
		section.Questions = questions
		out[i] = section
	}
	return out
}

// CreateForm — новая форма. Владелец — создающий; компания активной сессии
// запоминается, чтобы её квота платила за файлы ответов и чтобы было что
// предложить в «поделиться с компанией». Первый раздел заводится сразу: форма
// без раздела — это форма, в которую некуда положить вопрос.
func (s *Service) CreateForm(ctx context.Context, userID int64, companyID *int64, title string, quiz bool) (*domain.Form, error) {
	pos, err := s.repo.NextPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	form := &domain.Form{
		OwnerID: userID, CompanyID: companyID, Title: title, Position: pos,
		Status: domain.StatusDraft, AllowAnonymous: true, ShowProgress: true,
		CollectName: true,
		Quiz: quiz, QuizRelease: domain.QuizImmediately, QuizShowAnswers: true,
		CreatedBy: &userID,
	}
	if err := s.repo.CreateForm(ctx, form); err != nil {
		return nil, err
	}
	if _, err := s.repo.ReplaceStructure(ctx, form.ID, []domain.Section{{NextAction: domain.NextNext}}); err != nil {
		return nil, err
	}
	sections, err := s.repo.ListSections(ctx, form.ID)
	if err != nil {
		return nil, err
	}
	form.Sections = sections
	form.MyAccess = domain.AccessOwner
	s.publish(ctx, form.ID, "form:created", formPayload(form))
	return form, nil
}

/*
FormPatch — правка формы. Каждое поле — указатель: «ключа нет» означает «не

	трогать», иначе обычное сохранение настроек сбрасывало бы то, чего клиент не
	присылал. У сроков указатель двойной — им нужно ещё и явное «убрать».
*/
type FormPatch struct {
	Title            *string
	Description      *string
	Status           *string
	AllowAnonymous   *bool
	OneResponse      *bool
	AllowEdit        *bool
	CollectEmail     *bool
	CollectName      *bool
	ShowProgress     *bool
	ShuffleQuestions *bool
	Confirmation     *string
	ShowSummary      *bool
	Quiz             *bool
	QuizRelease      *string
	QuizShowAnswers  *bool
	OpensAt          **time.Time
	ClosesAt         **time.Time
	MaxResponses     *int
}

// UpdateForm — правка формы и её настроек (уровень edit).
func (s *Service) UpdateForm(ctx context.Context, userID, id int64, p FormPatch) (*domain.Form, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, id, domain.AccessEdit)
	if err != nil {
		return nil, err
	}

	singleBefore := form.OneResponse
	applyPatch(form, p)
	if err := s.repo.UpdateForm(ctx, form); err != nil {
		return nil, err
	}
	/* Настройка «один ответ от человека» продублирована в самих ответах — её
	   сторожит частичный уникальный индекс. Включение на форме, где кто-то уже
	   отвечал дважды, база отобьёт: сообщаем об этом честно, а не роняем 500. */
	if form.OneResponse != singleBefore {
		if err := s.repo.SetResponsesSingle(ctx, id, form.OneResponse); err != nil {
			form.OneResponse = singleBefore
			if uerr := s.repo.UpdateForm(ctx, form); uerr != nil {
				s.log.Warn("forms.rollback_single_failed", "form_id", id, "error", uerr)
			}
			return nil, domain.NewError("VALIDATION",
				"Кто-то уже отправил больше одного ответа — включить «один ответ от человека» нельзя", 409)
		}
	}
	s.publish(ctx, id, "form:updated", formPayload(form))
	return form, nil
}

func applyPatch(f *domain.Form, p FormPatch) {
	if p.Title != nil {
		if title := strings.TrimSpace(*p.Title); title != "" {
			f.Title = title
		}
	}
	if p.Description != nil {
		f.Description = strings.TrimSpace(*p.Description)
	}
	if p.Status != nil {
		f.Status = domain.NormalizeStatus(*p.Status)
	}
	if p.AllowAnonymous != nil {
		f.AllowAnonymous = *p.AllowAnonymous
	}
	if p.OneResponse != nil {
		f.OneResponse = *p.OneResponse
	}
	if p.AllowEdit != nil {
		f.AllowEdit = *p.AllowEdit
	}
	if p.CollectEmail != nil {
		f.CollectEmail = *p.CollectEmail
	}
	if p.CollectName != nil {
		f.CollectName = *p.CollectName
	}
	if p.ShowProgress != nil {
		f.ShowProgress = *p.ShowProgress
	}
	if p.ShuffleQuestions != nil {
		f.ShuffleQuestions = *p.ShuffleQuestions
	}
	if p.Confirmation != nil {
		f.Confirmation = strings.TrimSpace(*p.Confirmation)
	}
	if p.ShowSummary != nil {
		f.ShowSummary = *p.ShowSummary
	}
	if p.Quiz != nil {
		f.Quiz = *p.Quiz
	}
	if p.QuizRelease != nil {
		f.QuizRelease = domain.NormalizeQuizRelease(*p.QuizRelease)
	}
	if p.QuizShowAnswers != nil {
		f.QuizShowAnswers = *p.QuizShowAnswers
	}
	if p.OpensAt != nil {
		f.OpensAt = *p.OpensAt
	}
	if p.ClosesAt != nil {
		f.ClosesAt = *p.ClosesAt
	}
	if p.MaxResponses != nil && *p.MaxResponses >= 0 {
		f.MaxResponses = *p.MaxResponses
	}
}

// DeleteForm — только владелец: форма уходит вместе со всеми собранными
// ответами, и такое решение не доверяют приглашённому редактору.
func (s *Service) DeleteForm(ctx context.Context, userID, id int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	form, err := s.require(ctx, a, id, domain.AccessOwner)
	if err != nil {
		return err
	}
	// Аудиторию узнаём ДО удаления: после него шар уже нет и звать некого.
	audience := s.audience(ctx, id)
	// Файлы ответов — тоже: строк не станет, а объекты в хранилище останутся.
	var paths []string
	if err := s.repo.EachResponse(ctx, id, func(r *domain.Response) error {
		paths = append(paths, filePaths(r.Answers)...)
		return nil
	}); err != nil {
		return err
	}
	if err := s.repo.DeleteForm(ctx, id); err != nil {
		return err
	}
	if len(paths) > 0 {
		quotaUser, companyID := quotaScope(form)
		s.files.RemoveFor(ctx, quotaUser, companyID, paths)
	}
	s.bus.Publish(ctx, "form:deleted", audience, map[string]any{"id": id})
	return nil
}

// DuplicateForm — копия формы со всей структурой, но без ответов и без чужих
// доступов: копию заводят, чтобы провести опрос заново.
func (s *Service) DuplicateForm(ctx context.Context, userID, id int64, companyID *int64) (*domain.Form, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	src, err := s.require(ctx, a, id, domain.AccessView)
	if err != nil {
		return nil, err
	}
	sections, err := s.repo.ListSections(ctx, id)
	if err != nil {
		return nil, err
	}
	pos, err := s.repo.NextPosition(ctx, userID)
	if err != nil {
		return nil, err
	}

	copyForm := *src
	copyForm.ID, copyForm.OwnerID, copyForm.CompanyID = 0, userID, companyID
	copyForm.Title = strings.TrimSpace(src.Title) + " (копия)"
	copyForm.Position, copyForm.Status, copyForm.CreatedBy = pos, domain.StatusDraft, &userID
	copyForm.Sections, copyForm.Responses, copyForm.MyAccess = nil, 0, domain.AccessOwner
	if err := s.repo.CreateForm(ctx, &copyForm); err != nil {
		return nil, err
	}
	if _, err := s.repo.ReplaceStructure(ctx, copyForm.ID, cloneSections(sections)); err != nil {
		return nil, err
	}
	if copyForm.Sections, err = s.repo.ListSections(ctx, copyForm.ID); err != nil {
		return nil, err
	}
	s.publish(ctx, copyForm.ID, "form:created", formPayload(&copyForm))
	return &copyForm, nil
}

/*
cloneSections — структура для новой формы: id обнуляются, а ветвление

	переводится в ПОЗИЦИИ разделов — у копии разделов id ещё нет, и ссылаться
	по ним не на что (репозиторий переведёт позиции обратно в id).
*/
func cloneSections(sections []domain.Section) []domain.Section {
	index := make(map[int64]int, len(sections))
	for i, s := range sections {
		index[s.ID] = i
	}
	out := make([]domain.Section, 0, len(sections))
	for _, s := range sections {
		next := domain.Section{
			Title: s.Title, Description: s.Description, NextAction: s.NextAction,
			NextIndex: -1,
		}
		if s.NextSectionID != nil {
			if i, ok := index[*s.NextSectionID]; ok {
				next.NextIndex = i
			}
		}
		for _, q := range s.Questions {
			clone := q
			clone.ID, clone.FormID, clone.SectionID = 0, 0, 0
			clone.Config = retargetToIndexes(q.Config, index)
			next.Questions = append(next.Questions, clone)
		}
		out = append(out, next)
	}
	return out
}

// retargetToIndexes — переходы вопроса из id разделов в их позиции.
func retargetToIndexes(config map[string]any, index map[int64]int) map[string]any {
	out := make(map[string]any, len(config))
	for k, v := range config {
		out[k] = v
	}
	raw, ok := out["targets"].(map[string]any)
	if !ok {
		return out
	}
	targets := make(map[string]any, len(raw))
	for option, target := range raw {
		s := strings.TrimSpace(valueString(target))
		if s == domain.NextSubmit || s == domain.NextNext {
			targets[option] = s
			continue
		}
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			if i, ok := index[id]; ok {
				targets[option] = domain.TargetIndex(i)
			}
		}
	}
	out["targets"] = targets
	return out
}

func valueString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	}
	return ""
}

// ReplaceStructure — полная замена разделов и вопросов формы. Значения
// удалённых вопросов вычищаются из уже собранных ответов (вместе с файлами).
func (s *Service) ReplaceStructure(ctx context.Context, userID, id int64, sections []domain.Section) (*domain.Form, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, id, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	// Форма без разделов невозможна: вопрос некуда положить.
	if len(sections) == 0 {
		sections = []domain.Section{{NextAction: domain.NextNext, NextIndex: -1}}
	}
	for i := range sections {
		for j := range sections[i].Questions {
			sections[i].Questions[j].Normalize()
		}
	}
	removed, err := s.repo.ReplaceStructure(ctx, id, sections)
	if err != nil {
		return nil, err
	}
	if len(removed) > 0 {
		if err := s.stripRemovedQuestions(ctx, form, removed); err != nil {
			return nil, err
		}
	}
	if form.Sections, err = s.repo.ListSections(ctx, id); err != nil {
		return nil, err
	}
	s.publish(ctx, id, "form:structure", map[string]any{"id": id})
	return form, nil
}

// stripRemovedQuestions — убрать значения удалённых вопросов из ответов и
// пересчитать строку поиска; приложенные к ним файлы уходят из хранилища.
func (s *Service) stripRemovedQuestions(ctx context.Context, form *domain.Form, removed []int64) error {
	sections, err := s.repo.ListSections(ctx, form.ID)
	if err != nil {
		return err
	}
	questions := domain.AllQuestions(sections)

	var orphans []string
	var stale []*domain.Response
	if err := s.repo.EachResponse(ctx, form.ID, func(r *domain.Response) error {
		changed := false
		for _, qid := range removed {
			key := domain.QuestionID(qid)
			v, ok := r.Answers[key]
			if !ok {
				continue
			}
			orphans = append(orphans, filePaths(map[string]any{key: v})...)
			delete(r.Answers, key)
			changed = true
		}
		if changed {
			stale = append(stale, r)
		}
		return nil
	}); err != nil {
		return err
	}
	for _, r := range stale {
		if err := s.repo.UpdateResponse(ctx, r, domain.SearchText(questions, r.Answers), nil); err != nil {
			return err
		}
	}
	if len(orphans) > 0 {
		quotaUser, companyID := quotaScope(form)
		s.files.RemoveFor(ctx, quotaUser, companyID, orphans)
	}
	return nil
}

// SearchForms — глобальный поиск (строка Hola) по доступным формам.
func (s *Service) SearchForms(ctx context.Context, userID int64, query string, limit int) ([]*domain.SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*domain.SearchHit{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.SearchForms(ctx, a.UserID, a.Companies, query, limit)
}

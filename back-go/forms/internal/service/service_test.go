package service

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

const (
	ownerID     = 42 // владелец тестовой формы
	companyID   = 7  // компания, в которой она заведена
	strangerID  = 99 // посторонний
	assigneeID  = 13 // тот, кому форму назначили
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// ── Фейки портов ─────────────────────────────────────────────────

type fakeRepo struct {
	form       *domain.Form
	sections   []domain.Section
	responses  map[int64]*domain.Response
	userShares []*domain.UserShare
	share      *domain.Share
	nextID     int64
}

func (f *fakeRepo) ListForms(_ domain.Ctx, _ int64, _ []int64, _ string) ([]*domain.Form, error) {
	return []*domain.Form{f.form}, nil
}

func (f *fakeRepo) GetForm(_ domain.Ctx, id int64) (*domain.Form, error) {
	if f.form != nil && f.form.ID == id {
		copyForm := *f.form
		return &copyForm, nil
	}
	return nil, nil
}

func (f *fakeRepo) CountOwned(_ domain.Ctx, _ int64) (int, error)  { return 1, nil }
func (f *fakeRepo) CreateForm(_ domain.Ctx, form *domain.Form) error {
	f.nextID++
	form.ID = f.nextID
	return nil
}
func (f *fakeRepo) UpdateForm(_ domain.Ctx, form *domain.Form) error {
	f.form = form
	return nil
}
func (f *fakeRepo) DeleteForm(_ domain.Ctx, _ int64) error          { f.form = nil; return nil }
func (f *fakeRepo) NextPosition(_ domain.Ctx, _ int64) (int, error) { return 1, nil }
func (f *fakeRepo) SearchForms(_ domain.Ctx, _ int64, _ []int64, _ string, _ int) ([]*domain.SearchHit, error) {
	return []*domain.SearchHit{}, nil
}

func (f *fakeRepo) ListSections(_ domain.Ctx, _ int64) ([]domain.Section, error) {
	return f.sections, nil
}
func (f *fakeRepo) GetQuestion(_ domain.Ctx, _, questionID int64) (*domain.Question, error) {
	for _, q := range domain.AllQuestions(f.sections) {
		if q.ID == questionID {
			return &q, nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) ReplaceStructure(_ domain.Ctx, _ int64, sections []domain.Section) ([]int64, error) {
	removed := []int64{}
	kept := map[int64]bool{}
	for _, s := range sections {
		for _, q := range s.Questions {
			kept[q.ID] = true
		}
	}
	for _, q := range domain.AllQuestions(f.sections) {
		if !kept[q.ID] {
			removed = append(removed, q.ID)
		}
	}
	f.sections = sections
	return removed, nil
}

func (f *fakeRepo) ListResponses(_ domain.Ctx, filter domain.ResponseListFilter) ([]*domain.Response, int, error) {
	out := []*domain.Response{}
	for _, r := range f.responses {
		out = append(out, r)
	}
	return out, len(out), nil
}
func (f *fakeRepo) GetResponse(_ domain.Ctx, id int64) (*domain.Response, error) {
	return f.responses[id], nil
}
func (f *fakeRepo) ResponseOfUser(_ domain.Ctx, formID, userID int64) (*domain.Response, error) {
	for _, r := range f.responses {
		if r.FormID == formID && r.UserID != nil && *r.UserID == userID {
			return r, nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) CountResponses(_ domain.Ctx, _ int64) (int, error) { return len(f.responses), nil }
func (f *fakeRepo) BookingCounts(_ domain.Ctx, _ int64, keys []string, except int64) (map[string]map[string]int, error) {
	out := map[string]map[string]int{}
	for _, key := range keys {
		counts := map[string]int{}
		for _, r := range f.responses {
			if except > 0 && r.ID == except {
				continue
			}
			if option := domain.Text(r.Answers[key]); option != "" {
				counts[option]++
			}
		}
		out[key] = counts
	}
	return out, nil
}

func (f *fakeRepo) CreateResponse(_ domain.Ctx, r *domain.Response, _ string, bookings []domain.Booking) error {
	// Фейк повторяет правило базы: место занято — ответ не принимается.
	for _, b := range bookings {
		taken := 0
		for _, existing := range f.responses {
			if domain.Text(existing.Answers[b.QuestionKey]) == b.Option {
				taken++
			}
		}
		if taken >= b.Capacity {
			return domain.ErrNoSlots
		}
	}
	f.nextID++
	r.ID = f.nextID
	r.CreatedAt = time.Now()
	f.responses[r.ID] = r
	return nil
}
func (f *fakeRepo) UpdateResponse(_ domain.Ctx, r *domain.Response, _ string, _ []domain.Booking) error {
	f.responses[r.ID] = r
	return nil
}
func (f *fakeRepo) DeleteResponse(_ domain.Ctx, _, id int64) error {
	delete(f.responses, id)
	return nil
}
func (f *fakeRepo) DeleteResponses(_ domain.Ctx, _ int64, ids []int64, all bool) ([]*domain.Response, error) {
	out := []*domain.Response{}
	for id, r := range f.responses {
		if all || slices.Contains(ids, id) {
			out = append(out, r)
			delete(f.responses, id)
		}
	}
	return out, nil
}
func (f *fakeRepo) EachResponse(_ domain.Ctx, _ int64, fn func(*domain.Response) error) error {
	for _, r := range f.responses {
		if err := fn(r); err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeRepo) SetResponsesSingle(_ domain.Ctx, _ int64, _ bool) error { return nil }
func (f *fakeRepo) PublishGrades(_ domain.Ctx, _, responseID int64) error {
	for id, r := range f.responses {
		if responseID == 0 || id == responseID {
			r.Graded = true
		}
	}
	return nil
}
func (f *fakeRepo) ResponsesOfOwner(_ domain.Ctx, _ int64, _ []int64) ([]*domain.ResponseScope, error) {
	return nil, nil
}

// AccessOf — владелец получает всё, остальные — по своим шарам.
func (f *fakeRepo) AccessOf(_ domain.Ctx, formID, userID int64, companyIDs []int64) (string, error) {
	if f.form == nil || f.form.ID != formID {
		return domain.AccessNone, nil
	}
	if f.form.OwnerID == userID {
		return domain.AccessOwner, nil
	}
	best := domain.AccessNone
	for _, sh := range f.userShares {
		if sh.FormID != formID {
			continue
		}
		if (sh.UserID != nil && *sh.UserID == userID) ||
			(sh.CompanyID != nil && slices.Contains(companyIDs, *sh.CompanyID)) {
			best = domain.BestAccess(best, sh.Access)
		}
	}
	return best, nil
}

func (f *fakeRepo) Audience(_ domain.Ctx, _ int64) ([]int64, error) { return []int64{ownerID}, nil }
func (f *fakeRepo) ListUserShares(_ domain.Ctx, _ int64) ([]*domain.UserShare, error) {
	return f.userShares, nil
}
func (f *fakeRepo) PutUserShare(_ domain.Ctx, s *domain.UserShare) error {
	f.userShares = append(f.userShares, s)
	return nil
}
func (f *fakeRepo) DeleteUserShare(_ domain.Ctx, _ int64, _, _ *int64) error { return nil }
func (f *fakeRepo) Assignees(_ domain.Ctx, _ int64) ([]*domain.Assignee, error) {
	return []*domain.Assignee{}, nil
}
func (f *fakeRepo) ClaimDueReminders(_ domain.Ctx, _ time.Time, _ int) ([]domain.DueReminder, error) {
	return nil, nil
}

func (f *fakeRepo) CreateShare(_ domain.Ctx, s *domain.Share) error { f.share = s; return nil }
func (f *fakeRepo) ListShares(_ domain.Ctx, _ int64) ([]*domain.Share, error) {
	return []*domain.Share{f.share}, nil
}
func (f *fakeRepo) GetShareByCode(_ domain.Ctx, code string) (*domain.Share, error) {
	if f.share != nil && f.share.Code == code {
		return f.share, nil
	}
	return nil, nil
}
func (f *fakeRepo) UpdateShare(_ domain.Ctx, _, _ int64, _ string, _ bool) error { return nil }
func (f *fakeRepo) DeleteShare(_ domain.Ctx, _, _ int64) error                   { return nil }
func (f *fakeRepo) LogVisit(_ domain.Ctx, _ *domain.ShareVisit) error            { return nil }
func (f *fakeRepo) ListVisits(_ domain.Ctx, _ int64, _ int) ([]*domain.ShareVisit, error) {
	return nil, nil
}

type fakeUsers struct{ companies map[int64][]int64 }

func (u *fakeUsers) GetUser(_ domain.Ctx, id int64) (*domain.User, error) {
	return &domain.User{ID: id, IsActive: true}, nil
}
func (u *fakeUsers) CompanyActive(_ domain.Ctx, _ *int64) (bool, error) { return true, nil }
func (u *fakeUsers) CompaniesOf(_ domain.Ctx, userID int64) ([]int64, error) {
	return u.companies[userID], nil
}
func (u *fakeUsers) CompanyMembers(_ domain.Ctx, _ int64) ([]int64, error) {
	return []int64{assigneeID}, nil
}
func (u *fakeUsers) SearchDirectory(_ domain.Ctx, _ []int64, _ string, _ int) ([]*domain.User, error) {
	return nil, nil
}
func (u *fakeUsers) CompanyName(_ domain.Ctx, _ int64) (string, error) { return "Компания", nil }

type fakeFiles struct{ removed []string }

func (f *fakeFiles) SaveFor(_ context.Context, _, _ int64, name string, _ []byte) (string, error) {
	return "forms/" + name, nil
}
func (f *fakeFiles) SaveStreamFor(_ context.Context, _, _ int64, name string, _ io.Reader, _ int64) (string, error) {
	return "forms/" + name, nil
}
func (f *fakeFiles) RemoveFor(_ context.Context, _, _ int64, paths []string) {
	f.removed = append(f.removed, paths...)
}
func (f *fakeFiles) Remove(paths []string) { f.removed = append(f.removed, paths...) }

type busEvent struct {
	event   string
	rooms   []string
	payload any
}

type fakeBus struct{ events []busEvent }

func (b *fakeBus) Publish(_ domain.Ctx, event string, rooms []string, payload any) {
	b.events = append(b.events, busEvent{event: event, rooms: rooms, payload: payload})
}

func (b *fakeBus) has(event string) bool {
	for _, e := range b.events {
		if e.event == event {
			return true
		}
	}
	return false
}

// ── Стенд ────────────────────────────────────────────────────────

type stand struct {
	svc   *Service
	repo  *fakeRepo
	bus   *fakeBus
	files *fakeFiles
}

// newStand — открытая форма с одним обязательным вопросом и одним свободным.
func newStand() *stand {
	company := int64(companyID)
	repo := &fakeRepo{
		form: &domain.Form{
			ID: 1, OwnerID: ownerID, CompanyID: &company, Title: "Опрос",
			Status: domain.StatusOpen, AllowAnonymous: true,
			QuizRelease: domain.QuizImmediately, QuizShowAnswers: true,
		},
		sections: []domain.Section{{
			ID: 10, FormID: 1, NextAction: domain.NextSubmit,
			Questions: []domain.Question{
				{ID: 100, FormID: 1, SectionID: 10, Type: domain.QShortText, Title: "Имя", Required: true},
				{ID: 101, FormID: 1, SectionID: 10, Type: domain.QParagraph, Title: "Отзыв"},
			},
		}},
		responses: map[int64]*domain.Response{},
		nextID:    500,
	}
	bus := &fakeBus{}
	files := &fakeFiles{}
	svc := New(Deps{
		Repo:  repo,
		Users: &fakeUsers{companies: map[int64][]int64{ownerID: {companyID}, assigneeID: {companyID}}},
		Files: files,
		Bus:   bus,
		Log:   discardLogger(),
	})
	return &stand{svc: svc, repo: repo, bus: bus, files: files}
}

func ctx() context.Context { return context.Background() }

func code(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("ожидалась ошибка, её нет")
	}
	de := domain.AsDomainError(err)
	if de == nil {
		t.Fatalf("не доменная ошибка: %v", err)
	}
	return de.Code
}

// ── Доступ ───────────────────────────────────────────────────────

func TestStrangerSeesNothing(t *testing.T) {
	s := newStand()
	// Существование чужой формы не раскрываем: для постороннего её просто нет.
	if got := code(t, mustErr(s.svc.GetForm(ctx(), strangerID, 1))); got != "NOT_FOUND" {
		t.Fatalf("посторонний получил %s, ожидался NOT_FOUND", got)
	}
}

func TestRespondentCannotSeeResponses(t *testing.T) {
	s := newStand()
	user := int64(assigneeID)
	s.repo.userShares = []*domain.UserShare{
		{FormID: 1, UserID: &user, Access: domain.AccessRespond},
	}

	// Назначенный форму видит — значит нехватку уровня называем честно.
	if _, err := s.svc.GetForm(ctx(), assigneeID, 1); err != nil {
		t.Fatalf("назначенный не видит форму: %v", err)
	}
	_, err := s.svc.ListResponses(ctx(), assigneeID, 1, domain.ResponseListFilter{})
	if got := code(t, err); got != "FORBIDDEN" {
		t.Fatalf("назначенный получил %s на список ответов, ожидался FORBIDDEN", got)
	}
}

func TestAnswerKeysHiddenFromRespondent(t *testing.T) {
	s := newStand()
	s.repo.form.Quiz = true
	s.repo.sections[0].Questions[0].AnswerKey = map[string]any{"values": []any{"Иван"}}
	user := int64(assigneeID)
	s.repo.userShares = []*domain.UserShare{
		{FormID: 1, UserID: &user, Access: domain.AccessRespond},
	}

	form, err := s.svc.GetForm(ctx(), assigneeID, 1)
	if err != nil {
		t.Fatalf("чтение формы: %v", err)
	}
	if len(form.Sections[0].Questions[0].AnswerKey) != 0 {
		t.Fatal("правильные ответы теста ушли отвечающему")
	}
	// Автору они, наоборот, нужны — он их и задаёт.
	own, _ := s.svc.GetForm(ctx(), ownerID, 1)
	if len(own.Sections[0].Questions[0].AnswerKey) == 0 {
		t.Fatal("автор потерял правильные ответы")
	}
}

// ── Приём ответов ────────────────────────────────────────────────

func TestSubmitRequiresOpenForm(t *testing.T) {
	s := newStand()
	s.repo.form.Status = domain.StatusDraft

	_, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "Иван"}})
	if got := code(t, err); got != "FORM_CLOSED" {
		t.Fatalf("черновик принял ответ (%s)", got)
	}
}

func TestSubmitRespectsWindowAndLimit(t *testing.T) {
	s := newStand()
	future := time.Now().Add(time.Hour)
	s.repo.form.OpensAt = &future
	_, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "Иван"}})
	if got := code(t, err); got != "FORM_NOT_STARTED" {
		t.Fatalf("форма приняла ответ до начала приёма (%s)", got)
	}

	s.repo.form.OpensAt = nil
	s.repo.form.MaxResponses = 1
	s.repo.responses[1] = &domain.Response{ID: 1, FormID: 1}
	_, err = s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "Иван"}})
	if got := code(t, err); got != "FORM_LIMIT" {
		t.Fatalf("форма приняла ответ сверх потолка (%s)", got)
	}
}

func TestSubmitRequiresRequiredQuestion(t *testing.T) {
	s := newStand()
	_, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"101": "Всё хорошо"}})
	if got := code(t, err); got != "VALIDATION" {
		t.Fatalf("форма приняла ответ без обязательного вопроса (%s)", got)
	}
}

func TestSubmitOnceRule(t *testing.T) {
	s := newStand()
	s.repo.form.OneResponse = true

	if _, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "Иван"}}); err != nil {
		t.Fatalf("первый ответ отвергнут: %v", err)
	}
	_, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "Иван"}})
	if got := code(t, err); got != "ALREADY_ANSWERED" {
		t.Fatalf("второй ответ принят (%s)", got)
	}
}

func TestSubmitGradesQuiz(t *testing.T) {
	s := newStand()
	s.repo.form.Quiz = true
	s.repo.sections[0].Questions[0].Points = 4
	s.repo.sections[0].Questions[0].AnswerKey = map[string]any{"values": []any{"Иван"}}

	res, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "иван"}})
	if err != nil {
		t.Fatalf("ответ отвергнут: %v", err)
	}
	if res.Score != 4 || res.MaxScore != 4 || !res.Graded {
		t.Fatalf("оценка теста: %d/%d graded=%v, ожидалось 4/4 graded=true", res.Score, res.MaxScore, res.Graded)
	}
	// Разбор приходит только вместе с открытой оценкой.
	if len(res.AnswerKeys) == 0 {
		t.Fatal("правильные ответы не приехали с открытой оценкой")
	}

	// Ручная проверка держит оценку закрытой, пока автор её не откроет.
	s.repo.responses = map[int64]*domain.Response{}
	s.repo.form.QuizRelease = domain.QuizManual
	res, err = s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "Иван"}})
	if err != nil {
		t.Fatalf("ответ отвергнут: %v", err)
	}
	if res.Graded || len(res.AnswerKeys) != 0 {
		t.Fatal("при ручной проверке оценка и разбор ушли сразу")
	}
}

func TestSubmitPublishesEvent(t *testing.T) {
	s := newStand()
	if _, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{Answers: map[string]any{"100": "Иван"}}); err != nil {
		t.Fatalf("ответ отвергнут: %v", err)
	}
	if !s.bus.has("response:new") {
		t.Fatal("событие о новом ответе не опубликовано")
	}
}

// ── Публичная ссылка ─────────────────────────────────────────────

func TestSharedSubmitAnonymousRule(t *testing.T) {
	s := newStand()
	s.repo.share = &domain.Share{ID: 5, FormID: 1, Code: "abc"}
	s.repo.form.AllowAnonymous = false

	_, err := s.svc.SharedSubmit(ctx(), "abc", Visitor{}, SubmitInput{Answers: map[string]any{"100": "Гость"}})
	if got := code(t, err); got != "AUTH_REQUIRED" {
		t.Fatalf("форма приняла анонимный ответ (%s)", got)
	}

	s.repo.form.AllowAnonymous = true
	res, err := s.svc.SharedSubmit(ctx(), "abc", Visitor{}, SubmitInput{Answers: map[string]any{"100": "Гость"}})
	if err != nil {
		t.Fatalf("анонимный ответ отвергнут при разрешении: %v", err)
	}
	// Ответ помечается ссылкой, через которую пришёл: автор видит, что сработало.
	if res.Response.ShareID == nil || *res.Response.ShareID != 5 {
		t.Fatalf("ответ не привязан к ссылке: %#v", res.Response.ShareID)
	}
}

func TestSharedDraftNotOpened(t *testing.T) {
	s := newStand()
	s.repo.share = &domain.Share{ID: 5, FormID: 1, Code: "abc"}
	s.repo.form.Status = domain.StatusDraft

	_, err := s.svc.SharedForm(ctx(), "abc", Visitor{})
	if got := code(t, err); got != "FORM_CLOSED" {
		t.Fatalf("черновик открылся по ссылке (%s)", got)
	}
}

func TestSharedRequireAuth(t *testing.T) {
	s := newStand()
	s.repo.share = &domain.Share{ID: 5, FormID: 1, Code: "abc", RequireAuth: true}

	_, err := s.svc.SharedForm(ctx(), "abc", Visitor{})
	if got := code(t, err); got != "SHARE_AUTH_REQUIRED" {
		t.Fatalf("ссылка «только для своих» пустила гостя (%s)", got)
	}
}

// ── Назначения ───────────────────────────────────────────────────

func TestShareWithNotifiesAssignee(t *testing.T) {
	s := newStand()
	user := int64(assigneeID)
	due := time.Now().Add(48 * time.Hour)

	if _, err := s.svc.ShareWith(ctx(), ownerID, 1, []ShareTarget{
		{UserID: &user, Access: domain.AccessRespond, DueAt: &due},
	}); err != nil {
		t.Fatalf("назначение не прошло: %v", err)
	}
	if !s.bus.has("form:assigned") {
		t.Fatal("назначенному не отправлено уведомление")
	}

	// Уровень «видеть ответы» — не поручение, о нём не трубим.
	s.bus.events = nil
	if _, err := s.svc.ShareWith(ctx(), ownerID, 1, []ShareTarget{
		{UserID: &user, Access: domain.AccessView},
	}); err != nil {
		t.Fatalf("выдача доступа не прошла: %v", err)
	}
	if s.bus.has("form:assigned") {
		t.Fatal("уведомление о назначении ушло при обычной выдаче доступа")
	}
}

func TestShareWithForeignCompanyForbidden(t *testing.T) {
	s := newStand()
	foreign := int64(1234)
	_, err := s.svc.ShareWith(ctx(), ownerID, 1, []ShareTarget{
		{CompanyID: &foreign, Access: domain.AccessRespond},
	})
	if got := code(t, err); got != "FORBIDDEN" {
		t.Fatalf("форму назначили чужой компании (%s)", got)
	}
}

// ── Структура и сводка ───────────────────────────────────────────

func TestReplaceStructureCleansAnswers(t *testing.T) {
	s := newStand()
	s.repo.responses[1] = &domain.Response{
		ID: 1, FormID: 1,
		Answers: map[string]any{
			"100": "Иван",
			"101": []any{map[string]any{"path": "forms/a.pdf", "name": "a.pdf"}},
		},
	}

	// Второй вопрос убрали из формы — его значения и файлы уходят из ответов.
	sections := []domain.Section{{
		ID: 10, NextAction: domain.NextSubmit,
		Questions: []domain.Question{s.repo.sections[0].Questions[0]},
	}}
	if _, err := s.svc.ReplaceStructure(ctx(), ownerID, 1, sections); err != nil {
		t.Fatalf("сохранение структуры: %v", err)
	}
	if _, ok := s.repo.responses[1].Answers["101"]; ok {
		t.Fatal("значение удалённого вопроса осталось в ответе")
	}
	if len(s.files.removed) == 0 {
		t.Fatal("файл удалённого вопроса остался в хранилище")
	}
}

func TestSummaryCounts(t *testing.T) {
	s := newStand()
	s.repo.sections[0].Questions = append(s.repo.sections[0].Questions, domain.Question{
		ID: 102, Type: domain.QRadio, Title: "Оценка",
		Config: map[string]any{"options": []any{"Да", "Нет"}, "other": true},
	})
	s.repo.responses[1] = &domain.Response{ID: 1, FormID: 1, CreatedAt: time.Now(),
		Answers: map[string]any{"102": "Да"}}
	s.repo.responses[2] = &domain.Response{ID: 2, FormID: 1, CreatedAt: time.Now(),
		Answers: map[string]any{"102": "Да"}}
	s.repo.responses[3] = &domain.Response{ID: 3, FormID: 1, CreatedAt: time.Now(),
		Answers: map[string]any{"102": "Своё"}}

	sum, err := s.svc.Summary(ctx(), ownerID, 1)
	if err != nil {
		t.Fatalf("сводка: %v", err)
	}
	if sum.Total != 3 {
		t.Fatalf("всего ответов %d, ожидалось 3", sum.Total)
	}
	var radio *QuestionSummary
	for i := range sum.Questions {
		if sum.Questions[i].QuestionID == 102 {
			radio = &sum.Questions[i]
		}
	}
	if radio == nil {
		t.Fatal("вопрос выбора пропал из сводки")
	}
	if radio.Options[0].Label != "Да" || radio.Options[0].Count != 2 {
		t.Fatalf("распределение вариантов: %#v", radio.Options)
	}
	// Вписанное человеком идёт следом и помечается «своим».
	last := radio.Options[len(radio.Options)-1]
	if !last.Other || last.Label != "Своё" {
		t.Fatalf("свой вариант не отмечен: %#v", last)
	}
}

// mustErr — вернуть ошибку из пары (значение, ошибка) в проверках доступа.
func mustErr[T any](_ T, err error) error { return err }

// ── Условное отображение и «Запись» ──────────────────────────────

func TestSubmitDropsHiddenAnswers(t *testing.T) {
	s := newStand()
	// Второй вопрос показывается, только если на первый ответили «Да».
	s.repo.sections[0].Questions[1].Config = map[string]any{
		"visible_question_id": 100,
		"visible_values":      []any{"Да"},
	}

	res, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{
		Answers: map[string]any{"100": "Нет", "101": "текст из прошлой попытки"},
	})
	if err != nil {
		t.Fatalf("ответ отвергнут: %v", err)
	}
	// Значение скрытого вопроса не сохраняется: человек его не видел.
	if _, ok := res.Response.Answers["101"]; ok {
		t.Fatalf("ответ на скрытый вопрос сохранён: %#v", res.Response.Answers)
	}
}

func TestBookingSlotsRunOut(t *testing.T) {
	s := newStand()
	s.repo.sections[0].Questions = []domain.Question{{
		ID: 200, FormID: 1, SectionID: 10, Type: domain.QBooking, Title: "Смена",
		Config: map[string]any{
			"options":  []any{"Утро", "Вечер"},
			"capacity": map[string]any{"Утро": 1, "Вечер": 2},
		},
	}}

	// Второй отвечающий — назначенный: иначе форма для него просто не существует.
	user := int64(assigneeID)
	s.repo.userShares = []*domain.UserShare{
		{FormID: 1, UserID: &user, Access: domain.AccessRespond},
	}

	if _, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{
		Answers: map[string]any{"200": "Утро"},
	}); err != nil {
		t.Fatalf("первая запись отвергнута: %v", err)
	}
	// Единственное место утренней смены занято — вторая запись туда не проходит.
	_, err := s.svc.Submit(ctx(), assigneeID, 1, SubmitInput{
		Answers: map[string]any{"200": "Утро"},
	})
	if got := code(t, err); got != "NO_SLOTS" {
		t.Fatalf("запись сверх мест принята (%s)", got)
	}
	// А на вечернюю смену места ещё есть.
	if _, err := s.svc.Submit(ctx(), assigneeID, 1, SubmitInput{
		Answers: map[string]any{"200": "Вечер"},
	}); err != nil {
		t.Fatalf("запись при свободных местах отвергнута: %v", err)
	}
}

func TestFillShowsBookingCounts(t *testing.T) {
	s := newStand()
	s.repo.sections[0].Questions = []domain.Question{{
		ID: 200, FormID: 1, SectionID: 10, Type: domain.QBooking, Title: "Смена",
		Config: map[string]any{
			"options":  []any{"Утро", "Вечер"},
			"capacity": map[string]any{"Утро": 3, "Вечер": 3},
		},
	}}
	if _, err := s.svc.Submit(ctx(), ownerID, 1, SubmitInput{
		Answers: map[string]any{"200": "Утро"},
	}); err != nil {
		t.Fatalf("запись отвергнута: %v", err)
	}

	view, err := s.svc.Fill(ctx(), ownerID, 1)
	if err != nil {
		t.Fatalf("форма для заполнения: %v", err)
	}
	// Занятость приезжает вместе с формой — остаток показывают ДО отправки.
	if view.Booking["200"]["Утро"] != 1 {
		t.Fatalf("занятость мест не посчитана: %#v", view.Booking)
	}
}

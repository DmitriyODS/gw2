package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

// ── Фейки клиентов ──

type fakeTasks struct {
	domain.TasksClient
	tasks      []domain.TaskRef
	depts      []domain.CatalogItem
	createErr  error
	created    []string
	closed     []int64
	createDept int64
}

func (f *fakeTasks) SearchTasks(_ context.Context, _ int64, query string, _ int) ([]domain.TaskRef, error) {
	var out []domain.TaskRef
	for _, t := range f.tasks {
		if wordsMatch(query, t.Name) {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeTasks) CreateTask(_ context.Context, _, _ int64, name string, deptID int64) (*domain.TaskRef, error) {
	if f.createErr != nil && deptID == 0 {
		return nil, f.createErr
	}
	f.created = append(f.created, name)
	f.createDept = deptID
	return &domain.TaskRef{ID: 42, Name: name}, nil
}

func (f *fakeTasks) CloseTask(_ context.Context, _, _, taskID int64) (string, error) {
	f.closed = append(f.closed, taskID)
	for _, t := range f.tasks {
		if t.ID == taskID {
			return t.Name, nil
		}
	}
	return "", domain.NewError("NOT_FOUND", "Задача не найдена", 404)
}

func (f *fakeTasks) ListDepartments(_ context.Context, _ int64) ([]domain.CatalogItem, error) {
	return f.depts, nil
}

type fakeDiary struct {
	domain.DiaryClient
	diaries []domain.Diary
	entries []domain.Entry
	nextID  int64
}

func (f *fakeDiary) ListDiaries(_ context.Context, _ int64) ([]domain.Diary, error) {
	return f.diaries, nil
}

func (f *fakeDiary) CreateDiary(_ context.Context, _ int64, name string) (*domain.Diary, error) {
	d := domain.Diary{ID: int64(len(f.diaries) + 1), Name: name}
	f.diaries = append(f.diaries, d)
	return &d, nil
}

func (f *fakeDiary) ListEntries(_ context.Context, _, diaryID int64, from, to string) ([]domain.Entry, error) {
	var out []domain.Entry
	for _, e := range f.entries {
		if e.DiaryID != diaryID {
			continue
		}
		if from != "" && e.Date < from {
			continue
		}
		if to != "" && e.Date > to {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (f *fakeDiary) CreateEntry(_ context.Context, _, diaryID int64, date, title string) (*domain.Entry, error) {
	f.nextID++
	e := domain.Entry{ID: f.nextID, DiaryID: diaryID, Date: date, Title: title}
	f.entries = append(f.entries, e)
	return &e, nil
}

func (f *fakeDiary) SetEntryDone(_ context.Context, _, _, entryID int64, done bool) (*domain.Entry, error) {
	for i := range f.entries {
		if f.entries[i].ID == entryID {
			f.entries[i].Done = done
			return &f.entries[i], nil
		}
	}
	return nil, domain.NewError("NOT_FOUND", "Запись не найдена", 404)
}

type fakeNotes struct {
	domain.NotesClient
	notes   []domain.NoteRef
	deleted []int64
}

func (f *fakeNotes) FindNotes(_ context.Context, _, _ int64, query string, _ int) ([]domain.NoteRef, error) {
	var out []domain.NoteRef
	for _, n := range f.notes {
		if wordsMatch(query, n.Title) {
			out = append(out, n)
		}
	}
	return out, nil
}

func (f *fakeNotes) DeleteNote(_ context.Context, _, noteID int64) error {
	f.deleted = append(f.deleted, noteID)
	return nil
}

// ── Харнес ──

type harness struct {
	svc    *Service
	secret paseto.V4AsymmetricSecretKey
	tasks  *fakeTasks
	diary  *fakeDiary
	notes  *fakeNotes
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	secret := paseto.NewV4AsymmetricSecretKey()
	verifier, err := pasetoauth.NewVerifier(secret.Public().ExportHex())
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{
		secret: secret,
		tasks:  &fakeTasks{},
		diary:  &fakeDiary{},
		notes:  &fakeNotes{},
	}
	h.svc = New(Deps{
		Tasks: h.tasks, Diary: h.diary, Notes: h.notes,
		Verifier: verifier, Log: slog.New(slog.DiscardHandler),
	})
	return h
}

func (h *harness) token(t *testing.T, userID, companyID int64) string {
	t.Helper()
	tok := paseto.NewToken()
	tok.SetSubject(fmt.Sprint(userID))
	tok.SetString("type", "access")
	tok.SetIssuedAt(time.Now())
	tok.SetNotBefore(time.Now())
	tok.SetExpiration(time.Now().Add(15 * time.Minute))
	if companyID > 0 {
		tok.Set("company_id", companyID)
	}
	return tok.V4Sign(h.secret, nil)
}

func (h *harness) request(t *testing.T, token, command string, state *domain.DialogState) *domain.WebhookRequest {
	t.Helper()
	req := &domain.WebhookRequest{Version: "1.0"}
	req.Meta.Timezone = "Europe/Moscow"
	req.Session.User.AccessToken = token
	req.Request.Type = "SimpleUtterance"
	req.Request.Command = command
	if state != nil {
		raw, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		req.State.Session = raw
	}
	return req
}

// ── Тесты ──

func TestNoTokenStartsAccountLinking(t *testing.T) {
	h := newHarness(t)
	resp := h.svc.Handle(context.Background(), h.request(t, "", "мои задачи", nil))
	if resp.StartAccountLinking == nil || resp.Response != nil {
		t.Fatalf("ожидалась директива связки аккаунтов: %+v", resp)
	}
	resp = h.svc.Handle(context.Background(), h.request(t, "мусорный-токен", "мои задачи", nil))
	if resp.StartAccountLinking == nil {
		t.Fatalf("невалидный токен тоже должен вести на связку: %+v", resp)
	}
}

func TestDiaryAddCreatesDefaultDiary(t *testing.T) {
	h := newHarness(t)
	resp := h.svc.Handle(context.Background(),
		h.request(t, h.token(t, 7, 0), "запиши на завтра купить хлеб", nil))
	if len(h.diary.diaries) != 1 || len(h.diary.entries) != 1 {
		t.Fatalf("ожидались авто-ежедневник и запись: %+v %+v", h.diary.diaries, h.diary.entries)
	}
	if h.diary.entries[0].Title != "купить хлеб" {
		t.Fatalf("текст записи: %q", h.diary.entries[0].Title)
	}
	if !strings.Contains(resp.Response.Text, "завтра") {
		t.Fatalf("реплика без дня: %q", resp.Response.Text)
	}
}

func TestTaskCreateAsksDepartment(t *testing.T) {
	h := newHarness(t)
	h.tasks.createErr = domain.NewError("DEPARTMENT_REQUIRED", "Нужно выбрать отдел", 422)
	h.tasks.depts = []domain.CatalogItem{{ID: 10, Name: "Разработка"}, {ID: 20, Name: "Продажи"}}
	tok := h.token(t, 7, 3)

	resp := h.svc.Handle(context.Background(), h.request(t, tok, "добавь задачу сделать отчет", nil))
	if resp.SessionState == nil || resp.SessionState.Pending != "choose_department" {
		t.Fatalf("ожидался выбор отдела: %+v", resp)
	}
	// Ответ номером варианта: "2" → Продажи.
	resp = h.svc.Handle(context.Background(), h.request(t, tok, "2", resp.SessionState))
	if len(h.tasks.created) != 1 || h.tasks.createDept != 20 {
		t.Fatalf("задача не создана в выбранном отделе: %+v dept=%d", h.tasks.created, h.tasks.createDept)
	}
	if resp.SessionState != nil {
		t.Fatal("состояние диалога должно сброситься")
	}
}

func TestTaskCloseWithConfirmation(t *testing.T) {
	h := newHarness(t)
	h.tasks.tasks = []domain.TaskRef{{ID: 5, Name: "квартальный отчет"}}
	tok := h.token(t, 7, 3)

	resp := h.svc.Handle(context.Background(), h.request(t, tok, "закрой задачу отчет", nil))
	if resp.SessionState == nil || resp.SessionState.Pending != "confirm_close_task" {
		t.Fatalf("ожидалось подтверждение: %+v", resp)
	}
	// «нет» — отмена без закрытия.
	deny := h.svc.Handle(context.Background(), h.request(t, tok, "нет", resp.SessionState))
	if len(h.tasks.closed) != 0 || deny.SessionState != nil {
		t.Fatalf("отмена не должна закрывать задачу: %+v", h.tasks.closed)
	}
	// «да» — закрытие.
	resp = h.svc.Handle(context.Background(), h.request(t, tok, "закрой задачу отчет", nil))
	h.svc.Handle(context.Background(), h.request(t, tok, "да", resp.SessionState))
	if len(h.tasks.closed) != 1 || h.tasks.closed[0] != 5 {
		t.Fatalf("задача не закрыта: %+v", h.tasks.closed)
	}
}

func TestTasksRequireCompany(t *testing.T) {
	h := newHarness(t)
	resp := h.svc.Handle(context.Background(), h.request(t, h.token(t, 7, 0), "мои задачи", nil))
	if !strings.Contains(resp.Response.Text, "компани") {
		t.Fatalf("ожидалась реплика про активную компанию: %q", resp.Response.Text)
	}
}

func TestNoteDeleteWithConfirmation(t *testing.T) {
	h := newHarness(t)
	h.notes.notes = []domain.NoteRef{{ID: 3, Title: "идеи"}}
	tok := h.token(t, 7, 0)

	resp := h.svc.Handle(context.Background(), h.request(t, tok, "удали заметку идеи", nil))
	if resp.SessionState == nil || resp.SessionState.Pending != "confirm_delete_note" {
		t.Fatalf("ожидалось подтверждение удаления: %+v", resp)
	}
	h.svc.Handle(context.Background(), h.request(t, tok, "да", resp.SessionState))
	if len(h.notes.deleted) != 1 || h.notes.deleted[0] != 3 {
		t.Fatalf("заметка не удалена: %+v", h.notes.deleted)
	}
}

func TestDiaryDoneByText(t *testing.T) {
	h := newHarness(t)
	h.diary.diaries = []domain.Diary{{ID: 1, Name: "Ежедневник"}}
	h.diary.entries = []domain.Entry{{ID: 11, DiaryID: 1, Date: "2026-07-19", Title: "позвонить маме"}}
	h.svc.Handle(context.Background(),
		h.request(t, h.token(t, 7, 0), "отметь позвонить маме выполненным", nil))
	if !h.diary.entries[0].Done {
		t.Fatal("запись не отмечена выполненной")
	}
}

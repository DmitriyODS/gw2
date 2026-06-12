package service

// Порт back/tests/test_yougile_task_service.py и
// test_yougile_webhook_apply.py: импорт/экспорт/отвязка задач и применение
// вебхук-событий — на фейках портов, без БД и сети.

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// ── фейки YouGile-портов ─────────────────────────────────────────

type fakeYGRepo struct {
	store     *fakeStore
	accounts  map[int64]*domain.YougileAccount
	companies map[int64]*domain.YougileCompany
	updated   map[string]any // последние UpdateYougileCompanyFields
}

func newFakeYGRepo(store *fakeStore) *fakeYGRepo {
	return &fakeYGRepo{
		store:     store,
		accounts:  map[int64]*domain.YougileAccount{},
		companies: map[int64]*domain.YougileCompany{},
	}
}

func (f *fakeYGRepo) GetYougileAccount(_ context.Context, userID int64) (*domain.YougileAccount, error) {
	return f.accounts[userID], nil
}

func (f *fakeYGRepo) UpsertYougileAccount(_ context.Context, acc *domain.YougileAccount) error {
	if acc.ID == 0 {
		acc.ID = int64(len(f.accounts) + 1)
	}
	f.accounts[acc.UserID] = acc
	return nil
}

func (f *fakeYGRepo) DeleteYougileAccount(_ context.Context, userID int64) error {
	delete(f.accounts, userID)
	return nil
}

func (f *fakeYGRepo) GetYougileCompany(_ context.Context, companyID int64) (*domain.YougileCompany, error) {
	return f.companies[companyID], nil
}

func (f *fakeYGRepo) UpdateYougileCompanyFields(_ context.Context, _ int64, fields map[string]any) error {
	f.updated = fields
	return nil
}

func (f *fakeYGRepo) SetCompanyUsesYougile(_ context.Context, companyID int64, enabled bool) error {
	if c := f.companies[companyID]; c != nil {
		c.UsesYougile = enabled
	}
	return nil
}

func (f *fakeYGRepo) TaskByYougileID(_ context.Context, companyID int64, ygTaskID string) (*domain.Task, error) {
	for _, t := range f.store.tasks {
		if t.CompanyID == companyID && t.YougileTaskID != nil && *t.YougileTaskID == ygTaskID {
			return t, nil
		}
	}
	return nil, nil
}

// fakeYGAPI — фейковый клиент YouGile: канированные ответы + журнал вызовов.
type fakeYGAPI struct {
	key string

	task        map[string]any // GetTask
	created     map[string]any // CreateTask
	findResult  map[string]any // FindTaskByShortID
	companies   []map[string]any
	createdKey  string
	me          map[string]any
	chat        []map[string]any // PostChatMessage payloads
	chatIDs     []string
	updates     []map[string]any // UpdateTask payloads
	updateIDs   []string
	createBody  map[string]any // последний CreateTask body
	err         error          // ошибка для всех вызовов
	deletedKeys []string
}

func (f *fakeYGAPI) ListCompanies(string, string) ([]map[string]any, error) {
	return f.companies, f.err
}
func (f *fakeYGAPI) CreateKey(string, string, string) (string, error) {
	return f.createdKey, f.err
}
func (f *fakeYGAPI) DeleteKey(key string) error {
	f.deletedKeys = append(f.deletedKeys, key)
	return f.err
}
func (f *fakeYGAPI) Me() (map[string]any, error) { return f.me, f.err }
func (f *fakeYGAPI) ListProjects(int) ([]map[string]any, error) {
	return nil, f.err
}
func (f *fakeYGAPI) ListBoards(string, int) ([]map[string]any, error) { return nil, f.err }
func (f *fakeYGAPI) ListColumns(string, int) ([]map[string]any, error) {
	return nil, f.err
}
func (f *fakeYGAPI) GetTask(string) (map[string]any, error) { return f.task, f.err }
func (f *fakeYGAPI) CreateTask(body map[string]any) (map[string]any, error) {
	f.createBody = body
	return f.created, f.err
}
func (f *fakeYGAPI) UpdateTask(id string, body map[string]any) (map[string]any, error) {
	f.updateIDs = append(f.updateIDs, id)
	f.updates = append(f.updates, body)
	return map[string]any{}, f.err
}
func (f *fakeYGAPI) FindTaskByShortID(string, string, []string) (map[string]any, error) {
	return f.findResult, f.err
}
func (f *fakeYGAPI) PostChatMessage(chatID string, body map[string]any) error {
	f.chatIDs = append(f.chatIDs, chatID)
	f.chat = append(f.chat, body)
	return f.err
}
func (f *fakeYGAPI) CreateWebhook(string, string, []map[string]any) (map[string]any, error) {
	return map[string]any{"id": "wh-1"}, f.err
}
func (f *fakeYGAPI) UpdateWebhook(string, map[string]any) error { return f.err }

// fakeCipher — «шифрование» для тестов: префикс enc:.
type fakeCipher struct{}

func (fakeCipher) EncryptKey(plain string) ([]byte, error) { return []byte("enc:" + plain), nil }
func (fakeCipher) DecryptKey(enc []byte) (string, error) {
	s := string(enc)
	if strings.HasPrefix(s, "enc:") {
		return s[4:], nil
	}
	return "", nil
}

// ── сборка тестового окружения ───────────────────────────────────

type ygEnv struct {
	yg    *Yougile
	svc   *Service
	store *fakeStore
	repo  *fakeYGRepo
	api   *fakeYGAPI
	bus   *fakeBus
	users *fakeUsers
}

func newYGEnv() *ygEnv {
	svc, store, _, _, bus, users := newTestService()
	repo := newFakeYGRepo(store)
	api := &fakeYGAPI{}
	yg := NewYougile(YougileDeps{
		Service: svc, Repo: repo, Cipher: fakeCipher{},
		NewClient:  func(string) domain.YougileAPI { return api },
		PublicBase: "https://gw.example.com",
		GenSecret:  func() string { return "fixed-secret" },
		Log:        slog.New(slog.DiscardHandler),
	})
	return &ygEnv{yg: yg, svc: svc, store: store, repo: repo, api: api, bus: bus, users: users}
}

func (e *ygEnv) seedCompany(over func(c *domain.YougileCompany)) *domain.YougileCompany {
	c := &domain.YougileCompany{
		ID: 1, UsesYougile: true,
		YgCompanyID:     ptr("9347006b-dc75-4550-97d5-3008ba00d4a0"),
		YgProjectID:     ptr("proj-uuid"),
		YgBoardID:       ptr("board-uuid"),
		YgFirstColumnID: ptr("col-first-uuid"),
	}
	if over != nil {
		over(c)
	}
	e.repo.companies[c.ID] = c
	return c
}

func (e *ygEnv) seedUser() *domain.User {
	u := employee(e.users, 42, 1)
	// Подключённый аккаунт с расшифровываемым ключом.
	e.repo.accounts[u.ID] = &domain.YougileAccount{
		ID: 1, UserID: u.ID, CompanyID: 1,
		YgCompanyID: "9347006b-dc75-4550-97d5-3008ba00d4a0",
		YgUserID:    ptr("me-yg"), YgLogin: "a@b.c",
		KeyCiphertext: []byte("enc:KEY"), KeyFingerprint: "9aQ",
	}
	return u
}

// ── чистые функции ───────────────────────────────────────────────

func TestSyncHashStableAndDistinct(t *testing.T) {
	a := syncHash("A", 0, false)
	b := syncHash("A", 0, false)
	c := syncHash("A", 0, true)
	if a != b {
		t.Fatal("хеш нестабилен")
	}
	if a == c {
		t.Fatal("completed не влияет на хеш")
	}
	if len(a) != 40 {
		t.Fatalf("len(sha1 hex) = %d", len(a)) // sha1 hex
	}
}

// Паритет с Python: _sync_hash(title='A', deadline_ms=None, completed=False)
// == sha1("A||0").
func TestSyncHashParityWithPython(t *testing.T) {
	if got := syncHash("A", 0, false); got != "3083bf913172316016774e7c31a146d3bdd440a5" {
		t.Fatalf("hash(A,None,False) = %s", got)
	}
	if got := syncHash(" A ", 0, false); got != syncHash("A", 0, false) {
		t.Fatalf("strip() заголовка не применился")
	}
	if got := syncHash("A", 1717000000000, true); got != "7185b68c6e8e9674e3813de1ecef22ddccb55ea1" {
		t.Fatalf("hash(A,1717000000000,True) = %s", got)
	}
}

// ── проверки доступа ─────────────────────────────────────────────

func TestRequireCompanyDisabled(t *testing.T) {
	e := newYGEnv()
	company := e.seedCompany(func(c *domain.YougileCompany) { c.UsesYougile = false })
	err := requireCompanyEnabled(company)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "COMPANY_DISABLED" {
		t.Fatalf("ожидался COMPANY_DISABLED, получено %v", err)
	}
}

func TestRequireCompanyNotConfigured(t *testing.T) {
	e := newYGEnv()
	company := e.seedCompany(func(c *domain.YougileCompany) { c.YgBoardID = nil })
	err := requireCompanyEnabled(company)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "COMPANY_NOT_CONFIGURED" {
		t.Fatalf("ожидался COMPANY_NOT_CONFIGURED, получено %v", err)
	}
}

func TestRequireUserNotConnected(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := employee(e.users, 42, 1) // без аккаунта
	_, err := e.yg.ImportTask(context.Background(), u, dto.YougileImport{
		URL: "https://ru.yougile.com/x", DepartmentID: 5,
	}, "")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "USER_NOT_CONNECTED" || de.HTTPStatus != 412 {
		t.Fatalf("ожидался USER_NOT_CONNECTED 412, получено %v", err)
	}
}

// ── ImportTask ───────────────────────────────────────────────────

func TestImportBadURL(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := e.seedUser()
	_, err := e.yg.ImportTask(context.Background(), u,
		dto.YougileImport{URL: "not-a-url", DepartmentID: 5}, "")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "BAD_URL" {
		t.Fatalf("ожидался BAD_URL, получено %v", err)
	}
}

func TestImportForeignCompanyBlocked(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := e.seedUser()
	foreignURL := "https://ru.yougile.com/team/11111111-1111-1111-1111-111111111111/" +
		"#tasks?task=22222222-2222-2222-2222-222222222222"
	_, err := e.yg.ImportTask(context.Background(), u,
		dto.YougileImport{URL: foreignURL, DepartmentID: 5}, "")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "FOREIGN_COMPANY" {
		t.Fatalf("ожидался FOREIGN_COMPANY, получено %v", err)
	}
}

func TestImportSuccessWritesLinkAndPostsBack(t *testing.T) {
	e := newYGEnv()
	company := e.seedCompany(nil)
	u := e.seedUser()
	e.store.depts[5] = &domain.Department{ID: 5, Name: "Отдел", CompanyID: 1}

	const ygTaskID = "4f6f0391-0f94-4d30-9b0e-99430a36d4fb"
	url := "https://ru.yougile.com/team/" + *company.YgCompanyID + "/#tasks?task=" + ygTaskID
	e.api.task = map[string]any{
		"id": ygTaskID, "title": "Импорт из YG", "columnId": "col-yg",
		"deadline":  map[string]any{"deadline": float64(1717000000000), "startDate": float64(0), "withTime": false},
		"completed": false, "description": "desc",
	}

	out, err := e.yg.ImportTask(context.Background(), u,
		dto.YougileImport{URL: url, DepartmentID: 5, PullDeadline: true},
		"https://gw.example.com")
	if err != nil {
		t.Fatalf("импорт: %v", err)
	}

	task := e.store.tasks[out.ID]
	if task == nil {
		t.Fatal("задача не создана")
	}
	if task.CompanyID != 1 || task.DepartmentID != 5 {
		t.Fatalf("task = %+v", task)
	}
	if task.LinkYougile == nil || !strings.HasSuffix(*task.LinkYougile, "task="+ygTaskID) {
		t.Fatalf("link_yougile = %v", task.LinkYougile)
	}
	if task.Deadline == nil || task.Deadline.UnixMilli() != 1717000000000 {
		t.Fatalf("deadline = %v", task.Deadline)
	}
	if task.YougileTaskID == nil || *task.YougileTaskID != ygTaskID {
		t.Fatalf("yougile_task_id = %v", task.YougileTaskID)
	}
	if strOrEmpty(task.YougileProjectID) != "proj-uuid" ||
		strOrEmpty(task.YougileBoardID) != "board-uuid" ||
		strOrEmpty(task.YougileColumnID) != "col-yg" {
		t.Fatalf("структурные поля: %+v", task)
	}
	if task.YougileSyncHash == nil || *task.YougileSyncHash == "" {
		t.Fatal("sync_hash не записан")
	}

	// В YG-карточке оставили обратную ссылку на GW.
	if len(e.api.chat) != 1 || e.api.chatIDs[0] != ygTaskID {
		t.Fatalf("chat-сообщения: %v", e.api.chatIDs)
	}
	if !strings.Contains(e.api.chat[0]["text"].(string), "https://gw.example.com/tasks/") {
		t.Fatalf("текст линка: %v", e.api.chat[0]["text"])
	}

	// В GW появился системный комментарий.
	if len(e.store.comments) != 1 {
		t.Fatalf("комментариев = %d", len(e.store.comments))
	}

	// Сокет-событие task:created ушло в all.
	var created bool
	for _, ev := range e.bus.events {
		if ev.Event == "task:created" {
			created = true
		}
	}
	if !created {
		t.Fatal("task:created не опубликован")
	}
}

func TestImportExistingLinkReturnsExisting(t *testing.T) {
	e := newYGEnv()
	company := e.seedCompany(nil)
	u := e.seedUser()
	const ygTaskID = "4f6f0391-0f94-4d30-9b0e-99430a36d4fb"
	existing := seedTask(e.store, 1)
	existing.YougileTaskID = ptr(ygTaskID)

	url := "https://ru.yougile.com/team/" + *company.YgCompanyID + "/#tasks?task=" + ygTaskID
	out, err := e.yg.ImportTask(context.Background(), u,
		dto.YougileImport{URL: url, DepartmentID: 5}, "")
	if err != nil {
		t.Fatalf("импорт: %v", err)
	}
	if out.ID != existing.ID {
		t.Fatalf("ожидалась существующая задача %d, получена %d", existing.ID, out.ID)
	}
	if len(e.store.tasks) != 1 {
		t.Fatal("создана лишняя задача")
	}
	if e.api.createBody != nil {
		t.Fatal("create_task в YG не должен вызываться")
	}
}

// ── ExportTask ───────────────────────────────────────────────────

func TestExportSuccessWritesYougileFieldsAndPostsLinkBack(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := e.seedUser()
	task := seedTask(e.store, 1)
	task.Name = "Экспорт"
	deadline := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	task.Deadline = &deadline
	e.api.created = map[string]any{"id": "new-yg-task", "idTaskProject": "OIP1-9"}

	out, err := e.yg.ExportTask(context.Background(), u, task.ID, "https://gw.example.com")
	if err != nil {
		t.Fatalf("экспорт: %v", err)
	}
	if out.ID != task.ID {
		t.Fatalf("id = %d", out.ID)
	}

	// POST /tasks ушёл в первую колонку с assigned=[me-yg].
	body := e.api.createBody
	if body["columnId"] != "col-first-uuid" {
		t.Fatalf("columnId = %v", body["columnId"])
	}
	if assigned := body["assigned"].([]string); len(assigned) != 1 || assigned[0] != "me-yg" {
		t.Fatalf("assigned = %v", body["assigned"])
	}
	if body["title"] != "Экспорт" {
		t.Fatalf("title = %v", body["title"])
	}
	dl, ok := body["deadline"].(map[string]any)
	if !ok || dl["deadline"].(int64) != deadline.UnixMilli() {
		t.Fatalf("deadline = %v", body["deadline"])
	}

	// Задача обновилась: link_yougile + yougile_task_id + sync_hash.
	if strOrEmpty(task.YougileTaskID) != "new-yg-task" {
		t.Fatalf("yougile_task_id = %v", task.YougileTaskID)
	}
	if task.LinkYougile == nil || !strings.Contains(*task.LinkYougile, "OIP1-9") {
		t.Fatalf("link_yougile = %v", task.LinkYougile)
	}
	if task.YougileSyncHash == nil {
		t.Fatal("sync_hash не записан")
	}

	// Линк назад в GW + системный комментарий.
	if len(e.api.chat) != 1 {
		t.Fatalf("chat-сообщений = %d", len(e.api.chat))
	}
	if len(e.store.comments) != 1 {
		t.Fatalf("комментариев = %d", len(e.store.comments))
	}
}

func TestExportAlreadyLinkedRaises(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := e.seedUser()
	task := seedTask(e.store, 1)
	task.YougileTaskID = ptr("already")

	_, err := e.yg.ExportTask(context.Background(), u, task.ID, "")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "ALREADY_LINKED" {
		t.Fatalf("ожидался ALREADY_LINKED, получено %v", err)
	}
}

func TestExportTaskInOtherCompany(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := e.seedUser()
	task := seedTask(e.store, 999)

	_, err := e.yg.ExportTask(context.Background(), u, task.ID, "")
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "NOT_FOUND" || de.HTTPStatus != 404 {
		t.Fatalf("ожидался NOT_FOUND 404, получено %v", err)
	}
}

// ── UnlinkTask ───────────────────────────────────────────────────

func TestUnlinkClearsFieldsAndPostsComment(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := e.seedUser()
	task := seedTask(e.store, 1)
	task.YougileTaskID = ptr("x")
	task.LinkYougile = ptr("https://yg/x")

	out, err := e.yg.UnlinkTask(context.Background(), u, task.ID)
	if err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if out.ID != task.ID {
		t.Fatalf("id = %d", out.ID)
	}
	if task.YougileTaskID != nil || task.LinkYougile != nil {
		t.Fatalf("привязка не снята: %+v", task)
	}
	if len(e.store.comments) != 1 {
		t.Fatalf("комментариев = %d", len(e.store.comments))
	}
}

func TestUnlinkIdempotentNoLink(t *testing.T) {
	e := newYGEnv()
	e.seedCompany(nil)
	u := e.seedUser()
	task := seedTask(e.store, 1)

	out, err := e.yg.UnlinkTask(context.Background(), u, task.ID)
	if err != nil || out.ID != task.ID {
		t.Fatalf("unlink: %v", err)
	}
	if len(e.store.comments) != 0 {
		t.Fatal("комментарий на идемпотентной отвязке")
	}
}

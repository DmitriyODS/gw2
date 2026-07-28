package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
)

// ── Фейки портов (без БД/сети, как в callsvc/authsvc/msgsvc) ─────

type storedEmbedding struct {
	companyID int64
	vector    []float32
	model     string
}

type fakeRepo struct {
	companies map[int64]*domain.CompanyAI
	userAI    map[int64]*domain.UserAI // личные ИИ-настройки (ключ ассистента)
	tasks     map[int64]*domain.TaskText
	embedded  map[int64]storedEmbedding // task_id → эмбеддинг

	platform *domain.PlatformAI
	models   []*domain.AIModel

	searchHits []domain.SearchHit
	lastSearch struct {
		companyID int64
		model     string
		limit     int
	}
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		companies: map[int64]*domain.CompanyAI{},
		userAI:    map[int64]*domain.UserAI{},
		tasks:     map[int64]*domain.TaskText{},
		embedded:  map[int64]storedEmbedding{},
	}
}

// Платформенный ИИ в тестах: ключ задан, каталог из трёх моделей — этого
// достаточно, чтобы проверить и доступ, и тарификацию обращений.
func (r *fakeRepo) GetPlatformAI(_ context.Context) (*domain.PlatformAI, error) {
	if r.platform != nil {
		return r.platform, nil
	}
	return &domain.PlatformAI{
		Enabled: true, APIKeyEnc: []byte("enc:sk-platform"), BaseURL: "",
		ModelChat: domain.PlatformModelChat, ModelEmbedding: domain.PlatformModelEmbedding,
	}, nil
}

func (r *fakeRepo) UpdatePlatformAI(_ context.Context, p *domain.PlatformAI) error {
	r.platform = p
	return nil
}

func (r *fakeRepo) ListModels(_ context.Context) ([]*domain.AIModel, error) {
	if r.models != nil {
		return r.models, nil
	}
	return []*domain.AIModel{
		{Code: domain.PlatformModelChat, Title: "GPT", Kind: "chat", PricePerMTok: 6100, Selectable: true, IsActive: true},
		{Code: "gemini-3.1-flash-lite", Title: "GEMINI", Kind: "chat", PricePerMTok: 7600, Selectable: true, IsActive: true},
		{Code: domain.PlatformModelEmbedding, Title: "Эмбеддинги", Kind: "embedding", PricePerMTok: 516, IsActive: true},
	}, nil
}

func (r *fakeRepo) UpsertModel(_ context.Context, m *domain.AIModel) error {
	r.models = append(r.models, m)
	return nil
}

func (r *fakeRepo) CompanyOwner(_ context.Context, companyID int64) (int64, error) {
	if c, ok := r.companies[companyID]; ok {
		return c.OwnerID, nil
	}
	return 0, nil
}

func (r *fakeRepo) GetCompanyAI(_ context.Context, companyID int64) (*domain.CompanyAI, error) {
	c, ok := r.companies[companyID]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *fakeRepo) UpdateCompanyAI(_ context.Context, c *domain.CompanyAI) error {
	cp := *c
	r.companies[c.ID] = &cp
	return nil
}

func (r *fakeRepo) GetUserAI(_ context.Context, userID int64) (*domain.UserAI, error) {
	u, ok := r.userAI[userID]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (r *fakeRepo) UpsertUserAI(_ context.Context, u *domain.UserAI) error {
	cp := *u
	r.userAI[u.UserID] = &cp
	return nil
}

// MembershipLevel — в тестах администратор управляет одноимённой компанией
// (см. companyAdmin: id == companyID).
func (r *fakeRepo) MembershipLevel(_ context.Context, userID, companyID int64) (int, error) {
	if userID == companyID {
		return domain.LevelAdmin, nil
	}
	return 0, nil
}

func (r *fakeRepo) CountTasks(_ context.Context, companyID int64) (int, error) {
	n := 0
	for _, t := range r.tasks {
		if t.CompanyID != nil && *t.CompanyID == companyID {
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) CountEmbeddings(_ context.Context, companyID int64, model string) (int, error) {
	n := 0
	for _, e := range r.embedded {
		if e.companyID == companyID && (model == "" || e.model == model) {
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) FindUnindexedTaskIDs(_ context.Context, companyID int64, model string) ([]int64, error) {
	var out []int64
	for id, t := range r.tasks {
		if t.CompanyID == nil || *t.CompanyID != companyID {
			continue
		}
		if e, ok := r.embedded[id]; !ok || e.model != model {
			out = append(out, id)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetTaskText(_ context.Context, taskID int64) (*domain.TaskText, error) {
	t, ok := r.tasks[taskID]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (r *fakeRepo) ListTaskTexts(_ context.Context, ids []int64) ([]*domain.TaskText, error) {
	var out []*domain.TaskText
	for _, id := range ids {
		if t, ok := r.tasks[id]; ok {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeRepo) UpsertEmbedding(_ context.Context, taskID, companyID int64, vector []float32, model string) error {
	r.embedded[taskID] = storedEmbedding{companyID: companyID, vector: vector, model: model}
	return nil
}

func (r *fakeRepo) SearchEmbeddings(_ context.Context, companyID int64, _ []float32, model string, limit int) ([]domain.SearchHit, error) {
	r.lastSearch.companyID = companyID
	r.lastSearch.model = model
	r.lastSearch.limit = limit
	return r.searchHits, nil
}

func (r *fakeRepo) AICompanyIDs(_ context.Context) ([]int64, error) {
	var out []int64
	for id, c := range r.companies {
		if c.Enabled {
			out = append(out, id)
		}
	}
	return out, nil
}

func (r *fakeRepo) TVWeekContext(_ context.Context, _ int64, _, _ time.Time) (*domain.TVWeekContext, error) {
	return &domain.TVWeekContext{}, nil
}

// fakeFacts — in-memory кэш ТВ-фактов.
type fakeFacts struct {
	facts map[int64]*domain.TVFact
}

func newFakeFacts() *fakeFacts { return &fakeFacts{facts: map[int64]*domain.TVFact{}} }

func (f *fakeFacts) GetFact(_ context.Context, companyID int64) (*domain.TVFact, error) {
	return f.facts[companyID], nil
}

func (f *fakeFacts) SetFact(_ context.Context, companyID int64, fact *domain.TVFact, _ time.Duration) error {
	f.facts[companyID] = fact
	return nil
}

func (f *fakeFacts) DeleteFact(_ context.Context, companyID int64) {
	delete(f.facts, companyID)
}

type fakeLLM struct {
	chatResult *domain.ChatResult
	chatErr    error
	embedErr   error

	lastChat  domain.ChatParams
	lastEmbed struct {
		apiKey  string
		model   string
		texts   []string
		timeout time.Duration
	}
	embedCalls int
	chatCalls  int
}

func (l *fakeLLM) ChatOnce(_ context.Context, p domain.ChatParams) (*domain.ChatResult, error) {
	l.chatCalls++
	l.lastChat = p
	if l.chatErr != nil {
		return nil, l.chatErr
	}
	if l.chatResult != nil {
		res := *l.chatResult
		return &res, nil
	}
	return &domain.ChatResult{Content: "pong"}, nil
}

func (l *fakeLLM) Embed(_ context.Context, p domain.EmbedParams) ([][]float32, int, error) {
	l.embedCalls++
	l.lastEmbed.apiKey = p.APIKey
	l.lastEmbed.model = p.Model
	l.lastEmbed.texts = p.Texts
	l.lastEmbed.timeout = p.Timeout
	if l.embedErr != nil {
		return nil, 0, l.embedErr
	}
	out := make([][]float32, len(p.Texts))
	for i := range p.Texts {
		out[i] = []float32{0.1, 0.2, 0.3}
	}
	return out, 10 * len(p.Texts), nil
}

// fakeCipher — «шифрование» приписыванием префикса. misconfigured имитирует
// отсутствие AI_KEY_ENCRYPTION_KEY.
type fakeCipher struct {
	misconfigured bool
}

func (c *fakeCipher) Encrypt(plain string) ([]byte, error) {
	if c.misconfigured {
		return nil, domain.ErrSecretMisconfigured
	}
	return []byte("enc:" + plain), nil
}

func (c *fakeCipher) Decrypt(enc []byte) (string, bool) {
	if c.misconfigured {
		return "", false
	}
	s := string(enc)
	if !strings.HasPrefix(s, "enc:") {
		return "", false
	}
	return strings.TrimPrefix(s, "enc:"), true
}

// ── Хелперы ──────────────────────────────────────────────────────

func newTestService() (*Service, *fakeRepo, *fakeLLM) {
	repo := newFakeRepo()
	llm := &fakeLLM{}
	svc := New(repo, llm, &fakeCipher{}, newFakeFacts(), nil, nil, "", SupportConfig{}, slog.New(slog.DiscardHandler))
	return svc, repo, llm
}

// enabledCompany — компания с включёнными ИИ-возможностями. Ключа у неё нет:
// работа идёт на платформенном, токены тратит создатель (OwnerID).
func enabledCompany(id int64) *domain.CompanyAI {
	return &domain.CompanyAI{
		ID:         id,
		Enabled:    true,
		Shared:     true,
		FeatSearch: true,
		FeatTVFact: true,
		OwnerID:    id,
	}
}

// enabledUserAI — подключённый личный ключ ИИ-ассистента.
func enabledUserAI(userID int64) *domain.UserAI {
	hint := "sk-…1234"
	return &domain.UserAI{
		UserID:        userID,
		Enabled:       true,
		APIKeyEnc:     []byte("enc:sk-secret"),
		KeyHint:       &hint,
		ModelChat:     domain.PlatformModelChat,
		FeatAssistant: true,
		FeatNotes:     true,
	}
}

// companyAdmin — администратор (level 3) компании companyID. id == companyID,
// чтобы membership-проверка (fakeRepo.MembershipLevel) различала персоны: доступ
// к AI-настройкам скоупится компанией из пути, а не активной компанией сессии.
func companyAdmin(companyID int64) *domain.User {
	return &domain.User{ID: companyID, RoleLevel: domain.LevelAdmin, CompanyID: &companyID, CompanyActive: true}
}

func wantDomainError(t *testing.T, err error, code string, status int) {
	t.Helper()
	de := domain.AsDomainError(err)
	if de == nil {
		t.Fatalf("ожидалась domain.Error %s, получено: %v", code, err)
	}
	if de.Code != code || de.HTTPStatus != status {
		t.Fatalf("ожидалось %s/%d, получено %s/%d", code, status, de.Code, de.HTTPStatus)
	}
}

// ── Status ───────────────────────────────────────────────────────

func TestStatusEnabled(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)

	st, err := svc.Status(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	// Модели теперь платформенные: своих ключей и моделей у компании нет.
	if !st.Enabled || st.ModelChat != domain.PlatformModelChat ||
		st.ModelEmbedding != domain.PlatformModelEmbedding {
		t.Fatalf("неожиданный статус: %+v", st)
	}
}

func TestStatusDisabledWithoutError(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.Enabled = false
	repo.companies[1] = c

	st, err := svc.Status(context.Background(), 1)
	if err != nil || st.Enabled {
		t.Fatalf("выключенный AI: enabled=false без ошибки, получено %+v, %v", st, err)
	}

	// Компании нет — тоже enabled=false без ошибки.
	st, err = svc.Status(context.Background(), 999)
	if err != nil || st.Enabled {
		t.Fatalf("нет компании: enabled=false без ошибки, получено %+v, %v", st, err)
	}
}

// Нерасшифровываемый ПЛАТФОРМЕННЫЙ ключ (сменили AI_KEY_ENCRYPTION_KEY) тихо
// выключает ИИ у всех — падать при каждом обращении нельзя.
func TestStatusKeyUndecryptable(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)
	repo.platform = &domain.PlatformAI{Enabled: true, APIKeyEnc: []byte("garbage")}

	st, err := svc.Status(context.Background(), 1)
	if err != nil || st.Enabled {
		t.Fatalf("нерасшифровываемый ключ: enabled=false, получено %+v, %v", st, err)
	}
}

// ── Chat ─────────────────────────────────────────────────────────

func TestChatDisabled(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.Enabled = false
	repo.companies[1] = c

	_, err := svc.Chat(context.Background(), ChatArgs{CompanyID: 1, MessagesJSON: `[]`})
	wantDomainError(t, err, "AI_DISABLED", 403)
}

func TestChatPlainContent(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	llm.chatResult = &domain.ChatResult{Content: "  привет  "}

	res, err := svc.Chat(context.Background(), ChatArgs{
		CompanyID:    1,
		MessagesJSON: `[{"role":"user","content":"hi"}]`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != "привет" || res.ToolCallsJSON != "" {
		t.Fatalf("ожидался стрипнутый текст, получено %+v", res)
	}
	// Ключ и модель — ПЛАТФОРМЕННЫЕ (у компании своих нет), дефолты прежние.
	if llm.lastChat.APIKey != "sk-platform" || llm.lastChat.Model != domain.PlatformModelChat {
		t.Fatalf("ключ/модель: %+v", llm.lastChat)
	}
	if llm.lastChat.MaxTokens != 400 || llm.lastChat.Timeout != 30*time.Second {
		t.Fatalf("дефолты max_tokens/timeout: %+v", llm.lastChat)
	}
}

func TestChatPassesToolCalls(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	toolCalls := `[{"id":"call_1","type":"function","function":{"name":"get_stats","arguments":"{}"}}]`
	llm.chatResult = &domain.ChatResult{Content: "", ToolCallsJSON: toolCalls}

	res, err := svc.Chat(context.Background(), ChatArgs{
		CompanyID:    1,
		MessagesJSON: `[{"role":"user","content":"hi"}]`,
		ToolsJSON:    `[{"type":"function","function":{"name":"get_stats"}}]`,
		MaxTokens:    200,
		Temperature:  0.5,
		TimeoutSec:   12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ToolCallsJSON != toolCalls {
		t.Fatalf("tool_calls должны прокидываться сырым JSON, получено %q", res.ToolCallsJSON)
	}
	if llm.lastChat.MaxTokens != 200 || llm.lastChat.Temperature != 0.5 || llm.lastChat.Timeout != 12*time.Second {
		t.Fatalf("параметры не прокинуты: %+v", llm.lastChat)
	}
	if llm.lastChat.ToolsJSON == "" {
		t.Fatal("tools_json не прокинут в upstream")
	}
}

func TestChatInvalidMessagesJSON(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)

	_, err := svc.Chat(context.Background(), ChatArgs{CompanyID: 1, MessagesJSON: `{не json`})
	wantDomainError(t, err, "AI_BAD_REQUEST", 400)
}

// ── Настройки: доступ ────────────────────────────────────────────

func TestSettingsAccess(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)
	repo.companies[2] = enabledCompany(2)

	// Администратор своей компании — ок.
	if _, err := svc.GetSettings(context.Background(), companyAdmin(1), 1); err != nil {
		t.Fatalf("своя компания: %v", err)
	}
	// Администратор чужой компании — 403: AI-настройками управляет администратор
	// именно этой компании (членство), независимо от активной компании сессии.
	_, err := svc.GetSettings(context.Background(), companyAdmin(2), 1)
	wantDomainError(t, err, "FORBIDDEN", 403)
	// Компании нет — 404 без message (проверяется до доступа).
	_, err = svc.GetSettings(context.Background(), companyAdmin(99), 99)
	wantDomainError(t, err, "NOT_FOUND", 404)
	if de := domain.AsDomainError(err); de.Message != "" {
		t.Fatalf("NOT_FOUND без message, получено %q", de.Message)
	}
}

// ── Настройки: обновление ────────────────────────────────────────

// Тумблеры ИИ-возможностей компании: их правит администратор, а разрешение
// тратить токены (shared) — только создатель.
func TestUpdateCompanyFeatureToggles(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.FeatTVFact = false
	repo.companies[1] = c

	on := true
	out, err := svc.UpdateSettings(context.Background(), companyAdmin(1), 1, dto.AiSettingsUpdate{
		FeatTVFact: &on,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.FeatTVFact || !out.Enabled {
		t.Fatalf("тумблер ТВ-фактов должен включиться: %+v", out)
	}
	if !repo.companies[1].FeatTVFact {
		t.Fatal("значение должно сохраниться в компании")
	}
}

// Чужой администратор не может разрешить тратить токены создателя.
func TestSharedTokensOwnerOnly(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.OwnerID = 42 // создатель — другой человек
	repo.companies[1] = c

	on := true
	_, err := svc.UpdateSettings(context.Background(), companyAdmin(1), 1, dto.AiSettingsUpdate{Shared: &on})
	if err == nil {
		t.Fatal("ожидался отказ: разрешать трату своих токенов может только создатель")
	}
	if de := domain.AsDomainError(err); de == nil || de.Code != "OWNER_ONLY" {
		t.Fatalf("ожидался OWNER_ONLY, получено %v", err)
	}
}

// Выключенная возможность закрывает доступ, даже когда ИИ компании включён.
func TestFeatureToggleGatesAccess(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.FeatSearch = false
	repo.companies[1] = c

	client, err := svc.clientForCompany(context.Background(), 1, domain.FeatureSearch)
	if err != nil {
		t.Fatal(err)
	}
	if client != nil {
		t.Fatal("умный поиск выключен — доступа к модели быть не должно")
	}
	// А ТВ-факты по-прежнему разрешены.
	if client, _ := svc.clientForCompany(context.Background(), 1, domain.FeatureTVFact); client == nil {
		t.Fatal("включённая возможность должна работать")
	}
}

// Запрет тратить токены создателя выключает ИИ всей компании.
func TestSharedDisabledStopsCompanyAI(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.Shared = false
	repo.companies[1] = c

	if client, _ := svc.clientForCompany(context.Background(), 1, domain.FeatureSearch); client != nil {
		t.Fatal("создатель не разрешил тратить токены — ИИ компании недоступен")
	}
}

// Платформенный ключ шифруется при сохранении и действует сразу.
func TestUpdatePlatformSettingsEncryptsKey(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.platform = &domain.PlatformAI{}

	enabled := true
	key := "sk-proj-abcdef123456"
	out, err := svc.UpdatePlatformSettings(context.Background(), dto.PlatformAiUpdate{
		Enabled: &enabled, APIKey: &key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Enabled || !out.HasKey || out.KeyHint != "sk-…3456" {
		t.Fatalf("ожидался включённый ключ с маской: %+v", out)
	}
	if string(repo.platform.APIKeyEnc) != "enc:"+key {
		t.Fatalf("ключ должен храниться зашифрованным, получено %q", repo.platform.APIKeyEnc)
	}
	repo.companies[1] = enabledCompany(1)
	if st, _ := svc.Status(context.Background(), 1); !st.Enabled {
		t.Fatal("после сохранения ключа ИИ должен работать сразу (кэш сброшен)")
	}
}

// clear_key стирает платформенный ключ, и ИИ выключается без ожидания TTL.
func TestUpdatePlatformSettingsClearKey(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)
	if st, _ := svc.Status(context.Background(), 1); !st.Enabled {
		t.Fatal("прекондиция: ИИ включён")
	}

	out, err := svc.UpdatePlatformSettings(context.Background(), dto.PlatformAiUpdate{ClearKey: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.HasKey || out.KeyHint != "" {
		t.Fatalf("clear_key должен стереть ключ и маску: %+v", out)
	}
	if st, _ := svc.Status(context.Background(), 1); st.Enabled {
		t.Fatal("после clear_key ИИ должен выключиться сразу")
	}
}

func TestTestSettingsDisabled(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.Enabled = false // ИИ компании выключен — проверять нечего
	repo.companies[1] = c

	_, err := svc.TestSettings(context.Background(), companyAdmin(1), 1)
	wantDomainError(t, err, "AI_DISABLED", 409)
}

func TestTestSettingsCollectsErrors(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	llm.embedErr = fmt.Errorf("bad model")

	res, err := svc.TestSettings(context.Background(), companyAdmin(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Chat || res.Embedding {
		t.Fatalf("ожидалось chat=true embedding=false: %+v", res)
	}
	if res.Error == nil || *res.Error != " embedding: bad model" {
		t.Fatalf("формат ошибки как во Flask ((error or '') + ' embedding: ...'), получено %v", res.Error)
	}
}

// ── SemanticSearch ───────────────────────────────────────────────

func TestSemanticSearchFailOpen(t *testing.T) {
	svc, repo, llm := newTestService()
	c := enabledCompany(1)
	c.Enabled = false
	repo.companies[1] = c

	// Выключенный AI → пустая выдача без ошибки.
	hits, err := svc.SemanticSearch(context.Background(), 1, "котики")
	if err != nil || hits != nil {
		t.Fatalf("fail-open при выключенном AI: %v, %v", hits, err)
	}
	// Пустой запрос → пусто (LLM не дёргается).
	repo.companies[1] = enabledCompany(1)
	if hits, err := svc.SemanticSearch(context.Background(), 1, "   "); err != nil || hits != nil || llm.embedCalls != 0 {
		t.Fatalf("пустой запрос: %v, %v, embedCalls=%d", hits, err, llm.embedCalls)
	}
	// Ошибка эмбеддинга → пусто без ошибки.
	llm.embedErr = fmt.Errorf("boom")
	if hits, err := svc.SemanticSearch(context.Background(), 1, "котики"); err != nil || hits != nil {
		t.Fatalf("fail-open при ошибке эмбеддинга: %v, %v", hits, err)
	}
}

func TestSemanticSearchFiltersScore(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	repo.searchHits = []domain.SearchHit{
		{TaskID: 1, Score: 0.90}, // лучший
		{TaskID: 2, Score: 0.82}, // в пределах band от лучшего — пройдёт
		{TaskID: 3, Score: 0.50}, // отстал от лучшего > band — обрыв
		{TaskID: 4, Score: 0.20}, // ниже абсолютного порога
	}

	hits, err := svc.SemanticSearch(context.Background(), 1, "котики")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 || hits[0].TaskID != 1 || hits[1].TaskID != 2 {
		t.Fatalf("ожидались только близкие к лучшему хиты: %+v", hits)
	}
	if repo.lastSearch.model != "text-embedding-3-small" || repo.lastSearch.limit != 50 {
		t.Fatalf("фильтр model и лимит 50 обязательны: %+v", repo.lastSearch)
	}
	if llm.lastEmbed.timeout != 4*time.Second {
		t.Fatalf("таймаут эмбеддинга запроса 4с, получено %v", llm.lastEmbed.timeout)
	}
}

// ── Эмбеддинги задач ─────────────────────────────────────────────

func TestReindexTaskOnce(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	cid := int64(1)
	dep := "Разработка"
	fio := "Иванов Иван"
	repo.tasks[5] = &domain.TaskText{ID: 5, CompanyID: &cid, Name: "Починить баг", DepartmentName: &dep, ResponsibleFIO: &fio}

	if err := svc.reindexTaskOnce(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	e, ok := repo.embedded[5]
	if !ok || e.model != "text-embedding-3-small" || e.companyID != 1 {
		t.Fatalf("эмбеддинг не сохранён: %+v", repo.embedded)
	}
	want := "Починить баг\nОтдел: Разработка\nОтветственный: Иванов Иван"
	if len(llm.lastEmbed.texts) != 1 || llm.lastEmbed.texts[0] != want {
		t.Fatalf("текст эмбеддинга:\n%q\nожидался:\n%q", llm.lastEmbed.texts, want)
	}
}

func TestReindexTaskSkips(t *testing.T) {
	svc, repo, llm := newTestService()
	c := enabledCompany(1)
	c.Enabled = false
	repo.companies[1] = c
	cid := int64(1)
	repo.tasks[5] = &domain.TaskText{ID: 5, CompanyID: &cid, Name: "Задача"}

	// AI выключен — тихо пропускаем.
	if err := svc.reindexTaskOnce(context.Background(), 5); err != nil || len(repo.embedded) != 0 {
		t.Fatalf("выключенный AI: пропуск без ошибки, %v", err)
	}
	// Задачи нет — тоже не ошибка.
	if err := svc.reindexTaskOnce(context.Background(), 404); err != nil {
		t.Fatalf("нет задачи: %v", err)
	}
	// Пустой текст — пропуск, LLM не дёргается.
	repo.companies[1] = enabledCompany(1)
	repo.tasks[6] = &domain.TaskText{ID: 6, CompanyID: &cid, Name: ""}
	if err := svc.reindexTaskOnce(context.Background(), 6); err != nil || llm.embedCalls != 0 {
		t.Fatalf("пустой текст: пропуск, %v, embedCalls=%d", err, llm.embedCalls)
	}
}

func TestBackfillAndIndexingStatus(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)
	cid := int64(1)
	for i := int64(1); i <= 3; i++ {
		repo.tasks[i] = &domain.TaskText{ID: i, CompanyID: &cid, Name: fmt.Sprintf("Задача %d", i)}
	}
	// Один эмбеддинг устаревшей моделью — должен пересчитаться.
	repo.embedded[1] = storedEmbedding{companyID: 1, model: "old-model"}

	before, err := svc.IndexingStatus(context.Background(), companyAdmin(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if before.TotalTasks != 3 || before.Indexed != 0 || before.Pending != 3 || !before.AiEnabled {
		t.Fatalf("до бэкфилла: %+v", before)
	}

	svc.runBackfill(context.Background(), 1)

	after, err := svc.IndexingStatus(context.Background(), companyAdmin(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if after.Indexed != 3 || after.Pending != 0 {
		t.Fatalf("после бэкфилла: %+v", after)
	}
	if repo.embedded[1].model != "text-embedding-3-small" {
		t.Fatal("эмбеддинг устаревшей модели должен быть пересчитан")
	}
}

func TestIndexingStatusDisabledCompany(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.Enabled = false
	repo.companies[1] = c
	cid := int64(1)
	repo.tasks[1] = &domain.TaskText{ID: 1, CompanyID: &cid, Name: "Задача"}

	st, err := svc.IndexingStatus(context.Background(), companyAdmin(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	// find_unindexed_task_ids смотрит только ai_enabled-компании → pending 0.
	if st.Pending != 0 || st.AiEnabled {
		t.Fatalf("выключенный AI: pending=0, ai_enabled=false, получено %+v", st)
	}
}

func TestStartReindex(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)
	cid := int64(1)
	repo.tasks[1] = &domain.TaskText{ID: 1, CompanyID: &cid, Name: "Задача"}

	out, err := svc.StartReindex(context.Background(), companyAdmin(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Queued || out.Pending != 1 {
		t.Fatalf("ожидалось queued=true pending=1: %+v", out)
	}
	// Бэкфилл идёт в фоне — дожидаемся результата.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := repo.embedded[1]; ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("фоновый бэкфилл не проиндексировал задачу")
}

func TestStartReindexDisabled(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.Enabled = false
	repo.companies[1] = c

	_, err := svc.StartReindex(context.Background(), companyAdmin(1), 1)
	wantDomainError(t, err, "AI_DISABLED", 409)
}

// ── Embed (gRPC) ─────────────────────────────────────────────────

func TestEmbedDisabled(t *testing.T) {
	svc, repo, _ := newTestService()
	c := enabledCompany(1)
	c.Enabled = false
	repo.companies[1] = c

	_, _, err := svc.Embed(context.Background(), 1, "текст")
	wantDomainError(t, err, "AI_DISABLED", 403)
}

func TestEmbedReturnsModelAndVector(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)

	vec, model, err := svc.Embed(context.Background(), 1, "текст")
	if err != nil {
		t.Fatal(err)
	}
	if model != "text-embedding-3-small" || len(vec) != 3 {
		t.Fatalf("vec=%v model=%q", vec, model)
	}
}

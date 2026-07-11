package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// ── Фейки: диалог ассистента + gRPC-клиент tasksvc ────────────────

type fakeFeedbackRow struct {
	verdict string
	reason  *string
}

type fakeAssistantRepo struct {
	conv       map[[2]int64]*domain.AssistantConversation
	messages   map[int64][]domain.AssistantMessage
	feedback   map[int64]fakeFeedbackRow // message_id → голос (пишет только владелец)
	nextConvID int64
	nextMsgID  int64
}

func newFakeAssistantRepo() *fakeAssistantRepo {
	return &fakeAssistantRepo{
		conv:     map[[2]int64]*domain.AssistantConversation{},
		messages: map[int64][]domain.AssistantMessage{},
		feedback: map[int64]fakeFeedbackRow{},
	}
}

func (r *fakeAssistantRepo) GetOrCreateConversation(_ context.Context, userID, companyID int64) (*domain.AssistantConversation, error) {
	key := [2]int64{userID, companyID}
	if c, ok := r.conv[key]; ok {
		return c, nil
	}
	r.nextConvID++
	c := &domain.AssistantConversation{ID: r.nextConvID, UserID: userID, CompanyID: companyID, CreatedAt: time.Now()}
	r.conv[key] = c
	return c, nil
}

func (r *fakeAssistantRepo) RecentMessages(_ context.Context, conversationID int64, limit int) ([]domain.AssistantMessage, error) {
	msgs := r.messages[conversationID]
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	out := make([]domain.AssistantMessage, len(msgs))
	copy(out, msgs)
	return out, nil
}

func (r *fakeAssistantRepo) History(_ context.Context, conversationID int64, limit int, before *time.Time) ([]domain.AssistantMessage, error) {
	msgs := r.messages[conversationID]
	out := []domain.AssistantMessage{}
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if before != nil && !m.CreatedAt.Before(*before) {
			continue
		}
		if fb, ok := r.feedback[m.ID]; ok {
			verdict := fb.verdict
			m.MyFeedback = &verdict
		}
		out = append(out, m)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *fakeAssistantRepo) AppendMessage(_ context.Context, conversationID int64, role, text string, sources *string) (*domain.AssistantMessage, error) {
	r.nextMsgID++
	m := domain.AssistantMessage{ID: r.nextMsgID, ConversationID: conversationID, Role: role, Text: text, Sources: sources, CreatedAt: time.Now()}
	r.messages[conversationID] = append(r.messages[conversationID], m)
	return &m, nil
}

// UpsertFeedback — та же семантика, что у SQL-репозитория: голос проходит,
// только если сообщение — ответ ассистента в диалоге именно (userID, companyID).
func (r *fakeAssistantRepo) UpsertFeedback(_ context.Context, messageID, userID, companyID int64, verdict string, reason *string) (bool, error) {
	for _, c := range r.conv {
		if c.UserID != userID || c.CompanyID != companyID {
			continue
		}
		for _, m := range r.messages[c.ID] {
			if m.ID == messageID && m.Role == domain.AssistantRoleAssistant {
				r.feedback[messageID] = fakeFeedbackRow{verdict: verdict, reason: reason}
				return true, nil
			}
		}
	}
	return false, nil
}

// fakeTasksClient — gRPC-клиент tasksvc (инструменты статистики/поиска).
type fakeTasksClient struct {
	summary    *domain.StatsSummary
	summaryErr error
	depts      []domain.DepartmentStat
	employees  []domain.EmployeeStat
	unitTypes  []domain.UnitTypeStat
	calendar   []domain.CalendarDayStat
	searchRes  []domain.TaskRef
	taskLink   *domain.TaskRef
	linkErr    error

	calls      []string
	lastPeriod string
	lastLimit  int
	lastTaskID int64
}

func (c *fakeTasksClient) GetStatsSummary(_ context.Context, _ int64, period string) (*domain.StatsSummary, error) {
	c.calls = append(c.calls, "GetStatsSummary")
	c.lastPeriod = period
	if c.summaryErr != nil {
		return nil, c.summaryErr
	}
	if c.summary != nil {
		return c.summary, nil
	}
	return &domain.StatsSummary{}, nil
}

func (c *fakeTasksClient) ListDepartments(_ context.Context, _ int64, period string) ([]domain.DepartmentStat, error) {
	c.calls = append(c.calls, "ListDepartments")
	c.lastPeriod = period
	return c.depts, nil
}

func (c *fakeTasksClient) GetTopEmployees(_ context.Context, _ int64, period string, limit int) ([]domain.EmployeeStat, error) {
	c.calls = append(c.calls, "GetTopEmployees")
	c.lastPeriod = period
	c.lastLimit = limit
	return c.employees, nil
}

func (c *fakeTasksClient) GetStatsByUnitTypes(_ context.Context, _ int64, period string) ([]domain.UnitTypeStat, error) {
	c.calls = append(c.calls, "GetStatsByUnitTypes")
	c.lastPeriod = period
	return c.unitTypes, nil
}

func (c *fakeTasksClient) GetStatsCalendar(_ context.Context, _ int64, period string) ([]domain.CalendarDayStat, error) {
	c.calls = append(c.calls, "GetStatsCalendar")
	c.lastPeriod = period
	return c.calendar, nil
}

func (c *fakeTasksClient) SearchTasks(_ context.Context, _ int64, _ string, limit int) ([]domain.TaskRef, error) {
	c.calls = append(c.calls, "SearchTasks")
	c.lastLimit = limit
	return c.searchRes, nil
}

func (c *fakeTasksClient) GetTaskLink(_ context.Context, _ int64, taskID int64) (*domain.TaskRef, error) {
	c.calls = append(c.calls, "GetTaskLink")
	c.lastTaskID = taskID
	if c.linkErr != nil {
		return nil, c.linkErr
	}
	return c.taskLink, nil
}

// sequencedLLM — фиксированная последовательность ответов chat completion,
// по одному на каждую итерацию tools-цикла (в отличие от fakeLLM в
// service_test.go, который всегда отдаёт один и тот же результат).
type sequencedLLM struct {
	results    []*domain.ChatResult
	i          int
	toolsJSONs []string // ToolsJSON каждого вызова, в порядке вызовов
}

func (l *sequencedLLM) ChatOnce(_ context.Context, p domain.ChatParams) (*domain.ChatResult, error) {
	l.toolsJSONs = append(l.toolsJSONs, p.ToolsJSON)
	idx := l.i
	l.i++
	if idx < len(l.results) {
		return l.results[idx], nil
	}
	return &domain.ChatResult{Content: "done"}, nil
}

func (l *sequencedLLM) Embed(_ context.Context, _, _ string, texts []string, _ time.Duration) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0.1, 0.2, 0.3}
	}
	return out, nil
}

// toolCallMessage — JSON, который parseAssistantToolCalls ожидает найти в
// ChatResult.ToolCallsJSON: массив OpenAI tool_calls.
func toolCallMessage(t *testing.T, id, name string, args map[string]any) string {
	t.Helper()
	rawArgs, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	calls := []map[string]any{{
		"id":   id,
		"type": "function",
		"function": map[string]any{
			"name":      name,
			"arguments": string(rawArgs),
		},
	}}
	raw, err := json.Marshal(calls)
	if err != nil {
		t.Fatalf("marshal tool_calls: %v", err)
	}
	return string(raw)
}

func newAssistantTestService(tasks *fakeTasksClient, assistants *fakeAssistantRepo, llm domain.LLMClient) (*Service, *fakeRepo) {
	repo := newFakeRepo()
	repo.companies[1] = enabledCompany(1)
	return New(repo, llm, &fakeCipher{}, newFakeFacts(), assistants, tasks, "https://gw.example", SupportConfig{}, slog.New(slog.DiscardHandler)), repo
}

// ── SendAssistantMessage ───────────────────────────────────────────

func TestSendAssistantMessage_EmptyTextIsValidationError(t *testing.T) {
	svc, _ := newAssistantTestService(&fakeTasksClient{}, newFakeAssistantRepo(), &sequencedLLM{})
	_, err := svc.SendAssistantMessage(context.Background(), 10, 1, "   ")
	wantDomainError(t, err, "VALIDATION", 400)
}

func TestSendAssistantMessage_AiDisabledCompany(t *testing.T) {
	repo := newFakeRepo() // компания 1 НЕ добавлена → AI выключен
	svc := New(repo, &sequencedLLM{}, &fakeCipher{}, newFakeFacts(),
		newFakeAssistantRepo(), &fakeTasksClient{}, "https://gw.example", SupportConfig{}, slog.New(slog.DiscardHandler))

	_, err := svc.SendAssistantMessage(context.Background(), 10, 1, "Привет")
	wantDomainError(t, err, "AI_DISABLED", 409)
}

// TestSendAssistantMessage_StatsToolRoundTrip — модель просит статистику
// (tool_call get_stats_summary) → диспетчер зовёт tasksClient с тем же
// периодом → следующий ход модели получает результат инструмента и отвечает
// текстом; оба сообщения (user+assistant) сохраняются в историю.
func TestSendAssistantMessage_StatsToolRoundTrip(t *testing.T) {
	tasks := &fakeTasksClient{summary: &domain.StatsSummary{
		NewCount: 5, ClosedCount: 3, InProgressCount: 2, DebtCount: 1,
		TotalHours: 12.5, PeriodLabel: "эта неделя",
	}}
	assistants := newFakeAssistantRepo()
	llm := &sequencedLLM{results: []*domain.ChatResult{
		{ToolCallsJSON: toolCallMessage(t, "call_1", "get_stats_summary", map[string]any{"period": "this_week"})},
		{Content: "На этой неделе поступило 5 задач, закрыто 3."},
	}}
	svc, _ := newAssistantTestService(tasks, assistants, llm)

	reply, err := svc.SendAssistantMessage(context.Background(), 10, 1, "Сколько задач на этой неделе?")
	if err != nil {
		t.Fatalf("SendAssistantMessage: %v", err)
	}
	if reply.Text != "На этой неделе поступило 5 задач, закрыто 3." {
		t.Fatalf("неожиданный ответ: %q", reply.Text)
	}
	if len(tasks.calls) != 1 || tasks.calls[0] != "GetStatsSummary" {
		t.Fatalf("ожидался ровно один вызов GetStatsSummary, получено: %v", tasks.calls)
	}
	if tasks.lastPeriod != "this_week" {
		t.Fatalf("период не форвардится верно: %q", tasks.lastPeriod)
	}

	conv, _ := assistants.GetOrCreateConversation(context.Background(), 10, 1)
	saved := assistants.messages[conv.ID]
	if len(saved) != 2 {
		t.Fatalf("ожидалось 2 сохранённых сообщения (user+assistant), получено %d", len(saved))
	}
	if saved[0].Role != domain.AssistantRoleUser || saved[0].Text != "Сколько задач на этой неделе?" {
		t.Fatalf("первое сообщение не user-реплика: %+v", saved[0])
	}
	if saved[1].Role != domain.AssistantRoleAssistant || saved[1].Text != reply.Text {
		t.Fatalf("второе сообщение не ответ ассистента: %+v", saved[1])
	}
	// Провенанс: реально вызванный инструмент попал в sources сохранённого
	// сообщения и в ответ; реплика пользователя — без sources.
	if saved[0].Sources != nil {
		t.Fatalf("у реплики пользователя не должно быть sources: %+v", saved[0])
	}
	wantSources := "Данные: статистика за эту неделю"
	if saved[1].Sources == nil || *saved[1].Sources != wantSources {
		t.Fatalf("sources ответа: ожидалось %q, получено %+v", wantSources, saved[1].Sources)
	}
	if reply.Sources == nil || *reply.Sources != wantSources {
		t.Fatalf("reply.Sources: ожидалось %q, получено %+v", wantSources, reply.Sources)
	}
	if reply.ID != saved[1].ID || reply.ID == 0 {
		t.Fatalf("reply.ID должен быть id сохранённого сообщения из БД: reply=%d saved=%d", reply.ID, saved[1].ID)
	}
}

func TestSendAssistantMessage_NoToolCallsSkipsTasksClient(t *testing.T) {
	tasks := &fakeTasksClient{}
	llm := &sequencedLLM{results: []*domain.ChatResult{{Content: "Здравствуйте, чем могу помочь?"}}}
	svc, _ := newAssistantTestService(tasks, newFakeAssistantRepo(), llm)

	reply, err := svc.SendAssistantMessage(context.Background(), 10, 1, "Привет")
	if err != nil {
		t.Fatalf("SendAssistantMessage: %v", err)
	}
	if reply.Text != "Здравствуйте, чем могу помочь?" {
		t.Fatalf("неожиданный ответ: %q", reply.Text)
	}
	if len(tasks.calls) != 0 {
		t.Fatalf("tasksClient не должен вызываться без tool_calls: %v", tasks.calls)
	}
	if reply.Sources != nil {
		t.Fatalf("без инструментов sources должен быть nil: %+v", reply.Sources)
	}
}

// TestSendAssistantMessage_SourcesMultiToolDedup — несколько инструментов за
// один ответ → все подписи через запятую, повтор того же вызова не дублируется.
func TestSendAssistantMessage_SourcesMultiToolDedup(t *testing.T) {
	tasks := &fakeTasksClient{
		employees: []domain.EmployeeStat{{FIO: "Иванов И.И.", TaskCount: 2, Hours: 6}},
		taskLink:  &domain.TaskRef{ID: 77, Name: "Собрать отчёт"},
	}
	assistants := newFakeAssistantRepo()
	llm := &sequencedLLM{results: []*domain.ChatResult{
		{ToolCallsJSON: toolCallMessage(t, "call_1", "get_top_employees", map[string]any{"period": "this_month"})},
		{ToolCallsJSON: toolCallMessage(t, "call_2", "find_task", map[string]any{"query": "отчёт"})},
		{ToolCallsJSON: toolCallMessage(t, "call_3", "get_top_employees", map[string]any{"period": "this_month"})},
		{Content: "Готово."},
	}}
	svc, repo := newAssistantTestService(tasks, assistants, llm)
	repo.searchHits = []domain.SearchHit{{TaskID: 77, Score: 0.9}}

	reply, err := svc.SendAssistantMessage(context.Background(), 10, 1, "Кто лидер и где отчёт?")
	if err != nil {
		t.Fatalf("SendAssistantMessage: %v", err)
	}
	want := "Данные: лидеры за этот месяц, поиск задачи „отчёт“"
	if reply.Sources == nil || *reply.Sources != want {
		t.Fatalf("sources: ожидалось %q, получено %+v", want, reply.Sources)
	}
}

// TestSendAssistantMessage_FailedToolNotInSources — упавший инструмент данных
// не дал и в провенанс не попадает.
func TestSendAssistantMessage_FailedToolNotInSources(t *testing.T) {
	tasks := &fakeTasksClient{summaryErr: errors.New("tasksvc unavailable")}
	llm := &sequencedLLM{results: []*domain.ChatResult{
		{ToolCallsJSON: toolCallMessage(t, "call_1", "get_stats_summary", map[string]any{"period": "today"})},
		{Content: "Не удалось получить статистику."},
	}}
	svc, _ := newAssistantTestService(tasks, newFakeAssistantRepo(), llm)

	reply, err := svc.SendAssistantMessage(context.Background(), 10, 1, "Как дела сегодня?")
	if err != nil {
		t.Fatalf("SendAssistantMessage: %v", err)
	}
	if reply.Sources != nil {
		t.Fatalf("упавший инструмент не должен попадать в sources: %+v", reply.Sources)
	}
}

// ── dispatchAssistantTool: инструменты статистики ──────────────────

func TestDispatchAssistantTool_StatsToolsForwardPeriodAndMapFields(t *testing.T) {
	tasks := &fakeTasksClient{
		depts:     []domain.DepartmentStat{{ID: 1, Name: "Разработка", NewCount: 4}},
		employees: []domain.EmployeeStat{{FIO: "Иванов И.И.", TaskCount: 2, Hours: 6}},
		unitTypes: []domain.UnitTypeStat{{Name: "Звонок", Hours: 1.5, TaskCount: 1}},
		calendar:  []domain.CalendarDayStat{{Date: "2026-07-06", NewCount: 2, ClosedCount: 1, Hours: 3}},
	}
	svc, _ := newAssistantTestService(tasks, newFakeAssistantRepo(), &sequencedLLM{})
	ctx := context.Background()

	svc.dispatchAssistantTool(ctx, "list_departments", map[string]any{"period": "last_month"}, 1)
	if tasks.lastPeriod != "last_month" {
		t.Fatalf("list_departments: период не форвардится: %q", tasks.lastPeriod)
	}
	res := svc.dispatchAssistantTool(ctx, "get_top_employees", map[string]any{"period": "7d", "limit": float64(5)}, 1)
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("get_top_employees: неожиданный тип результата %T", res)
	}
	if tasks.lastLimit != 5 {
		t.Fatalf("limit не форвардится: %d", tasks.lastLimit)
	}
	employees, _ := m["employees"].([]map[string]any)
	if len(employees) != 1 || employees[0]["fio"] != "Иванов И.И." {
		t.Fatalf("employees mismatch: %+v", m)
	}

	res = svc.dispatchAssistantTool(ctx, "get_stats_by_unit_types", map[string]any{"period": "today"}, 1)
	m = res.(map[string]any)
	unitTypes, _ := m["unit_types"].([]map[string]any)
	if len(unitTypes) != 1 || unitTypes[0]["name"] != "Звонок" {
		t.Fatalf("unit_types mismatch: %+v", m)
	}

	res = svc.dispatchAssistantTool(ctx, "get_stats_calendar", map[string]any{"period": "yesterday"}, 1)
	m = res.(map[string]any)
	days, _ := m["days"].([]map[string]any)
	if len(days) != 1 || days[0]["date"] != "2026-07-06" {
		t.Fatalf("days mismatch: %+v", m)
	}
}

func TestDispatchAssistantTool_GRPCErrorReturnsToolErrorNotPanic(t *testing.T) {
	tasks := &fakeTasksClient{summaryErr: errors.New("tasksvc unavailable")}
	svc, _ := newAssistantTestService(tasks, newFakeAssistantRepo(), &sequencedLLM{})

	res := svc.dispatchAssistantTool(context.Background(), "get_stats_summary", nil, 1)
	m, ok := res.(map[string]any)
	if !ok || m["error"] != "tool_execution_failed" {
		t.Fatalf("ожидался {\"error\":\"tool_execution_failed\"}, получено: %+v", res)
	}
}

func TestDispatchAssistantTool_UnknownToolName(t *testing.T) {
	svc, _ := newAssistantTestService(&fakeTasksClient{}, newFakeAssistantRepo(), &sequencedLLM{})
	res := svc.dispatchAssistantTool(context.Background(), "delete_everything", nil, 1)
	m := res.(map[string]any)
	if m["error"] != "unknown_tool:delete_everything" {
		t.Fatalf("неожиданный результат для неизвестного инструмента: %+v", res)
	}
}

// ── find_task ────────────────────────────────────────────────────

func TestDispatchAssistantTool_FindTaskBuildsURLFromBestHit(t *testing.T) {
	tasks := &fakeTasksClient{
		taskLink: &domain.TaskRef{ID: 77, Name: "Собрать отчёт", ResponsibleFIO: "Сидоров С.С."},
	}
	svc, repo := newAssistantTestService(tasks, newFakeAssistantRepo(), &sequencedLLM{})
	// Лучшее совпадение — первый элемент (SearchEmbeddings уже упорядочен по
	// убыванию score в постгресе; фейк-репозиторий отдаёт как есть).
	repo.searchHits = []domain.SearchHit{{TaskID: 77, Score: 0.9}, {TaskID: 5, Score: 0.4}}

	res := svc.dispatchAssistantTool(context.Background(), "find_task", map[string]any{"query": "отчёт"}, 1)
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("неожиданный тип результата: %T", res)
	}
	if m["found"] != true {
		t.Fatalf("found=false, ожидалось true: %+v", m)
	}
	if m["id"] != int64(77) || m["name"] != "Собрать отчёт" || m["responsible_fio"] != "Сидоров С.С." {
		t.Fatalf("поля задачи некорректны: %+v", m)
	}
	if m["url"] != "https://gw.example/tasks/77" {
		t.Fatalf("url собран неверно: %+v", m["url"])
	}
	if len(tasks.calls) != 1 || tasks.calls[0] != "GetTaskLink" || tasks.lastTaskID != 77 {
		t.Fatalf("GetTaskLink должен звать именно лучший хит (77): calls=%v lastTaskID=%d", tasks.calls, tasks.lastTaskID)
	}
}

func TestDispatchAssistantTool_FindTaskNoHitsReturnsNotFound(t *testing.T) {
	svc, repo := newAssistantTestService(&fakeTasksClient{}, newFakeAssistantRepo(), &sequencedLLM{})
	repo.searchHits = nil

	res := svc.dispatchAssistantTool(context.Background(), "find_task", map[string]any{"query": "нет такой задачи"}, 1)
	m := res.(map[string]any)
	if m["found"] != false {
		t.Fatalf("ожидалось found=false при пустой семантической выдаче: %+v", m)
	}
}

func TestDispatchAssistantTool_FindTaskLinkGoneReturnsNotFound(t *testing.T) {
	// Семантика нашла задачу, но GetTaskLink говорит, что её больше нет
	// (удалена/другая компания) — не должно ронять диспетчер.
	tasks := &fakeTasksClient{taskLink: nil}
	svc, repo := newAssistantTestService(tasks, newFakeAssistantRepo(), &sequencedLLM{})
	repo.searchHits = []domain.SearchHit{{TaskID: 999, Score: 0.5}}

	res := svc.dispatchAssistantTool(context.Background(), "find_task", map[string]any{"query": "что-то"}, 1)
	m := res.(map[string]any)
	if m["found"] != false {
		t.Fatalf("ожидалось found=false, когда GetTaskLink вернул nil: %+v", m)
	}
}

func TestDispatchAssistantTool_FindTaskEmptyQuery(t *testing.T) {
	svc, _ := newAssistantTestService(&fakeTasksClient{}, newFakeAssistantRepo(), &sequencedLLM{})
	res := svc.dispatchAssistantTool(context.Background(), "find_task", map[string]any{"query": "   "}, 1)
	m := res.(map[string]any)
	if m["error"] != "query_required" {
		t.Fatalf("пустой query должен вернуть query_required: %+v", res)
	}
}

// ── GetAssistantHistory ────────────────────────────────────────────

func TestGetAssistantHistory_ReturnsNewestFirst(t *testing.T) {
	assistants := newFakeAssistantRepo()
	svc, _ := newAssistantTestService(&fakeTasksClient{}, assistants, &sequencedLLM{})

	conv, err := assistants.GetOrCreateConversation(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}
	for _, text := range []string{"первое", "второе", "третье"} {
		if _, err := assistants.AppendMessage(context.Background(), conv.ID, domain.AssistantRoleUser, text, nil); err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
	}

	history, err := svc.GetAssistantHistory(context.Background(), 10, 1, 2, nil)
	if err != nil {
		t.Fatalf("GetAssistantHistory: %v", err)
	}
	if len(history) != 2 || history[0].Text != "третье" || history[1].Text != "второе" {
		t.Fatalf("ожидался порядок новые→старые с лимитом 2: %+v", history)
	}
}

// ── SendAssistantFeedback ──────────────────────────────────────────

// seedAssistantAnswer — диалог (userID, companyID) с одним ответом ассистента;
// возвращает id сообщения.
func seedAssistantAnswer(t *testing.T, assistants *fakeAssistantRepo, userID, companyID int64) int64 {
	t.Helper()
	conv, err := assistants.GetOrCreateConversation(context.Background(), userID, companyID)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}
	m, err := assistants.AppendMessage(context.Background(), conv.ID, domain.AssistantRoleAssistant, "Ответ", nil)
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	return m.ID
}

// TestSendAssistantFeedback_UpsertReplacesVote — повторный голос заменяет
// прежний (идемпотентный upsert), 👍 сбрасывает причину; голос виден в
// History как MyFeedback.
func TestSendAssistantFeedback_UpsertReplacesVote(t *testing.T) {
	assistants := newFakeAssistantRepo()
	svc, _ := newAssistantTestService(&fakeTasksClient{}, assistants, &sequencedLLM{})
	msgID := seedAssistantAnswer(t, assistants, 10, 1)
	ctx := context.Background()

	reason := "inaccurate"
	if err := svc.SendAssistantFeedback(ctx, 10, 1, msgID, "down", &reason); err != nil {
		t.Fatalf("down: %v", err)
	}
	fb := assistants.feedback[msgID]
	if fb.verdict != "down" || fb.reason == nil || *fb.reason != "inaccurate" {
		t.Fatalf("после 👎 ожидался down/inaccurate: %+v", fb)
	}

	staleReason := "irrelevant"
	if err := svc.SendAssistantFeedback(ctx, 10, 1, msgID, "up", &staleReason); err != nil {
		t.Fatalf("up: %v", err)
	}
	fb = assistants.feedback[msgID]
	if fb.verdict != "up" || fb.reason != nil {
		t.Fatalf("повторный 👍 должен заменить голос и сбросить причину: %+v", fb)
	}

	history, err := svc.GetAssistantHistory(ctx, 10, 1, 10, nil)
	if err != nil {
		t.Fatalf("GetAssistantHistory: %v", err)
	}
	if len(history) != 1 || history[0].MyFeedback == nil || *history[0].MyFeedback != "up" {
		t.Fatalf("history должен нести my_feedback=up: %+v", history)
	}
}

// TestSendAssistantFeedback_ForeignMessageRejected — сообщение из диалога
// другого пользователя (или другой компании) — NOT_FOUND, голос не пишется.
func TestSendAssistantFeedback_ForeignMessageRejected(t *testing.T) {
	assistants := newFakeAssistantRepo()
	svc, _ := newAssistantTestService(&fakeTasksClient{}, assistants, &sequencedLLM{})
	foreignMsgID := seedAssistantAnswer(t, assistants, 20, 1) // диалог пользователя 20

	err := svc.SendAssistantFeedback(context.Background(), 10, 1, foreignMsgID, "up", nil)
	wantDomainError(t, err, "NOT_FOUND", 404)
	if _, voted := assistants.feedback[foreignMsgID]; voted {
		t.Fatalf("чужой голос не должен сохраняться")
	}

	// Та же пара user+message, но другая активная компания — тоже мимо.
	err = svc.SendAssistantFeedback(context.Background(), 20, 2, foreignMsgID, "up", nil)
	wantDomainError(t, err, "NOT_FOUND", 404)
}

// TestSendAssistantFeedback_UserMessageRejected — голосовать можно только за
// ответы ассистента, не за собственные реплики.
func TestSendAssistantFeedback_UserMessageRejected(t *testing.T) {
	assistants := newFakeAssistantRepo()
	svc, _ := newAssistantTestService(&fakeTasksClient{}, assistants, &sequencedLLM{})
	conv, _ := assistants.GetOrCreateConversation(context.Background(), 10, 1)
	m, _ := assistants.AppendMessage(context.Background(), conv.ID, domain.AssistantRoleUser, "Вопрос", nil)

	err := svc.SendAssistantFeedback(context.Background(), 10, 1, m.ID, "up", nil)
	wantDomainError(t, err, "NOT_FOUND", 404)
}

func TestSendAssistantFeedback_Validation(t *testing.T) {
	assistants := newFakeAssistantRepo()
	svc, _ := newAssistantTestService(&fakeTasksClient{}, assistants, &sequencedLLM{})
	msgID := seedAssistantAnswer(t, assistants, 10, 1)
	ctx := context.Background()

	wantDomainError(t, svc.SendAssistantFeedback(ctx, 10, 1, msgID, "meh", nil), "VALIDATION", 400)
	badReason := "boring"
	wantDomainError(t, svc.SendAssistantFeedback(ctx, 10, 1, msgID, "down", &badReason), "VALIDATION", 400)
	wantDomainError(t, svc.SendAssistantFeedback(ctx, 10, 1, 0, "up", nil), "VALIDATION", 400)
}

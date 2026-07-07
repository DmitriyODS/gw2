// Package service — ИИ-ассистент (Сущность 3 плана переосмысления «Мой
// Groove»): деловой корпоративный ассистент по статистике/задачам, БЕЗ
// «сладкой» тональности бывшего Грувика и без привязки к питомцу.
//
// Перенесено из бывшего groove/internal/{clients/ai.go,service/tools.go,
// service/ai.go} (tools-цикл + диспетчер + системный промпт), с одним
// архитектурным отличием: инструменты статистики больше не читают таблицы
// tasksvc напрямую — только через честный gRPC-клиент (s.tasks).
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

const (
	// assistantHistoryLimit — сколько последних сообщений диалога берём как
	// контекст для tools-цикла (аналог petChatHistoryLimit=12 у Грувика, с
	// запасом — ассистент обсуждает более развёрнутые деловые темы).
	assistantHistoryLimit  = 20
	assistantMaxIterations = 4
	assistantMaxTokens     = 500
	// assistantTemperature — ниже, чем у Грувика (0.9): деловой ассистент
	// должен быть предсказуем, а не «с характером».
	assistantTemperature = 0.3
	assistantTimeout     = 30 * time.Second
)

// assistantToolSchemasJSON — OpenAI-совместимые tool-definition'ы: 5
// инструментов статистики (перенесены из groove-tools.go как есть по
// смыслу) + новый find_task (семантический поиск задачи по описанию).
const assistantToolSchemasJSON = `[
  {"type": "function", "function": {
    "name": "get_stats_summary",
    "description": "Общие метрики компании за период: поступило задач, закрыто, ещё в работе (на конец периода), долг до периода, суммарные часы команды. Использовать для вопросов «сколько задач поступило/закрыто», «сколько часов отработали», «как дела за неделю».",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"], "description": "Период статистики. По умолчанию this_week."}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "list_departments",
    "description": "Список отделов компании с количеством поступивших задач за период.",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"]}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "get_top_employees",
    "description": "Топ сотрудников по отработанному времени за период: ФИО, кол-во задач, часы. Для вопросов «кто больше всего работал», «лидеры недели».",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"]},
      "limit": {"type": "integer", "minimum": 1, "maximum": 30, "description": "Сколько строк вернуть. По умолчанию 10."}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "get_stats_by_unit_types",
    "description": "Распределение работы по типам юнитов (звонок, разработка, встреча и т. п.) за период: часы и количество задач.",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"]}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "get_stats_calendar",
    "description": "Динамика по дням за период: на каждый день — поступило, закрыто, часы.",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"]}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "find_task",
    "description": "Найти конкретную задачу компании по описанию/названию (семантический поиск) и получить прямую ссылку на неё. Используй, когда пользователь спрашивает про конкретную задачу или просит найти задачу по теме.",
    "parameters": {"type": "object", "properties": {
      "query": {"type": "string", "description": "Название или описание задачи, которую ищет пользователь."}
    }, "required": ["query"], "additionalProperties": false}}}
]`

// assistantSystemPrompt — деловой корпоративный тон: без экспрессии, юмора
// и эмодзи, только факты и цифры из инструментов; явно указывать
// номер/название задачи и присылать прямую ссылку, когда задача найдена;
// не выдумывать данные вне инструментов.
func assistantSystemPrompt() string {
	now := time.Now().In(time.FixedZone("MSK", 3*60*60))
	return "Ты — корпоративный ИИ-ассистент платформы Groove Work. Отвечаешь на вопросы " +
		"сотрудников о задачах, статистике и загрузке команды. Тон — деловой и нейтральный: " +
		"без юмора, восклицаний, эмодзи и уменьшительных суффиксов. Опирайся ТОЛЬКО на " +
		"факты и цифры, полученные через инструменты (get_stats_summary, list_departments, " +
		"get_top_employees, get_stats_by_unit_types, get_stats_calendar, find_task) — никогда " +
		"не выдумывай цифры, названия задач или сотрудников. Если данных недостаточно — " +
		"так и скажи, не домысливай. Когда находишь задачу через find_task — обязательно " +
		"назови её точное название и приведи прямую ссылку из результата инструмента. " +
		"Отвечай кратко и по делу, на русском языке. Сегодня — " + now.Format("02.01.2006") + "."
}

// AssistantReply — ответ SendAssistantMessage: сохранённое сообщение
// ассистента (ID из БД нужен фронту для обратной связи 👍/👎).
type AssistantReply struct {
	ID        int64
	Text      string
	Sources   *string
	CreatedAt time.Time
}

// SendAssistantMessage — реплика пользователя → ответ ассистента. Диалог —
// один на пару (userID, companyID); история — контекст tools-цикла.
// Компания без включённого AI → AI_DISABLED (409-эквивалент из errAiDisabled).
func (s *Service) SendAssistantMessage(ctx context.Context, userID, companyID int64, text string) (*AssistantReply, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.NewError("VALIDATION", "Текст сообщения не может быть пустым", 400)
	}
	client, err := s.clientFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(409)
	}

	conv, err := s.assistants.GetOrCreateConversation(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	history, err := s.assistants.RecentMessages(ctx, conv.ID, assistantHistoryLimit)
	if err != nil {
		return nil, err
	}

	messages := make([]map[string]any, 0, len(history)+2)
	messages = append(messages, map[string]any{"role": "system", "content": assistantSystemPrompt()})
	for _, m := range history {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Text})
	}
	messages = append(messages, map[string]any{"role": "user", "content": text})

	reply, sources, err := s.chatWithTools(ctx, companyID, messages)
	if err != nil {
		return nil, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		reply = "Не удалось сформировать ответ — попробуйте переформулировать вопрос."
	}
	var sourcesStr *string
	if len(sources) > 0 {
		joined := "Данные: " + strings.Join(sources, ", ")
		sourcesStr = &joined
	}

	if _, err := s.assistants.AppendMessage(ctx, conv.ID, domain.AssistantRoleUser, text, nil); err != nil {
		return nil, err
	}
	saved, err := s.assistants.AppendMessage(ctx, conv.ID, domain.AssistantRoleAssistant, reply, sourcesStr)
	if err != nil {
		return nil, err
	}
	return &AssistantReply{ID: saved.ID, Text: reply, Sources: saved.Sources, CreatedAt: saved.CreatedAt}, nil
}

// assistantFeedbackReasons — допустимые причины 👎 (чипы на фронте).
var assistantFeedbackReasons = map[string]struct{}{
	"inaccurate": {}, "irrelevant": {}, "incomplete": {},
}

// SendAssistantFeedback — голос 👍/👎 по ответу ассистента. Идемпотентный
// upsert: повторный голос заменяет прежний. Чужой message_id (не из диалога
// этой пары userID+companyID) — NOT_FOUND, id не раскрываем деталями.
func (s *Service) SendAssistantFeedback(ctx context.Context, userID, companyID, messageID int64, verdict string, reason *string) error {
	if messageID <= 0 {
		return domain.NewError("VALIDATION", "Некорректный message_id", 400)
	}
	if verdict != domain.AssistantFeedbackUp && verdict != domain.AssistantFeedbackDown {
		return domain.NewError("VALIDATION", "verdict должен быть up или down", 400)
	}
	if verdict == domain.AssistantFeedbackUp {
		reason = nil // причина имеет смысл только для 👎
	} else if reason != nil {
		if _, ok := assistantFeedbackReasons[*reason]; !ok {
			return domain.NewError("VALIDATION", "Недопустимая причина отзыва", 400)
		}
	}
	ok, err := s.assistants.UpsertFeedback(ctx, messageID, userID, companyID, verdict, reason)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NewError("NOT_FOUND", "Сообщение не найдено", 404)
	}
	return nil
}

// GetAssistantHistory — постраничная лента (новые → старые), для REST.
func (s *Service) GetAssistantHistory(ctx context.Context, userID, companyID int64, limit int, before *time.Time) ([]domain.AssistantMessage, error) {
	if limit <= 0 {
		limit = assistantHistoryLimit
	}
	conv, err := s.assistants.GetOrCreateConversation(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	return s.assistants.History(ctx, conv.ID, limit, before)
}

// ── Tools-цикл (перенесено из groove/internal/clients/ai.go:ChatWithTools) ─

// chatWithTools — цикл function-calling поверх s.Chat (один ход за раз).
// В отличие от старой версии (groovesvc звал aisvc по gRPC), это теперь
// прямой вызов метода того же сервиса — цикл живёт внутри aisvc.
// Вторым результатом — провенанс: человекочитаемые подписи инструментов,
// РЕАЛЬНО давших данные (упавшие/невалидные вызовы не считаются), без
// дублей, в порядке вызовов.
func (s *Service) chatWithTools(ctx context.Context, companyID int64, messages []map[string]any) (string, []string, error) {
	var sources []string
	convo := append([]map[string]any{}, messages...)
	for i := 0; i < assistantMaxIterations; i++ {
		res, err := s.assistantChatOnce(ctx, companyID, convo, assistantToolSchemasJSON)
		if err != nil {
			return "", nil, err
		}
		toolCalls := parseAssistantToolCalls(res.ToolCallsJSON, s.log)
		if len(toolCalls) == 0 {
			return res.Content, sources, nil
		}

		convo = append(convo, map[string]any{
			"role": "assistant", "content": res.Content, "tool_calls": toolCalls,
		})
		for _, tc := range toolCalls {
			fn, _ := tc["function"].(map[string]any)
			name, _ := fn["name"].(string)
			args := map[string]any{}
			if rawArgs, ok := fn["arguments"].(string); ok && rawArgs != "" {
				_ = json.Unmarshal([]byte(rawArgs), &args)
			}
			result := s.dispatchAssistantTool(ctx, name, args, companyID)
			if label := assistantSourceLabel(name, args); label != "" && !isToolError(result) {
				sources = appendUnique(sources, label)
			}
			rawResult, err := json.Marshal(result)
			if err != nil {
				rawResult = []byte(`{"error":"tool_result_marshal_failed"}`)
			}
			id, _ := tc["id"].(string)
			convo = append(convo, map[string]any{
				"role": "tool", "tool_call_id": id, "content": string(rawResult),
			})
		}
	}

	// Лимит итераций исчерпан — финальный заход без tools, чтобы модель
	// точно ответила текстом, а не очередным tool_call.
	res, err := s.assistantChatOnce(ctx, companyID, convo, "")
	if err != nil {
		return "", nil, err
	}
	return res.Content, sources, nil
}

// ── Провенанс («Данные: …») ───────────────────────────────────────

// assistantPeriodAccusative — русские подписи периодов в винительном падеже
// («за …»); коды и алиасы — те же, что понимает ResolveAssistantPeriod в
// tasksvc (незнакомый/пустой код там падает в this_week — здесь так же).
var assistantPeriodAccusative = map[string]string{
	"today":     "сегодня",
	"yesterday": "вчера",
	"this_week": "эту неделю", "week": "эту неделю",
	"last_week":  "прошлую неделю",
	"this_month": "этот месяц", "month": "этот месяц",
	"last_month": "прошлый месяц",
	"7d":         "последние 7 дней", "7days": "последние 7 дней", "last_7_days": "последние 7 дней",
	"30d": "последние 30 дней", "30days": "последние 30 дней", "last_30_days": "последние 30 дней",
}

func assistantPeriodPhrase(args map[string]any) string {
	code := strings.ToLower(strings.TrimSpace(argPeriod(args)))
	if label, ok := assistantPeriodAccusative[code]; ok {
		return label
	}
	return assistantPeriodAccusative["this_week"]
}

// assistantSourceLabel — человекочитаемая подпись вызова инструмента для
// строки провенанса; "" — инструмент неизвестен (в строку не попадает).
func assistantSourceLabel(name string, args map[string]any) string {
	switch name {
	case "get_stats_summary":
		return "статистика за " + assistantPeriodPhrase(args)
	case "list_departments":
		return "отделы"
	case "get_top_employees":
		return "лидеры за " + assistantPeriodPhrase(args)
	case "get_stats_by_unit_types":
		return "работа по типам юнитов за " + assistantPeriodPhrase(args)
	case "get_stats_calendar":
		return "динамика по дням за " + assistantPeriodPhrase(args)
	case "find_task":
		query, _ := args["query"].(string)
		query = strings.TrimSpace(query)
		if query == "" {
			return "поиск задачи"
		}
		return "поиск задачи „" + truncateRunes(query, 60) + "“"
	}
	return ""
}

// isToolError — результат диспетчера вида {"error": …} (инструмент упал или
// вызван невалидно) — данных не дал, в провенанс не идёт.
func isToolError(result any) bool {
	m, ok := result.(map[string]any)
	if !ok {
		return false
	}
	_, hasErr := m["error"]
	return hasErr
}

func appendUnique(list []string, v string) []string {
	for _, item := range list {
		if item == v {
			return list
		}
	}
	return append(list, v)
}

func (s *Service) assistantChatOnce(ctx context.Context, companyID int64, messages []map[string]any, toolsJSON string) (*domain.ChatResult, error) {
	raw, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	return s.Chat(ctx, ChatArgs{
		CompanyID:    companyID,
		MessagesJSON: string(raw),
		ToolsJSON:    toolsJSON,
		MaxTokens:    assistantMaxTokens,
		Temperature:  assistantTemperature,
		TimeoutSec:   assistantTimeout.Seconds(),
	})
}

func parseAssistantToolCalls(raw string, log *slog.Logger) []map[string]any {
	if raw == "" {
		return nil
	}
	var parsed []map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		log.Warn("ai.assistant.tool_calls_bad_json", "raw", truncateRunes(raw, 200))
		return nil
	}
	return parsed
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// ── Диспетчер инструментов ────────────────────────────────────────

// dispatchAssistantTool — запустить инструмент по имени; сбой → {"error":
// "..."}, НИКОГДА не роняет tools-цикл (как dispatchTool у Грувика).
func (s *Service) dispatchAssistantTool(ctx context.Context, name string, args map[string]any, companyID int64) any {
	if s.tasks == nil {
		return map[string]any{"error": "tasks_unavailable"}
	}
	if args == nil {
		args = map[string]any{}
	}
	var result any
	var err error
	switch name {
	case "get_stats_summary":
		result, err = s.toolStatsSummary(ctx, args, companyID)
	case "list_departments":
		result, err = s.toolDepartments(ctx, args, companyID)
	case "get_top_employees":
		result, err = s.toolTopEmployees(ctx, args, companyID)
	case "get_stats_by_unit_types":
		result, err = s.toolByUnitTypes(ctx, args, companyID)
	case "get_stats_calendar":
		result, err = s.toolCalendar(ctx, args, companyID)
	case "find_task":
		result, err = s.toolFindTask(ctx, args, companyID)
	default:
		return map[string]any{"error": "unknown_tool:" + name}
	}
	if err != nil {
		s.log.Warn("ai.assistant.tool_failed", "tool", name, "company_id", companyID, "error", err)
		return map[string]any{"error": "tool_execution_failed"}
	}
	return result
}

func argPeriod(args map[string]any) string {
	code, _ := args["period"].(string)
	return code
}

func (s *Service) toolStatsSummary(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	period := argPeriod(args)
	r, err := s.tasks.GetStatsSummary(ctx, companyID, period)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"period":             r.PeriodLabel,
		"received":           r.NewCount,
		"closed":             r.ClosedCount,
		"in_progress_now":    r.InProgressCount,
		"debt_before_period": r.DebtCount,
		"team_hours":         r.TotalHours,
	}, nil
}

func (s *Service) toolDepartments(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	period := argPeriod(args)
	rows, err := s.tasks.ListDepartments(ctx, companyID, period)
	if err != nil {
		return nil, err
	}
	departments := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		departments = append(departments, map[string]any{
			"id": r.ID, "name": r.Name, "received_count": r.NewCount,
		})
	}
	return map[string]any{"departments": departments}, nil
}

func (s *Service) toolTopEmployees(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	period := argPeriod(args)
	limit := 0
	if raw, ok := args["limit"].(float64); ok && raw > 0 {
		limit = int(raw)
	}
	rows, err := s.tasks.GetTopEmployees(ctx, companyID, period, limit)
	if err != nil {
		return nil, err
	}
	employees := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		employees = append(employees, map[string]any{
			"fio": r.FIO, "tasks_count": r.TaskCount, "hours": r.Hours,
		})
	}
	return map[string]any{"employees": employees}, nil
}

func (s *Service) toolByUnitTypes(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	period := argPeriod(args)
	rows, err := s.tasks.GetStatsByUnitTypes(ctx, companyID, period)
	if err != nil {
		return nil, err
	}
	unitTypes := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		unitTypes = append(unitTypes, map[string]any{
			"name": r.Name, "hours": r.Hours, "tasks_count": r.TaskCount,
		})
	}
	return map[string]any{"unit_types": unitTypes}, nil
}

func (s *Service) toolCalendar(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	period := argPeriod(args)
	rows, err := s.tasks.GetStatsCalendar(ctx, companyID, period)
	if err != nil {
		return nil, err
	}
	days := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		days = append(days, map[string]any{
			"date": r.Date, "received": r.NewCount, "closed": r.ClosedCount, "hours": r.Hours,
		})
	}
	return map[string]any{"days": days}, nil
}

// toolFindTask — семантический поиск задачи (s.SemanticSearch, тот же путь,
// что и REST-поиск задач) → лучшее совпадение → ссылка через tasksvc
// (GetTaskLink). Намеренно НЕ отдаём LLM сырые id без проверки — так модель
// не может «изобрести» ссылку на несуществующую задачу: URL строится только
// из данных, реально вернувшихся от tasksvc.
func (s *Service) toolFindTask(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	query, _ := args["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return map[string]any{"error": "query_required"}, nil
	}
	hits, err := s.SemanticSearch(ctx, companyID, query)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return map[string]any{"found": false}, nil
	}
	link, err := s.tasks.GetTaskLink(ctx, companyID, hits[0].TaskID)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return map[string]any{"found": false}, nil
	}
	return map[string]any{
		"found":           true,
		"id":              link.ID,
		"name":            link.Name,
		"responsible_fio": link.ResponsibleFIO,
		"url":             s.taskURL(link.ID),
	}, nil
}

// taskURL — прямая ссылка на карточку задачи (APP_PUBLIC_BASE_URL/tasks/id),
// тот же паттерн, что уже используется authsvc для ссылок в письмах.
func (s *Service) taskURL(taskID int64) string {
	base := strings.TrimRight(s.appBaseURL, "/")
	return fmt.Sprintf("%s/tasks/%s", base, strconv.FormatInt(taskID, 10))
}

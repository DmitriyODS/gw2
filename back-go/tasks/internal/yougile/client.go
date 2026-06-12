// Package yougile — инфраструктура интеграции с YouGile: тонкий HTTP-клиент
// REST API v2 (`https://ru.yougile.com/api-v2`), парсер ссылок на карточки и
// Fernet-шифрование личных API-ключей. Портировано из
// back/app/integrations/yougile/{client,parser,crypto}.py без изменения
// поведения; бизнес-логика — в internal/service/yougile_*.go.
//
// Аутентификация. На большинство методов идёт `Authorization: Bearer <key>`,
// кроме трёх auth-эндпоинтов (`/auth/companies`, `/auth/keys`,
// `/auth/keys/get`) и `DELETE /auth/keys/{key}` — там Bearer не нужен.
//
// Rate-limit. Сервер отдаёт 429 без `Retry-After`. Делаем до трёх попыток с
// экспоненциальным backoff'ом (1с/2с/4с). На 4xx (кроме 429) сразу ошибка,
// ретрай ничего не даст. 5xx — ретраим столько же.
package yougile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	BaseURL        = "https://ru.yougile.com/api-v2"
	defaultTimeout = 15 * time.Second // YG обычно отвечает <1с, но webhook-эндпоинт долгий.
	maxRetries     = 3
)

var retryStatuses = map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}

// Error — ошибка YouGile API. Текст совпадает со str(YougileError) во
// Flask — он попадает в сообщения пользовательских ошибок
// («YouGile: YouGile 400»). Auth=true — 401/403 (ключ невалиден или прав
// не хватает), RateLimited=true — 429 после всех ретраев.
type Error struct {
	Message     string
	Status      int
	Auth        bool
	RateLimited bool
}

func (e *Error) Error() string { return e.Message }

func IsAuth(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Auth
}

// Client — тонкий клиент. Создаётся либо с key (Bearer), либо без — для
// auth-флоу. httpClient и sleep инжектируются в тестах.
type Client struct {
	key        string
	base       string
	httpClient *http.Client
	sleep      func(time.Duration)
}

// Option — настройка клиента (тестовые транспорт/база/sleep).
type Option func(*Client)

func WithHTTPClient(c *http.Client) Option { return func(cl *Client) { cl.httpClient = c } }
func WithBaseURL(base string) Option       { return func(cl *Client) { cl.base = base } }
func WithSleep(fn func(time.Duration)) Option {
	return func(cl *Client) { cl.sleep = fn }
}

func NewClient(key string, opts ...Option) *Client {
	c := &Client{
		key:        key,
		base:       BaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		sleep:      time.Sleep,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ── низкоуровневое ───────────────────────────────────────────────────────

func (c *Client) request(method, path string, body any, params url.Values, anonymous bool) (any, error) {
	full := c.base + path
	if len(params) > 0 {
		full += "?" + params.Encode()
	}

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, &Error{Message: fmt.Sprintf("YouGile недоступен: %v", err)}
		}
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest(method, full, bytes.NewReader(payload))
		if err != nil {
			return nil, &Error{Message: fmt.Sprintf("YouGile недоступен: %v", err)}
		}
		req.Header.Set("User-Agent", "GrooveWork-Yougile/1.0")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		if !anonymous && c.key != "" {
			req.Header.Set("Authorization", "Bearer "+c.key)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Сетевые/таймауты — ретраим как 5xx.
			if attempt == maxRetries-1 {
				return nil, &Error{Message: fmt.Sprintf("YouGile недоступен: %v", err)}
			}
			c.sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			if attempt == maxRetries-1 {
				return nil, &Error{Message: fmt.Sprintf("YouGile недоступен: %v", readErr)}
			}
			c.sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}

		if retryStatuses[resp.StatusCode] && attempt < maxRetries-1 {
			c.sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}

		return parseResponse(resp.StatusCode, data)
	}
	return nil, &Error{Message: "retry-loop exhausted"}
}

func parseResponse(status int, body []byte) (any, error) {
	switch {
	case status == 401:
		return nil, &Error{Message: "Неверный логин/пароль или ключ", Status: 401, Auth: true}
	case status == 403:
		return nil, &Error{Message: "Нет доступа", Status: 403, Auth: true}
	case status == 429:
		return nil, &Error{Message: "Превышен лимит запросов YouGile", Status: 429, RateLimited: true}
	case status >= 400:
		return nil, &Error{Message: "YouGile " + strconv.Itoa(status), Status: status}
	}
	if len(body) == 0 {
		return nil, nil
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return string(body), nil
	}
	return out, nil
}

// asObject / asContentList — приведение any-ответов.
func asObject(data any) map[string]any {
	if m, ok := data.(map[string]any); ok {
		return m
	}
	return nil
}

func contentList(data any) []map[string]any {
	var raw []any
	if m := asObject(data); m != nil {
		if c, ok := m["content"].([]any); ok {
			raw = c
		}
	} else if l, ok := data.([]any); ok {
		raw = l
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func str(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprint(v)
	}
	return ""
}

// ── auth ─────────────────────────────────────────────────────────────────

// ListCompanies — `POST /auth/companies`. Без Bearer'а.
func (c *Client) ListCompanies(login, password string) ([]map[string]any, error) {
	body := map[string]any{"login": login, "password": password}
	data, err := c.request("POST", "/auth/companies", body, nil, true)
	if err != nil {
		return nil, err
	}
	return contentList(data), nil
}

// CreateKey — `POST /auth/keys`. Возвращает строку-ключ.
func (c *Client) CreateKey(login, password, companyID string) (string, error) {
	body := map[string]any{"login": login, "password": password, "companyId": companyID}
	data, err := c.request("POST", "/auth/keys", body, nil, true)
	if err != nil {
		return "", err
	}
	m := asObject(data)
	if m == nil || m["key"] == nil {
		return "", &Error{Message: "Неожиданный ответ /auth/keys"}
	}
	return str(m, "key"), nil
}

// DeleteKey — `DELETE /auth/keys/{key}`. Анонимно — кто знает ключ, тот его
// и удалит.
func (c *Client) DeleteKey(key string) error {
	_, err := c.request("DELETE", "/auth/keys/"+key, nil, nil, true)
	return err
}

// ── профиль / структура ──────────────────────────────────────────────────

func (c *Client) Me() (map[string]any, error) {
	data, err := c.request("GET", "/users/me", nil, nil, false)
	if err != nil {
		return nil, err
	}
	return asObject(data), nil
}

func (c *Client) ListProjects(limit int) ([]map[string]any, error) {
	return c.page("/projects", nil, limit)
}

func (c *Client) ListBoards(projectID string, limit int) ([]map[string]any, error) {
	params := url.Values{}
	if projectID != "" {
		params.Set("projectId", projectID)
	}
	return c.page("/boards", params, limit)
}

func (c *Client) ListColumns(boardID string, limit int) ([]map[string]any, error) {
	params := url.Values{}
	params.Set("boardId", boardID)
	return c.page("/columns", params, limit)
}

// ── задачи ───────────────────────────────────────────────────────────────

func (c *Client) GetTask(taskID string) (map[string]any, error) {
	data, err := c.request("GET", "/tasks/"+taskID, nil, nil, false)
	if err != nil {
		return nil, err
	}
	return asObject(data), nil
}

func (c *Client) CreateTask(body map[string]any) (map[string]any, error) {
	data, err := c.request("POST", "/tasks", body, nil, false)
	if err != nil {
		return nil, err
	}
	return asObject(data), nil
}

func (c *Client) UpdateTask(taskID string, body map[string]any) (map[string]any, error) {
	data, err := c.request("PUT", "/tasks/"+taskID, body, nil, false)
	if err != nil {
		return nil, err
	}
	return asObject(data), nil
}

// maxSubtaskLookups — предохранитель BFS по подзадачам: каждая — отдельный
// GET /tasks/{id}.
const maxSubtaskLookups = 500

// FindTaskByShortID — резолв человекочитаемого id карточки (`OIP1-2454`)
// в UUID.
//
// YouGile API v2 не даёт фильтр `/tasks?idTaskProject=...` (400), и сам
// `/tasks?boardId=...` тоже не работает — только `columnId`. Поэтому
// перебираем колонки доски и страницы внутри.
//
// Подзадачи к колонке не привязаны — они живут только в поле `subtasks`
// родителя. Если на верхнем уровне не нашли, обходим собранные subtask-id
// в ширину (GET /tasks/{id} по одной, вложенность любая).
//
// Возвращает task-объект или nil.
func (c *Client) FindTaskByShortID(boardID, shortID string, columnIDs []string) (map[string]any, error) {
	if columnIDs == nil {
		cols, err := c.ListColumns(boardID, 1000)
		if err != nil {
			return nil, err
		}
		for _, col := range cols {
			if id := str(col, "id"); id != "" {
				columnIDs = append(columnIDs, id)
			}
		}
	}
	target := strings.ToUpper(strings.TrimSpace(shortID))
	if target == "" {
		return nil, nil
	}

	matches := func(t map[string]any) bool {
		return strings.ToUpper(str(t, "idTaskProject")) == target ||
			strings.ToUpper(str(t, "idTaskCommon")) == target
	}

	var subtaskIDs []string
	seen := map[string]bool{}
	collectSubtasks := func(t map[string]any) {
		subs, _ := t["subtasks"].([]any)
		for _, s := range subs {
			sid := fmt.Sprint(s)
			if !seen[sid] {
				seen[sid] = true
				subtaskIDs = append(subtaskIDs, sid)
			}
		}
	}

	const pageSize = 1000
	for _, colID := range columnIDs {
		offset := 0
		for {
			params := url.Values{}
			params.Set("columnId", colID)
			params.Set("limit", strconv.Itoa(pageSize))
			params.Set("offset", strconv.Itoa(offset))
			params.Set("includeDeleted", "false")
			data, err := c.request("GET", "/tasks", nil, params, false)
			if err != nil {
				return nil, err
			}
			content := contentList(data)
			for _, t := range content {
				if matches(t) {
					return t, nil
				}
				collectSubtasks(t)
			}
			if len(content) < pageSize {
				break
			}
			offset += pageSize
		}
	}

	for idx := 0; idx < len(subtaskIDs) && idx < maxSubtaskLookups; idx++ {
		t, err := c.GetTask(subtaskIDs[idx])
		if err != nil {
			if IsAuth(err) {
				return nil, err
			}
			continue
		}
		if t == nil {
			continue
		}
		if matches(t) {
			return t, nil
		}
		collectSubtasks(t)
	}
	return nil, nil
}

// ── чат задачи ───────────────────────────────────────────────────────────

func (c *Client) PostChatMessage(chatID string, body map[string]any) error {
	_, err := c.request("POST", "/chats/"+chatID+"/messages", body, nil, false)
	return err
}

// ── webhooks ─────────────────────────────────────────────────────────────

func (c *Client) CreateWebhook(hookURL, event string, filters []map[string]any) (map[string]any, error) {
	if filters == nil {
		filters = []map[string]any{}
	}
	body := map[string]any{"url": hookURL, "event": event, "filters": filters}
	data, err := c.request("POST", "/webhooks", body, nil, false)
	if err != nil {
		return nil, err
	}
	return asObject(data), nil
}

func (c *Client) UpdateWebhook(webhookID string, body map[string]any) error {
	_, err := c.request("PUT", "/webhooks/"+webhookID, body, nil, false)
	return err
}

// ── пагинация (внутреннее) ───────────────────────────────────────────────

// page — собрать все страницы list-эндпоинта. В нашем сценарии элементов
// десятки/сотни, лимит 1000 покрывает почти всё с одного запроса;
// полноценная пагинация на будущее, без unbounded-цикла.
func (c *Client) page(path string, params url.Values, limit int) ([]map[string]any, error) {
	var out []map[string]any
	offset := 0
	pageSize := limit
	if pageSize > 1000 {
		pageSize = 1000
	}
	for {
		p := url.Values{}
		for k, vs := range params {
			for _, v := range vs {
				p.Add(k, v)
			}
		}
		p.Set("limit", strconv.Itoa(pageSize))
		p.Set("offset", strconv.Itoa(offset))
		data, err := c.request("GET", path, nil, p, false)
		if err != nil {
			return nil, err
		}
		content := contentList(data)
		out = append(out, content...)
		if len(content) < pageSize || len(out) >= limit {
			break
		}
		offset += pageSize
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

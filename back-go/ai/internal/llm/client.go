// Package llm — HTTP-клиент OpenAI-совместимого API (ProxyAPI) без SDK:
// POST /chat/completions и POST /embeddings. Замена openai-обёртки из
// back/app/services/ai_client.py.
//
// Таймаут — per-request через context (Timeout в параметрах). Ошибки сети и
// не-2xx ответы upstream'а заворачиваются в domain.Error AI_UPSTREAM
// (502; таймаут — 504) с текстом upstream'а — как Flask пробрасывал текст
// OpenAIError.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// DefaultBaseURL — базовый URL ProxyAPI (PROXYAPI_BASE_URL во Flask);
// переопределяется env AI_API_BASE_URL.
const DefaultBaseURL = "https://api.proxyapi.ru/openai/v1"

// DefaultTimeout — _REQUEST_TIMEOUT из Flask.
const DefaultTimeout = 30 * time.Second

// упрощает чтение тел ошибок: не тащим мегабайты в сообщение.
const maxErrorBody = 2048

type Client struct {
	baseURL string
	http    *http.Client
	log     *slog.Logger

	// quirks — что модель не принимает (ключ — сервер+модель). Узнаётся из
	// ответа 400 и дальше учитывается сразу, без лишнего круга.
	quirks sync.Map
}

// modelQuirks — отличия модели от классического Chat Completions.
// Reasoning-модели OpenAI (o-серия, gpt-5…) не принимают max_tokens (только
// max_completion_tokens) и произвольную temperature. Имя модели ничего не
// гарантирует — у пользователя бывает свой сервер со своими правилами, —
// поэтому поведение выясняется по отказу, а не по списку моделей.
type modelQuirks struct {
	completionTokens bool
	noTemperature    bool
}

var _ domain.LLMClient = (*Client)(nil)

func New(baseURL string, log *slog.Logger) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		// без Timeout: per-request дедлайны задаёт context вызова.
		http: &http.Client{},
		log:  log,
	}
}

type chatRequest struct {
	Model               string          `json:"model"`
	Messages            json.RawMessage `json:"messages"`
	Tools               json.RawMessage `json:"tools,omitempty"`
	MaxTokens           int             `json:"max_tokens,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
}

// reasoningBudget — во что превращается лимит ответа у модели с
// max_completion_tokens: туда входят и скрытые рассуждения, и с прежним
// бюджетом ответ обрезался до пустоты. Списывается всё равно фактический расход.
const reasoningBudget = 4

func buildChatRequest(p domain.ChatParams, q modelQuirks) chatRequest {
	req := chatRequest{Model: p.Model, Messages: json.RawMessage(p.MessagesJSON)}
	if q.completionTokens {
		req.MaxCompletionTokens = p.MaxTokens * reasoningBudget
	} else {
		req.MaxTokens = p.MaxTokens
	}
	if !q.noTemperature {
		t := p.Temperature
		req.Temperature = &t
	}
	if p.ToolsJSON != "" {
		req.Tools = json.RawMessage(p.ToolsJSON)
	}
	return req
}

// adapt — отказ из-за параметра, который модель не принимает: запомнить и
// сказать, есть ли смысл повторить.
func (c *Client) adapt(key string, q *modelQuirks, err error) bool {
	var he *httpError
	if !errors.As(err, &he) || he.status != http.StatusBadRequest {
		return false
	}
	var body struct {
		Error struct {
			Param string `json:"param"`
			Code  string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(he.body, &body) != nil {
		return false
	}
	switch {
	case body.Error.Param == "max_tokens" && !q.completionTokens:
		q.completionTokens = true
	case body.Error.Param == "temperature" && !q.noTemperature:
		q.noTemperature = true
	default:
		return false
	}
	c.quirks.Store(key, *q)
	c.log.Info("llm.model_quirk", "model", key, "param", body.Error.Param, "code", body.Error.Code)
	return true
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   *string         `json:"content"`
			ToolCalls json.RawMessage `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	// usage — расход токенов; по нему считается списание с баланса тарифа.
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) ChatOnce(ctx context.Context, p domain.ChatParams) (*domain.ChatResult, error) {
	key := p.BaseURL + "|" + p.Model
	var q modelQuirks
	if v, ok := c.quirks.Load(key); ok {
		q = v.(modelQuirks)
	}
	var resp chatResponse
	// Отказов по параметрам бывает не больше двух (лимит и температура).
	for attempt := 0; ; attempt++ {
		err := c.post(ctx, p.BaseURL, "/chat/completions", p.APIKey, buildChatRequest(p, q), &resp, p.Timeout)
		if err == nil {
			break
		}
		if attempt < 2 && c.adapt(key, &q, err) {
			continue
		}
		return nil, asDomain(err)
	}
	if len(resp.Choices) == 0 {
		return nil, upstreamError("пустой ответ модели: нет choices")
	}
	msg := resp.Choices[0].Message
	out := &domain.ChatResult{Usage: domain.TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
	}}
	if msg.Content != nil {
		out.Content = *msg.Content
	}
	// tool_calls: null/отсутствие — обычный текстовый ответ.
	if tc := strings.TrimSpace(string(msg.ToolCalls)); tc != "" && tc != "null" {
		out.ToolCallsJSON = tc
	}
	return out, nil
}

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) Embed(ctx context.Context, p domain.EmbedParams) ([][]float32, int, error) {
	if len(p.Texts) == 0 {
		return nil, 0, nil
	}
	var resp embeddingsResponse
	err := c.post(ctx, p.BaseURL, "/embeddings", p.APIKey,
		embeddingsRequest{Model: p.Model, Input: p.Texts}, &resp, p.Timeout)
	if err != nil {
		return nil, 0, asDomain(err)
	}
	// API возвращает items с полем index — на всякий случай сортируем.
	sort.Slice(resp.Data, func(i, j int) bool { return resp.Data[i].Index < resp.Data[j].Index })
	out := make([][]float32, 0, len(resp.Data))
	for _, d := range resp.Data {
		out = append(out, d.Embedding)
	}
	if len(out) != len(p.Texts) {
		return nil, 0, upstreamError(fmt.Sprintf("эмбеддингов %d вместо %d", len(out), len(p.Texts)))
	}
	used := resp.Usage.TotalTokens
	if used == 0 {
		used = resp.Usage.PromptTokens
	}
	return out, used, nil
}

// post — запрос к upstream. baseURL пуст — общий адрес клиента; непустой
// используется, когда пользователь подключил СВОЙ сервер модели.
func (c *Client) post(ctx context.Context, baseURL, path, apiKey string, body, out any, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	raw, err := json.Marshal(body)
	if err != nil {
		return upstreamError("кодирование запроса: " + err.Error())
	}
	base := c.baseURL
	if baseURL != "" {
		base = strings.TrimRight(baseURL, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(raw))
	if err != nil {
		return upstreamError(err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.NewError("AI_UPSTREAM", "таймаут запроса к AI-провайдеру", 504)
		}
		return upstreamError(err.Error())
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		c.log.Warn("llm.upstream_error", "path", path, "status", resp.StatusCode)
		return &httpError{status: resp.StatusCode, body: snippet}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return upstreamError("декодирование ответа: " + err.Error())
	}
	return nil
}

// httpError — не-2xx ответ upstream'а; тело нужно, чтобы понять причину отказа.
type httpError struct {
	status int
	body   []byte
}

func (e *httpError) Error() string {
	return fmt.Sprintf("status %d: %s", e.status, strings.TrimSpace(string(e.body)))
}

// asDomain — наружу ошибка уходит доменной (AI_UPSTREAM с текстом upstream'а).
func asDomain(err error) error {
	var he *httpError
	if errors.As(err, &he) {
		return upstreamError(he.Error())
	}
	return err
}

func upstreamError(msg string) *domain.Error {
	return domain.NewError("AI_UPSTREAM", msg, 502)
}

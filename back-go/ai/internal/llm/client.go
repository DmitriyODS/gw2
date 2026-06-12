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
	Model       string          `json:"model"`
	Messages    json.RawMessage `json:"messages"`
	Tools       json.RawMessage `json:"tools,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   *string         `json:"content"`
			ToolCalls json.RawMessage `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) ChatOnce(ctx context.Context, p domain.ChatParams) (*domain.ChatResult, error) {
	req := chatRequest{
		Model:       p.Model,
		Messages:    json.RawMessage(p.MessagesJSON),
		MaxTokens:   p.MaxTokens,
		Temperature: p.Temperature,
	}
	if p.ToolsJSON != "" {
		req.Tools = json.RawMessage(p.ToolsJSON)
	}
	var resp chatResponse
	if err := c.post(ctx, "/chat/completions", p.APIKey, req, &resp, p.Timeout); err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, upstreamError("пустой ответ модели: нет choices")
	}
	msg := resp.Choices[0].Message
	out := &domain.ChatResult{}
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
}

func (c *Client) Embed(ctx context.Context, apiKey, model string, texts []string, timeout time.Duration) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	var resp embeddingsResponse
	err := c.post(ctx, "/embeddings", apiKey, embeddingsRequest{Model: model, Input: texts}, &resp, timeout)
	if err != nil {
		return nil, err
	}
	// API возвращает items с полем index — на всякий случай сортируем.
	sort.Slice(resp.Data, func(i, j int) bool { return resp.Data[i].Index < resp.Data[j].Index })
	out := make([][]float32, 0, len(resp.Data))
	for _, d := range resp.Data {
		out = append(out, d.Embedding)
	}
	if len(out) != len(texts) {
		return nil, upstreamError(fmt.Sprintf("эмбеддингов %d вместо %d", len(out), len(texts)))
	}
	return out, nil
}

func (c *Client) post(ctx context.Context, path, apiKey string, body, out any, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	raw, err := json.Marshal(body)
	if err != nil {
		return upstreamError("кодирование запроса: " + err.Error())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
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
		return upstreamError(fmt.Sprintf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet))))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return upstreamError("декодирование ответа: " + err.Error())
	}
	return nil
}

func upstreamError(msg string) *domain.Error {
	return domain.NewError("AI_UPSTREAM", msg, 502)
}

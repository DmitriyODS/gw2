// Package clients — gRPC-клиенты groovesvc к другим микросервисам
// (aisvc — LLM-шлюз, msgsvc — pet-чат). Межсервисное общение — только gRPC.
package clients

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/aipb"
)

const (
	aiStatusTimeout  = 5 * time.Second
	aiStatusCacheTTL = time.Minute
	// Запас gRPC-дедлайна сверх timeout_sec самого LLM-вызова.
	aiGRPCSlack = 5 * time.Second
)

// AI — клиент aisvc. Fail-open (как прежний get_ai_client во Flask):
// сервис недоступен / ИИ выключен → Enabled=false, бот уходит в статику.
// Status кэшируется per-company на минуту; негатив не кэшируется —
// включение ИИ подхватывается сразу.
type AI struct {
	conn *grpc.ClientConn
	stub aipb.AiServiceClient
	log  *slog.Logger

	mu    sync.Mutex
	cache map[int64]time.Time // company_id → enabled-до
}

var _ domain.AIClient = (*AI)(nil)

func NewAI(addr string, log *slog.Logger) (*AI, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &AI{
		conn:  conn,
		stub:  aipb.NewAiServiceClient(conn),
		log:   log,
		cache: map[int64]time.Time{},
	}, nil
}

func (c *AI) Close() { _ = c.conn.Close() }

func (c *AI) Enabled(ctx context.Context, companyID int64) bool {
	c.mu.Lock()
	until, ok := c.cache[companyID]
	c.mu.Unlock()
	if ok && time.Now().Before(until) {
		return true
	}

	rctx, cancel := context.WithTimeout(ctx, aiStatusTimeout)
	defer cancel()
	resp, err := c.stub.Status(rctx, &aipb.StatusRequest{CompanyId: companyID})
	if err != nil || resp.GetError() != nil || !resp.GetEnabled() {
		c.mu.Lock()
		delete(c.cache, companyID)
		c.mu.Unlock()
		return false
	}
	c.mu.Lock()
	c.cache[companyID] = time.Now().Add(aiStatusCacheTTL)
	c.mu.Unlock()
	return true
}

func (c *AI) chatOnce(ctx context.Context, companyID int64, messages []map[string]any,
	toolsJSON string, maxTokens int, temperature float64,
	timeout time.Duration) (*aipb.ChatResponse, error) {

	raw, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	rctx, cancel := context.WithTimeout(ctx, timeout+aiGRPCSlack)
	defer cancel()
	resp, err := c.stub.Chat(rctx, &aipb.ChatRequest{
		CompanyId:    companyID,
		MessagesJson: string(raw),
		ToolsJson:    toolsJSON,
		MaxTokens:    int32(maxTokens),
		Temperature:  temperature,
		TimeoutSec:   timeout.Seconds(),
	})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	return resp, nil
}

func (c *AI) Chat(ctx context.Context, companyID int64, messages []map[string]any,
	maxTokens int, temperature float64, timeout time.Duration) (string, error) {

	resp, err := c.chatOnce(ctx, companyID, messages, "", maxTokens, temperature, timeout)
	if err != nil {
		return "", err
	}
	return resp.GetContent(), nil
}

// ChatWithTools — чат с OpenAI function-calling; цикл крутится здесь.
// aisvc.Chat выполняет РОВНО ОДИН ход (messages → content | tool_calls);
// onTool вызывается синхронно для каждого tool_call. Цикл останавливается,
// когда модель ответила текстом или достигнут maxIterations.
func (c *AI) ChatWithTools(ctx context.Context, companyID int64,
	messages []map[string]any, toolsJSON string,
	onTool func(name string, args map[string]any) any,
	maxTokens int, temperature float64, timeout time.Duration,
	maxIterations int) (string, error) {

	convo := append([]map[string]any{}, messages...)
	for i := 0; i < maxIterations; i++ {
		resp, err := c.chatOnce(ctx, companyID, convo, toolsJSON,
			maxTokens, temperature, timeout)
		if err != nil {
			return "", err
		}
		toolCalls := parseToolCalls(resp.GetToolCallsJson(), c.log)
		if len(toolCalls) == 0 {
			return resp.GetContent(), nil
		}

		convo = append(convo, map[string]any{
			"role":       "assistant",
			"content":    resp.GetContent(),
			"tool_calls": toolCalls,
		})
		for _, tc := range toolCalls {
			fn, _ := tc["function"].(map[string]any)
			name, _ := fn["name"].(string)
			args := map[string]any{}
			if rawArgs, ok := fn["arguments"].(string); ok && rawArgs != "" {
				_ = json.Unmarshal([]byte(rawArgs), &args)
			}
			var result any
			func() {
				defer func() {
					if r := recover(); r != nil {
						result = map[string]any{"error": "tool_handler_failed"}
					}
				}()
				result = onTool(name, args)
			}()
			rawResult, err := json.Marshal(result)
			if err != nil {
				rawResult = []byte(`{"error":"tool_result_marshal_failed"}`)
			}
			id, _ := tc["id"].(string)
			convo = append(convo, map[string]any{
				"role":         "tool",
				"tool_call_id": id,
				"content":      string(rawResult),
			})
		}
	}

	// Лимит итераций — финальный заход без tools, чтобы модель точно
	// ответила текстом, а не очередным tool_call.
	resp, err := c.chatOnce(ctx, companyID, convo, "", maxTokens, temperature, timeout)
	if err != nil {
		return "", err
	}
	return resp.GetContent(), nil
}

func parseToolCalls(raw string, log *slog.Logger) []map[string]any {
	if raw == "" {
		return nil
	}
	var parsed []map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		log.Warn("ai.tool_calls.bad_json", "raw", truncate(raw, 200))
		return nil
	}
	return parsed
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

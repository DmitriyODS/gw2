package llm

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// reasoningServer — ведёт себя как reasoning-модель OpenAI: max_tokens и
// temperature отвергает ошибкой unsupported_parameter.
func reasoningServer(t *testing.T, bodies *[]map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		*bodies = append(*bodies, body)
		for _, param := range []string{"max_tokens", "temperature"} {
			if _, ok := body[param]; ok {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"message":"Unsupported parameter","type":"invalid_request_error","param":"` +
					param + `","code":"unsupported_parameter"}}`))
				return
			}
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ок"}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestChatAdaptsToReasoningModel(t *testing.T) {
	var bodies []map[string]any
	srv := reasoningServer(t, &bodies)
	c := New(srv.URL, slog.New(slog.DiscardHandler))
	p := domain.ChatParams{Model: "gpt-5", MessagesJSON: `[]`, MaxTokens: 100, Temperature: 0.7}

	res, err := c.ChatOnce(context.Background(), p)
	if err != nil || res.Content != "ок" {
		t.Fatalf("ответ = %+v, err = %v", res, err)
	}
	last := bodies[len(bodies)-1]
	if last["max_completion_tokens"] != float64(100*reasoningBudget) || last["temperature"] != nil {
		t.Fatalf("итоговый запрос: %v", last)
	}

	// Вторая реплика уходит сразу в нужной форме — без повторных отказов.
	bodies = nil
	if _, err := c.ChatOnce(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 1 {
		t.Fatalf("запросов на вторую реплику: %d, ожидался один", len(bodies))
	}
}

func TestChatClassicModelKeepsMaxTokens(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ок"}}]}`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, slog.New(slog.DiscardHandler))

	_, err := c.ChatOnce(context.Background(), domain.ChatParams{Model: "gpt-4o-mini", MessagesJSON: `[]`, MaxTokens: 50})
	if err != nil {
		t.Fatal(err)
	}
	// Совместимые серверы знают только max_tokens; нулевая температура — тоже значение.
	if got["max_tokens"] != float64(50) || got["temperature"] != float64(0) || got["max_completion_tokens"] != nil {
		t.Fatalf("запрос классической модели: %v", got)
	}
}

func TestChatOtherBadRequestNotRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad messages","param":"messages","code":"invalid"}}`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, slog.New(slog.DiscardHandler))

	_, err := c.ChatOnce(context.Background(), domain.ChatParams{Model: "m", MessagesJSON: `[]`, MaxTokens: 1})
	if de, ok := err.(*domain.Error); !ok || de.Code != "AI_UPSTREAM" || calls != 1 {
		t.Fatalf("err = %v, calls = %d", err, calls)
	}
}

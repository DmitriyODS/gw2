package service

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

func newSupportSvc(llm domain.LLMClient, cfg SupportConfig) *Service {
	return New(newFakeRepo(), llm, &fakeCipher{}, newFakeFacts(), nil, nil, "", cfg,
		slog.New(slog.DiscardHandler))
}

// Ключ не задан → AI_DISABLED (msgsvc по нему откатывается на канированный
// автоответ).
func TestSupportReply_NoKeyDisabled(t *testing.T) {
	svc := newSupportSvc(&fakeLLM{}, SupportConfig{})
	_, err := svc.SupportReply(context.Background(), `[{"role":"user","content":"привет"}]`)
	wantDomainError(t, err, "AI_DISABLED", 409)
}

// Системный промпт добавляется ПЕРВЫМ сообщением, история — за ним; ключ и
// модель — платформенные из конфига.
func TestSupportReply_PrependsSystemPrompt(t *testing.T) {
	llm := &fakeLLM{chatResult: &domain.ChatResult{Content: "  Ответ бота  "}}
	svc := newSupportSvc(llm, SupportConfig{APIKey: "sk-support", Model: "gpt-test"})

	got, err := svc.SupportReply(context.Background(),
		`[{"role":"user","content":"как создать задачу?"}]`)
	if err != nil {
		t.Fatalf("SupportReply: %v", err)
	}
	if got != "Ответ бота" {
		t.Fatalf("content = %q, ждали стрипнутый ответ", got)
	}
	if llm.lastChat.APIKey != "sk-support" || llm.lastChat.Model != "gpt-test" {
		t.Fatalf("ключ/модель не платформенные: %+v", llm.lastChat)
	}
	msgs := llm.lastChat.MessagesJSON
	if !strings.Contains(msgs, `"role":"system"`) ||
		!strings.Contains(msgs, "Groove Work") ||
		!strings.Contains(msgs, "как создать задачу?") {
		t.Fatalf("messages без системного промпта или истории: %s", msgs)
	}
	if strings.Index(msgs, `"system"`) > strings.Index(msgs, "как создать задачу?") {
		t.Fatalf("системный промпт не первым сообщением: %s", msgs)
	}
}

// Модель по умолчанию — supportDefaultModel, если SUPPORT_AI_MODEL пуст.
func TestSupportReply_DefaultModel(t *testing.T) {
	llm := &fakeLLM{chatResult: &domain.ChatResult{Content: "ок"}}
	svc := newSupportSvc(llm, SupportConfig{APIKey: "sk"})
	if _, err := svc.SupportReply(context.Background(), `[{"role":"user","content":"хай"}]`); err != nil {
		t.Fatalf("SupportReply: %v", err)
	}
	if llm.lastChat.Model != supportDefaultModel {
		t.Fatalf("model = %q, ждали %q", llm.lastChat.Model, supportDefaultModel)
	}
}

// Невалидная/пустая история — AI_BAD_REQUEST, до LLM не доходим.
func TestSupportReply_BadHistory(t *testing.T) {
	llm := &fakeLLM{chatResult: &domain.ChatResult{Content: "ок"}}
	svc := newSupportSvc(llm, SupportConfig{APIKey: "sk"})
	for _, bad := range []string{"", "не json", "[]", `{"role":"user"}`} {
		if _, err := svc.SupportReply(context.Background(), bad); err == nil {
			t.Fatalf("ждали ошибку для %q", bad)
		}
	}
}

// Пустой ответ модели — ошибка (бот не должен слать пустое сообщение).
func TestSupportReply_EmptyContent(t *testing.T) {
	llm := &fakeLLM{chatResult: &domain.ChatResult{Content: "   "}}
	svc := newSupportSvc(llm, SupportConfig{APIKey: "sk"})
	_, err := svc.SupportReply(context.Background(), `[{"role":"user","content":"хай"}]`)
	wantDomainError(t, err, "AI_EMPTY", 502)
}

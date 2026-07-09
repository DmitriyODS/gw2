package service

import (
	"context"
	"strings"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

func TestTransformTextSendsInstructionAndReturnsResult(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	llm.chatResult = &domain.ChatResult{Content: "  Исправленный текст  "}

	out, err := svc.TransformText(context.Background(), 1, "fix", "", "текст с ошибкай")
	if err != nil {
		t.Fatalf("TransformText: %v", err)
	}
	if out != "Исправленный текст" {
		t.Fatalf("ожидали обрезанный результат, получили %q", out)
	}
	if !strings.Contains(llm.lastChat.MessagesJSON, "текст с ошибкай") {
		t.Fatalf("исходный текст не дошёл до LLM: %s", llm.lastChat.MessagesJSON)
	}
	if !strings.Contains(llm.lastChat.MessagesJSON, "Исправь орфографические") {
		t.Fatalf("инструкция fix не дошла до LLM: %s", llm.lastChat.MessagesJSON)
	}
	if llm.lastChat.ToolsJSON != "" {
		t.Fatalf("text-tools не должен передавать tools: %s", llm.lastChat.ToolsJSON)
	}
}

func TestTransformTextToneAndTranslateRequireKnownStyle(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	llm.chatResult = &domain.ChatResult{Content: "ok"}

	_, err := svc.TransformText(context.Background(), 1, "tone", "sarcastic", "текст")
	wantDomainError(t, err, "VALIDATION", 400)
	_, err = svc.TransformText(context.Background(), 1, "translate", "fr", "текст")
	wantDomainError(t, err, "VALIDATION", 400)

	if _, err := svc.TransformText(context.Background(), 1, "tone", "formal", "текст"); err != nil {
		t.Fatalf("tone formal: %v", err)
	}
	if !strings.Contains(llm.lastChat.MessagesJSON, "деловом") {
		t.Fatalf("тон не попал в инструкцию: %s", llm.lastChat.MessagesJSON)
	}
}

func TestTransformTextValidation(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = enabledCompany(1)

	_, err := svc.TransformText(context.Background(), 1, "fix", "", "   ")
	wantDomainError(t, err, "VALIDATION", 400)
	_, err = svc.TransformText(context.Background(), 1, "explode", "", "текст")
	wantDomainError(t, err, "VALIDATION", 400)
	_, err = svc.TransformText(context.Background(), 1, "fix", "", strings.Repeat("а", textToolMaxChars+1))
	wantDomainError(t, err, "VALIDATION", 400)
}

func TestTransformTextAiDisabled(t *testing.T) {
	svc, repo, _ := newTestService()
	repo.companies[1] = &domain.CompanyAI{ID: 1, Enabled: false}

	_, err := svc.TransformText(context.Background(), 1, "fix", "", "текст")
	wantDomainError(t, err, "AI_DISABLED", 409)
}

func TestTransformTextEmptyLLMAnswer(t *testing.T) {
	svc, repo, llm := newTestService()
	repo.companies[1] = enabledCompany(1)
	llm.chatResult = &domain.ChatResult{Content: "   "}

	_, err := svc.TransformText(context.Background(), 1, "fix", "", "текст")
	wantDomainError(t, err, "AI_EMPTY", 502)
}

package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

func tvService() (*Service, *fakeRepo, *fakeLLM, *fakeFacts) {
	repo := newFakeRepo()
	llm := &fakeLLM{}
	facts := newFakeFacts()
	svc := New(repo, llm, &fakeCipher{}, facts, nil, nil, "", slog.New(slog.DiscardHandler))
	return svc, repo, llm, facts
}

func TestGenerateTVFactStoresStrippedText(t *testing.T) {
	svc, repo, llm, facts := tvService()
	repo.companies[1] = enabledCompany(1)
	llm.chatResult = &domain.ChatResult{Content: "  «Работа любит счёт.»  "}

	if err := svc.GenerateTVFact(context.Background(), 1); err != nil {
		t.Fatalf("GenerateTVFact: %v", err)
	}
	fact := facts.facts[1]
	if fact == nil {
		t.Fatal("факт не записан в кэш")
	}
	if fact.Text != "Работа любит счёт." {
		t.Fatalf("text = %q — кавычки/пробелы не срезаны", fact.Text)
	}
	if fact.Kind != "general" && fact.Kind != "context" {
		t.Fatalf("kind = %q", fact.Kind)
	}
	if fact.GeneratedAt == "" {
		t.Fatal("generated_at пуст")
	}
	if llm.lastChat.MaxTokens != tvMaxTokens || llm.lastChat.Temperature != tvTemperature {
		t.Fatalf("параметры chat: %+v", llm.lastChat)
	}
}

func TestGenerateTVFactDisabledAIClearsCache(t *testing.T) {
	svc, repo, _, facts := tvService()
	repo.companies[1] = &domain.CompanyAI{ID: 1, Enabled: false}
	facts.facts[1] = &domain.TVFact{Kind: "general", Text: "старый факт"}

	if err := svc.GenerateTVFact(context.Background(), 1); err != nil {
		t.Fatalf("GenerateTVFact: %v", err)
	}
	if facts.facts[1] != nil {
		t.Fatal("факт выключенной компании должен быть затёрт")
	}
	got, err := svc.GetTVFact(context.Background(), 1)
	if err != nil || got != nil {
		t.Fatalf("GetTVFact после выключения = %v, %v", got, err)
	}
}

func TestTVWeekWindowMSK(t *testing.T) {
	now := time.Date(2026, 6, 12, 1, 30, 0, 0, time.UTC) // 04:30 МСК
	start, end := tvWeekWindowMSK(now)
	if got := end.Format("2006-01-02 15:04:05"); got != "2026-06-12 23:59:59" {
		t.Fatalf("end = %s", got)
	}
	if got := start.Format("2006-01-02 15:04:05"); got != "2026-06-06 00:00:00" {
		t.Fatalf("start = %s", got)
	}
}

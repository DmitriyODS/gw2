package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
)

// fakeMeter — учёт токенов доступа в памяти.
type fakeMeter struct {
	left   int64
	spends []domain.TokenSpend
}

func (m *fakeMeter) Check(_ context.Context, userID, _ int64) (int64, int64, bool) {
	return userID, m.left, m.left > 0
}

func (m *fakeMeter) Consume(_ context.Context, u domain.TokenSpend) (bool, int64) {
	m.spends = append(m.spends, u)
	if u.OwnKey {
		return true, m.left
	}
	if u.Billed > m.left {
		return false, m.left
	}
	m.left -= u.Billed
	return true, m.left
}

// Один токен доступа = 1000 токенов самой дешёвой модели; дорогие модели
// расходуют его пропорционально цене (61 руб. против 5,16 руб. за 1 млн).
func TestAccessTokensScaleWithModelPrice(t *testing.T) {
	cat := domain.NewModelCatalog([]*domain.AIModel{
		{Code: "cheap", PricePerMTok: 516, IsActive: true},
		{Code: "gpt", PricePerMTok: 6100, IsActive: true},
		{Code: "gemini", PricePerMTok: 7600, IsActive: true},
	})

	if got := cat.AccessTokens("cheap", 1000); got != 1 {
		t.Fatalf("1000 токенов базовой модели = 1 токен доступа, получено %d", got)
	}
	// 1000 × (6100/516) / 1000 ≈ 11,82 → округление вверх.
	if got := cat.AccessTokens("gpt", 1000); got != 12 {
		t.Fatalf("gpt: ожидалось 12 токенов доступа, получено %d", got)
	}
	if got := cat.AccessTokens("gemini", 1000); got != 15 {
		t.Fatalf("gemini: ожидалось 15 токенов доступа, получено %d", got)
	}
	// Любое обращение стоит хотя бы один токен — бесплатных не бывает.
	if got := cat.AccessTokens("gpt", 1); got != 1 {
		t.Fatalf("минимальное списание — 1 токен, получено %d", got)
	}
}

// Обращение ассистента списывает токены с баланса пользователя.
func TestAssistantSpendsUserTokens(t *testing.T) {
	svc, repo, llm := newTestService()
	meter := &fakeMeter{left: 1000}
	svc.WithTokenMeter(meter)
	repo.userAI[10] = &domain.UserAI{UserID: 10, Enabled: true, FeatAssistant: true, FeatNotes: true}
	llm.chatResult = &domain.ChatResult{
		Content: "ответ",
		Usage:   domain.TokenUsage{PromptTokens: 400, CompletionTokens: 600},
	}

	if _, err := svc.TransformText(context.Background(), 10, "fix", "", "текст"); err != nil {
		t.Fatalf("TransformText: %v", err)
	}
	if len(meter.spends) != 1 {
		t.Fatalf("ожидалось одно списание, получено %d", len(meter.spends))
	}
	spend := meter.spends[0]
	if spend.PayerID != 10 || spend.Feature != domain.FeatureNotes || spend.OwnKey {
		t.Fatalf("неожиданное списание: %+v", spend)
	}
	// 1000 токенов gpt-5.4-nano ≈ 12 токенов доступа.
	if spend.Billed != 12 {
		t.Fatalf("ожидалось 12 токенов доступа, получено %d", spend.Billed)
	}
	if meter.left != 988 {
		t.Fatalf("остаток должен уменьшиться: %d", meter.left)
	}
}

// Свой ключ пользователя обходит квоту: расход фиксируется, но не списывается.
func TestOwnKeySkipsBilling(t *testing.T) {
	svc, repo, llm := newTestService()
	meter := &fakeMeter{left: 0} // токенов нет вовсе
	svc.WithTokenMeter(meter)
	repo.userAI[10] = enabledUserAI(10)
	llm.chatResult = &domain.ChatResult{
		Content: "ответ",
		Usage:   domain.TokenUsage{PromptTokens: 100, CompletionTokens: 100},
	}

	if _, err := svc.TransformText(context.Background(), 10, "fix", "", "текст"); err != nil {
		t.Fatalf("на своём ключе обращение должно проходить без токенов: %v", err)
	}
	if len(meter.spends) != 1 || !meter.spends[0].OwnKey || meter.spends[0].Billed != 0 {
		t.Fatalf("на своём ключе списания быть не должно: %+v", meter.spends)
	}
	if llm.lastChat.APIKey != "sk-secret" {
		t.Fatalf("запрос должен уйти на личный ключ: %+v", llm.lastChat)
	}
}

// Кончились токены — обращение отклоняется до похода в модель.
func TestNoTokensBlocksRequest(t *testing.T) {
	svc, repo, llm := newTestService()
	svc.WithTokenMeter(&fakeMeter{left: 0})
	repo.userAI[10] = &domain.UserAI{UserID: 10, Enabled: true, FeatAssistant: true, FeatNotes: true}

	_, err := svc.TransformText(context.Background(), 10, "fix", "", "текст")
	wantDomainError(t, err, "AI_NO_TOKENS", 402)
	if llm.chatCalls != 0 {
		t.Fatal("без токенов запрос к модели уходить не должен")
	}
}

// Компанийные возможности тратят токены СОЗДАТЕЛЯ компании.
func TestCompanyFeatureSpendsOwnerTokens(t *testing.T) {
	svc, repo, _ := newTestService()
	meter := &fakeMeter{left: 500}
	svc.WithTokenMeter(meter)
	c := enabledCompany(1)
	c.OwnerID = 77
	repo.companies[1] = c
	repo.tasks[5] = &domain.TaskText{ID: 5, CompanyID: ptrInt64(1), Name: "Задача"}

	if err := svc.reindexTaskOnce(context.Background(), 5); err != nil {
		t.Fatalf("reindexTaskOnce: %v", err)
	}
	if len(meter.spends) != 1 {
		t.Fatalf("ожидалось одно списание, получено %d", len(meter.spends))
	}
	if meter.spends[0].PayerID != 77 || meter.spends[0].CompanyID != 1 {
		t.Fatalf("платить должен создатель компании: %+v", meter.spends[0])
	}
}

// Карточка настроек показывает остаток токенов и доступные модели.
func TestMySettingsShowsTokensAndModels(t *testing.T) {
	svc, repo, _ := newTestService()
	svc.WithTokenMeter(&fakeMeter{left: 742})
	repo.userAI[10] = &domain.UserAI{UserID: 10, Enabled: true, FeatAssistant: true}

	got, err := svc.GetMySettings(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetMySettings: %v", err)
	}
	if got.TokensLeft != 742 {
		t.Fatalf("остаток токенов: %d", got.TokensLeft)
	}
	if len(got.Models) != 2 {
		t.Fatalf("пользователю доступны две модели, получено %d", len(got.Models))
	}
	if !got.PlatformReady {
		t.Fatal("платформенный ключ настроен — карточка должна это показывать")
	}
	var _ dto.MyAiSettings = *got
}

func ptrInt64(v int64) *int64 { return &v }

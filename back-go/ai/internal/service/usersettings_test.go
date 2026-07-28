package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
)

func newUserSettingsService() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	return New(repo, &sequencedLLM{}, &fakeCipher{}, newFakeFacts(), newFakeAssistantRepo(),
		&fakeTasksClient{}, "https://gw.example", SupportConfig{}, slog.New(slog.DiscardHandler)), repo
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

// TestGetMySettings_DefaultsWhenNeverConfigured — строки в БД ещё нет: ключа
// нет, но карточка настроек всё равно должна отрисоваться (дефолтная модель).
func TestGetMySettings_DefaultsWhenNeverConfigured(t *testing.T) {
	svc, _ := newUserSettingsService()
	got, err := svc.GetMySettings(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetMySettings: %v", err)
	}
	if got.HasKey || got.KeyHint != nil {
		t.Fatalf("ключа быть не должно: %+v", got)
	}
	if got.ModelChat != domain.PlatformModelChat {
		t.Fatalf("ожидалась дефолтная модель, получено %q", got.ModelChat)
	}
}

// TestUpdateMySettings_StoresEncryptedKeyAndHint — сырой ключ наружу не
// возвращается и в БД лежит зашифрованным, пользователю видна только маска.
func TestUpdateMySettings_StoresEncryptedKeyAndHint(t *testing.T) {
	svc, repo := newUserSettingsService()
	got, err := svc.UpdateMySettings(context.Background(), 10, dto.MyAiSettingsUpdate{
		APIKey: strPtr("sk-personal-secret"), ModelChat: strPtr(domain.PlatformModelChat),
	})
	if err != nil {
		t.Fatalf("UpdateMySettings: %v", err)
	}
	if !got.HasKey || got.KeyHint == nil {
		t.Fatalf("ожидались признак ключа и маска: %+v", got)
	}
	if got.ModelChat != domain.PlatformModelChat {
		t.Fatalf("модель не сохранилась: %q", got.ModelChat)
	}
	stored := repo.userAI[10]
	if stored == nil || string(stored.APIKeyEnc) != "enc:sk-personal-secret" {
		t.Fatalf("ключ должен лежать зашифрованным: %+v", stored)
	}
}

// TestUpdateMySettings_EmptyKeyKeepsPrevious — пустой api_key означает «не
// менять» (форма настроек не присылает существующий ключ обратно).
func TestUpdateMySettings_EmptyKeyKeepsPrevious(t *testing.T) {
	svc, repo := newUserSettingsService()
	ctx := context.Background()
	if _, err := svc.UpdateMySettings(ctx, 10, dto.MyAiSettingsUpdate{APIKey: strPtr("sk-first")}); err != nil {
		t.Fatalf("первое сохранение: %v", err)
	}
	if _, err := svc.UpdateMySettings(ctx, 10, dto.MyAiSettingsUpdate{
		APIKey: strPtr("   "), Enabled: boolPtr(false),
	}); err != nil {
		t.Fatalf("второе сохранение: %v", err)
	}
	stored := repo.userAI[10]
	if string(stored.APIKeyEnc) != "enc:sk-first" {
		t.Fatalf("ключ не должен был меняться: %+v", stored)
	}
	if stored.Enabled {
		t.Fatalf("тумблер должен был выключиться")
	}
}

// TestUpdateMySettings_ClearKeyDisconnects — clear_key отвязывает ассистента:
// ключ и маска стираются, следующее сообщение получит AI_DISABLED.
func TestUpdateMySettings_ClearKeyDisconnects(t *testing.T) {
	svc, repo := newUserSettingsService()
	ctx := context.Background()
	if _, err := svc.UpdateMySettings(ctx, 10, dto.MyAiSettingsUpdate{APIKey: strPtr("sk-first")}); err != nil {
		t.Fatalf("сохранение ключа: %v", err)
	}
	got, err := svc.UpdateMySettings(ctx, 10, dto.MyAiSettingsUpdate{ClearKey: true})
	if err != nil {
		t.Fatalf("сброс ключа: %v", err)
	}
	if got.HasKey || got.KeyHint != nil {
		t.Fatalf("после сброса ключа быть не должно: %+v", got)
	}
	if repo.userAI[10].APIKeyEnc != nil {
		t.Fatalf("ключ должен быть стёрт и в хранилище")
	}
	// Свой ключ убран — ассистент продолжает работать на ПЛАТФОРМЕННОМ ключе
	// в пределах токенов тарифа. Чтобы выключить ИИ совсем, есть тумблер.
	if _, err := svc.SendAssistantMessage(ctx, 10, nil, "Привет"); err != nil {
		t.Fatalf("без своего ключа ассистент работает на платформенном: %v", err)
	}
	if _, err := svc.UpdateMySettings(ctx, 10, dto.MyAiSettingsUpdate{Enabled: boolPtr(false)}); err != nil {
		t.Fatalf("выключение ИИ: %v", err)
	}
	_, err = svc.SendAssistantMessage(ctx, 10, nil, "Привет")
	wantDomainError(t, err, "AI_DISABLED", 409)
}

// TestUpdateMySettings_InvalidatesClientCache — правка настроек подхватывается
// сразу, а не через cacheTTL: иначе человек вводит ключ и минуту получает отказ.
func TestUpdateMySettings_InvalidatesClientCache(t *testing.T) {
	svc, _ := newUserSettingsService()
	ctx := context.Background()

	// Прогреваем отрицательный ответ кэша.
	if _, err := svc.SendAssistantMessage(ctx, 10, nil, "Привет"); err == nil {
		t.Fatalf("без ключа ожидалась ошибка")
	}
	if _, err := svc.UpdateMySettings(ctx, 10, dto.MyAiSettingsUpdate{APIKey: strPtr("sk-fresh")}); err != nil {
		t.Fatalf("UpdateMySettings: %v", err)
	}
	if _, err := svc.SendAssistantMessage(ctx, 10, nil, "Привет"); err != nil {
		t.Fatalf("после подключения ключа ассистент должен отвечать: %v", err)
	}
}

// TestTestMySettings_NeedsKey — проверять связь нечем, пока ключ не подключён.
func TestTestMySettings_NeedsKey(t *testing.T) {
	svc, _ := newUserSettingsService()
	_, err := svc.TestMySettings(context.Background(), 10)
	wantDomainError(t, err, "AI_DISABLED", 409)
}

func TestTestMySettings_PingsChatOnly(t *testing.T) {
	svc, repo := newUserSettingsService()
	repo.userAI[10] = enabledUserAI(10)

	got, err := svc.TestMySettings(context.Background(), 10)
	if err != nil {
		t.Fatalf("TestMySettings: %v", err)
	}
	if !got.Chat || got.Error != nil {
		t.Fatalf("ожидался успешный chat: %+v", got)
	}
	// Эмбеддинги остаются на компанийном ключе — личный их не проверяет.
	if got.Embedding {
		t.Fatalf("личный ключ эмбеддинги не обслуживает: %+v", got)
	}
}

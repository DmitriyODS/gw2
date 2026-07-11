package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

type fakeSupportAI struct {
	reply        string
	err          error
	calls        int
	lastMessages string
}

func (f *fakeSupportAI) SupportReply(_ context.Context, messagesJSON string) (string, error) {
	f.calls++
	f.lastMessages = messagesJSON
	if f.err != nil {
		return "", f.err
	}
	return f.reply, nil
}

// ИИ-ответ поддержки: история уходит в формате OpenAI (владелец — user,
// поддержка — assistant), ответ сохраняется бот-сообщением kind=dev_reply.
func TestSupportAIReplyAnswers(t *testing.T) {
	svc, repo, _, _ := newTestEnv()
	ai := &fakeSupportAI{reply: "Откройте раздел «Задачи» и нажмите «Создать»."}
	svc.ai = ai
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2, i64p(10))
	conv, _ := repo.GetConversation(ctx, dev.ID)
	owner := int64(2)
	// История: старый вопрос владельца + старый ответ супер-админа (человека).
	repo.CreateMessage(ctx, domain.NewMessage{ConversationID: dev.ID, SenderID: &owner, Text: str("привет")})
	admin := int64(1)
	repo.CreateMessage(ctx, domain.NewMessage{ConversationID: dev.ID, SenderID: &admin,
		Text: str("Здравствуйте!"), Kind: domain.KindDevReply})
	msg, err := repo.CreateMessage(ctx, domain.NewMessage{ConversationID: dev.ID, SenderID: &owner,
		Text: str("как создать задачу?")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	reply, err := svc.supportAIReply(ctx, conv, msg)
	if err != nil {
		t.Fatalf("supportAIReply: %v", err)
	}
	if reply == nil || !reply.IsBot || reply.Kind != domain.KindDevReply ||
		reply.SenderID != nil || *reply.Text != ai.reply {
		t.Fatalf("неверная форма ИИ-ответа: %+v", reply)
	}
	m := ai.lastMessages
	if !strings.Contains(m, `"role":"user","content":"как создать задачу?"`) ||
		!strings.Contains(m, `"role":"assistant","content":"Здравствуйте!"`) {
		t.Fatalf("история для ИИ неверна: %s", m)
	}
	if strings.Contains(m, `"role":"system"`) {
		t.Fatalf("системный промпт добавляет aisvc, не msgsvc: %s", m)
	}
}

// Человек-поддержка отвечал в последние 15 минут — бот молчит.
func TestSupportAIReplySilentWhileHumanActive(t *testing.T) {
	svc, repo, _, _ := newTestEnv()
	ai := &fakeSupportAI{reply: "не должен уйти"}
	svc.ai = ai
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2, i64p(10))
	conv, _ := repo.GetConversation(ctx, dev.ID)
	admin, owner := int64(1), int64(2)
	human, _ := repo.CreateMessage(ctx, domain.NewMessage{ConversationID: dev.ID, SenderID: &admin,
		Text: str("Разбираюсь с вашим вопросом"), Kind: domain.KindDevReply})
	// fakeRepo тикает от now-1h — «свежесть» ответа человека ставим руками.
	repo.msgs[human.ID].CreatedAt = time.Now().UTC()
	msg, _ := repo.CreateMessage(ctx, domain.NewMessage{ConversationID: dev.ID, SenderID: &owner,
		Text: str("а ещё вопрос")})

	reply, err := svc.supportAIReply(ctx, conv, msg)
	if err != nil {
		t.Fatalf("supportAIReply: %v", err)
	}
	if reply != nil || ai.calls != 0 {
		t.Fatalf("бот должен молчать при живом человеке: reply=%+v calls=%d", reply, ai.calls)
	}
}

// Ошибка ИИ — supportAIReply отдаёт её наверх (schedule откатывается на
// канированный автоответ), бот-сообщение не создаётся.
func TestSupportAIReplyErrorNoMessage(t *testing.T) {
	svc, repo, _, _ := newTestEnv()
	svc.ai = &fakeSupportAI{err: errors.New("ai down")}
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2, i64p(10))
	conv, _ := repo.GetConversation(ctx, dev.ID)
	owner := int64(2)
	msg, _ := repo.CreateMessage(ctx, domain.NewMessage{ConversationID: dev.ID, SenderID: &owner,
		Text: str("вопрос")})

	if _, err := svc.supportAIReply(ctx, conv, msg); err == nil {
		t.Fatal("ждали ошибку ИИ")
	}
	for _, m := range repo.msgs {
		if m.IsBot {
			t.Fatalf("бот-сообщение не должно создаваться при ошибке: %+v", m)
		}
	}
}

// SendMessage с настроенным ИИ НЕ создаёт канированный автоответ синхронно
// (ответ уходит фоном); сообщение без текста — канированная ветка как раньше.
func TestSendMessageWithAIIsAsync(t *testing.T) {
	svc, repo, _, pub := newTestEnv()
	// ИИ, который молчит из-за «занятого человека», — фоновая горутина
	// гарантированно не создаст сообщений (без гонок в тесте).
	svc.ai = &fakeSupportAI{reply: "ответ"}
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2, i64p(10))
	admin := int64(1)
	human, _ := repo.CreateMessage(ctx, domain.NewMessage{ConversationID: dev.ID, SenderID: &admin,
		Text: str("на связи"), Kind: domain.KindDevReply})
	repo.msgs[human.ID].CreatedAt = time.Now().UTC()

	if _, err := svc.SendMessage(ctx, dev.ID, 2, dto.MessageCreate{Text: str("помогите")}); err != nil {
		t.Fatalf("send: %v", err)
	}
	// Синхронно опубликовано только само сообщение (без канированного бота).
	if news := pub.byName("message:new"); len(news) != 1 {
		t.Fatalf("message:new событий: %d, ожидалось 1 (ИИ отвечает фоном)", len(news))
	}
	for _, m := range repo.msgs {
		if m.IsBot {
			t.Fatalf("канированный автоответ не должен создаваться при ИИ: %+v", m)
		}
	}
}

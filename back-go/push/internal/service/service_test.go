package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/push/internal/domain"
)

type fakeTokens struct {
	byUser  map[int64][]string
	deleted []string
}

func (f *fakeTokens) Upsert(_ context.Context, t domain.DeviceToken) error {
	f.byUser[t.UserID] = append(f.byUser[t.UserID], t.Token)
	return nil
}
func (f *fakeTokens) Delete(_ context.Context, token string) error {
	f.deleted = append(f.deleted, token)
	return nil
}
func (f *fakeTokens) ListByUsers(_ context.Context, ids []int64) ([]domain.DeviceToken, error) {
	var out []domain.DeviceToken
	for _, id := range ids {
		for _, tok := range f.byUser[id] {
			out = append(out, domain.DeviceToken{Token: tok, UserID: id})
		}
	}
	return out, nil
}

type fakeUsers struct{ names map[int64]string }

func (f *fakeUsers) Names(_ context.Context, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	for _, id := range ids {
		if n, ok := f.names[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}

type fakePresence struct{ online map[int64]bool }

func (f *fakePresence) Offline(_ context.Context, ids []int64) ([]int64, error) {
	var out []int64
	for _, id := range ids {
		if !f.online[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

type sent struct {
	token string
	n     domain.Notification
}

type fakeSender struct{ sent []sent }

func (f *fakeSender) Enabled() bool { return true }
func (f *fakeSender) Send(_ context.Context, token string, n domain.Notification) (bool, error) {
	f.sent = append(f.sent, sent{token, n})
	return false, nil
}

func newSvc() (*Service, *fakeTokens, *fakeSender, *fakePresence) {
	tokens := &fakeTokens{byUser: map[int64][]string{}}
	sender := &fakeSender{}
	pres := &fakePresence{online: map[int64]bool{}}
	svc := New(Deps{
		Tokens:   tokens,
		Users:    &fakeUsers{names: map[int64]string{7: "Иван"}},
		Presence: pres,
		Sender:   sender,
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return svc, tokens, sender, pres
}

func TestMessagePushExcludesSenderAndUsesName(t *testing.T) {
	svc, tokens, sender, _ := newSvc()
	tokens.byUser[5] = []string{"tok5"}
	tokens.byUser[7] = []string{"tok7"} // отправитель — пуш не должен прийти

	payload := []byte(`{"conversation_id":3,"from_user_id":7,"message":{"sender_id":7,"text":"привет","kind":"text"}}`)
	svc.Dispatch(context.Background(), "message:new", payload, []string{"user_5", "user_7"})

	if len(sender.sent) != 1 || sender.sent[0].token != "tok5" {
		t.Fatalf("ожидался 1 пуш на tok5, получено %+v", sender.sent)
	}
	if sender.sent[0].n.Title != "Иван" || sender.sent[0].n.Body != "привет" {
		t.Fatalf("неверный заголовок/текст: %+v", sender.sent[0].n)
	}
}

func TestTaskPushToResponsibleOnly(t *testing.T) {
	svc, tokens, sender, _ := newSvc()
	tokens.byUser[9] = []string{"tok9"}

	payload := []byte(`{"id":42,"name":"Сделать отчёт","author_id":1,"responsible_user_id":9}`)
	svc.Dispatch(context.Background(), "task:created", payload, []string{"all"})

	if len(sender.sent) != 1 || sender.sent[0].n.Data["task_id"] != "42" {
		t.Fatalf("ожидался пуш ответственному с task_id=42, получено %+v", sender.sent)
	}
}

func TestTaskPushSkippedWhenAuthorIsResponsible(t *testing.T) {
	svc, tokens, sender, _ := newSvc()
	tokens.byUser[1] = []string{"tok1"}
	payload := []byte(`{"id":42,"name":"x","author_id":1,"responsible_user_id":1}`)
	svc.Dispatch(context.Background(), "task:created", payload, []string{"all"})
	if len(sender.sent) != 0 {
		t.Fatalf("автор=ответственный — пуша быть не должно, получено %+v", sender.sent)
	}
}

func TestOnlineRecipientSkipped(t *testing.T) {
	svc, tokens, sender, pres := newSvc()
	tokens.byUser[5] = []string{"tok5"}
	pres.online[5] = true
	payload := []byte(`{"conversation_id":3,"from_user_id":7,"message":{"text":"hi","kind":"text"}}`)
	svc.Dispatch(context.Background(), "message:new", payload, []string{"user_5"})
	if len(sender.sent) != 0 {
		t.Fatalf("онлайн-получателю пуш слать не нужно, получено %+v", sender.sent)
	}
}

func TestCallPushHighPriority(t *testing.T) {
	svc, tokens, sender, _ := newSvc()
	tokens.byUser[5] = []string{"tok5"}
	payload := []byte(`{"id":11,"media":"video","initiator_id":7,"initiator_fio":"Иван"}`)
	svc.Dispatch(context.Background(), "call:incoming", payload, []string{"user_5"})
	if len(sender.sent) != 1 || !sender.sent[0].n.HighPriority {
		t.Fatalf("ожидался high-priority пуш звонка, получено %+v", sender.sent)
	}
	if sender.sent[0].n.Data["call_id"] != "11" || sender.sent[0].n.Channel != domain.ChannelCalls {
		t.Fatalf("неверные данные звонка: %+v", sender.sent[0].n)
	}
}

func TestInvalidTokenPruned(t *testing.T) {
	svc, tokens, _, _ := newSvc()
	tokens.byUser[5] = []string{"dead"}
	svc.sender = &pruningSender{}
	payload := []byte(`{"id":42,"name":"x","author_id":1,"responsible_user_id":5}`)
	svc.Dispatch(context.Background(), "task:created", payload, []string{"all"})
	if len(tokens.deleted) != 1 || tokens.deleted[0] != "dead" {
		t.Fatalf("мёртвый токен должен быть удалён, deleted=%+v", tokens.deleted)
	}
}

type pruningSender struct{}

func (pruningSender) Enabled() bool { return true }
func (pruningSender) Send(context.Context, string, domain.Notification) (bool, error) {
	return true, nil
}

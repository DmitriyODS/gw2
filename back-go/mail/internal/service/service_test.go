package service

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/mail/internal/domain"
)

type captureSender struct{ last domain.Message }

func (c *captureSender) Send(_ context.Context, msg domain.Message) error {
	c.last = msg
	return nil
}

func TestSendVerifyEmailRendersTemplate(t *testing.T) {
	cap := &captureSender{}
	svc := New(cap, slog.Default())

	err := svc.Send(context.Background(), "user@example.com", "Иван Иванов", "verify_email", map[string]string{
		"code": "482913",
		"link": "https://gw.example.com/verify-email?token=abc",
		"fio":  "Иван",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if cap.last.Subject == "" {
		t.Error("пустая тема письма")
	}
	for _, want := range []string{"482913", "https://gw.example.com/verify-email?token=abc", "Иван", "Groove"} {
		if !strings.Contains(cap.last.HTML, want) {
			t.Errorf("в HTML письма нет %q", want)
		}
	}
}

func TestSendResetPassword(t *testing.T) {
	cap := &captureSender{}
	svc := New(cap, slog.Default())
	err := svc.Send(context.Background(), "u@e.com", "Иван", "reset_password", map[string]string{
		"fio": "Иван", "link": "https://gw.example.com/reset-password?token=xyz",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(cap.last.HTML, "reset-password?token=xyz") {
		t.Error("в письме сброса нет ссылки")
	}
}

func TestSendCompanyInvite(t *testing.T) {
	cap := &captureSender{}
	svc := New(cap, slog.Default())
	err := svc.Send(context.Background(), "u@e.com", "", "company_invite", map[string]string{
		"company": "Acme", "role": "Менеджер", "link": "https://gw.example.com/invite/abc",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	for _, want := range []string{"Acme", "Менеджер", "/invite/abc"} {
		if !strings.Contains(cap.last.HTML, want) {
			t.Errorf("в приглашении нет %q", want)
		}
	}
}

func TestSendUnknownTemplate(t *testing.T) {
	svc := New(&captureSender{}, slog.Default())
	if err := svc.Send(context.Background(), "u@e.com", "", "nope", nil); err == nil {
		t.Fatal("ожидалась ошибка для неизвестного шаблона")
	}
}

// Package service — рендер брендированных HTML-шаблонов и отправка через Sender.
package service

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/mail/internal/domain"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

// subjects — тема письма по типу шаблона; ключ совпадает с именем файла без .html.
var subjects = map[string]string{
	"verify_email":   "Подтверждение почты — Groove Work",
	"reset_password": "Сброс пароля — Groove Work",
	"company_invite": "Приглашение в команду — Groove Work",
}

type Service struct {
	sender domain.Sender
	log    *slog.Logger
}

func New(sender domain.Sender, log *slog.Logger) *Service {
	return &Service{sender: sender, log: log}
}

// Send рендерит шаблон tmpl с params и отправляет письмо на адрес to.
// Параметры шаблона (code, link, fio, …) доступны как поля map: {{.code}}.
func (s *Service) Send(ctx context.Context, to, toName, tmpl string, params map[string]string) error {
	subject, ok := subjects[tmpl]
	if !ok {
		return fmt.Errorf("неизвестный шаблон %q", tmpl)
	}
	data := make(map[string]any, len(params)+1)
	for k, v := range params {
		data[k] = v
	}
	if _, exists := data["fio"]; !exists {
		data["fio"] = toName
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, tmpl+".html", data); err != nil {
		return fmt.Errorf("рендер %s: %w", tmpl, err)
	}
	return s.sender.Send(ctx, domain.Message{
		To:      to,
		ToName:  toName,
		Subject: subject,
		HTML:    buf.String(),
	})
}

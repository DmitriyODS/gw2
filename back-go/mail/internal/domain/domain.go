// Package domain — модели и порты mailsvc. Сервис stateless: домен сводится к
// сообщению и порту отправки.
package domain

import "context"

// Message — готовое к отправке письмо (HTML).
type Message struct {
	To      string
	ToName  string
	Subject string
	HTML    string
}

// Sender — транспорт доставки письма (SMTP).
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

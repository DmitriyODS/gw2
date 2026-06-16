// Package smtp — реализация domain.Sender поверх стандартного net/smtp.
// Поддерживает три режима TLS: starttls (587/25), tls (implicit, 465), none
// (dev/mailpit без шифрования). Письма — HTML, UTF-8, quoted-printable.
package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/mail/internal/domain"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	FromName string
	TLSMode  string // starttls | tls | none
}

type Client struct {
	cfg Config
	log *slog.Logger
}

var _ domain.Sender = (*Client)(nil)

func New(cfg Config, log *slog.Logger) *Client { return &Client{cfg: cfg, log: log} }

func (c *Client) Send(_ context.Context, msg domain.Message) error {
	addr := net.JoinHostPort(c.cfg.Host, c.cfg.Port)
	raw := c.buildMessage(msg)

	var auth smtp.Auth
	if c.cfg.User != "" {
		auth = smtp.PlainAuth("", c.cfg.User, c.cfg.Password, c.cfg.Host)
	}

	if c.cfg.TLSMode == "tls" {
		return c.sendImplicitTLS(addr, auth, msg.To, raw)
	}
	// starttls/none: net/smtp сам поднимает STARTTLS, если сервер его анонсирует
	// (для mailpit без TLS — отправка без шифрования, auth не задаётся).
	return smtp.SendMail(addr, auth, c.cfg.From, []string{msg.To}, raw)
}

// sendImplicitTLS — соединение, зашифрованное с первого байта (порт 465).
func (c *Client) sendImplicitTLS(addr string, auth smtp.Auth, to string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: c.cfg.Host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := client.Mail(c.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (c *Client) buildMessage(msg domain.Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + address(c.cfg.FromName, c.cfg.From) + "\r\n")
	b.WriteString("To: " + address(msg.ToName, msg.To) + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")

	var body strings.Builder
	qp := quotedprintable.NewWriter(&body)
	_, _ = qp.Write([]byte(msg.HTML))
	_ = qp.Close()
	b.WriteString(body.String())

	return []byte(b.String())
}

// address — заголовок адреса с опциональным отображаемым именем (RFC 2047).
func address(name, email string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", name), email)
}

// Package smtp — реализация domain.Sender поверх стандартного net/smtp.
// Поддерживает три режима TLS: starttls (587/25), tls (implicit, 465), none
// (dev/mailpit без шифрования). Письма — HTML, UTF-8, quoted-printable.
//
// Механизм авторизации выбирается по анонсу сервера: предпочитаем LOGIN
// (Beget и ряд провайдеров не поддерживают PLAIN — отвечают 504), иначе PLAIN.
// LOGIN в net/smtp не входит — реализован тут (loginAuth).
package smtp

import (
	"context"
	"crypto/tls"
	"errors"
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

	client, err := c.dial(addr)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if c.cfg.User != "" {
		if ok, mechs := client.Extension("AUTH"); ok {
			if err := client.Auth(c.pickAuth(mechs)); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := client.Mail(c.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(msg.To); err != nil {
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

// dial устанавливает SMTP-соединение по режиму TLS:
//
//	tls   — зашифровано с первого байта (465);
//	иначе — открытое соединение + STARTTLS, если сервер его анонсирует
//	        (587/2525/25); none — без шифрования (dev/mailpit).
func (c *Client) dial(addr string) (*smtp.Client, error) {
	if c.cfg.TLSMode == "tls" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: c.cfg.Host})
		if err != nil {
			return nil, fmt.Errorf("tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, c.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("smtp client: %w", err)
		}
		return client, nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}
	if c.cfg.TLSMode != "none" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: c.cfg.Host}); err != nil {
				_ = client.Close()
				return nil, fmt.Errorf("starttls: %w", err)
			}
		}
	}
	return client, nil
}

// pickAuth выбирает механизм по списку, анонсированному сервером в EHLO.
// LOGIN предпочтительнее PLAIN: Beget и др. поддерживают только его.
func (c *Client) pickAuth(mechs string) smtp.Auth {
	hasPlain := strings.Contains(mechs, "PLAIN")
	hasLogin := strings.Contains(mechs, "LOGIN")
	if hasLogin && !hasPlain {
		return &loginAuth{user: c.cfg.User, password: c.cfg.Password, host: c.cfg.Host}
	}
	if hasPlain {
		return smtp.PlainAuth("", c.cfg.User, c.cfg.Password, c.cfg.Host)
	}
	if hasLogin {
		return &loginAuth{user: c.cfg.User, password: c.cfg.Password, host: c.cfg.Host}
	}
	// Неизвестный набор — пробуем PLAIN (исторический дефолт).
	return smtp.PlainAuth("", c.cfg.User, c.cfg.Password, c.cfg.Host)
}

// loginAuth — механизм AUTH LOGIN (логин/пароль отдельными base64-шагами).
// net/smtp его не реализует. Креды шлются почти в открытом виде, поэтому,
// как и PlainAuth, требуем TLS-соединение (кроме localhost).
type loginAuth struct {
	user     string
	password string
	host     string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, errors.New("smtp: LOGIN auth requires TLS connection")
	}
	if server.Name != a.host {
		return "", nil, errors.New("smtp: wrong host name")
	}
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimRight(string(fromServer), ": ")) {
	case "username":
		return []byte(a.user), nil
	case "password":
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("smtp: unexpected LOGIN challenge %q", fromServer)
	}
}

func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
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

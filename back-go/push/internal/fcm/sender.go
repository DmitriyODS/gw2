// Package fcm — отправка пуш-уведомлений через FCM HTTP v1 API.
//
// Намеренно без тяжёлого firebase-admin-go: берём OAuth2-токен из
// service-account JSON (golang.org/x/oauth2/google) и шлём обычный REST в
// fcm.googleapis.com. Без ключей сервис стартует, но Sender отключён (no-op)
// — pushsvc крутится, пуши просто не уходят (удобно для dev без Firebase).
package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/DmitriyODS/gw2/back-go/push/internal/domain"
)

const scope = "https://www.googleapis.com/auth/firebase.messaging"

type Sender struct {
	client   *http.Client
	endpoint string
	log      *slog.Logger
}

// New — Sender из service-account JSON. credsJSON пуст → отключённый no-op
// Sender (Enabled()==false), без ошибки.
func New(ctx context.Context, credsJSON []byte, log *slog.Logger) (*Sender, error) {
	if len(bytes.TrimSpace(credsJSON)) == 0 {
		log.Warn("fcm.disabled", "reason", "no credentials")
		return &Sender{log: log}, nil
	}
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, scope)
	if err != nil {
		return nil, fmt.Errorf("fcm: bad credentials: %w", err)
	}
	projectID := creds.ProjectID
	if projectID == "" {
		// project_id отсутствует в ужатом ключе — достаём вручную.
		var meta struct {
			ProjectID string `json:"project_id"`
		}
		_ = json.Unmarshal(credsJSON, &meta)
		projectID = meta.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("fcm: project_id not found in credentials")
	}
	client := oauth2.NewClient(ctx, creds.TokenSource)
	client.Timeout = 10 * time.Second
	log.Info("fcm.enabled", "project_id", projectID)
	return &Sender{
		client:   client,
		endpoint: fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", projectID),
		log:      log,
	}, nil
}

func (s *Sender) Enabled() bool { return s.client != nil }

func (s *Sender) Send(ctx context.Context, token string, n domain.Notification) (bool, error) {
	if !s.Enabled() {
		return false, nil
	}
	body, err := json.Marshal(buildMessage(token, n))
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return false, nil
	}
	raw, _ := io.ReadAll(resp.Body)
	// 404/NOT_FOUND/UNREGISTERED — токен мёртв, его надо удалить.
	invalid := resp.StatusCode == http.StatusNotFound ||
		strings.Contains(string(raw), "UNREGISTERED") ||
		strings.Contains(string(raw), "NOT_FOUND")
	return invalid, fmt.Errorf("fcm: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
}

// buildMessage — FCM v1 message. Звонки (HighPriority) — data-only + high,
// чтобы onMessageReceived вызвался в фоне и поднял полноэкранный экран.
// Остальное — notification + data: систему показывает трей даже при убитом
// приложении, а на переднем плане уведомление строит сам клиент.
func buildMessage(token string, n domain.Notification) map[string]any {
	data := map[string]string{}
	for k, v := range n.Data {
		data[k] = v
	}
	data["channel"] = n.Channel
	// Заголовок/текст всегда и в data — клиент строит уведомление в
	// onMessageReceived единообразно (звонки и сообщения с приложением на экране).
	data["title"] = n.Title
	data["body"] = n.Body

	msg := map[string]any{"token": token, "data": data}
	android := map[string]any{"priority": "high"}

	if !n.HighPriority {
		// Не-звонки: notification-payload, чтобы систему показала трей даже
		// при убитом приложении (тогда onMessageReceived не вызывается).
		msg["notification"] = map[string]any{"title": n.Title, "body": n.Body}
		androidNotif := map[string]any{"channel_id": n.Channel}
		// tag — новое уведомление заменяет прежнее с тем же тегом (группировка
		// по диалогу: сообщения из одного чата схлопываются в одно).
		if n.Tag != "" {
			androidNotif["tag"] = n.Tag
		}
		android["notification"] = androidNotif
	}
	msg["android"] = android
	return map[string]any{"message": msg}
}

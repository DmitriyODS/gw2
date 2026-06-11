// Package livekit — серверная обвязка медиа-сервера LiveKit.
//
// Весь медиа-транспорт (SFU, ICE, reconnect, mute, data-чат) выполняет сам
// LiveKit; здесь: access-токены для комнат (JWT HS256), управление комнатами
// через Twirp REST и верификация подписи вебхуков.
package livekit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

// roomEmptyTimeoutSec — пустая комната живёт не дольше этого: покрывает паузу
// между созданием и подключением инициатора и даёт быстрый room_finished.
const roomEmptyTimeoutSec = 30

const twirpTimeout = 5 * time.Second

type Config struct {
	APIKey    string
	APISecret string
	// APIURL — базовый URL для server-to-server запросов (Twirp).
	APIURL string
	// ClientURL — URL подключения браузера ('/livekit' за nginx или ws://…).
	ClientURL string
	// TokenTTL — должен покрывать самый длинный звонок.
	TokenTTL time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
	log  *slog.Logger
}

var _ domain.MediaServer = (*Client)(nil)

func New(cfg Config, log *slog.Logger) *Client {
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 6 * time.Hour
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: twirpTimeout},
		log:  log,
	}
}

func (c *Client) ClientURL() string { return c.cfg.ClientURL }

// AccessToken — JWT для подключения участника к комнате.
func (c *Client) AccessToken(identity, name, room string, metadata map[string]any) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":  c.cfg.APIKey,
		"sub":  identity,
		"nbf":  now.Add(-10 * time.Second).Unix(),
		"exp":  now.Add(c.cfg.TokenTTL).Unix(),
		"name": name,
		"video": map[string]any{
			"room":                 room,
			"roomJoin":             true,
			"canPublish":           true,
			"canSubscribe":         true,
			"canPublishData":       true,
			"canUpdateOwnMetadata": false,
		},
	}
	if metadata != nil {
		raw, err := json.Marshal(metadata)
		if err != nil {
			return "", err
		}
		claims["metadata"] = string(raw)
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(c.cfg.APISecret))
}

func (c *Client) adminToken(room string) (string, error) {
	now := time.Now()
	video := map[string]any{"roomCreate": true, "roomList": true, "roomAdmin": true}
	if room != "" {
		video["room"] = room
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":   c.cfg.APIKey,
		"sub":   "gw2-callsvc",
		"nbf":   now.Add(-10 * time.Second).Unix(),
		"exp":   now.Add(time.Minute).Unix(),
		"video": video,
	}).SignedString([]byte(c.cfg.APISecret))
}

// twirp — вызов метода RoomService. Ошибки не фатальны: комнаты автосоздаются
// при первом подключении по токену, поэтому недоступность HTTP-API LiveKit не
// должна ронять звонок — логируем и едем дальше.
func (c *Client) twirp(ctx context.Context, method string, payload map[string]any, room string) map[string]any {
	token, err := c.adminToken(room)
	if err != nil {
		c.log.Warn("livekit.admin_token_failed", "error", err)
		return nil
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/twirp/livekit.RoomService/%s", c.cfg.APIURL, method),
		bytes.NewReader(body))
	if err != nil {
		c.log.Warn("livekit.twirp_request_failed", "method", method, "error", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		c.log.Warn("livekit.twirp_unreachable", "method", method, "error", err)
		return nil
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		c.log.Warn("livekit.twirp_error", "method", method,
			"status", resp.StatusCode, "body", truncate(string(data), 300))
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

// CreateRoom — заранее, ради лимита участников и empty_timeout.
func (c *Client) CreateRoom(ctx context.Context, name string, maxParticipants int) {
	c.twirp(ctx, "CreateRoom", map[string]any{
		"name":             name,
		"empty_timeout":    roomEmptyTimeoutSec,
		"max_participants": maxParticipants,
	}, "")
}

// DeleteRoom — завершить комнату для всех; LiveKit пришлёт room_finished.
func (c *Client) DeleteRoom(ctx context.Context, name string) {
	c.twirp(ctx, "DeleteRoom", map[string]any{"room": name}, name)
}

// ListParticipantIdentities — identity всех в комнате; ok=false, если LiveKit
// недоступен (или комнаты уже нет — Twirp вернёт not_found).
func (c *Client) ListParticipantIdentities(ctx context.Context, room string) ([]string, bool) {
	data := c.twirp(ctx, "ListParticipants", map[string]any{"room": room}, room)
	if data == nil {
		return nil, false
	}
	raw, _ := data["participants"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		p, _ := item.(map[string]any)
		if identity, _ := p["identity"].(string); identity != "" {
			out = append(out, identity)
		}
	}
	return out, true
}

// VerifyWebhook — проверить подпись вебхука LiveKit и вернуть распарсенное
// событие. LiveKit кладёт в Authorization JWT, подписанный API-secret, с
// claim `sha256` — хэшем тела запроса (base64; hex принимаем на всякий случай).
func (c *Client) VerifyWebhook(body []byte, authHeader string) (map[string]any, error) {
	if authHeader == "" {
		return nil, fmt.Errorf("empty Authorization header")
	}
	raw := authHeader
	if len(raw) > 7 && raw[:7] == "Bearer " {
		raw = raw[7:]
	}
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(c.cfg.APISecret), nil
	}, jwt.WithIssuer(c.cfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("bad webhook token: %w", err)
	}
	digest := sha256.Sum256(body)
	got, _ := claims["sha256"].(string)
	if got != base64.StdEncoding.EncodeToString(digest[:]) &&
		got != hex.EncodeToString(digest[:]) {
		return nil, fmt.Errorf("webhook body digest mismatch")
	}
	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("bad webhook body: %w", err)
	}
	return event, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

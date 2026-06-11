package livekit

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testKey    = "devkey"
	testSecret = "dev_livekit_secret_min_32_chars_ok"
)

func newTestClient() *Client {
	return New(Config{
		APIKey: testKey, APISecret: testSecret,
		APIURL: "http://localhost:7880", ClientURL: "/livekit",
	}, slog.Default())
}

func decodeToken(t *testing.T, token string) jwt.MapClaims {
	t.Helper()
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	if err != nil {
		t.Fatalf("токен не парсится: %v", err)
	}
	return claims
}

func TestAccessTokenGrants(t *testing.T) {
	c := newTestClient()
	token, err := c.AccessToken("u42", "Иванов Иван", "call-7",
		map[string]any{"user_id": 42, "avatar_path": nil})
	if err != nil {
		t.Fatal(err)
	}
	claims := decodeToken(t, token)

	if claims["iss"] != testKey || claims["sub"] != "u42" || claims["name"] != "Иванов Иван" {
		t.Errorf("базовые клеймы неверны: %v", claims)
	}
	video, _ := claims["video"].(map[string]any)
	if video["room"] != "call-7" || video["roomJoin"] != true ||
		video["canPublish"] != true || video["canPublishData"] != true {
		t.Errorf("video-грант неверен: %v", video)
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(claims["metadata"].(string)), &meta); err != nil {
		t.Fatalf("metadata не JSON: %v", err)
	}
	if meta["user_id"] != float64(42) {
		t.Errorf("metadata.user_id = %v", meta["user_id"])
	}
}

func signWebhook(t *testing.T, body []byte, digest string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":    testKey,
		"exp":    time.Now().Add(time.Minute).Unix(),
		"sha256": digest,
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestVerifyWebhookOK(t *testing.T) {
	c := newTestClient()
	body := []byte(`{"event":"room_finished","room":{"name":"call-5"}}`)
	digest := sha256.Sum256(body)
	auth := "Bearer " + signWebhook(t, body, base64.StdEncoding.EncodeToString(digest[:]))

	event, err := c.VerifyWebhook(body, auth)
	if err != nil {
		t.Fatalf("валидный вебхук отвергнут: %v", err)
	}
	if event["event"] != "room_finished" {
		t.Errorf("событие не распарсилось: %v", event)
	}
}

func TestVerifyWebhookBadDigest(t *testing.T) {
	c := newTestClient()
	body := []byte(`{"event":"room_finished"}`)
	other := sha256.Sum256([]byte("другое тело"))
	auth := signWebhook(t, body, base64.StdEncoding.EncodeToString(other[:]))

	if _, err := c.VerifyWebhook(body, auth); err == nil {
		t.Error("вебхук с чужим дайджестом должен быть отвергнут")
	}
}

func TestVerifyWebhookBadSignature(t *testing.T) {
	c := newTestClient()
	body := []byte(`{"event":"x"}`)
	digest := sha256.Sum256(body)
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":    testKey,
		"sha256": base64.StdEncoding.EncodeToString(digest[:]),
	}).SignedString([]byte("другой-секрет-другой-секрет-1234"))

	if _, err := c.VerifyWebhook(body, token); err == nil {
		t.Error("вебхук с неверной подписью должен быть отвергнут")
	}
	if _, err := c.VerifyWebhook(body, ""); err == nil {
		t.Error("пустой Authorization должен быть отвергнут")
	}
}

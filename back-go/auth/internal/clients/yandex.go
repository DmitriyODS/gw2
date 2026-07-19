package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// Yandex — OAuth-клиент Яндекс ID (вход/регистрация через Яндекс):
// обмен кода на токен (oauth.yandex.ru) и чтение профиля (login.yandex.ru).
type Yandex struct {
	clientID     string
	clientSecret string
	http         *http.Client
	tokenURL     string
	profileURL   string
}

func NewYandex(clientID, clientSecret string) *Yandex {
	return &Yandex{
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         &http.Client{Timeout: 10 * time.Second},
		tokenURL:     "https://oauth.yandex.ru/token",
		profileURL:   "https://login.yandex.ru/info?format=json",
	}
}

var _ domain.YandexOAuthClient = (*Yandex)(nil)

func (y *Yandex) Exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {y.clientID},
		"client_secret": {y.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, y.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := y.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("yandex token: status %d", resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("yandex token: пустой access_token")
	}
	return out.AccessToken, nil
}

func (y *Yandex) Profile(ctx context.Context, accessToken string) (*domain.YandexProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, y.profileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)
	resp, err := y.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yandex profile: status %d", resp.StatusCode)
	}
	var out struct {
		ID            string `json:"id"`
		DefaultEmail  string `json:"default_email"`
		RealName      string `json:"real_name"`
		DisplayName   string `json:"display_name"`
		Login         string `json:"login"`
		AvatarID      string `json:"default_avatar_id"`
		IsAvatarEmpty bool   `json:"is_avatar_empty"`
		DefaultPhone  struct {
			Number string `json:"number"`
		} `json:"default_phone"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	name := out.RealName
	if name == "" {
		name = out.DisplayName
	}
	if name == "" {
		name = out.Login
	}
	p := &domain.YandexProfile{
		ID: out.ID, Email: out.DefaultEmail, Name: name,
		Phone: out.DefaultPhone.Number,
	}
	if !out.IsAvatarEmpty {
		p.AvatarID = out.AvatarID
	}
	return p, nil
}

// FetchAvatar — картинка профиля Яндекса (публичный CDN, токен не нужен).
// islands-200 — квадрат 200×200, дальше её обычным путём кладёт UploadAvatar.
func (y *Yandex) FetchAvatar(ctx context.Context, avatarID string) ([]byte, error) {
	url := "https://avatars.yandex.net/get-yapic/" + avatarID + "/islands-200"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := y.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yandex avatar: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
}

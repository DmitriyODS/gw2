package domain

import (
	"context"
	"time"
)

// OAuthCode — одноразовый код согласия OAuth 2.0 (authorization code grant,
// связка аккаунтов навыка Алисы): кому и с какой активной компанией выдать
// пару токенов на token-эндпоинте. Живёт в Redis с коротким TTL.
type OAuthCode struct {
	UserID    int64  `json:"user_id"`
	CompanyID *int64 `json:"company_id"`
}

// OAuthCodeStore — хранилище одноразовых кодов: Pop атомарно забирает и
// удаляет (повторное использование кода невозможно).
type OAuthCodeStore interface {
	Save(ctx context.Context, code string, oc OAuthCode, ttl time.Duration) error
	Pop(ctx context.Context, code string) (*OAuthCode, error)
}

// YandexProfile — профиль пользователя Яндекс ID (login.yandex.ru/info).
type YandexProfile struct {
	ID       string // постоянный id аккаунта Яндекса (users.yandex_id)
	Email    string // default_email; может быть пустым (нужен scope login:email)
	Name     string // real_name либо display_name (scope login:info)
	Phone    string // default_phone.number; пусто без scope login:default_phone
	AvatarID string // default_avatar_id; пусто, если is_avatar_empty (scope login:avatar)
}

// YandexOAuthClient — обратная сторона OAuth: мы — клиент Яндекс ID
// (вход/регистрация через Яндекс). Exchange меняет код авторизации на
// OAuth-токен Яндекса, Profile читает профиль по токену, FetchAvatar
// скачивает картинку профиля (avatars.yandex.net).
type YandexOAuthClient interface {
	Exchange(ctx context.Context, code string) (string, error)
	Profile(ctx context.Context, accessToken string) (*YandexProfile, error)
	FetchAvatar(ctx context.Context, avatarID string) ([]byte, error)
}

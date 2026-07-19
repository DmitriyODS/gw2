package dto

// OAuthAuthorizeRequest — тело POST /api/auth/oauth/authorize (страница
// согласия фронта): параметры, с которыми Яндекс привёл пользователя.
type OAuthAuthorizeRequest struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
	Scope       string `json:"scope"`
}

// OAuthTokenRequest — разобранная форма POST /api/auth/oauth/token
// (application/x-www-form-urlencoded, его зовёт Яндекс).
type OAuthTokenRequest struct {
	GrantType    string
	Code         string
	RefreshToken string
	ClientID     string
	ClientSecret string
}

// YandexAuthConfig — публичная конфигурация кнопки «Войти с Яндексом».
type YandexAuthConfig struct {
	Enabled  bool   `json:"enabled"`
	ClientID string `json:"client_id"`
}

// OAuthTokens — ответ token-эндпоинта (RFC 6749).
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

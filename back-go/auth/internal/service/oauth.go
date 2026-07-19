// OAuth 2.0-провайдер (authorization code grant) для связки аккаунтов навыка
// Алисы (Яндекс.Диалоги). Мы — сервер авторизации: страница согласия фронта
// выпускает одноразовый код (Redis, TTL), Яндекс меняет его на пару
// access+refresh на token-эндпоинте и дальше сам обновляет access по refresh
// (rotation: каждый refresh-грант выдаёт новую пару). Клеймы токенов — обычная
// сессия (активная компания на момент согласия), поэтому alicesvc проверяет их
// штатным pkg/pasetoauth.
package service

import (
	"context"
	"crypto/subtle"
	"net/url"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

const (
	oauthCodeTTL = 10 * time.Minute
	// yandexRedirectPrefix — единственный допустимый redirect_uri связки
	// аккаунтов Яндекс.Диалогов.
	yandexRedirectPrefix = "https://social.yandex.net/broker/redirect"
)

// Ошибки token-эндпоинта — с кодами RFC 6749 (их читает Яндекс).
var (
	errOAuthDisabled  = domain.NewError("OAUTH_DISABLED", "Связка аккаунтов не настроена на сервере", 403)
	errOAuthClient    = domain.NewError("invalid_client", "Неверный client_id или client_secret", 401)
	errOAuthGrant     = domain.NewError("invalid_grant", "Код или refresh-токен недействителен", 400)
	errOAuthGrantType = domain.NewError("unsupported_grant_type", "Поддерживаются authorization_code и refresh_token", 400)
)

// WithOAuth — включить OAuth-провайдер (вызывается из main; без него оба
// эндпоинта отвечают OAUTH_DISABLED).
func (s *Service) WithOAuth(codes domain.OAuthCodeStore, clientID, clientSecret string) *Service {
	s.oauthCodes, s.oauthClientID, s.oauthClientSecret = codes, clientID, clientSecret
	return s
}

func (s *Service) oauthEnabled() bool {
	return s.oauthCodes != nil && s.oauthClientID != "" && s.oauthClientSecret != ""
}

// OAuthAuthorize — согласие авторизованного пользователя: одноразовый код с
// его id и активной компанией сессии → URL возврата к Яндексу.
func (s *Service) OAuthAuthorize(ctx context.Context, userID int64, companyID *int64, req dto.OAuthAuthorizeRequest) (string, error) {
	if !s.oauthEnabled() {
		return "", errOAuthDisabled
	}
	if req.ClientID != s.oauthClientID {
		return "", domain.NewError("INVALID_CLIENT", "Неизвестный client_id", 403)
	}
	if !strings.HasPrefix(req.RedirectURI, yandexRedirectPrefix) {
		return "", domain.NewError("INVALID_REDIRECT_URI", "Недопустимый redirect_uri", 403)
	}
	code, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.oauthCodes.Save(ctx, code, domain.OAuthCode{UserID: userID, CompanyID: companyID}, oauthCodeTTL); err != nil {
		return "", err
	}
	// code/state/client_id/scope возвращаются как пришли (требование Диалогов).
	q := url.Values{"code": {code}, "state": {req.State}, "client_id": {req.ClientID}}
	if req.Scope != "" {
		q.Set("scope", req.Scope)
	}
	sep := "?"
	if strings.Contains(req.RedirectURI, "?") {
		sep = "&"
	}
	return req.RedirectURI + sep + q.Encode(), nil
}

// OAuthToken — token-эндпоинт: обмен кода на пару токенов и обновление по
// refresh. Выдаёт обычную сессию (без cookie) — access живёт accessTTL,
// Яндекс обновляет его сам по expires_in.
func (s *Service) OAuthToken(ctx context.Context, req dto.OAuthTokenRequest) (*dto.OAuthTokens, error) {
	if !s.oauthEnabled() {
		return nil, errOAuthClient
	}
	if subtle.ConstantTimeCompare([]byte(req.ClientID), []byte(s.oauthClientID)) != 1 ||
		subtle.ConstantTimeCompare([]byte(req.ClientSecret), []byte(s.oauthClientSecret)) != 1 {
		return nil, errOAuthClient
	}

	var userID int64
	var companyID *int64
	switch req.GrantType {
	case "authorization_code":
		oc, err := s.oauthCodes.Pop(ctx, req.Code)
		if err != nil {
			return nil, err
		}
		if oc == nil {
			return nil, errOAuthGrant
		}
		userID, companyID = oc.UserID, oc.CompanyID
	case "refresh_token":
		uid, cid, err := s.tokens.ParseRefresh(req.RefreshToken)
		if err != nil {
			return nil, errOAuthGrant
		}
		userID, companyID = uid, cid
	default:
		return nil, errOAuthGrantType
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errOAuthGrant
	}
	// Активной компании могло не стать (вышел/исключён/компания отключена) —
	// выдаём сессию без компании, а не роняем связку (как Refresh).
	if user.IsSuperAdmin {
		companyID = nil
	} else if companyID != nil {
		m, err := s.repo.GetMembership(ctx, userID, *companyID)
		if err != nil {
			return nil, err
		}
		if m == nil || m.Company == nil || !m.Company.IsActive {
			companyID = nil
		}
	}
	sess, err := s.session(ctx, user, companyID, true)
	if err != nil {
		return nil, err
	}
	return &dto.OAuthTokens{
		AccessToken:  sess.AccessToken,
		TokenType:    "bearer",
		ExpiresIn:    int(s.tokens.AccessTTL().Seconds()),
		RefreshToken: sess.RefreshToken,
	}, nil
}

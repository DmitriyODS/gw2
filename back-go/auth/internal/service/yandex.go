// Вход и регистрация через Яндекс ID: мы — OAuth-клиент Яндекса.
// Фронт ведёт пользователя на oauth.yandex.ru/authorize, Яндекс возвращает
// его на /yandex-callback?code=…, фронт шлёт код сюда. Матчинг аккаунта:
// сначала по users.yandex_id, затем по подтверждённому email Яндекса
// (привязываем yandex_id к существующему аккаунту), иначе — автосоздание
// (email считается подтверждённым Яндексом, письмо-верификация не нужна).
package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

var errYandexDisabled = domain.NewError("YANDEX_DISABLED", "Вход через Яндекс не настроен на сервере", 403)

// WithYandex — включить вход через Яндекс ID (вызывается из main).
func (s *Service) WithYandex(client domain.YandexOAuthClient, clientID string) *Service {
	s.yandex, s.yandexClientID = client, clientID
	return s
}

// YandexAuthConfig — публичная конфигурация для кнопки «Войти с Яндексом».
func (s *Service) YandexAuthConfig() *dto.YandexAuthConfig {
	return &dto.YandexAuthConfig{
		Enabled:  s.yandex != nil && s.yandexClientID != "",
		ClientID: s.yandexClientID,
	}
}

func (s *Service) YandexLogin(ctx context.Context, code string) (*dto.Session, error) {
	if s.yandex == nil || s.yandexClientID == "" {
		return nil, errYandexDisabled
	}
	token, err := s.yandex.Exchange(ctx, code)
	if err != nil {
		return nil, domain.NewError("YANDEX_CODE_INVALID", "Не удалось подтвердить вход через Яндекс", 401)
	}
	profile, err := s.yandex.Profile(ctx, token)
	if err != nil || profile == nil || profile.ID == "" {
		return nil, domain.NewError("YANDEX_PROFILE_FAILED", "Яндекс не отдал профиль пользователя", 502)
	}

	user, err := s.repo.GetByYandexID(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	if user == nil && profile.Email != "" {
		// Существующий аккаунт с этой почтой — привязываем Яндекс к нему.
		if user, err = s.repo.GetByEmail(ctx, profile.Email); err != nil {
			return nil, err
		}
		if user != nil {
			fields := map[string]any{"yandex_id": profile.ID}
			// Почта подтверждена Яндексом — гейт EMAIL_NOT_VERIFIED снимается.
			if !user.EmailVerified {
				fields["email_verified"] = true
			}
			if err := s.repo.UpdateFields(ctx, user.ID, fields); err != nil {
				return nil, err
			}
		}
	}
	if user == nil {
		if user, err = s.registerFromYandex(ctx, profile); err != nil {
			return nil, err
		}
	}
	if !user.IsActive {
		return nil, domain.NewError("USER_DISABLED", "Аккаунт отключён", 403)
	}
	s.log.Info("auth.yandex_login", "user_id", user.ID)
	return s.startSession(ctx, user)
}

// YandexLinkStatus — привязан ли Яндекс ID к аккаунту (карточка профиля).
func (s *Service) YandexLinkStatus(ctx context.Context, userID int64) (bool, error) {
	return s.repo.YandexLinked(ctx, userID)
}

// YandexLink — привязать Яндекс ID к СУЩЕСТВУЮЩЕМУ аккаунту (из профиля,
// state=link): дальше пользователь входит кнопкой, а не создаёт дубликат.
func (s *Service) YandexLink(ctx context.Context, userID int64, code string) error {
	if s.yandex == nil || s.yandexClientID == "" {
		return errYandexDisabled
	}
	token, err := s.yandex.Exchange(ctx, code)
	if err != nil {
		return domain.NewError("YANDEX_CODE_INVALID", "Не удалось подтвердить вход через Яндекс", 401)
	}
	profile, err := s.yandex.Profile(ctx, token)
	if err != nil || profile == nil || profile.ID == "" {
		return domain.NewError("YANDEX_PROFILE_FAILED", "Яндекс не отдал профиль пользователя", 502)
	}
	existing, err := s.repo.GetByYandexID(ctx, profile.ID)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != userID {
		return domain.NewError("YANDEX_TAKEN",
			"Этот Яндекс-аккаунт уже привязан к другому пользователю", 409)
	}
	if existing != nil {
		return nil // уже привязан к этому же аккаунту
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{"yandex_id": profile.ID}); err != nil {
		return err
	}
	s.log.Info("auth.yandex_link", "user_id", userID)
	return nil
}

// YandexUnlink — отвязать Яндекс ID (вход остаётся по логину/паролю).
func (s *Service) YandexUnlink(ctx context.Context, userID int64) error {
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{"yandex_id": nil}); err != nil {
		return err
	}
	s.log.Info("auth.yandex_unlink", "user_id", userID)
	return nil
}

// registerFromYandex — автосоздание аккаунта из профиля Яндекса: логин из
// имени (транслит), случайный пароль (вход — через Яндекс либо сброс по почте).
func (s *Service) registerFromYandex(ctx context.Context, p *domain.YandexProfile) (*domain.User, error) {
	fio := strings.TrimSpace(p.Name)
	if fio == "" {
		fio = "Пользователь Яндекса"
	}
	login, err := s.SuggestLogin(ctx, fio)
	if err != nil {
		return nil, err
	}
	randomPass, err := randomToken()
	if err != nil {
		return nil, err
	}
	hashed, err := s.repo.HashPassword(ctx, randomPass)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		FIO: fio, Login: login, HashPassword: hashed,
		IsDefaultPass: false, EmailVerified: true,
	}
	if p.Email != "" {
		email := strings.ToLower(strings.TrimSpace(p.Email))
		user.Email = &email
	}
	if phone := strings.TrimSpace(p.Phone); phone != "" {
		user.Phone = &phone
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFields(ctx, user.ID, map[string]any{"yandex_id": p.ID}); err != nil {
		return nil, err
	}
	// Картинка профиля Яндекса — аватаркой нового аккаунта. Не фатально:
	// без неё останется identicon.
	if p.AvatarID != "" {
		if raw, err := s.yandex.FetchAvatar(ctx, p.AvatarID); err != nil {
			s.log.Warn("auth.yandex_avatar_fetch_failed", "user_id", user.ID, "error", err)
		} else if _, err := s.UploadAvatar(ctx, user.ID, raw); err != nil {
			s.log.Warn("auth.yandex_avatar_save_failed", "user_id", user.ID, "error", err)
		}
	}
	s.log.Info("auth.yandex_register", "user_id", user.ID)
	return user, nil
}

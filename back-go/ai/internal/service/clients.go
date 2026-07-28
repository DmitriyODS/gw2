package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// Доступ к моделям в новой схеме: ключ ОДИН, платформенный (его задаёт
// супер-админ в «Аудите платформы»), а тариф выдаёт пользователю ТОКЕНЫ
// ДОСТУПА. Личный ключ — необязательная настройка: с ним запросы уходят на
// свой сервер и токены тарифа не тратятся.
//
// Кто платит:
//   - личные функции (чат ассистента, ИИ в заметках) — сам пользователь;
//   - компанийные (умный поиск задач, факты ТВ-режима) — СОЗДАТЕЛЬ компании,
//     и только если он разрешил тратить свои токены (companies.ai_shared).

// platformAI — платформенные настройки с коротким кэшем: их читает каждый
// запрос к модели, а меняются они редко.
func (s *Service) platformAI(ctx context.Context) (*domain.PlatformAI, error) {
	s.mu.Lock()
	if s.platform != nil && s.platformAt.After(time.Now()) {
		p := s.platform
		s.mu.Unlock()
		return p, nil
	}
	s.mu.Unlock()

	p, err := s.repo.GetPlatformAI(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.platform = p
	s.platformAt = time.Now().Add(cacheTTL)
	s.mu.Unlock()
	return p, nil
}

// catalog — каталог моделей с ценами (по ним считается списание).
func (s *Service) catalog(ctx context.Context) (*domain.ModelCatalog, error) {
	s.mu.Lock()
	if s.models != nil && s.modelsAt.After(time.Now()) {
		c := s.models
		s.mu.Unlock()
		return c, nil
	}
	s.mu.Unlock()

	list, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	c := domain.NewModelCatalog(list)
	s.mu.Lock()
	s.models = c
	s.modelsAt = time.Now().Add(cacheTTL)
	s.mu.Unlock()
	return c, nil
}

// invalidatePlatform — сбросить кэш платформенных настроек и каталога.
func (s *Service) invalidatePlatform() {
	s.mu.Lock()
	s.platform, s.models = nil, nil
	s.mu.Unlock()
}

// platformClient — доступ на платформенном ключе. nil без ошибки — ИИ на
// платформе не настроен (ключа нет или он выключен).
func (s *Service) platformClient(ctx context.Context) (*aiClient, error) {
	p, err := s.platformAI(ctx)
	if err != nil {
		return nil, err
	}
	if !p.Ready() {
		return nil, nil
	}
	apiKey, ok := s.cipher.Decrypt(p.APIKeyEnc)
	if !ok {
		s.log.Warn("ai.platform_decrypt_failed")
		return nil, nil
	}
	return &aiClient{
		apiKey:         apiKey,
		baseURL:        p.BaseURL,
		modelChat:      firstNonEmpty(p.ModelChat, domain.PlatformModelChat),
		modelEmbedding: firstNonEmpty(p.ModelEmbedding, domain.PlatformModelEmbedding),
		modelSupport:   firstNonEmpty(p.ModelSupport, p.ModelChat, domain.PlatformModelChat),
	}, nil
}

// clientForCompany — доступ для КОМПАНИЙНЫХ функций (умный поиск, ТВ-факты).
// nil — возможность выключена, создатель не разрешил тратить токены или
// платформенный ИИ не настроен. feature пуст — проверяется только общий
// тумблер компании.
func (s *Service) clientForCompany(ctx context.Context, companyID int64, feature string) (*aiClient, error) {
	if companyID <= 0 {
		return nil, nil
	}
	company, err := s.repo.GetCompanyAI(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil || !company.AllowsFeature(feature) {
		return nil, nil
	}
	client, err := s.platformClient(ctx)
	if err != nil || client == nil {
		return nil, err
	}
	// Платит создатель компании: его тариф и его баланс токенов.
	c := *client
	c.companyID = companyID
	c.payerID = company.OwnerID
	return &c, nil
}

// clientFor — обратная совместимость с прежним компанийным доступом: те же
// правила, без проверки конкретной возможности.
func (s *Service) clientFor(ctx context.Context, companyID int64) (*aiClient, error) {
	return s.clientForCompany(ctx, companyID, "")
}

// userClientFor — доступ ЛИЧНЫХ функций пользователя. Свой ключ имеет
// приоритет (тогда токены тарифа не тратятся), иначе платформенный ключ и
// выбранная человеком модель. nil — ИИ у пользователя выключен или платформа
// не настроена.
func (s *Service) userClientFor(ctx context.Context, userID int64) (*aiClient, error) {
	return s.userClientForFeature(ctx, userID, "")
}

func (s *Service) userClientForFeature(ctx context.Context, userID int64, feature string) (*aiClient, error) {
	settings, err := s.repo.GetUserAI(ctx, userID)
	if err != nil {
		return nil, err
	}
	if settings == nil || !settings.Enabled {
		return nil, nil
	}
	if feature != "" && !settings.AllowsFeature(feature) {
		return nil, nil
	}

	if settings.OwnKey() {
		apiKey, ok := s.cipher.Decrypt(settings.APIKeyEnc)
		if !ok {
			s.log.Warn("ai.user_decrypt_failed", "user_id", userID)
			return nil, nil
		}
		return &aiClient{
			apiKey:    apiKey,
			baseURL:   settings.APIBaseURL,
			modelChat: settings.ChatModel(),
			ownKey:    true,
		}, nil
	}

	client, err := s.platformClient(ctx)
	if err != nil || client == nil {
		return nil, err
	}
	c := *client
	c.payerID = userID
	// Модель выбирает пользователь — но только из включённого каталога.
	if cat, err := s.catalog(ctx); err == nil && cat.Allowed(settings.ChatModel()) {
		c.modelChat = settings.ChatModel()
	}
	return &c, nil
}

// supportClient — ИИ техподдержки: платформенный ключ и своя модель. Токены
// пользователей он не тратит — это расход самой платформы.
func (s *Service) supportClient(ctx context.Context) (*aiClient, error) {
	client, err := s.platformClient(ctx)
	if err != nil || client == nil {
		return nil, err
	}
	c := *client
	c.modelChat = c.modelSupport
	return &c, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

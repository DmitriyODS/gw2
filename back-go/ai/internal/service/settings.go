package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
)

// ИИ-возможности КОМПАНИИ. Своего ключа у компании нет: умный поиск задач и
// факты ТВ-режима работают на платформенном ключе и тратят токены СОЗДАТЕЛЯ
// компании — поэтому у него есть тумблер «разрешить» (shared), а у каждой
// возможности свой выключатель.

// resolveCompany — компания (404) + проверка доступа: ИИ-настройками управляет
// администратор ИМЕННО этой компании (членство user_companies с ролью ≥ 3) или
// супер-админ платформы. Доступ скоупится компанией из пути, а НЕ активной
// компанией сессии — раздел «Компании» правит любую свою компанию независимо от
// того, какая активна.
func (s *Service) resolveCompany(ctx context.Context, actor *domain.User, companyID int64) (*domain.CompanyAI, error) {
	company, err := s.repo.GetCompanyAI(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errNotFound()
	}
	if actor != nil && actor.IsSuperAdmin {
		return company, nil
	}
	if actor != nil {
		level, err := s.repo.MembershipLevel(ctx, actor.ID, company.ID)
		if err != nil {
			return nil, err
		}
		if level >= domain.LevelAdmin {
			return company, nil
		}
	}
	return nil, errNoAccess()
}

func (s *Service) dumpSettings(ctx context.Context, c *domain.CompanyAI) *dto.AiSettings {
	out := &dto.AiSettings{
		Enabled:         c.Enabled,
		Shared:          c.Shared,
		FeatSearch:      c.FeatSearch,
		FeatTVFact:      c.FeatTVFact,
		OwnerTokensLeft: -1,
	}
	if p, err := s.platformAI(ctx); err == nil {
		out.PlatformReady = p.Ready()
		out.ModelChat = firstNonEmpty(p.ModelChat, domain.PlatformModelChat)
		out.ModelEmbedding = firstNonEmpty(p.ModelEmbedding, domain.PlatformModelEmbedding)
	}
	// Остаток токенов владельца — чтобы администратор видел, на чей счёт
	// работает ИИ компании и не кончился ли он.
	if s.meter != nil && c.OwnerID > 0 {
		if _, left, ok := s.meter.Check(ctx, c.OwnerID, 0); ok {
			out.OwnerTokensLeft = left
		}
	}
	return out
}

func (s *Service) GetSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.AiSettings, error) {
	company, err := s.resolveCompany(ctx, actor, companyID)
	if err != nil {
		return nil, err
	}
	return s.dumpSettings(ctx, company), nil
}

func (s *Service) UpdateSettings(ctx context.Context, actor *domain.User, companyID int64, upd dto.AiSettingsUpdate) (*dto.AiSettings, error) {
	company, err := s.resolveCompany(ctx, actor, companyID)
	if err != nil {
		return nil, err
	}
	if upd.Enabled != nil {
		company.Enabled = *upd.Enabled
	}
	// Тратить свои токены на компанию разрешает только её СОЗДАТЕЛЬ: это его
	// деньги, и решать за него не может ни другой администратор, ни супер-админ.
	if upd.Shared != nil {
		if actor == nil || actor.ID != company.OwnerID {
			return nil, domain.NewError("OWNER_ONLY",
				"Разрешить тратить токены на компанию может только её создатель", 403)
		}
		company.Shared = *upd.Shared
	}
	if upd.FeatSearch != nil {
		company.FeatSearch = *upd.FeatSearch
	}
	if upd.FeatTVFact != nil {
		company.FeatTVFact = *upd.FeatTVFact
	}

	if err := s.repo.UpdateCompanyAI(ctx, company); err != nil {
		return nil, err
	}
	s.invalidateClient(company.ID)
	return s.dumpSettings(ctx, company), nil
}

// TestSettings — реальная проверка связи: один tiny-chat + один embedding на
// том доступе, которым пользуется компания. Ничего не сохраняет.
func (s *Service) TestSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.AiTestResult, error) {
	company, err := s.resolveCompany(ctx, actor, companyID)
	if err != nil {
		return nil, err
	}
	client, err := s.clientFor(ctx, company.ID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(409)
	}
	return s.pingClient(ctx, client), nil
}

// pingClient — общая проверка связи: chat + embedding, ошибки уходят флагами
// в результат, а не роняют запрос.
func (s *Service) pingClient(ctx context.Context, client *aiClient) *dto.AiTestResult {
	result := &dto.AiTestResult{}
	setErr := func(text string) {
		prev := ""
		if result.Error != nil {
			prev = *result.Error
		}
		combined := prev + text
		result.Error = &combined
	}

	t0 := time.Now()
	_, chatErr := s.llm.ChatOnce(ctx, domain.ChatParams{
		APIKey:       client.apiKey,
		BaseURL:      client.baseURL,
		Model:        client.modelChat,
		MessagesJSON: `[{"role":"user","content":"ping"}]`,
		MaxTokens:    2,
		Temperature:  0,
		Timeout:      10 * time.Second,
	})
	if chatErr != nil {
		setErr("chat: " + chatErr.Error())
	} else {
		result.Chat = true
	}
	if client.modelEmbedding != "" {
		_, _, embErr := s.llm.Embed(ctx, domain.EmbedParams{
			APIKey: client.apiKey, BaseURL: client.baseURL,
			Model: client.modelEmbedding, Texts: []string{"ping"}, Timeout: 10 * time.Second,
		})
		if embErr != nil {
			setErr(" embedding: " + embErr.Error())
		} else {
			result.Embedding = true
		}
	}
	result.LatencyMS = time.Since(t0).Milliseconds()
	return result
}

func (s *Service) IndexingStatus(ctx context.Context, actor *domain.User, companyID int64) (*dto.IndexingStatus, error) {
	company, err := s.resolveCompany(ctx, actor, companyID)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountTasks(ctx, company.ID)
	if err != nil {
		return nil, err
	}
	model := domain.PlatformModelEmbedding
	if p, err := s.platformAI(ctx); err == nil {
		model = firstNonEmpty(p.ModelEmbedding, model)
	}
	indexed, err := s.repo.CountEmbeddings(ctx, company.ID, model)
	if err != nil {
		return nil, err
	}
	// pending считаем только при включённом ИИ компании — иначе индексировать
	// нечего и показывать очередь незачем.
	pending := 0
	client, err := s.clientForCompany(ctx, company.ID, domain.FeatureSearch)
	if err != nil {
		return nil, err
	}
	if client != nil {
		ids, err := s.repo.FindUnindexedTaskIDs(ctx, company.ID, model)
		if err != nil {
			return nil, err
		}
		pending = len(ids)
	}
	return &dto.IndexingStatus{
		TotalTasks: total,
		Indexed:    indexed,
		Pending:    pending,
		Model:      model,
		AiEnabled:  client != nil,
	}, nil
}

// StartReindex — 202 Accepted: бэкфилл уходит в фон, реальный прогресс —
// через IndexingStatus. Повторный запрос при идущем бэкфилле новый не
// запускает (атомарный флаг), но отвечает той же формой.
func (s *Service) StartReindex(ctx context.Context, actor *domain.User, companyID int64) (*dto.ReindexQueued, error) {
	company, err := s.resolveCompany(ctx, actor, companyID)
	if err != nil {
		return nil, err
	}
	client, err := s.clientForCompany(ctx, company.ID, domain.FeatureSearch)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(409)
	}

	ids, err := s.repo.FindUnindexedTaskIDs(ctx, company.ID, client.modelEmbedding)
	if err != nil {
		return nil, err
	}
	cid := company.ID
	if _, running := s.backfills.LoadOrStore(cid, struct{}{}); !running {
		go func() {
			defer s.backfills.Delete(cid)
			s.runBackfill(context.Background(), cid)
		}()
	} else {
		s.log.Info("ai.reindex.already_running", "company_id", cid)
	}
	return &dto.ReindexQueued{Queued: true, Pending: len(ids)}, nil
}

package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/secret"
)

// resolveCompany — компания (404) + проверка доступа: AI-настройками управляет
// администратор ИМЕННО этой компании, т.е. компания из пути должна совпасть с
// активной компанией сессии. Супер-админ компанийные AI-настройки не управляет.
// Уровень роли (Администратор, ≥ 3) проверяет транспорт.
func (s *Service) resolveCompany(ctx context.Context, actor *domain.User, companyID int64) (*domain.CompanyAI, error) {
	company, err := s.repo.GetCompanyAI(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errNotFound()
	}
	if actor.CompanyID != nil && *actor.CompanyID == company.ID {
		return company, nil
	}
	return nil, errNoAccess()
}

func dumpSettings(c *domain.CompanyAI) *dto.AiSettings {
	return &dto.AiSettings{
		Enabled:        c.Enabled,
		KeyHint:        c.KeyHint,
		HasKey:         len(c.APIKeyEnc) > 0,
		ModelChat:      c.ModelChat,
		ModelEmbedding: c.ModelEmbedding,
	}
}

func (s *Service) GetSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.AiSettings, error) {
	company, err := s.resolveCompany(ctx, actor, companyID)
	if err != nil {
		return nil, err
	}
	return dumpSettings(company), nil
}

func (s *Service) UpdateSettings(ctx context.Context, actor *domain.User, companyID int64, upd dto.AiSettingsUpdate) (*dto.AiSettings, error) {
	company, err := s.resolveCompany(ctx, actor, companyID)
	if err != nil {
		return nil, err
	}
	if upd.Enabled != nil {
		company.Enabled = *upd.Enabled
	}
	if upd.ModelChat != nil {
		company.ModelChat = strings.TrimSpace(*upd.ModelChat)
	}
	if upd.ModelEmbedding != nil {
		company.ModelEmbedding = strings.TrimSpace(*upd.ModelEmbedding)
	}

	// api_key: None / "" → не менять; clear_key=true → стереть; иначе зашифровать.
	if upd.ClearKey {
		company.APIKeyEnc = nil
		company.KeyHint = nil
	} else if upd.APIKey != nil {
		if newKey := strings.TrimSpace(*upd.APIKey); newKey != "" {
			enc, err := s.cipher.Encrypt(newKey)
			if err != nil {
				s.log.Error("ai.encrypt_failed", "err", err)
				return nil, domain.NewError("AI_KEY_NOT_CONFIGURED",
					"На сервере не задан AI_KEY_ENCRYPTION_KEY", 500)
			}
			company.APIKeyEnc = enc
			hint := secret.MakeHint(newKey)
			company.KeyHint = &hint
		}
	}

	if err := s.repo.UpdateCompanyAI(ctx, company); err != nil {
		return nil, err
	}
	s.invalidateClient(company.ID)
	return dumpSettings(company), nil
}

// TestSettings — реальная проверка связи: один tiny-chat + один embedding.
// Ничего не сохраняет; ошибки не роняют запрос — уходят флагами в результат.
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
	_, embErr := s.llm.Embed(ctx, client.apiKey, client.modelEmbedding, []string{"ping"}, 10*time.Second)
	if embErr != nil {
		// Конкатенация как во Flask: (error or "") + " embedding: ..."
		setErr(" embedding: " + embErr.Error())
	} else {
		result.Embedding = true
	}
	result.LatencyMS = time.Since(t0).Milliseconds()
	return result, nil
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
	indexed, err := s.repo.CountEmbeddings(ctx, company.ID, company.ModelEmbedding)
	if err != nil {
		return nil, err
	}
	// pending: find_unindexed_task_ids смотрит только ai_enabled-компании —
	// при выключенном AI отдаёт 0.
	pending := 0
	if company.Enabled {
		ids, err := s.repo.FindUnindexedTaskIDs(ctx, company.ID, company.EmbeddingModel())
		if err != nil {
			return nil, err
		}
		pending = len(ids)
	}
	return &dto.IndexingStatus{
		TotalTasks: total,
		Indexed:    indexed,
		Pending:    pending,
		Model:      company.ModelEmbedding,
		// Как во Flask: только наличие ключа, без попытки расшифровать.
		AiEnabled: company.Enabled && len(company.APIKeyEnc) > 0,
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
	client, err := s.clientFor(ctx, company.ID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(409)
	}

	ids, err := s.repo.FindUnindexedTaskIDs(ctx, company.ID, company.EmbeddingModel())
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

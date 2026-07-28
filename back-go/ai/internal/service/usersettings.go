package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/secret"
)

// Личные ИИ-настройки пользователя: включён ли ИИ, какой моделью отвечать,
// какие возможности работают (чат Hola, ИИ в заметках) и НЕОБЯЗАТЕЛЬНЫЙ свой
// ключ со своим адресом API — на нём запросы уходят мимо квоты токенов.
// Прав тут проверять нечего: человек правит СВОИ настройки, скоуп — userID из
// токена. ИИ-возможности КОМПАНИИ живут отдельно (settings.go).

// defaultUserAI — вид настроек, когда строки в user_ai_settings ещё нет: ИИ
// включён, возможности разрешены, ключ платформенный.
func defaultUserAI(userID int64) *domain.UserAI {
	return &domain.UserAI{UserID: userID, Enabled: true, FeatAssistant: true, FeatNotes: true}
}

func (s *Service) dumpMySettings(ctx context.Context, u *domain.UserAI) *dto.MyAiSettings {
	out := &dto.MyAiSettings{
		Enabled:       u.Enabled,
		KeyHint:       u.KeyHint,
		HasKey:        u.OwnKey(),
		ModelChat:     u.ChatModel(),
		APIBaseURL:    u.APIBaseURL,
		FeatAssistant: u.FeatAssistant,
		FeatNotes:     u.FeatNotes,
		TokensLimit:   -1,
		TokensLeft:    -1,
		Models:        []*dto.AIModel{},
	}
	if p, err := s.platformAI(ctx); err == nil {
		out.PlatformReady = p.Ready()
	}
	if cat, err := s.catalog(ctx); err == nil {
		for _, m := range cat.Selectable() {
			out.Models = append(out.Models, &dto.AIModel{
				Code: m.Code, Title: m.Title, Rate: cat.Coefficient(m.Code),
			})
		}
	}
	// Остаток токенов: на своём ключе квота не тратится, показывать её незачем.
	if s.meter != nil && !u.OwnKey() {
		if _, left, ok := s.meter.Check(ctx, u.UserID, 0); ok {
			out.TokensLeft = left
		}
	}
	return out
}

func (s *Service) GetMySettings(ctx context.Context, userID int64) (*dto.MyAiSettings, error) {
	settings, err := s.repo.GetUserAI(ctx, userID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = defaultUserAI(userID)
	}
	return s.dumpMySettings(ctx, settings), nil
}

// UpdateMySettings — правила api_key прежние: не передан/пустой — не менять,
// clear_key — стереть, иначе зашифровать. Модель принимается только из
// каталога платформы (кроме случая своего ключа: там модель любая).
func (s *Service) UpdateMySettings(ctx context.Context, userID int64, upd dto.MyAiSettingsUpdate) (*dto.MyAiSettings, error) {
	settings, err := s.repo.GetUserAI(ctx, userID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = defaultUserAI(userID)
	}

	if upd.Enabled != nil {
		settings.Enabled = *upd.Enabled
	}
	if upd.FeatAssistant != nil {
		settings.FeatAssistant = *upd.FeatAssistant
	}
	if upd.FeatNotes != nil {
		settings.FeatNotes = *upd.FeatNotes
	}
	if upd.APIBaseURL != nil {
		settings.APIBaseURL = strings.TrimSpace(*upd.APIBaseURL)
	}
	if upd.ModelChat != nil {
		model := strings.TrimSpace(*upd.ModelChat)
		if model != "" && !settings.OwnKey() {
			cat, err := s.catalog(ctx)
			if err != nil {
				return nil, err
			}
			if !cat.Allowed(model) {
				return nil, domain.NewError("MODEL_UNKNOWN", "Такая модель недоступна", 422)
			}
		}
		settings.ModelChat = model
	}
	if upd.ClearKey {
		settings.APIKeyEnc = nil
		settings.KeyHint = nil
		settings.APIBaseURL = ""
	} else if upd.APIKey != nil {
		if newKey := strings.TrimSpace(*upd.APIKey); newKey != "" {
			enc, err := s.cipher.Encrypt(newKey)
			if err != nil {
				s.log.Error("ai.user_encrypt_failed", "err", err)
				return nil, domain.NewError("AI_KEY_NOT_CONFIGURED",
					"На сервере не задан AI_KEY_ENCRYPTION_KEY", 500)
			}
			settings.APIKeyEnc = enc
			hint := secret.MakeHint(newKey)
			settings.KeyHint = &hint
		}
	}

	if err := s.repo.UpsertUserAI(ctx, settings); err != nil {
		return nil, err
	}
	s.invalidateUserClient(userID)
	return s.dumpMySettings(ctx, settings), nil
}

// TestMySettings — проверка связи тем доступом, которым человек пользуется:
// своим ключом либо платформенным.
func (s *Service) TestMySettings(ctx context.Context, userID int64) (*dto.AiTestResult, error) {
	client, err := s.userClientFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(409)
	}

	t0 := time.Now()
	result := &dto.AiTestResult{}
	if _, err := s.llm.ChatOnce(ctx, domain.ChatParams{
		APIKey:       client.apiKey,
		BaseURL:      client.baseURL,
		Model:        client.modelChat,
		MessagesJSON: `[{"role":"user","content":"ping"}]`,
		MaxTokens:    2,
		Temperature:  0,
		Timeout:      10 * time.Second,
	}); err != nil {
		msg := "chat: " + err.Error()
		result.Error = &msg
	} else {
		result.Chat = true
	}
	result.LatencyMS = time.Since(t0).Milliseconds()
	return result, nil
}

// ── Платформенный ИИ (супер-админ, раздел «Аудит платформы») ─────

func (s *Service) GetPlatformSettings(ctx context.Context) (*dto.PlatformAiSettings, error) {
	p, err := s.repo.GetPlatformAI(ctx)
	if err != nil {
		return nil, err
	}
	cat, err := s.catalog(ctx)
	if err != nil {
		return nil, err
	}
	out := &dto.PlatformAiSettings{
		Enabled:        p.Enabled,
		HasKey:         len(p.APIKeyEnc) > 0,
		KeyHint:        p.KeyHint,
		BaseURL:        p.BaseURL,
		ModelChat:      firstNonEmpty(p.ModelChat, domain.PlatformModelChat),
		ModelEmbedding: firstNonEmpty(p.ModelEmbedding, domain.PlatformModelEmbedding),
		ModelSupport:   firstNonEmpty(p.ModelSupport, p.ModelChat, domain.PlatformModelChat),
		Models:         []*dto.PlatformAIModel{},
	}
	for _, m := range cat.All() {
		out.Models = append(out.Models, &dto.PlatformAIModel{
			Code: m.Code, Title: m.Title, Kind: m.Kind, PricePerMTok: m.PricePerMTok,
			Selectable: m.Selectable, IsActive: m.IsActive, Sort: m.Sort,
			Rate: cat.Coefficient(m.Code),
		})
	}
	return out, nil
}

// UpdatePlatformSettings — ключ proxy-api, адрес API, модели по умолчанию и
// цены каталога. Цена задаёт стоимость обращения в токенах доступа, поэтому
// правка каталога сразу меняет тарификацию.
func (s *Service) UpdatePlatformSettings(ctx context.Context, upd dto.PlatformAiUpdate) (*dto.PlatformAiSettings, error) {
	p, err := s.repo.GetPlatformAI(ctx)
	if err != nil {
		return nil, err
	}
	if upd.Enabled != nil {
		p.Enabled = *upd.Enabled
	}
	if upd.BaseURL != nil {
		p.BaseURL = strings.TrimSpace(*upd.BaseURL)
	}
	if upd.ModelChat != nil {
		p.ModelChat = strings.TrimSpace(*upd.ModelChat)
	}
	if upd.ModelEmbedding != nil {
		p.ModelEmbedding = strings.TrimSpace(*upd.ModelEmbedding)
	}
	if upd.ModelSupport != nil {
		p.ModelSupport = strings.TrimSpace(*upd.ModelSupport)
	}
	if upd.ClearKey {
		p.APIKeyEnc = nil
		p.KeyHint = ""
	} else if upd.APIKey != nil {
		if newKey := strings.TrimSpace(*upd.APIKey); newKey != "" {
			enc, err := s.cipher.Encrypt(newKey)
			if err != nil {
				s.log.Error("ai.platform_encrypt_failed", "err", err)
				return nil, domain.NewError("AI_KEY_NOT_CONFIGURED",
					"На сервере не задан AI_KEY_ENCRYPTION_KEY", 500)
			}
			p.APIKeyEnc = enc
			p.KeyHint = secret.MakeHint(newKey)
		}
	}
	if err := s.repo.UpdatePlatformAI(ctx, p); err != nil {
		return nil, err
	}
	for _, m := range upd.Models {
		if strings.TrimSpace(m.Code) == "" {
			continue
		}
		if err := s.repo.UpsertModel(ctx, &domain.AIModel{
			Code: m.Code, Title: m.Title, Kind: m.Kind, PricePerMTok: m.PricePerMTok,
			Selectable: m.Selectable, IsActive: m.IsActive, Sort: m.Sort,
		}); err != nil {
			return nil, err
		}
	}
	// Сбрасываем кэш: новый ключ и цены должны действовать сразу, а не через
	// минуту — иначе супер-админ проверяет связь старым ключом.
	s.invalidatePlatform()
	return s.GetPlatformSettings(ctx)
}

// TestPlatformSettings — проверка связи платформенным ключом.
func (s *Service) TestPlatformSettings(ctx context.Context) (*dto.AiTestResult, error) {
	client, err := s.platformClient(ctx)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(409)
	}
	return s.pingClient(ctx, client), nil
}

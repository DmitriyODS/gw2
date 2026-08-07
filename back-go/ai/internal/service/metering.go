package service

import (
	"context"
	"net/http"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// Тарификация обращений к моделям. Пользователь тратит ТОКЕНЫ ДОСТУПА: один
// такой токен = 1000 токенов самой дешёвой модели каталога, остальные модели
// дороже пропорционально цене (domain.ModelCatalog).
//
// Порядок такой: перед запросом проверяем остаток, после ответа списываем
// фактический расход. Предоплаты нет — модель может ответить и короче, и
// длиннее ожидаемого, а точные цифры приходят только в ответе.

// ErrNoTokens — на балансе не осталось токенов доступа. Пользователю нужно
// докупить их в магазине (или подключить свой ключ).
var ErrNoTokens = domain.NewError("AI_NO_TOKENS",
	"Закончились токены ИИ. Их можно докупить в магазине или подключить свой ключ.",
	http.StatusPaymentRequired)

// ensureTokens — есть ли чем платить за обращение. На своём ключе и без
// подключённого учёта — всегда да.
func (s *Service) ensureTokens(ctx context.Context, client *aiClient) error {
	if client == nil || client.ownKey || s.meter == nil || client.payerID <= 0 {
		return nil
	}
	_, left, ok := s.meter.Check(ctx, client.payerID, client.companyID)
	if !ok || left <= 0 {
		return ErrNoTokens
	}
	return nil
}

// spend — списать фактический расход обращения. Ошибки учёта не возвращаются:
// ответ модели пользователь уже получил, ронять его задним числом нельзя.
func (s *Service) spend(ctx context.Context, client *aiClient, feature, model string, usage domain.TokenUsage, actorID int64) {
	if client == nil || s.meter == nil || usage.Total() <= 0 {
		return
	}
	billed := int64(0)
	if !client.ownKey {
		cat, err := s.catalog(ctx)
		if err != nil {
			s.log.Warn("ai.catalog_failed", "error", err)
			return
		}
		billed = cat.AccessTokens(model, usage.Total())
	}
	s.meter.Consume(ctx, domain.TokenSpend{
		PayerID:   client.payerID,
		ActorID:   actorID,
		CompanyID: client.companyID,
		Feature:   feature,
		Model:     model,
		Usage:     usage,
		Billed:    billed,
		OwnKey:    client.ownKey,
	})
}

// chatMetered — один ход модели с проверкой и списанием токенов.
func (s *Service) chatMetered(ctx context.Context, client *aiClient, feature string, actorID int64,
	p domain.ChatParams) (*domain.ChatResult, error) {

	if err := s.ensureTokens(ctx, client); err != nil {
		return nil, err
	}
	p.APIKey, p.BaseURL = client.apiKey, client.baseURL
	if p.Model == "" {
		p.Model = client.modelChat
	}
	res, err := s.llm.ChatOnce(ctx, p)
	if err != nil {
		return nil, err
	}
	s.spend(ctx, client, feature, p.Model, res.Usage, actorID)
	return res, nil
}

// embedMetered — эмбеддинги с проверкой и списанием токенов.
func (s *Service) embedMetered(ctx context.Context, client *aiClient, feature string, actorID int64,
	texts []string, timeout time.Duration) ([][]float32, error) {

	if err := s.ensureTokens(ctx, client); err != nil {
		return nil, err
	}
	vecs, used, err := s.llm.Embed(ctx, domain.EmbedParams{
		APIKey: client.apiKey, BaseURL: client.baseURL,
		Model: client.modelEmbedding, Texts: texts, Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	s.spend(ctx, client, feature, client.modelEmbedding,
		domain.TokenUsage{PromptTokens: used}, actorID)
	return vecs, nil
}

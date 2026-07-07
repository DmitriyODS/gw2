package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// Status — включён ли ИИ (ai_enabled + ключ расшифровывается) и какие модели
// настроены. Выключен / компании нет — enabled=false БЕЗ ошибки.
func (s *Service) Status(ctx context.Context, companyID int64) (*StatusResult, error) {
	company, err := s.repo.GetCompanyAI(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return &StatusResult{}, nil
	}
	client, err := s.clientFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return &StatusResult{
		Enabled:        client != nil,
		ModelChat:      company.ChatModel(),
		ModelEmbedding: company.EmbeddingModel(),
	}, nil
}

// Chat — РОВНО ОДИН ход chat completion: messages → content | tool_calls.
// Цикл tool-calling крутит вызывающий — для делового ассистента это сам aisvc
// (chatWithTools в assistant.go, внутрипроцессный вызов, не gRPC).
func (s *Service) Chat(ctx context.Context, args ChatArgs) (*domain.ChatResult, error) {
	client, err := s.clientFor(ctx, args.CompanyID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(403)
	}
	if !json.Valid([]byte(args.MessagesJSON)) {
		return nil, domain.NewError("AI_BAD_REQUEST", "messages_json — невалидный JSON", 400)
	}
	if args.ToolsJSON != "" && !json.Valid([]byte(args.ToolsJSON)) {
		return nil, domain.NewError("AI_BAD_REQUEST", "tools_json — невалидный JSON", 400)
	}

	maxTokens := args.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	timeout := requestTimeout
	if args.TimeoutSec > 0 {
		timeout = time.Duration(args.TimeoutSec * float64(time.Second))
	}
	res, err := s.llm.ChatOnce(ctx, domain.ChatParams{
		APIKey:       client.apiKey,
		Model:        client.modelChat,
		MessagesJSON: args.MessagesJSON,
		ToolsJSON:    args.ToolsJSON,
		MaxTokens:    maxTokens,
		Temperature:  args.Temperature,
		Timeout:      timeout,
	})
	if err != nil {
		return nil, err
	}
	// Текстовый ответ стрипается, как chat() во Flask; при tool_calls
	// content отдаём как есть — вызывающий кладёт его в историю вербатим.
	if res.ToolCallsJSON == "" {
		res.Content = strings.TrimSpace(res.Content)
	}
	return res, nil
}

// Embed — эмбеддинг произвольного текста моделью компании.
func (s *Service) Embed(ctx context.Context, companyID int64, text string) ([]float32, string, error) {
	client, err := s.clientFor(ctx, companyID)
	if err != nil {
		return nil, "", err
	}
	if client == nil {
		return nil, "", errAiDisabled(403)
	}
	vecs, err := s.llm.Embed(ctx, client.apiKey, client.modelEmbedding, []string{text}, requestTimeout)
	if err != nil {
		return nil, "", err
	}
	return vecs[0], client.modelEmbedding, nil
}

// SemanticSearch — (task_id, score) по убыванию релевантности; score =
// 1 - cosine_distance. Fail-open как во Flask semantic_search: пустой запрос /
// выключенный AI / ошибка эмбеддинга → пустая выдача без ошибки.
func (s *Service) SemanticSearch(ctx context.Context, companyID int64, query string) ([]domain.SearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	client, err := s.clientFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, nil
	}
	vecs, err := s.llm.Embed(ctx, client.apiKey, client.modelEmbedding, []string{query}, 4*time.Second)
	if err != nil {
		s.log.Warn("ai.search.embed_failed", "company_id", companyID, "err", err)
		return nil, nil
	}
	// Фильтр по model обязателен: после смены модели старые эмбеддинги
	// не должны попадать в выдачу до перегенерации.
	hits, err := s.repo.SearchEmbeddings(ctx, companyID, vecs[0], client.modelEmbedding, semanticLimit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.SearchHit, 0, len(hits))
	for _, h := range hits {
		if h.Score > minSemanticScore {
			out = append(out, h)
		}
	}
	return out, nil
}

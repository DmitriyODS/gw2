package clients

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/billingpb"
)

// TokenMeter — учёт токенов доступа к ИИ поверх общего клиента биллинга.
// Отдельный адаптер нужен, чтобы домен aisvc не знал про gRPC-контракт.
type TokenMeter struct {
	billing *billingclient.Client
}

var _ domain.TokenMeter = (*TokenMeter)(nil)

func NewTokenMeter(billing *billingclient.Client) *TokenMeter {
	return &TokenMeter{billing: billing}
}

// Check — плательщик и остаток его токенов. Недоступный биллинг разрешает
// обращение (fail-open): ИИ не должен падать вместе с учётом.
func (m *TokenMeter) Check(ctx context.Context, userID, companyID int64) (int64, int64, bool) {
	return m.billing.AI(ctx, userID, companyID)
}

// Consume — списать фактический расход обращения.
func (m *TokenMeter) Consume(ctx context.Context, u domain.TokenSpend) (bool, int64) {
	return m.billing.ConsumeAI(ctx, &billingpb.ConsumeAIRequest{
		PayerId:          u.PayerID,
		ActorId:          u.ActorID,
		CompanyId:        u.CompanyID,
		Feature:          u.Feature,
		Model:            u.Model,
		PromptTokens:     int32(u.Usage.PromptTokens),
		CompletionTokens: int32(u.Usage.CompletionTokens),
		BilledTokens:     u.Billed,
		OwnKey:           u.OwnKey,
	})
}

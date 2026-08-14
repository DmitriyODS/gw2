package postgres

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// AddConsent — строка журнала согласий. Пишется в дополнение к users.legal_*:
// доказать факт получения согласия обязан оператор (ч.1 ст.9 152-ФЗ), а
// текущее состояние в users при следующем согласии перезаписывается.
func (r *UserRepository) AddConsent(ctx context.Context, c domain.Consent) error {
	docs, err := json.Marshal(c.Documents)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO user_consents (user_id, version, documents, accepted_at, ip, user_agent)
		VALUES ($1, $2, $3, now(), $4, $5)`,
		c.UserID, c.Version, docs, nullIfEmpty(c.IP), nullIfEmpty(c.UserAgent))
	return err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

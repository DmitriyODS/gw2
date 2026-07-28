package domain

import (
	"net/http"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

// Ошибки биллинга. Коды стабильны — на них опирается фронт.
var (
	ErrNotFound      = apierror.New("NOT_FOUND", "Не найдено", http.StatusNotFound)
	ErrForbidden     = apierror.New("FORBIDDEN", "Недостаточно прав", http.StatusForbidden)
	ErrValidation    = apierror.New("VALIDATION", "Проверьте заполненные поля", http.StatusUnprocessableEntity)
	ErrStoreDisabled = apierror.New("STORE_DISABLED", "Магазин временно недоступен", http.StatusServiceUnavailable)

	ErrPlanUnknown    = apierror.New("PLAN_UNKNOWN", "Такого тарифа нет", http.StatusUnprocessableEntity)
	ErrAddonUnknown   = apierror.New("ADDON_UNKNOWN", "Такого дополнения нет", http.StatusUnprocessableEntity)
	ErrSamePlan       = apierror.New("SAME_PLAN", "Этот тариф уже подключён", http.StatusConflict)
	ErrFreePlan       = apierror.New("FREE_PLAN", "Бесплатный тариф не оформляется — он действует по умолчанию", http.StatusUnprocessableEntity)
	ErrOrderNotPaid   = apierror.New("ORDER_NOT_PAID", "Заказ ещё не оплачен", http.StatusConflict)
	ErrOrderFinished  = apierror.New("ORDER_FINISHED", "Заказ уже завершён", http.StatusConflict)
	ErrAlreadyOwned   = apierror.New("ALREADY_OWNED", "Товар уже куплен", http.StatusConflict)
	ErrProductLocked  = apierror.New("PRODUCT_LOCKED", "Товар нельзя изменить на этом шаге", http.StatusConflict)
	ErrNotEnoughFunds = apierror.New("NOT_ENOUGH_FUNDS", "На балансе недостаточно средств", http.StatusConflict)

	ErrPromoUnknown  = apierror.New("PROMO_UNKNOWN", "Промокод не найден", http.StatusNotFound)
	ErrPromoExpired  = apierror.New("PROMO_EXPIRED", "Промокод больше не действует", http.StatusConflict)
	ErrPromoUsed     = apierror.New("PROMO_USED", "Промокод уже использован", http.StatusConflict)
	ErrPromoMismatch = apierror.New("PROMO_MISMATCH", "Промокод не подходит к этой покупке", http.StatusConflict)
	ErrPromoExists   = apierror.New("PROMO_EXISTS", "Такой промокод уже есть", http.StatusConflict)

	ErrPaymentFailed = apierror.New("PAYMENT_FAILED", "Не удалось создать платёж", http.StatusBadGateway)
)

// LimitError — превышение лимита тарифа. Extra несёт машинные поля, чтобы
// фронт показал апсейл: какой лимит, сколько сейчас и что за тариф.
func LimitError(kind string, limit int64, current int64, plan string, message string) *apierror.Error {
	return apierror.NewExtra("LIMIT_REACHED", message, http.StatusPaymentRequired, map[string]any{
		"limit_kind": kind,
		"limit":      limit,
		"current":    current,
		"plan":       plan,
	})
}

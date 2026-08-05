package apitest

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// unlimited — «без ограничения» в ответах биллинга (domain.Unlimited).
const unlimited = -1

// subscriptionsHidden — в выпуске 7.0 подписки скрыты от пользователя, и тариф
// не ограничивает счётные лимиты. Режим определяем ПО ОТВЕТУ сервера, а не по
// константе сервиса: харнес ходит снаружи и внутренние пакеты ему недоступны.
func subscriptionsHidden(boards int) bool { return boards == unlimited }

// Сквозные сценарии подписок: витрина, покупка тарифа через подтверждение
// оплаты супер-админом и реальный энфорсмент лимитов в других сервисах.

// Витрина отдаёт линейку тарифов и текущие лимиты пользователя.
func TestBillingShowcase(t *testing.T) {
	user := newVerifiedUser(t)
	dropPlan(t, user) // проверяем именно бесплатный тариф

	r := billingAPI.doJSON(t, http.MethodGet, "/api/billing/showcase", user.Token, nil)
	requireStatus(t, r, http.StatusOK, "запрос биллинга")

	var out struct {
		Entitlements struct {
			Plan   string `json:"plan"`
			Limits struct {
				Companies int   `json:"companies"`
				Boards    int   `json:"boards"`
				Storage   int64 `json:"storage_bytes"`
			} `json:"limits"`
		} `json:"entitlements"`
		Plans []struct {
			Code       string `json:"code"`
			PriceMonth int64  `json:"price_month"`
		} `json:"plans"`
		Addons []struct {
			Code string `json:"code"`
		} `json:"addons"`
	}
	if err := jsonUnmarshal(r.Raw, &out); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}

	if out.Entitlements.Plan != "junior" {
		t.Fatalf("новый пользователь должен быть на бесплатном тарифе, получено %q", out.Entitlements.Plan)
	}
	// Пока подписки скрыты, тариф не ограничивает счётные лимиты: покупать их
	// негде, значит и упираться человек не должен. Промежуточных состояний быть
	// не может — либо сняты все, либо действуют лимиты «Джуна».
	if subscriptionsHidden(out.Entitlements.Limits.Boards) {
		if out.Entitlements.Limits.Companies != unlimited {
			t.Fatalf("при скрытых подписках лимиты обязаны быть сняты все: %+v", out.Entitlements.Limits)
		}
	} else if out.Entitlements.Limits.Companies != 1 || out.Entitlements.Limits.Boards != 1 {
		t.Fatalf("лимиты «Джуна» не совпали: %+v", out.Entitlements.Limits)
	}
	if len(out.Plans) != 3 || len(out.Addons) == 0 {
		t.Fatalf("витрина должна отдавать линейку и докупки: планов %d, аддонов %d",
			len(out.Plans), len(out.Addons))
	}
}

// Лимит досок бесплатного тарифа: вторая доска не создаётся, а после выдачи
// платного тарифа — создаётся.
func TestBoardLimitEnforcedByPlan(t *testing.T) {
	user := newVerifiedUser(t)
	dropPlan(t, user) // лимит досок проверяем на «Джуне»
	admin := newSuperAdmin(t)

	ent := billingAPI.doJSON(t, http.MethodGet, "/api/billing/entitlements", user.Token, nil)
	requireStatus(t, ent, http.StatusOK, "запрос биллинга")
	var limits struct {
		Limits struct {
			Boards int `json:"boards"`
		} `json:"limits"`
	}
	if err := jsonUnmarshal(ent.Raw, &limits); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if subscriptionsHidden(limits.Limits.Boards) {
		t.Skip("подписки скрыты в этом выпуске — лимиты тарифа не применяются")
	}

	r := boardAPI.doJSON(t, http.MethodPost, "/api/boards", user.Token,
		map[string]any{"title": "Первая"})
	requireStatus(t, r, http.StatusCreated, "создание")

	// Вторая доска упирается в лимит «Джуна» (1 доска) — HTTP 402. Ждём в
	// цикле: клиент лимитов держит короткий кэш и на старте процесса мог
	// закэшировать fail-open-ответ (биллинг ещё поднимался).
	var limitErr struct {
		Error     string `json:"error"`
		LimitKind string `json:"limit_kind"`
	}
	blocked := false
	for i := 0; i < 40 && !blocked; i++ {
		r = boardAPI.doJSON(t, http.MethodPost, "/api/boards", user.Token,
			map[string]any{"title": fmt.Sprintf("Вторая %d", i)})
		if r.Status == http.StatusPaymentRequired {
			blocked = true
			break
		}
		time.Sleep(time.Second)
	}
	if !blocked {
		t.Fatalf("вторая доска должна упереться в лимит тарифа, последний статус %d", r.Status)
	}
	if err := jsonUnmarshal(r.Raw, &limitErr); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if limitErr.Error != "LIMIT_REACHED" || limitErr.LimitKind != "boards" {
		t.Fatalf("ожидалась ошибка лимита досок, получено %+v", limitErr)
	}

	// Супер-админ выдаёт «Синьора» — лимит снимается (у него доски безлимитны).
	r = billingAPI.doJSON(t, http.MethodPost, "/api/billing/admin/subscriptions/grant", admin.Token,
		map[string]any{"user_id": user.ID, "plan": "senior", "days": 30, "note": "apitest"})
	requireStatus(t, r, http.StatusOK, "запрос биллинга")

	// Клиент лимитов держит короткий кэш (30 с), поэтому боард-сервис может
	// секунду-другую видеть прежний тариф — ждём, но недолго.
	var created bool
	for i := 0; i < 40 && !created; i++ {
		resp := boardAPI.doJSON(t, http.MethodPost, "/api/boards", user.Token,
			map[string]any{"title": fmt.Sprintf("Третья %d", i)})
		created = resp.Status == http.StatusCreated
		if !created {
			time.Sleep(time.Second)
		}
	}
	if !created {
		t.Fatal("после выдачи «Синьора» лимит досок должен сняться")
	}
}

// Покупка тарифа: заказ ждёт оплаты, подтверждение супер-админом выдаёт
// подписку, повторное подтверждение её не удваивает.
func TestSubscriptionPurchaseFlow(t *testing.T) {
	user := newVerifiedUser(t)
	dropPlan(t, user)
	admin := newSuperAdmin(t)

	r := billingAPI.doJSON(t, http.MethodPost, "/api/billing/purchase", user.Token,
		map[string]any{"kind": "subscription", "item_code": "middle", "period": "month"})
	requireStatus(t, r, http.StatusCreated, "создание")
	var order struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Amount int64  `json:"amount"`
	}
	if err := jsonUnmarshal(r.Raw, &order); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if order.Status != "pending" || order.Amount != 29900 {
		t.Fatalf("неожиданный заказ: %+v", order)
	}

	// Пока заказ не оплачен, тариф прежний.
	r = billingAPI.doJSON(t, http.MethodGet, "/api/billing/entitlements", user.Token, nil)
	var ent struct {
		Plan string `json:"plan"`
	}
	if err := jsonUnmarshal(r.Raw, &ent); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if ent.Plan != "junior" {
		t.Fatalf("до оплаты тариф не меняется, получено %q", ent.Plan)
	}

	// Платёжный шлюз — заглушка: оплату подтверждает супер-админ.
	r = billingAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/billing/admin/orders/%d/confirm", order.ID), admin.Token, nil)
	requireStatus(t, r, http.StatusOK, "запрос биллинга")

	r = billingAPI.doJSON(t, http.MethodGet, "/api/billing/entitlements", user.Token, nil)
	if err := jsonUnmarshal(r.Raw, &ent); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if ent.Plan != "middle" {
		t.Fatalf("после оплаты ожидался тариф middle, получено %q", ent.Plan)
	}

	// Повторное подтверждение того же заказа отклоняется.
	r = billingAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/billing/admin/orders/%d/confirm", order.ID), admin.Token, nil)
	if r.Status != http.StatusConflict {
		t.Fatalf("повторное подтверждение должно отклоняться, получено %d", r.Status)
	}
}

// Промокод-подарок начисляет дни тарифа его владельцу.
func TestPromoGrantsPlanDays(t *testing.T) {
	user := newVerifiedUser(t)
	dropPlan(t, user)
	admin := newSuperAdmin(t)

	r := billingAPI.doJSON(t, http.MethodPost, "/api/billing/admin/promos", admin.Token,
		map[string]any{
			"code": "APITEST7", "kind": "days", "value": 7, "plan_code": "middle",
			"per_user_limit": 1, "is_active": true,
		})
	requireStatus(t, r, http.StatusCreated, "создание")

	r = billingAPI.doJSON(t, http.MethodPost, "/api/billing/promo/activate", user.Token,
		map[string]any{"code": "apitest7"}) // регистр не важен
	requireStatus(t, r, http.StatusOK, "запрос биллинга")

	r = billingAPI.doJSON(t, http.MethodGet, "/api/billing/entitlements", user.Token, nil)
	var ent struct {
		Plan string `json:"plan"`
	}
	if err := jsonUnmarshal(r.Raw, &ent); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if ent.Plan != "middle" {
		t.Fatalf("промокод должен выдать тариф middle, получено %q", ent.Plan)
	}

	// Повторная активация тем же человеком отклоняется.
	r = billingAPI.doJSON(t, http.MethodPost, "/api/billing/promo/activate", user.Token,
		map[string]any{"code": "APITEST7"})
	if r.Status != http.StatusConflict {
		t.Fatalf("повторная активация должна отклоняться, получено %d", r.Status)
	}
}

// Административные ручки закрыты от обычного пользователя.
func TestBillingAdminRequiresSuperAdmin(t *testing.T) {
	user := newVerifiedUser(t)
	r := billingAPI.doJSON(t, http.MethodGet, "/api/billing/admin/subscriptions", user.Token, nil)
	if r.Status != http.StatusForbidden {
		t.Fatalf("админские ручки биллинга должны быть закрыты, получено %d", r.Status)
	}
}

package apitest

import (
	"fmt"
	"net/http"
	"testing"
)

// ── Хелперы «Мой Groove» ─────────────────────────────────────────

// grooveCompany — компания с админом-создателем и одним сотрудником.
func grooveCompany(t *testing.T) (admin, member *actor, companyID int64) {
	t.Helper()
	admin = newVerifiedUser(t)
	companyID = admin.createCompany(t, uniq("Groove "))
	member = newMember(t, admin, companyID, roleEmployee)
	return admin, member, companyID
}

// grantBeans — выдать питомцу грувы напрямую (питомец должен уже существовать,
// т.е. после первого GET /pet). Для детерминированных тестов экономики.
func grantBeans(t *testing.T, userID int64, beans int) {
	t.Helper()
	tag, err := db.Exec(dbCtx(t), `UPDATE pets SET beans=$1 WHERE user_id=$2`, beans, userID)
	if err != nil {
		t.Fatalf("выдача грувов: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("выдача грувов: питомец user_id=%d не найден (нужен GET /pet)", userID)
	}
}

// ── Питомец и экономика ──────────────────────────────────────────

func TestGroovePetLifecycle(t *testing.T) {
	_, m, _ := grooveCompany(t)

	// Первое обращение создаёт питомца (яйцо, 0 грувов).
	r := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET /pet")
	if r.Str("name") != "Грувик" || r.Str("species") != "egg" || r.Num("beans") != 0 {
		t.Fatalf("новый питомец не в исходном состоянии: %s", r.Raw)
	}

	// Без грувов кормить нельзя.
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/pet/feed", m.Token, nil)
	requireError(t, r, 422, "NO_BEANS", "кормление без грувов")

	// Выдаём грувы и кормим до дневного лимита (6/день по 3 грува).
	grantBeans(t, m.ID, 100)
	for i := 0; i < 6; i++ {
		r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/pet/feed", m.Token, nil)
		requireStatus(t, r, 200, fmt.Sprintf("кормление %d", i+1))
	}
	// Списание грувов: 100 - 6*3 = 82.
	if r.Num("beans") != 82 {
		t.Fatalf("после 6 кормлений ожидалось 82 грува, получено %v", r.Num("beans"))
	}
	// 7-е кормление — дневной лимит исчерпан.
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/pet/feed", m.Token, nil)
	requireError(t, r, 429, "FED_ENOUGH", "7-е кормление за день")

	// Переименование.
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/pet/name", m.Token,
		map[string]any{"name": "Барсик"})
	requireStatus(t, r, 200, "переименование")
	if r.Str("name") != "Барсик" {
		t.Fatalf("имя не изменилось: %s", r.Raw)
	}
}

func TestGrooveShopAndQuest(t *testing.T) {
	_, m, _ := grooveCompany(t)

	// Магазин доступен без company-scope.
	r := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/shop", m.Token, nil)
	requireStatus(t, r, 200, "GET /shop")
	if r.JSON["prices"] == nil || r.JSON["species_prices"] == nil {
		t.Fatalf("магазин без прайса: %s", r.Raw)
	}

	// Питомец создаётся, покупка без грувов запрещена.
	grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", m.Token, nil)
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/shop/buy", m.Token,
		map[string]any{"item": "party"})
	requireError(t, r, 422, "NO_BEANS", "покупка без грувов")

	// С грувами: покупка проходит, повторная — ALREADY_OWNED.
	grantBeans(t, m.ID, 100)
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/shop/buy", m.Token,
		map[string]any{"item": "party"})
	requireStatus(t, r, 200, "покупка party")
	if r.Num("beans") != 70 { // 100 - 30
		t.Fatalf("грувы за покупку не списаны: %s", r.Raw)
	}
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/shop/buy", m.Token,
		map[string]any{"item": "party"})
	requireError(t, r, 422, "ALREADY_OWNED", "повторная покупка")

	// Несуществующий товар.
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/shop/buy", m.Token,
		map[string]any{"item": "нетакого"})
	requireError(t, r, 404, "NO_ITEM", "неизвестный товар")

	// Квест дня свежий (прогресс 0) → забрать награду нельзя.
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/pet/quest/claim", m.Token, nil)
	requireError(t, r, 422, "NOT_DONE", "клейм невыполненного квеста")
}

// ── Лента, кудосы, реакции, комментарии ──────────────────────────

func TestGrooveFeedKudosReactionsComments(t *testing.T) {
	admin, a, companyID := grooveCompany(t)
	b := newMember(t, admin, companyID, roleEmployee)
	c := newMember(t, admin, companyID, roleEmployee)

	// Пустая лента.
	r := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/feed", a.Token, nil)
	requireStatus(t, r, 200, "GET /feed")
	if len(r.List("items")) != 0 {
		t.Fatalf("лента новой компании не пуста: %s", r.Raw)
	}
	if len(r.List("allowed_reactions")) == 0 {
		t.Fatalf("нет allowed_reactions: %s", r.Raw)
	}

	// Кудос себе запрещён.
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/kudos", a.Token,
		map[string]any{"to_user_id": a.ID, "category": "helped", "text": "молодец я"})
	requireError(t, r, 422, "SELF_KUDOS", "кудос себе")

	// Неизвестная категория → 422 BAD_CATEGORY (валидна по форме, но не в наборе).
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/kudos", a.Token,
		map[string]any{"to_user_id": b.ID, "category": "wow", "text": "спасибо"})
	requireError(t, r, 422, "BAD_CATEGORY", "неизвестная категория")

	// Пустой текст → 400 (валидация формы).
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/kudos", a.Token,
		map[string]any{"to_user_id": b.ID, "category": "helped", "text": "   "})
	requireStatus(t, r, 400, "кудос без текста")

	// Валидный кудос A→B с категорией.
	r = grooveAPI.doJSON(t, http.MethodPost, "/api/groove/kudos", a.Token,
		map[string]any{"to_user_id": b.ID, "category": "quality", "text": "отличная работа"})
	requireStatus(t, r, 201, "кудос A→B")

	// Кудос виден в ленте с категорией.
	r = grooveAPI.doJSON(t, http.MethodGet, "/api/groove/feed", b.Token, nil)
	requireStatus(t, r, 200, "лента B")
	items := r.List("items")
	var kudosEvent map[string]any
	for _, it := range items {
		m := it.(map[string]any)
		if m["kind"] == "kudos" {
			kudosEvent = m
			break
		}
	}
	if kudosEvent == nil {
		t.Fatalf("кудос не появился в ленте: %s", r.Raw)
	}
	if pl, _ := kudosEvent["payload"].(map[string]any); pl == nil || pl["category"] != "quality" {
		t.Fatalf("категория кудоса потеряна: %v", kudosEvent["payload"])
	}
	eventID := int64(kudosEvent["id"].(float64))

	// Реакция: невалидный эмодзи.
	r = grooveAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/groove/feed/%d/reactions", eventID), c.Token,
		map[string]any{"emoji": "x"})
	requireError(t, r, 422, "BAD_EMOJI", "недопустимая реакция")

	// Реакция toggle: поставить и снять.
	r = grooveAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/groove/feed/%d/reactions", eventID), c.Token,
		map[string]any{"emoji": "🔥"})
	requireStatus(t, r, 200, "реакция+")
	if !r.Bool("added") || r.Num("count") != 1 {
		t.Fatalf("реакция не поставилась: %s", r.Raw)
	}
	r = grooveAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/groove/feed/%d/reactions", eventID), c.Token,
		map[string]any{"emoji": "🔥"})
	requireStatus(t, r, 200, "реакция-")
	if r.Bool("added") || r.Num("count") != 0 {
		t.Fatalf("реакция не снялась: %s", r.Raw)
	}

	// Комментарий: B добавляет, чужой сотрудник C удалить не может, автор может.
	r = grooveAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/groove/feed/%d/comments", eventID), b.Token,
		map[string]any{"text": "спасибо!"})
	requireStatus(t, r, 201, "комментарий B")
	commentID := int64(r.Num("id"))

	r = grooveAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/groove/feed/%d/comments", eventID), a.Token, nil)
	requireStatus(t, r, 200, "список комментариев")
	var comments []map[string]any
	_ = jsonUnmarshal(r.Raw, &comments)
	if len(comments) != 1 {
		t.Fatalf("ожидался один комментарий: %s", r.Raw)
	}

	r = grooveAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/groove/comments/%d", commentID), c.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "чужой сотрудник удаляет комментарий")

	r = grooveAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/groove/comments/%d", commentID), b.Token, nil)
	requireStatus(t, r, 200, "автор удаляет свой комментарий")
}

// ── Рейтинг, рейд, зоопарк, упразднённые ручки ───────────────────

func TestGrooveRatingRaidZoo(t *testing.T) {
	_, m, _ := grooveCompany(t)
	// Питомец должен существовать, чтобы попасть в рейтинг/зоопарк.
	grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", m.Token, nil)

	r := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/rating", m.Token, nil)
	requireStatus(t, r, 200, "GET /rating")
	if r.JSON["items"] == nil || r.JSON["me"] == nil || r.JSON["total"] == nil {
		t.Fatalf("рейтинг без ключей items/me/total: %s", r.Raw)
	}

	r = grooveAPI.doJSON(t, http.MethodGet, "/api/groove/raid", m.Token, nil)
	requireStatus(t, r, 200, "GET /raid")
	if r.Str("boss") == "" || r.JSON["week_start"] == nil {
		t.Fatalf("рейд без boss/week_start: %s", r.Raw)
	}
	if _, ok := r.JSON["my_closed"]; !ok {
		t.Fatalf("рейд без my_closed: %s", r.Raw)
	}

	r = grooveAPI.doJSON(t, http.MethodGet, "/api/groove/zoo", m.Token, nil)
	requireStatus(t, r, 200, "GET /zoo")
	var zoo []map[string]any
	if err := jsonUnmarshal(r.Raw, &zoo); err != nil {
		t.Fatalf("зоопарк не массив: %s", r.Raw)
	}

	// Упразднённые ручки (поглаживание/заряд) больше не смонтированы.
	for _, p := range []string{"/api/groove/pet/stroke", "/api/groove/zap"} {
		r = grooveAPI.doJSON(t, http.MethodPost, p, m.Token, nil)
		requireStatus(t, r, 404, "упразднённая ручка "+p)
	}
}

// ── Company-scope и гейт ─────────────────────────────────────────

func TestGrooveScopeAndGate(t *testing.T) {
	admin, m, companyID := grooveCompany(t)

	// Без токена — 401.
	r := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", "", nil)
	requireStatus(t, r, 401, "GET /pet без токена")

	// Супер-админ без активной компании — нужен ?company_id.
	root := newSuperAdmin(t)
	r = grooveAPI.doJSON(t, http.MethodGet, "/api/groove/feed", root.Token, nil)
	requireError(t, r, 400, "BAD_REQUEST", "супер-админ без company_id")

	// Выключенный режим «Мой Groove» → 403 GROOVE_DISABLED.
	if _, err := db.Exec(dbCtx(t),
		`UPDATE companies SET settings = jsonb_set(COALESCE(settings,'{}'::jsonb), '{uses_groove}', 'false') WHERE id=$1`,
		companyID); err != nil {
		t.Fatalf("выключение groove: %v", err)
	}
	r = grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", m.Token, nil)
	requireError(t, r, 403, "GROOVE_DISABLED", "выключенный режим Groove")
	_ = admin
}

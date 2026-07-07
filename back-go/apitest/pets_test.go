package apitest

import (
	"fmt"
	"net/http"
	"testing"
)

// ── Хелперы «Питомцы-грувики» ────────────────────────────────────

// petsCompany — компания с админом-создателем и одним сотрудником.
func petsCompany(t *testing.T) (admin, member *actor, companyID int64) {
	t.Helper()
	admin = newVerifiedUser(t)
	companyID = admin.createCompany(t, uniq("Pets "))
	member = newMember(t, admin, companyID, roleEmployee)
	return admin, member, companyID
}

// grantKudos — выдать питомцу кудосы напрямую (питомец должен уже
// существовать, т.е. после первого GET /pet). Для детерминированных тестов.
func grantKudos(t *testing.T, userID int64, kudos int) {
	t.Helper()
	tag, err := db.Exec(dbCtx(t), `UPDATE pets SET kudos=$1 WHERE user_id=$2`, kudos, userID)
	if err != nil {
		t.Fatalf("выдача кудосов: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("выдача кудосов: питомец user_id=%d не найден (нужен GET /pet)", userID)
	}
}

// ── Питомец и кормление ──────────────────────────────────────────

func TestPetsPetLifecycle(t *testing.T) {
	_, m, _ := petsCompany(t)

	// Первое обращение создаёт питомца (яйцо, 0 кудосов).
	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET /pet")
	if r.Str("species") != "egg" || r.Num("kudos") != 0 {
		t.Fatalf("новый питомец не в исходном состоянии: %s", r.Raw)
	}

	// Без кудосов кормить нельзя.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
	requireError(t, r, 422, "NO_KUDOS", "кормление без кудосов")

	// Выдаём кудосы и кормим до дневного лимита (6/день по 3 кудоса).
	grantKudos(t, m.ID, 100)
	for i := 0; i < 6; i++ {
		r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
		requireStatus(t, r, 200, "кормление")
	}
	// Списание кудосов: 100 - 6*3 = 82.
	if r.Num("kudos") != 82 {
		t.Fatalf("после 6 кормлений ожидалось 82 кудоса, получено %v", r.Num("kudos"))
	}
	// 7-е кормление — дневной лимит исчерпан.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
	requireError(t, r, 429, "FED_ENOUGH", "7-е кормление за день")

	// Переименование.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/name", m.Token,
		map[string]any{"name": "Барсик"})
	requireStatus(t, r, 200, "переименование")
	if r.Str("name") != "Барсик" {
		t.Fatalf("имя не изменилось: %s", r.Raw)
	}
}

// ── Прогулка и лечение ───────────────────────────────────────────

func TestPetsWalkAndHeal(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)

	// Прогулка без кудосов запрещена.
	r := petsAPI.doJSON(t, http.MethodPost, "/api/pets/walk", m.Token, nil)
	requireError(t, r, 422, "NO_KUDOS", "прогулка без кудосов")

	grantKudos(t, m.ID, 1000)
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/walk", m.Token, nil)
	requireStatus(t, r, 200, "прогулка")

	// Лечение здорового питомца запрещено.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/heal", m.Token, nil)
	requireError(t, r, 422, "NOT_SICK", "лечение здорового питомца")

	// Заболевание питомца напрямую в БД (проще, чем ждать реальный простой).
	if _, err := db.Exec(dbCtx(t),
		`UPDATE pets SET sick_since = now(), recovery = 0 WHERE user_id=$1`, m.ID); err != nil {
		t.Fatalf("заражение питомца: %v", err)
	}
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	if !r.Bool("sick") {
		t.Fatalf("питомец не помечен больным: %s", r.Raw)
	}
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/heal", m.Token, nil)
	requireStatus(t, r, 200, "лечение больного питомца")
}

// ── Магазин и мистери-слот ───────────────────────────────────────

func TestPetsShopAndMystery(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)

	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/shop", m.Token, nil)
	requireStatus(t, r, 200, "GET /shop")
	if r.JSON["items"] == nil {
		t.Fatalf("магазин без списка товаров: %s", r.Raw)
	}

	// Покупка без кудосов запрещена.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/shop/buy", m.Token,
		map[string]any{"item": "party"})
	requireError(t, r, 422, "NO_KUDOS", "покупка без кудосов")

	// С кудосами: покупка проходит, повторная — ALREADY_OWNED.
	grantKudos(t, m.ID, 1000)
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/shop/buy", m.Token,
		map[string]any{"item": "party"})
	requireStatus(t, r, 200, "покупка party")
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/shop/buy", m.Token,
		map[string]any{"item": "party"})
	requireError(t, r, 422, "ALREADY_OWNED", "повторная покупка")

	// Несуществующий товар.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/shop/buy", m.Token,
		map[string]any{"item": "нетакого"})
	requireError(t, r, 404, "NO_ITEM", "неизвестный товар")

	// Мистери-слот: первый раз выдаёт предмет, второй — ALREADY_TAKEN.
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/shop/mystery", m.Token, nil)
	requireStatus(t, r, 200, "GET /shop/mystery")
	if r.Str("key") == "" {
		t.Fatalf("мистери-слот без предмета: %s", r.Raw)
	}
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/shop/mystery", m.Token, nil)
	requireError(t, r, 429, "ALREADY_TAKEN", "повторный мистери-слот в тот же день")
}

// ── Квест дня, рейтинг, зоопарк ───────────────────────────────────

func TestPetsQuestRatingZoo(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)

	// Квест дня свежий (прогресс 0) → забрать награду нельзя.
	r := petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/quest/claim", m.Token, nil)
	requireError(t, r, 422, "NOT_DONE", "клейм невыполненного квеста")

	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/rating", m.Token, nil)
	requireStatus(t, r, 200, "GET /rating")
	if r.JSON["items"] == nil || r.JSON["me"] == nil || r.JSON["total"] == nil {
		t.Fatalf("рейтинг без ключей items/me/total: %s", r.Raw)
	}

	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/zoo", m.Token, nil)
	requireStatus(t, r, 200, "GET /zoo")
	var zoo []map[string]any
	if err := jsonUnmarshal(r.Raw, &zoo); err != nil {
		t.Fatalf("зоопарк не массив: %s", r.Raw)
	}
}

// ── Поглаживание чужого питомца ──────────────────────────────────

func TestPetsStrokeColleague(t *testing.T) {
	admin, a, companyID := petsCompany(t)
	b := newMember(t, admin, companyID, roleEmployee)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", a.Token, nil)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", b.Token, nil)

	// Себя погладить нельзя.
	r := petsAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/pets/stroke/%d", a.ID), a.Token, nil)
	requireError(t, r, 422, "SELF_STROKE", "поглаживание себя")

	// Без кудосов гладить нельзя.
	r = petsAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/pets/stroke/%d", b.ID), a.Token, nil)
	requireError(t, r, 422, "NO_KUDOS", "поглаживание без кудосов")

	grantKudos(t, a.ID, 100)
	r = petsAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/pets/stroke/%d", b.ID), a.Token, nil)
	requireStatus(t, r, 200, "поглаживание коллеги")
}

// ── Приключение питомца ──────────────────────────────────────────

// TestPetsAdventure — appointment-механика: старт приключения, гейты
// платных действий PET_AWAY (включая поглаживание чужого питомца в пути),
// ленивый возврат с наградой на GET владельца (ровно один раз).
func TestPetsAdventure(t *testing.T) {
	admin, m, companyID := petsCompany(t)
	b := newMember(t, admin, companyID, roleEmployee)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", b.Token, nil)
	grantKudos(t, m.ID, 100)
	grantKudos(t, b.ID, 100)

	// Старт: бесплатно, в ответе срок и локация.
	r := petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/adventure", m.Token, nil)
	requireStatus(t, r, 200, "старт приключения")
	if r.Str("adventure_until") == "" || r.Str("adventure_place") == "" {
		t.Fatalf("нет полей приключения в ответе: %s", r.Raw)
	}
	if r.Num("kudos") != 100 {
		t.Fatalf("старт должен быть бесплатным: %s", r.Raw)
	}

	// Повторный старт — питомец уже в пути.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/adventure", m.Token, nil)
	requireError(t, r, 422, "PET_AWAY", "повторный старт")

	// Платные действия владельца в пути недоступны.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
	requireError(t, r, 422, "PET_AWAY", "кормление в пути")
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/walk", m.Token, nil)
	requireError(t, r, 422, "PET_AWAY", "прогулка в пути")
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/heal", m.Token, nil)
	requireError(t, r, 422, "PET_AWAY", "лечение в пути")

	// Поглаживание ЧУЖОГО питомца в пути тоже недоступно.
	r = petsAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/pets/stroke/%d", m.ID), b.Token, nil)
	requireError(t, r, 422, "PET_AWAY", "поглаживание питомца в пути")

	// GET владельца, пока срок не истёк, возврат не фиксирует.
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET в пути")
	if r.Str("adventure_until") == "" || r.JSON["adventure_reward"] != nil {
		t.Fatalf("возврат зафиксирован раньше срока: %s", r.Raw)
	}

	// Форс-возврат: срок истёк — следующий GET владельца фиксирует возврат
	// и разово начисляет награду.
	if _, err := db.Exec(dbCtx(t),
		`UPDATE pets SET adventure_until = now() - interval '1 second' WHERE user_id=$1`,
		m.ID); err != nil {
		t.Fatalf("форс-возврат: %v", err)
	}
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET после срока")
	reward, _ := r.JSON["adventure_reward"].(map[string]any)
	if reward == nil {
		t.Fatalf("возврат без награды: %s", r.Raw)
	}
	kudos, _ := reward["kudos"].(float64)
	xp, _ := reward["xp"].(float64)
	if kudos < 3 || kudos > 10 || xp < 5 || xp > 15 {
		t.Fatalf("награда вне диапазонов 3–10/5–15: %v", reward)
	}
	if place, _ := reward["place"].(string); place == "" {
		t.Fatalf("награда без локации: %v", reward)
	}
	if r.Str("adventure_until") != "" || r.Str("adventure_place") != "" {
		t.Fatalf("поля приключения не очищены после возврата: %s", r.Raw)
	}
	if r.Num("kudos") != 100+kudos {
		t.Fatalf("кудосы после возврата: %v, ожидалось %v", r.Num("kudos"), 100+kudos)
	}

	// Повторный GET награду не дублирует.
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "повторный GET")
	if r.JSON["adventure_reward"] != nil {
		t.Fatalf("награда начислена дважды: %s", r.Raw)
	}
	if r.Num("kudos") != 100+kudos {
		t.Fatalf("кудосы изменились без начисления: %v", r.Num("kudos"))
	}
}

// ── Сейчас в эфире ────────────────────────────────────────────────

func TestPetsLiveActiveUnits(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)
	member := newMember(t, admin, companyID, roleEmployee)
	typeID := createUnitType(t, admin, uniq("Эфир "))
	taskID := createTask(t, admin, deptID, "Задача в эфире", nil)

	unitID := startUnit(t, member, taskID, typeID, "юнит в эфире")
	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/live", admin.Token, nil)
	requireStatus(t, r, 200, "live")
	found := false
	for _, it := range r.List("items") {
		m := it.(map[string]any)
		if int64(m["unit_id"].(float64)) == unitID {
			found = true
			if u, _ := m["user"].(map[string]any); u == nil || int64(u["id"].(float64)) != member.ID {
				t.Fatalf("live-элемент без владельца: %v", m)
			}
		}
	}
	if !found {
		t.Fatalf("активный юнит не попал в live: %s", r.Raw)
	}

	stopUnit(t, member, unitID)
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/live", admin.Token, nil)
	for _, it := range r.List("items") {
		if int64(it.(map[string]any)["unit_id"].(float64)) == unitID {
			t.Fatalf("остановленный юнит остался в live: %s", r.Raw)
		}
	}
}

// ── Company-scope и гейт ──────────────────────────────────────────

func TestPetsScopeAndGate(t *testing.T) {
	_, m, companyID := petsCompany(t)

	// Без токена — 401.
	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", "", nil)
	requireStatus(t, r, 401, "GET /pet без токена")

	// Обычный пользователь без активной компании не может подсунуть чужую
	// компанию query-параметром: ?company_id= — привилегия супер-админа.
	outsider := newVerifiedUser(t)
	r = petsAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/pets/zoo?company_id=%d", companyID), outsider.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "чужая компания через ?company_id без прав")

	// Супер-админ без активной компании — нужен ?company_id.
	root := newSuperAdmin(t)
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/zoo", root.Token, nil)
	requireError(t, r, 400, "BAD_REQUEST", "супер-админ без company_id")

	// С ?company_id супер-админ компанию видит.
	r = petsAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/pets/zoo?company_id=%d", companyID), root.Token, nil)
	requireStatus(t, r, 200, "зоопарк для супер-админа по company_id")

	// Выключенный режим «Мой Groove» → 403 GROOVE_DISABLED.
	if _, err := db.Exec(dbCtx(t),
		`UPDATE companies SET settings = jsonb_set(COALESCE(settings,'{}'::jsonb), '{uses_groove}', 'false') WHERE id=$1`,
		companyID); err != nil {
		t.Fatalf("выключение groove: %v", err)
	}
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireError(t, r, 403, "GROOVE_DISABLED", "выключенный режим Groove")
}

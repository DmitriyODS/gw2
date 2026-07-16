package apitest

// API-тесты потребностей грувика: ленивое убывание шкал через реальный
// GET /api/pets/pet, болезни «своей» шкалы, рецепты лечения (сон/купание/
// бульон), побег заброшенного питомца и награда владельца за поглаживание.

import (
	"fmt"
	"net/http"
	"testing"
)

// agePetNeeds — отодвинуть отметку пересчёта потребностей в прошлое: питомец
// «проживает» minutes минут без ухода (ждать их в тесте, очевидно, нельзя).
// Границы тика избегаем намеренно (+полтика): часы контейнера БД и хоста
// расходятся на доли секунды, и ровно кратный сдвиг давал бы то N, то N−1.
func agePetNeeds(t *testing.T, userID int64, minutes int) {
	t.Helper()
	tag, err := db.Exec(dbCtx(t),
		fmt.Sprintf(`UPDATE pets SET needs_at = now() - interval '%d minutes' WHERE user_id=$1`, minutes),
		userID)
	if err != nil {
		t.Fatalf("сдвиг needs_at: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("сдвиг needs_at: питомец user_id=%d не найден (нужен GET /pet)", userID)
	}
}

// makeSick — уложить питомца в конкретную болезнь на sickDays дней (побег и
// рецепты проверяются от заданного состояния, а не от случайного).
func makeSick(t *testing.T, userID int64, ailment string, sickDays int) {
	t.Helper()
	_, err := db.Exec(dbCtx(t), fmt.Sprintf(`
		UPDATE pets SET ailment=$2, sick_since = now() - interval '%d days', recovery=0
		WHERE user_id=$1`, sickDays), userID, ailment)
	if err != nil {
		t.Fatalf("заражение питомца (%s): %v", ailment, err)
	}
}

// setNeed — выставить одну шкалу (needs_at сдвигать не нужно: убывание не
// успевает набрать тик за время теста).
func setNeed(t *testing.T, userID int64, column string, value int) {
	t.Helper()
	if _, err := db.Exec(dbCtx(t),
		`UPDATE pets SET `+column+`=$2, needs_at=now() WHERE user_id=$1`, userID, value); err != nil {
		t.Fatalf("выставление %s: %v", column, err)
	}
}

// ── Потребности и болезни ────────────────────────────────────────

// Свежий питомец сыт и доволен, а брошенный на сутки — голодает и заболевает
// истощением: «не кормишь — болеет» проверяется целиком через API.
func TestPetsNeedsDecayCausesHunger(t *testing.T) {
	_, m, _ := petsCompany(t)

	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET /pet")
	needs, ok := r.JSON["needs"].(map[string]any)
	if !ok || needs["satiety"] != float64(100) || r.Num("mood") != 100 {
		t.Fatalf("новый питомец должен быть полон сил: %s", r.Raw)
	}
	if r.JSON["ailment"] != nil {
		t.Fatalf("новый питомец не может быть больным: %s", r.Raw)
	}

	// 5 часов без ухода: сытость −20 (2 за каждые полчаса), энергия −10.
	agePetNeeds(t, m.ID, 5*60+15)
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	needs = r.JSON["needs"].(map[string]any)
	if needs["satiety"] != float64(80) || needs["energy"] != float64(90) {
		t.Fatalf("потребности не убывают со временем: %s", r.Raw)
	}
	if r.Bool("sick") {
		t.Fatalf("5 часов — ещё не болезнь: %s", r.Raw)
	}

	// Сутки с лишним без еды — истощение (шкала сытости в нуле).
	agePetNeeds(t, m.ID, 30*60)
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	needs = r.JSON["needs"].(map[string]any)
	if needs["satiety"] != float64(0) {
		t.Fatalf("сытость должна опустеть: %s", r.Raw)
	}
	if !r.Bool("sick") || r.Str("ailment") != "hunger" {
		t.Fatalf("голодный питомец должен заболеть истощением: %s", r.Raw)
	}
	if r.Str("ailment_title") == "" || r.Str("ailment_hint") == "" {
		t.Fatalf("болезнь без подписи и рецепта: %s", r.Raw)
	}
	// Больному XP заморожен, но настроение всё равно отдаётся клиенту.
	if r.Num("mood") == 100 {
		t.Fatalf("настроение должно просесть: %s", r.Raw)
	}
}

// Каждая запущенная шкала ведёт в свою болезнь; общение болезни не даёт —
// только роняет настроение (и множитель XP за работу).
func TestPetsNeedsOwnAilments(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)

	cases := []struct {
		column  string
		ailment string
	}{
		{"need_energy", "cold"},
		{"need_hygiene", "grime"},
	}
	for _, c := range cases {
		t.Run(c.ailment, func(t *testing.T) {
			// Вылечиваем прошлую болезнь и обнуляем ровно одну шкалу.
			if _, err := db.Exec(dbCtx(t), `
				UPDATE pets SET ailment=NULL, sick_since=NULL, recovery=0,
					need_satiety=100, need_energy=100, need_hygiene=100, need_social=100,
					needs_at=now()
				WHERE user_id=$1`, m.ID); err != nil {
				t.Fatalf("сброс состояния: %v", err)
			}
			setNeed(t, m.ID, c.column, 0)

			r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
			if r.Str("ailment") != c.ailment {
				t.Fatalf("ожидалась болезнь %q, получено: %s", c.ailment, r.Raw)
			}
		})
	}

	// Одиночество — не болезнь.
	if _, err := db.Exec(dbCtx(t), `
		UPDATE pets SET ailment=NULL, sick_since=NULL, recovery=0,
			need_satiety=100, need_energy=100, need_hygiene=100, need_social=0, needs_at=now()
		WHERE user_id=$1`, m.ID); err != nil {
		t.Fatalf("сброс состояния: %v", err)
	}
	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	if r.Bool("sick") {
		t.Fatalf("одиночество не должно быть болезнью: %s", r.Raw)
	}
	// Зато настроение (а с ним множитель XP) заметно проседает — запущенная
	// шкала весит больше остальных.
	if r.Num("mood") >= 60 || r.Num("mood_factor") >= 1.5 {
		t.Fatalf("одинокий питомец должен хуже брать XP: %s", r.Raw)
	}
}

// ── Рецепты лечения ──────────────────────────────────────────────

// Сон бесплатен, восполняет энергию и лечит простуду; лимит — 2 раза в день.
func TestPetsSleepCuresCold(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	setNeed(t, m.ID, "need_energy", 0)
	makeSick(t, m.ID, "cold", 0)

	r := petsAPI.doJSON(t, http.MethodPost, "/api/pets/sleep", m.Token, nil)
	requireStatus(t, r, 200, "сон")
	if r.Num("kudos") != 0 {
		t.Fatalf("сон должен быть бесплатным: %s", r.Raw)
	}
	needs := r.JSON["needs"].(map[string]any)
	if needs["energy"] != float64(55) {
		t.Fatalf("сон должен восполнять энергию: %s", r.Raw)
	}
	if r.Num("recovery") != 2 {
		t.Fatalf("сон — верный рецепт от простуды (2 очка): %s", r.Raw)
	}

	// Второй сон добивает лечение, третий — упирается в дневной лимит.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/sleep", m.Token, nil)
	requireStatus(t, r, 200, "второй сон")
	if r.Bool("sick") {
		t.Fatalf("после двух снов простуда должна пройти: %s", r.Raw)
	}
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/sleep", m.Token, nil)
	requireError(t, r, 429, "SLEPT_ENOUGH", "третий сон за день")
}

// Купание — платное, чистит и одним разом поднимает грязнулю на ноги.
func TestPetsBathCuresGrime(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	setNeed(t, m.ID, "need_hygiene", 0)
	makeSick(t, m.ID, "grime", 0)

	r := petsAPI.doJSON(t, http.MethodPost, "/api/pets/bath", m.Token, nil)
	requireError(t, r, 422, "NO_KUDOS", "купание без кудосов")

	grantKudos(t, m.ID, 100)
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/bath", m.Token, nil)
	requireStatus(t, r, 200, "купание")
	if r.Num("kudos") != 88 {
		t.Fatalf("цена купания — 12 кудосов: %s", r.Raw)
	}
	if r.Bool("sick") {
		t.Fatalf("купание должно вылечить грязнулю: %s", r.Raw)
	}
	needs := r.JSON["needs"].(map[string]any)
	if needs["hygiene"] != float64(70) {
		t.Fatalf("купание должно чистить: %s", r.Raw)
	}
}

// Рецепт имеет значение: бульон поднимает истощённого, но простуду почти не
// лечит — от неё нужен сон.
func TestPetsWrongCureBarelyHelps(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	grantKudos(t, m.ID, 100)

	// Истощение: бульон — 2 очка из 3.
	setNeed(t, m.ID, "need_satiety", 0)
	makeSick(t, m.ID, "hunger", 0)
	r := petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
	requireStatus(t, r, 200, "бульон истощённому")
	if r.Num("recovery") != 2 {
		t.Fatalf("бульон — верный рецепт от истощения: %s", r.Raw)
	}
	needs := r.JSON["needs"].(map[string]any)
	if needs["satiety"] != float64(20) {
		t.Fatalf("бульон должен питать: %s", r.Raw)
	}

	// Простуда: тот же бульон — лишь 1 очко.
	makeSick(t, m.ID, "cold", 0)
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
	requireStatus(t, r, 200, "бульон простуженному")
	if r.Num("recovery") != 1 {
		t.Fatalf("от простуды бульон помогает слабо: %s", r.Raw)
	}
}

// ── Побег ────────────────────────────────────────────────────────

// Заброшенный питомец уходит: прогресс с нуля, имущество (кудосы, гардероб)
// остаётся, повторный GET побега не дублирует.
func TestPetsRunawayAfterLongSickness(t *testing.T) {
	_, m, _ := petsCompany(t)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)

	if _, err := db.Exec(dbCtx(t), `
		UPDATE pets SET stage=4, xp=700, species='owl', kudos=500, generation=2
		WHERE user_id=$1`, m.ID); err != nil {
		t.Fatalf("прокачка питомца: %v", err)
	}

	// Болеет 10 дней — ещё дома, но уже с предупреждением.
	makeSick(t, m.ID, "blues", 10)
	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	if r.JSON["runaway"] != nil {
		t.Fatalf("рано сбежал: %s", r.Raw)
	}
	if r.JSON["runaway_in_days"] == nil || r.Num("runaway_in_days") != 4 {
		t.Fatalf("ожидалось предупреждение за 4 дня: %s", r.Raw)
	}
	if r.Num("stage") != 4 {
		t.Fatalf("прогресс тронут раньше времени: %s", r.Raw)
	}

	// Две недели болезни — питомец уходит.
	makeSick(t, m.ID, "blues", 15)
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET /pet после долгой болезни")
	runaway, ok := r.JSON["runaway"].(map[string]any)
	if !ok || runaway["ailment"] != "blues" {
		t.Fatalf("ожидался побег: %s", r.Raw)
	}
	if r.Num("stage") != 0 || r.Num("xp") != 0 || r.Str("species") != "egg" {
		t.Fatalf("прогресс должен обнулиться: %s", r.Raw)
	}
	if r.Bool("sick") {
		t.Fatalf("новое яйцо не может быть больным: %s", r.Raw)
	}
	if r.Num("kudos") != 500 {
		t.Fatalf("кудосы должны уцелеть: %s", r.Raw)
	}
	if r.Num("generation") != 2 {
		t.Fatalf("поколения престижа не сбрасываются: %s", r.Raw)
	}

	// Повторный GET — побег уже зафиксирован (разовое поле).
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	if r.JSON["runaway"] != nil {
		t.Fatalf("побег отдан дважды: %s", r.Raw)
	}
}

// ── Поглаживание: смысл для обеих сторон ─────────────────────────

// Гладящий платит, а владелец поглаженного получает кудосы (больше, чем
// потрачено), XP, общение и строку недельного рейтинга.
func TestPetsStrokeRewardsOwner(t *testing.T) {
	admin, a, companyID := petsCompany(t)
	b := newMember(t, admin, companyID, roleEmployee)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", a.Token, nil)
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", b.Token, nil)

	grantKudos(t, a.ID, 100)
	setNeed(t, b.ID, "need_social", 10)

	r := petsAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/pets/stroke/%d", b.ID), a.Token, nil)
	requireStatus(t, r, 200, "поглаживание коллеги")
	// Ответ — снапшот ПОГЛАЖЕННОГО питомца.
	if r.Num("kudos") != 3 {
		t.Fatalf("владелец должен получить 3 кудоса: %s", r.Raw)
	}
	if r.Num("xp") != 2 {
		t.Fatalf("владельцу полагается XP настроения: %s", r.Raw)
	}
	needs := r.JSON["needs"].(map[string]any)
	if needs["social"] != float64(35) {
		t.Fatalf("поглаживание должно закрывать потребность в общении: %s", r.Raw)
	}

	// У гладящего списалось 2 кудоса, его питомцу — 1 XP за компанию.
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", a.Token, nil)
	if r.Num("kudos") != 98 || r.Num("xp") != 1 {
		t.Fatalf("гладящий: ожидалось 98 кудосов и 1 XP, получено %s", r.Raw)
	}

	// Признание идёт в недельный рейтинг владельца.
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/rating", b.Token, nil)
	me, ok := r.JSON["me"].(map[string]any)
	if !ok || me["kudos_week"] != float64(3) {
		t.Fatalf("поглаживание должно кормить рейтинг признания: %s", r.Raw)
	}
}

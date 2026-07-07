package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestPetsTaskClosedHookAcrossServices — сквозной сценарий геймификации:
// закрытие задачи в tasksvc → gRPC-хук OnTaskClosed (fire-and-forget после
// коммита) → petsvc начисляет герою кудосы (+5) и XP (+8) и эмитит
// pet:update, которое доезжает клиентам через мост шлюза (комната
// user_<id>).
func TestPetsTaskClosedHookAcrossServices(t *testing.T) {
	admin, _, deptID := newTaskCompany(t)

	// Питомец существует заранее — начисления пойдут в него.
	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", admin.Token, nil)
	requireStatus(t, r, 200, "GET /pet")
	if r.Num("kudos") != 0 || r.Num("xp") != 0 {
		t.Fatalf("свежий питомец не пустой: %s", r.Raw)
	}

	ws := connectWS(t, admin.Token)

	taskID := createTask(t, admin, deptID, "Задача для питомца", nil)
	ar := tasksAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/tasks/%d/archive", taskID), admin.Token, nil)
	requireStatus(t, ar, 200, "архив задачи")

	// 1. pet:update дошло по WS (хук асинхронный, комната user_<id>).
	ws.waitFrameMatch(t, "pet:update", func(f wsFrame) bool {
		id, _ := f.Obj()["user_id"].(float64)
		return int64(id) == admin.ID
	}, 15*time.Second)

	// 2. Начисления герою: +5 кудосов (кап task_closed) и +8 XP (xp_task).
	deadline := time.Now().Add(10 * time.Second)
	var kudos, xp float64
	for time.Now().Before(deadline) {
		pr := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", admin.Token, nil)
		requireStatus(t, pr, 200, "pet после закрытия")
		kudos, xp = pr.Num("kudos"), pr.Num("xp")
		if kudos == 5 && xp == 8 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if kudos != 5 || xp != 8 {
		t.Fatalf("начисления за закрытие: kudos=%v xp=%v, ожидалось 5/8", kudos, xp)
	}

	// 3. Рейтинг компании видит кудосы недели героя.
	rt := petsAPI.doJSON(t, http.MethodGet, "/api/pets/rating", admin.Token, nil)
	requireStatus(t, rt, 200, "рейтинг")
	me, _ := rt.JSON["me"].(map[string]any)
	if me == nil || me["kudos_week"].(float64) != 5 {
		t.Fatalf("kudos_week героя после закрытия: %v", me)
	}
}

// TestPetsUnitStoppedAwardsKudosAndXP — завершение юнита начисляет кудосы
// (1 за каждые 5 минут) и XP (1 за каждые 3 минуты) исполнителю. Старт
// юнита отодвигается на 30 минут назад PATCH'ем (юнит ещё активен — это
// разрешено), чтобы длительность при остановке была детерминированной.
func TestPetsUnitStoppedAwardsKudosAndXP(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)
	member := newMember(t, admin, companyID, roleEmployee)
	typeID := createUnitType(t, admin, uniq("Хук "))
	taskID := createTask(t, admin, deptID, "Задача для юнита", nil)

	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", member.Token, nil)

	unitID := startUnit(t, member, taskID, typeID, "юнит хука")
	pastStart := time.Now().UTC().Add(-30 * time.Minute).Format("2006-01-02T15:04:05")
	pr := tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/units/%d", unitID), member.Token,
		map[string]any{"datetime_start": pastStart})
	requireStatus(t, pr, 200, "отодвинуть старт юнита")
	stopUnit(t, member, unitID)

	deadline := time.Now().Add(10 * time.Second)
	var kudos, xp float64
	for time.Now().Before(deadline) {
		gr := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", member.Token, nil)
		requireStatus(t, gr, 200, "pet после юнита")
		kudos, xp = gr.Num("kudos"), gr.Num("xp")
		if kudos > 0 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	// 30 минут → 6 кудосов (30/5), 10 XP (30/3).
	if kudos != 6 || xp != 10 {
		t.Fatalf("начисления за 30-минутный юнит: kudos=%v xp=%v, ожидалось 6/10", kudos, xp)
	}
}

// TestPetsEvolutionCreatesPortalPost — сквозной сценарий petsvc → portalsvc:
// эволюция питомца (XP подводится под порог прямым UPDATE, кормление
// переводит через него) публикует в ленте корпоративного портала системный
// пост system_kind='pet_evolved' от имени владельца (gRPC CreateSystemPost,
// fire-and-forget — поэтому пост ждём поллингом).
func TestPetsEvolutionCreatesPortalPost(t *testing.T) {
	_, member, _ := petsCompany(t)

	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", member.Token, nil)
	requireStatus(t, r, 200, "GET /pet (создание)")

	// Кудосы на кормление + XP на 1 меньше порога «Малыша» (StageXP[1]=40).
	if _, err := db.Exec(dbCtx(t), `UPDATE pets SET kudos=10, xp=39 WHERE user_id=$1`, member.ID); err != nil {
		t.Fatalf("подводка XP к порогу: %v", err)
	}

	fr := petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", member.Token, nil)
	requireStatus(t, fr, 200, "кормление до эволюции")
	if !fr.Bool("evolved") {
		t.Fatalf("кормление не привело к эволюции: %s", fr.Raw)
	}

	// Пост публикуется асинхронной горутиной — ждём его в ленте портала.
	deadline := time.Now().Add(10 * time.Second)
	for {
		pr := portalAPI.doJSON(t, http.MethodGet, "/api/portal/posts", member.Token, nil)
		requireStatus(t, pr, 200, "лента портала")
		for _, raw := range pr.List("posts") {
			post, _ := raw.(map[string]any)
			if post == nil {
				continue
			}
			kind, _ := post["system_kind"].(string)
			if kind != "pet_evolved" {
				continue
			}
			if int64(post["author_id"].(float64)) != member.ID {
				t.Fatalf("автор системного поста: %v, ожидался %d", post["author_id"], member.ID)
			}
			body, _ := post["body"].(string)
			if !strings.Contains(body, "Малыш") {
				t.Fatalf("тело поста без названия стадии: %q", body)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("системный пост pet_evolved не появился в ленте за 10с")
		}
		time.Sleep(300 * time.Millisecond)
	}
}

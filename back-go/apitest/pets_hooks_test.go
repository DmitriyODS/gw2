package apitest

import (
	"fmt"
	"net/http"
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
		if kudos == 5 && xp == 12 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	// 5 кудосов и 8 XP за задачу; XP свежего (полного сил) питомца множится
	// настроением ×1.5 → 12.
	if kudos != 5 || xp != 12 {
		t.Fatalf("начисления за закрытие: kudos=%v xp=%v, ожидалось 5/12", kudos, xp)
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
	// 30 минут → 6 кудосов (30/5) и 10 XP (30/3), помноженные на настроение
	// свежего питомца (×1.5) → 15.
	if kudos != 6 || xp != 15 {
		t.Fatalf("начисления за 30-минутный юнит: kudos=%v xp=%v, ожидалось 6/15", kudos, xp)
	}
}

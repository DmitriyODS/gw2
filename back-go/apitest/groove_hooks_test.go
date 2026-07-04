package apitest

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestGrooveTaskClosedHookAcrossServices — сквозной сценарий геймификации:
// закрытие задачи в tasksvc → gRPC-хук OnTaskClosed (fire-and-forget после
// коммита) → groovesvc пишет событие ленты task_closed, начисляет герою
// грувы (+5) и XP (+8), двигает недельный рейд; событие feed:new доезжает
// клиентам через мост шлюза (комната all).
func TestGrooveTaskClosedHookAcrossServices(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)

	// Питомец существует заранее — начисления пойдут в него.
	r := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", admin.Token, nil)
	requireStatus(t, r, 200, "GET /pet")
	if r.Num("beans") != 0 || r.Num("xp") != 0 {
		t.Fatalf("свежий питомец не пустой: %s", r.Raw)
	}

	ws := connectWS(t, admin.Token)

	taskID := createTask(t, admin, deptID, "Задача для Грувика", nil)
	ar := tasksAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/tasks/%d/archive", taskID), admin.Token, nil)
	requireStatus(t, ar, 200, "архив задачи")

	// 1. Событие ленты task_closed (хук асинхронный — поллим).
	var eventID int64
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && eventID == 0 {
		fr := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/feed", admin.Token, nil)
		requireStatus(t, fr, 200, "лента")
		for _, it := range fr.List("items") {
			m := it.(map[string]any)
			pl, _ := m["payload"].(map[string]any)
			if m["kind"] == "task_closed" && pl != nil && int64(pl["task_id"].(float64)) == taskID {
				eventID = int64(m["id"].(float64))
				if u, _ := m["user"].(map[string]any); u == nil || int64(u["id"].(float64)) != admin.ID {
					t.Fatalf("герой закрытия — не актор: %v", m)
				}
			}
		}
		if eventID == 0 {
			time.Sleep(300 * time.Millisecond)
		}
	}
	if eventID == 0 {
		t.Fatalf("событие task_closed не появилось в ленте за 15с")
	}

	// 2. feed:new дошло по WS (комната all, company_id в payload).
	ws.waitFrameMatch(t, "feed:new", func(f wsFrame) bool {
		id, _ := f.Obj()["id"].(float64)
		return int64(id) == eventID
	}, 10*time.Second)

	// 3. Начисления герою: +5 грувов (кап task_closed) и +8 XP (xp_task).
	deadline = time.Now().Add(10 * time.Second)
	var beans, xp float64
	for time.Now().Before(deadline) {
		pr := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", admin.Token, nil)
		requireStatus(t, pr, 200, "pet после закрытия")
		beans, xp = pr.Num("beans"), pr.Num("xp")
		if beans == 5 && xp == 8 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if beans != 5 || xp != 8 {
		t.Fatalf("начисления за закрытие: beans=%v xp=%v, ожидалось 5/8", beans, xp)
	}

	// 4. Реакция коллеги на событие даёт герою ещё +1 грув.
	member := newMember(t, admin, companyID, roleEmployee)
	rr := grooveAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/groove/feed/%d/reactions", eventID), member.Token,
		map[string]any{"emoji": "🔥"})
	requireStatus(t, rr, 200, "реакция коллеги")
	pr := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", admin.Token, nil)
	if pr.Num("beans") != 6 {
		t.Fatalf("реакция не начислила грув герою: beans=%v", pr.Num("beans"))
	}

	// 5. Рейд недели видит закрытие: progress ≥ 1, личный вклад героя ≥ 1.
	rd := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/raid", admin.Token, nil)
	requireStatus(t, rd, 200, "рейд")
	if rd.Num("progress") < 1 {
		t.Fatalf("закрытие не попало в прогресс рейда: %s", rd.Raw)
	}
	if rd.Num("my_closed") < 1 {
		t.Fatalf("личный вклад героя в рейд не посчитан: %s", rd.Raw)
	}

	// 6. Wrapped «Моя неделя» героя видит закрытую задачу.
	wr := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/wrapped", admin.Token, nil)
	requireStatus(t, wr, 200, "wrapped")
	if wr.Num("closed") < 1 {
		t.Fatalf("wrapped не видит закрытие: %s", wr.Raw)
	}
	// AI выключен → ai_phrase отсутствует (null), а не ошибка.
	if v, ok := wr.JSON["ai_phrase"]; ok && v != nil {
		if s, _ := v.(string); s != "" {
			t.Fatalf("ai_phrase при выключенном AI: %v", v)
		}
	}
}

// TestGrooveKudosAwardsBeans — кудос начисляет адресату +2 грува (источник
// kudos, дневной кап), а карточка рейтинга считает признание недели.
func TestGrooveKudosAwardsBeans(t *testing.T) {
	admin, member, _ := grooveCompany(t)

	// Питомец адресата создаётся при начислении сам (GetOrCreate) — заранее
	// его НЕ создаём: проверяем и этот путь.
	r := grooveAPI.doJSON(t, http.MethodPost, "/api/groove/kudos", admin.Token,
		map[string]any{"to_user_id": member.ID, "category": "helped", "text": "выручил с релизом"})
	requireStatus(t, r, 201, "кудос")

	deadline := time.Now().Add(5 * time.Second)
	var beans float64
	for time.Now().Before(deadline) {
		pr := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/pet", member.Token, nil)
		requireStatus(t, pr, 200, "pet адресата")
		if beans = pr.Num("beans"); beans == 2 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if beans != 2 {
		t.Fatalf("кудос не начислил 2 грува адресату: beans=%v", beans)
	}

	// Рейтинг: счётчик признания недели у адресата = 1.
	rt := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/rating", member.Token, nil)
	requireStatus(t, rt, 200, "рейтинг")
	me, _ := rt.JSON["me"].(map[string]any)
	if me == nil || me["kudos_week"].(float64) != 1 {
		t.Fatalf("kudos_week адресата: %v", me)
	}
}

// TestGrooveLiveActiveUnits — «Сейчас в эфире»: активный юнит виден компании,
// после остановки — исчезает.
func TestGrooveLiveActiveUnits(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)
	member := newMember(t, admin, companyID, roleEmployee)
	typeID := createUnitType(t, admin, uniq("Эфир "))
	taskID := createTask(t, admin, deptID, "Задача в эфире", nil)

	unitID := startUnit(t, member, taskID, typeID, "юнит в эфире")
	r := grooveAPI.doJSON(t, http.MethodGet, "/api/groove/live", admin.Token, nil)
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
	r = grooveAPI.doJSON(t, http.MethodGet, "/api/groove/live", admin.Token, nil)
	for _, it := range r.List("items") {
		if int64(it.(map[string]any)["unit_id"].(float64)) == unitID {
			t.Fatalf("остановленный юнит остался в live: %s", r.Raw)
		}
	}
}

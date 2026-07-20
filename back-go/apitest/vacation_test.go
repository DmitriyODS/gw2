package apitest

// API-тесты режима «в отпуске» (users.on_vacation): тумблер в профиле
// (PATCH /users/me), гарды tasksvc на создание/правку/закрытие задач и старт
// юнитов (остановка активного юнита разрешена) и «отпуск грувика» в petsvc —
// заморозка потребностей/болезней и запрет ухода/поглаживаний.

import (
	"fmt"
	"net/http"
	"testing"
)

// setVacation — переключить режим отпуска актора через реальный PATCH /users/me.
func setVacation(t *testing.T, a *actor, on bool) {
	t.Helper()
	r := authAPI.doJSON(t, http.MethodPatch, "/api/users/me", a.Token, map[string]any{"on_vacation": on})
	requireStatus(t, r, 200, fmt.Sprintf("PATCH /users/me on_vacation=%v", on))
	if r.Bool("on_vacation") != on {
		t.Fatalf("on_vacation не переключился: %s", r.Raw)
	}
}

// Отпуск закрывает мутации задач и старт юнитов кодом ON_VACATION 403; личные
// и read-действия работают, остановить активный юнит можно, выключение
// отпуска возвращает всё как было.
func TestVacationBlocksTasksAndUnits(t *testing.T) {
	admin, _, deptID := newTaskCompany(t)
	typeID := createUnitType(t, admin, uniq("Тип "))
	taskID := createTask(t, admin, deptID, "До отпуска", nil)
	unitID := startUnit(t, admin, taskID, typeID, "юнит до отпуска")

	setVacation(t, admin, true)

	// Создание/правка/ответственный/этап/архив — 403 ON_VACATION.
	r := tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", admin.Token,
		map[string]any{"name": "В отпуске", "department_id": deptID})
	requireError(t, r, 403, "ON_VACATION", "создание задачи в отпуске")
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", taskID), admin.Token,
		map[string]any{"name": "Правка в отпуске"})
	requireError(t, r, 403, "ON_VACATION", "правка задачи в отпуске")
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d/responsible", taskID),
		admin.Token, map[string]any{"responsible_user_id": nil})
	requireError(t, r, 403, "ON_VACATION", "смена ответственного в отпуске")
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/archive", taskID), admin.Token, nil)
	requireError(t, r, 403, "ON_VACATION", "архивация в отпуске")

	// Чтение живо, активный юнит можно остановить, но новый не стартует.
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d", taskID), admin.Token, nil)
	requireStatus(t, r, 200, "чтение задачи в отпуске")
	stopUnit(t, admin, unitID)
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/units", taskID), admin.Token,
		map[string]any{"name": "юнит в отпуске", "unit_type_id": typeID})
	requireError(t, r, 403, "ON_VACATION", "старт юнита в отпуске")

	// Вернулся из отпуска — всё снова работает.
	setVacation(t, admin, false)
	createTask(t, admin, deptID, "После отпуска", nil)
	unitID = startUnit(t, admin, taskID, typeID, "юнит после отпуска")
	stopUnit(t, admin, unitID)
}

// Создатель компании явно проставляет и снимает отпуск сотруднику
// (PATCH /companies/:id/users/:userId {on_vacation}); эффект — тот же, что
// у личного тумблера. Не-создателю ручка закрыта.
func TestVacationSetByCompanyCreator(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)
	member := newMember(t, admin, companyID, roleEmployee)

	// Сотрудник этой ручкой не управляет — только создатель или супер-админ.
	r := authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/users/%d", companyID, admin.ID), member.Token,
		map[string]any{"on_vacation": true})
	requireError(t, r, 403, "FORBIDDEN", "отпуск ставит не создатель")

	// Создатель отправляет сотрудника в отпуск — задачи ему закрыты.
	r = authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/users/%d", companyID, member.ID), admin.Token,
		map[string]any{"on_vacation": true})
	requireStatus(t, r, 200, "создатель ставит отпуск")
	if !r.Bool("on_vacation") {
		t.Fatalf("отпуск не проставился: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", member.Token,
		map[string]any{"name": "В отпуске", "department_id": deptID})
	requireError(t, r, 403, "ON_VACATION", "задача сотрудника в отпуске")

	// Метка видна в списке участников (колонка «Отпуск» на фронте).
	r = authAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/companies/%d/members", companyID), admin.Token, nil)
	requireStatus(t, r, 200, "список участников")
	var membersList []map[string]any
	if err := jsonUnmarshal(r.Raw, &membersList); err != nil {
		t.Fatalf("список участников: ответ не массив: %s", r.Raw)
	}
	found := false
	for _, m := range membersList {
		if id, _ := m["id"].(float64); int64(id) == member.ID {
			found = true
			if m["on_vacation"] != true {
				t.Fatalf("участник без метки on_vacation: %v", m)
			}
		}
	}
	if !found {
		t.Fatalf("сотрудник не найден в списке участников: %s", r.Raw)
	}

	// Создатель снимает отпуск — сотрудник снова работает.
	r = authAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/companies/%d/users/%d", companyID, member.ID), admin.Token,
		map[string]any{"on_vacation": false})
	requireStatus(t, r, 200, "создатель снимает отпуск")
	createTask(t, member, deptID, "После отпуска", nil)
}

// В отпуске грувик заморожен: шкалы не тают, болезнь не наступает, действия
// владельца и поглаживания коллег отвечают PET_ON_VACATION, а DTO явно несёт
// метку on_vacation (её рисует фронт).
func TestVacationFreezesPet(t *testing.T) {
	admin, m, _ := petsCompany(t)

	r := petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET /pet")
	if r.Bool("on_vacation") {
		t.Fatalf("новый питомец не в отпуске: %s", r.Raw)
	}

	setVacation(t, m, true)

	// Сутки с лишним без ухода — без отпуска сытость была бы в нуле и питомец
	// болел бы истощением; в отпуске шкалы стоят на месте.
	agePetNeeds(t, m.ID, 30*60)
	r = petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", m.Token, nil)
	requireStatus(t, r, 200, "GET /pet в отпуске")
	if !r.Bool("on_vacation") {
		t.Fatalf("DTO не помечен on_vacation: %s", r.Raw)
	}
	needs, ok := r.JSON["needs"].(map[string]any)
	if !ok || needs["satiety"] != float64(100) {
		t.Fatalf("в отпуске шкалы не должны таять: %s", r.Raw)
	}
	if r.Bool("sick") {
		t.Fatalf("в отпуске питомец не заболевает: %s", r.Raw)
	}

	// Уход и приключения закрыты.
	grantKudos(t, m.ID, 100)
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
	requireError(t, r, 422, "PET_ON_VACATION", "кормление в отпуске")
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/adventure", m.Token, nil)
	requireError(t, r, 422, "PET_ON_VACATION", "приключение в отпуске")

	// Коллега не погладит отпускника (и кудосы владельцу не капнут).
	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", admin.Token, nil)
	grantKudos(t, admin.ID, 100)
	r = petsAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/pets/stroke/%d", m.ID), admin.Token, nil)
	requireError(t, r, 422, "PET_ON_VACATION", "поглаживание отпускника")

	setVacation(t, m, false)
	// После отпуска питомец жив-здоров и снова кормится.
	r = petsAPI.doJSON(t, http.MethodPost, "/api/pets/pet/feed", m.Token, nil)
	requireStatus(t, r, 200, "кормление после отпуска")
}

package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// ── Хелперы tasksvc ──────────────────────────────────────────────

// newTaskCompany — компания с админом-создателем и отделом (минимум для задач).
func newTaskCompany(t *testing.T) (admin *actor, companyID, deptID int64) {
	t.Helper()
	admin = newVerifiedUser(t)
	companyID = admin.createCompany(t, uniq("Задачи "))
	deptID = createDept(t, admin, uniq("Отдел "))
	return admin, companyID, deptID
}

func createDept(t *testing.T, a *actor, name string) int64 {
	t.Helper()
	r := tasksAPI.doJSON(t, http.MethodPost, "/api/departments", a.Token, map[string]any{"name": name})
	requireStatus(t, r, 201, "создание отдела "+name)
	return int64(r.Num("id"))
}

// createTask — задача с обязательными полями; extra — поверх.
func createTask(t *testing.T, a *actor, deptID int64, name string, extra map[string]any) int64 {
	t.Helper()
	body := map[string]any{"name": name, "department_id": deptID}
	for k, v := range extra {
		body[k] = v
	}
	r := tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", a.Token, body)
	requireStatus(t, r, 201, "создание задачи "+name)
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("создание задачи: нет id: %s", r.Raw)
	}
	return id
}

func createUnitType(t *testing.T, a *actor, name string) int64 {
	t.Helper()
	r := tasksAPI.doJSON(t, http.MethodPost, "/api/unit-types", a.Token, map[string]any{"name": name})
	requireStatus(t, r, 201, "создание типа юнита "+name)
	return int64(r.Num("id"))
}

// startUnit — старт юнита по задаче.
func startUnit(t *testing.T, a *actor, taskID, typeID int64, name string) int64 {
	t.Helper()
	r := tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/units", taskID), a.Token,
		map[string]any{"name": name, "unit_type_id": typeID})
	requireStatus(t, r, 201, "старт юнита "+name)
	return int64(r.Num("id"))
}

func stopUnit(t *testing.T, a *actor, unitID int64) {
	t.Helper()
	r := tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/units/%d/stop", unitID), a.Token, nil)
	requireStatus(t, r, 200, "остановка юнита")
}

// setUnitTimes — правка времени юнита (детерминированные часы для статистики).
func setUnitTimes(t *testing.T, a *actor, unitID int64, start, end string) {
	t.Helper()
	r := tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/units/%d", unitID), a.Token,
		map[string]any{"datetime_start": start, "datetime_end": end})
	requireStatus(t, r, 200, "правка времени юнита")
}

func taskListIDs(r apiResp) []int64 {
	items := r.List("items")
	out := make([]int64, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		id, _ := m["id"].(float64)
		out = append(out, int64(id))
	}
	return out
}

// ── CRUD задач, валидация, справочники, скоуп компании ───────────

func TestTasksCRUDAndValidation(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)

	// Валидация: без полей, мусорное имя, неизвестное поле.
	r := tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", admin.Token, map[string]any{})
	requireError(t, r, 400, "VALIDATION_ERROR", "задача без полей")
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", admin.Token,
		map[string]any{"name": strings.Repeat("ъ", 501), "department_id": deptID})
	requireError(t, r, 400, "VALIDATION_ERROR", "слишком длинное имя")
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", admin.Token,
		map[string]any{"name": "x", "department_id": deptID, "surprise": 1})
	requireError(t, r, 400, "VALIDATION_ERROR", "неизвестное поле")

	// Несуществующий отдел → 404, отдел чужой компании → 422.
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", admin.Token,
		map[string]any{"name": "x", "department_id": 99999999})
	requireError(t, r, 404, "DEPT_NOT_FOUND", "несуществующий отдел")
	stranger := newVerifiedUser(t)
	stranger.createCompany(t, uniq("Чужая "))
	foreignDept := createDept(t, stranger, uniq("Чужой отдел "))
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/tasks", admin.Token,
		map[string]any{"name": "x", "department_id": foreignDept})
	requireError(t, r, 422, "DEPT_FOREIGN", "отдел чужой компании")

	// Создание: ответственный по умолчанию — автор.
	taskID := createTask(t, admin, deptID, "Первая задача", map[string]any{"deadline": "2026-08-01"})
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d", taskID), admin.Token, nil)
	requireStatus(t, r, 200, "карточка задачи")
	if int64(r.Num("responsible_user_id")) != admin.ID || int64(r.Num("author_id")) != admin.ID {
		t.Fatalf("ответственный по умолчанию — автор: %s", r.Raw)
	}

	// Правка имени и дедлайна.
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", taskID), admin.Token,
		map[string]any{"name": "Переименована", "deadline": nil})
	requireStatus(t, r, 200, "правка задачи")
	if r.Str("name") != "Переименована" || r.JSON["deadline"] != nil {
		t.Fatalf("правка: %s", r.Raw)
	}

	// Ответственный: не член компании → 422; член — ок; null — снимается.
	outsider := newVerifiedUser(t)
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d/responsible", taskID),
		admin.Token, map[string]any{"responsible_user_id": outsider.ID})
	requireError(t, r, 422, "USER_FOREIGN", "ответственный вне компании")
	member := newMember(t, admin, companyID, roleEmployee)
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d/responsible", taskID),
		admin.Token, map[string]any{"responsible_user_id": member.ID})
	requireStatus(t, r, 200, "смена ответственного")
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d/responsible", taskID),
		admin.Token, map[string]any{"responsible_user_id": nil})
	requireStatus(t, r, 200, "снятие ответственного")
	if r.JSON["responsible_user_id"] != nil {
		t.Fatalf("ответственный не снялся: %s", r.Raw)
	}

	// Этапы: создание (менеджер+), назначение, чужой этап → 422.
	rs := tasksAPI.doJSON(t, http.MethodPost, "/api/stages", admin.Token,
		map[string]any{"name": uniq("Этап "), "color": "teal"})
	requireStatus(t, rs, 201, "создание этапа")
	stageID := int64(rs.Num("id"))
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d/stage", taskID),
		admin.Token, map[string]any{"stage_id": stageID})
	requireStatus(t, r, 200, "назначение этапа")
	if st, _ := r.JSON["stage"].(map[string]any); st == nil || int64(st["id"].(float64)) != stageID {
		t.Fatalf("этап не назначился: %s", r.Raw)
	}
	rs = tasksAPI.doJSON(t, http.MethodPost, "/api/stages", stranger.Token,
		map[string]any{"name": uniq("Чужой этап ")})
	requireStatus(t, rs, 201, "чужой этап")
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/tasks/%d/stage", taskID),
		admin.Token, map[string]any{"stage_id": int64(rs.Num("id"))})
	requireError(t, r, 422, "STAGE_FOREIGN", "этап чужой компании")

	// Комментарии: создание, правка автором, чужой сотрудник без прав → 403,
	// менеджер может (роль из токена, а не из users).
	cr := tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/comments", taskID),
		member.Token, map[string]any{"text": "первый комментарий"})
	requireStatus(t, cr, 201, "создание комментария")
	commentID := int64(cr.Num("id"))
	r = tasksAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/tasks/%d/comments/%d", taskID, commentID), member.Token,
		map[string]any{"text": "поправил сам"})
	requireStatus(t, r, 200, "правка своего комментария")
	other := newMember(t, admin, companyID, roleEmployee)
	r = tasksAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/tasks/%d/comments/%d", taskID, commentID), other.Token,
		map[string]any{"text": "взлом"})
	requireError(t, r, 403, "FORBIDDEN", "правка чужого комментария сотрудником")
	manager := newMember(t, admin, companyID, roleManager)
	r = tasksAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/tasks/%d/comments/%d", taskID, commentID), manager.Token,
		map[string]any{"text": "менеджер поправил"})
	requireStatus(t, r, 200, "правка чужого комментария менеджером")
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d/comments", taskID), member.Token, nil)
	requireStatus(t, r, 200, "список комментариев")
	if len(r.List("items")) != 1 {
		t.Fatalf("ожидался один комментарий: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/tasks/%d/comments/%d", taskID, commentID), manager.Token, nil)
	requireStatus(t, r, 200, "удаление комментария менеджером")

	// Архив/восстановление.
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/archive", taskID), admin.Token, nil)
	requireStatus(t, r, 200, "архивирование")
	if !r.Bool("is_archived") {
		t.Fatalf("is_archived не проставился: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/archive", taskID), admin.Token, nil)
	requireError(t, r, 422, "ALREADY_ARCHIVED", "повторный архив")
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/restore", taskID), admin.Token, nil)
	requireStatus(t, r, 200, "восстановление")
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/restore", taskID), admin.Token, nil)
	requireError(t, r, 422, "NOT_ARCHIVED", "повторное восстановление")

	// Удаление.
	r = tasksAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/tasks/%d", taskID), admin.Token, nil)
	requireStatus(t, r, 200, "удаление задачи")
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d", taskID), admin.Token, nil)
	requireStatus(t, r, 404, "карточка удалённой")

	// Без токена → 401.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/tasks", "", nil)
	requireError(t, r, 401, "UNAUTHORIZED", "tasks без токена")
}

// TestTasksCrossCompanyIsolation — задачи и юниты по id недоступны из другой
// компании: любой доступ отвечает 404, не раскрывая существование.
func TestTasksCrossCompanyIsolation(t *testing.T) {
	adminA, _, deptA := newTaskCompany(t)
	typeA := createUnitType(t, adminA, uniq("Тип "))
	taskA := createTask(t, adminA, deptA, "Секретная задача A", nil)
	unitA := startUnit(t, adminA, taskA, typeA, "юнит A")

	// Актор из компании B (админ/менеджер у себя — не важно: чужая компания).
	adminB := newVerifiedUser(t)
	adminB.createCompany(t, uniq("Компания B "))

	for _, tc := range []struct {
		method, path string
		body         map[string]any
	}{
		{http.MethodGet, fmt.Sprintf("/api/tasks/%d", taskA), nil},
		{http.MethodPatch, fmt.Sprintf("/api/tasks/%d", taskA), map[string]any{"name": "hack"}},
		{http.MethodDelete, fmt.Sprintf("/api/tasks/%d", taskA), nil},
		{http.MethodPost, fmt.Sprintf("/api/tasks/%d/archive", taskA), nil},
		{http.MethodPost, fmt.Sprintf("/api/tasks/%d/restore", taskA), nil},
		{http.MethodPut, fmt.Sprintf("/api/tasks/%d/color", taskA), map[string]any{"color": "red"}},
		{http.MethodPost, fmt.Sprintf("/api/tasks/%d/favorite", taskA), nil},
		{http.MethodGet, fmt.Sprintf("/api/tasks/%d/units", taskA), nil},
		{http.MethodPost, fmt.Sprintf("/api/tasks/%d/units", taskA), map[string]any{"name": "x", "unit_type_id": typeA}},
		{http.MethodPatch, fmt.Sprintf("/api/tasks/%d/responsible", taskA), map[string]any{"responsible_user_id": nil}},
		{http.MethodPatch, fmt.Sprintf("/api/tasks/%d/stage", taskA), map[string]any{"stage_id": nil}},
		{http.MethodGet, fmt.Sprintf("/api/tasks/%d/contributors", taskA), nil},
		{http.MethodGet, fmt.Sprintf("/api/tasks/%d/comments", taskA), nil},
		{http.MethodPost, fmt.Sprintf("/api/tasks/%d/comments", taskA), map[string]any{"text": "x"}},
		{http.MethodPatch, fmt.Sprintf("/api/units/%d", unitA), map[string]any{"name": "hack"}},
		{http.MethodPost, fmt.Sprintf("/api/units/%d/stop", unitA), nil},
		{http.MethodDelete, fmt.Sprintf("/api/units/%d", unitA), nil},
	} {
		var body any
		if tc.body != nil {
			body = tc.body
		}
		rr := tasksAPI.doJSON(t, tc.method, tc.path, adminB.Token, body)
		if rr.Status != 404 {
			t.Fatalf("%s %s из чужой компании: ожидался 404, получен %d: %s",
				tc.method, tc.path, rr.Status, rr.Raw)
		}
	}

	// Список задач компании B пуст — задачи A не просачиваются.
	r := tasksAPI.doJSON(t, http.MethodGet, "/api/tasks", adminB.Token, nil)
	requireStatus(t, r, 200, "список задач B")
	if len(taskListIDs(r)) != 0 {
		t.Fatalf("в списке B чужие задачи: %s", r.Raw)
	}

	// Задача A по-прежнему цела и юнит активен.
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d", taskA), adminA.Token, nil)
	requireStatus(t, r, 200, "задача A после атак")
	stopUnit(t, adminA, unitA)
}

// ── Список: поиск, избранное, личные цвета, пагинация ────────────

func TestTasksListSearchFavoritesColors(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)
	member := newMember(t, admin, companyID, roleEmployee)

	t1 := createTask(t, admin, deptID, "Синхронизация платёжного шлюза", nil)
	t2 := createTask(t, admin, deptID, "Отчёт по продажам", nil)
	t3 := createTask(t, admin, deptID, "Ревью платёжного кода", nil)

	// Поиск (AI выключен → LIKE по названию, без регистра).
	r := tasksAPI.doJSON(t, http.MethodGet, "/api/tasks?search="+urlQuery("платёжн"), admin.Token, nil)
	requireStatus(t, r, 200, "поиск LIKE")
	ids := taskListIDs(r)
	if len(ids) != 2 {
		t.Fatalf("поиск: ожидались 2 задачи, получено %v: %s", ids, r.Raw)
	}

	// Избранное: toggle туда-обратно + вкладка favorites.
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/favorite", t2), admin.Token, nil)
	requireStatus(t, r, 200, "в избранное")
	if !r.Bool("is_favorite") {
		t.Fatalf("is_favorite=false после включения: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/tasks?tab=favorites", admin.Token, nil)
	if ids := taskListIDs(r); len(ids) != 1 || ids[0] != t2 {
		t.Fatalf("вкладка favorites: %v", ids)
	}
	// Избранное личное: у сотрудника вкладка пуста.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/tasks?tab=favorites", member.Token, nil)
	if ids := taskListIDs(r); len(ids) != 0 {
		t.Fatalf("избранное протекло другому пользователю: %v", ids)
	}
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/favorite", t2), admin.Token, nil)
	if r.Bool("is_favorite") {
		t.Fatalf("повторный toggle должен снять избранное: %s", r.Raw)
	}

	// Личный цвет: у автора красный, у другого пользователя — null.
	r = tasksAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/tasks/%d/color", t1), admin.Token,
		map[string]any{"color": "red"})
	requireStatus(t, r, 200, "установка цвета")
	r = tasksAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/tasks/%d/color", t1), admin.Token,
		map[string]any{"color": "magenta"})
	requireError(t, r, 400, "VALIDATION_ERROR", "недопустимый цвет")
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d", t1), admin.Token, nil)
	if r.Str("color") != "red" {
		t.Fatalf("цвет автора: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d", t1), member.Token, nil)
	requireStatus(t, r, 200, "задача глазами сотрудника")
	if r.JSON["color"] != nil {
		t.Fatalf("личный цвет виден другому пользователю: %s", r.Raw)
	}

	// Пагинация.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/tasks?per_page=2&page=1&sort=created_at", admin.Token, nil)
	requireStatus(t, r, 200, "страница 1")
	if len(r.List("items")) != 2 || r.Num("total") != 3 || r.Num("per_page") != 2 {
		t.Fatalf("пагинация стр.1: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/tasks?per_page=2&page=2&sort=created_at", admin.Token, nil)
	if len(r.List("items")) != 1 || r.Num("page") != 2 {
		t.Fatalf("пагинация стр.2: %s", r.Raw)
	}
	_ = t3
}

// ── Юниты: жизненный цикл, права, каскад типа ────────────────────

func TestUnitsLifecycleAndRoles(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)
	employee := newMember(t, admin, companyID, roleEmployee)
	manager := newMember(t, admin, companyID, roleManager)
	typeID := createUnitType(t, admin, uniq("Разработка "))
	taskID := createTask(t, admin, deptID, "Задача с юнитами", nil)

	// Валидация старта.
	r := tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/units", taskID),
		employee.Token, map[string]any{"name": "x"})
	requireError(t, r, 400, "VALIDATION_ERROR", "юнит без типа")
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/units", taskID),
		employee.Token, map[string]any{"name": "x", "unit_type_id": 99999999})
	requireError(t, r, 404, "TYPE_NOT_FOUND", "несуществующий тип")

	// Один активный юнит: повторный старт → 409.
	unit1 := startUnit(t, employee, taskID, typeID, "первый")
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/units/active", employee.Token, nil)
	requireStatus(t, r, 200, "активный юнит")
	if int64(r.Num("id")) != unit1 {
		t.Fatalf("активный юнит: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/units", taskID),
		employee.Token, map[string]any{"name": "второй", "unit_type_id": typeID})
	requireError(t, r, 409, "ACTIVE_UNIT_EXISTS", "второй активный юнит")

	// Задачу с активным юнитом нельзя архивировать.
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/archive", taskID), admin.Token, nil)
	requireError(t, r, 422, "HAS_ACTIVE_UNIT", "архив с активным юнитом")

	// Чужой юнит: сотрудник не может стоп/правку/удаление, менеджер — может.
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/units/%d/stop", unit1), manager.Token, nil)
	requireStatus(t, r, 200, "менеджер остановил чужой юнит")
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/units/%d/stop", unit1), employee.Token, nil)
	requireError(t, r, 422, "ALREADY_STOPPED", "повторная остановка")

	unit2 := startUnit(t, employee, taskID, typeID, "второй")
	other := newMember(t, admin, companyID, roleEmployee)
	r = tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/units/%d/stop", unit2), other.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "сотрудник остановил чужой юнит")
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/units/%d", unit2), other.Token,
		map[string]any{"name": "переименован чужим"})
	requireError(t, r, 403, "FORBIDDEN", "правка чужого юнита сотрудником")
	r = tasksAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/units/%d", unit2), other.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "удаление чужого юнита сотрудником")

	// Владелец правит своё время; is_edited поднимается.
	setUnitTimes(t, employee, unit2, "2026-07-01T10:00:00", "2026-07-01T11:30:00")
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d/units", taskID), employee.Token, nil)
	requireStatus(t, r, 200, "юниты задачи")
	var units []map[string]any
	if err := jsonUnmarshal(r.Raw, &units); err != nil {
		t.Fatalf("юниты задачи: %v: %s", err, r.Raw)
	}
	foundEdited := false
	for _, u := range units {
		if int64(u["id"].(float64)) == int64(unit2) && u["is_edited"] == true {
			foundEdited = true
		}
	}
	if !foundEdited {
		t.Fatalf("is_edited не проставился: %s", r.Raw)
	}

	// Чужой тип юнита (другая компания) не привязывается ни при правке.
	strangerAdmin := newVerifiedUser(t)
	strangerAdmin.createCompany(t, uniq("Чужая "))
	foreignType := createUnitType(t, strangerAdmin, uniq("Чужой тип "))
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/units/%d", unit2), employee.Token,
		map[string]any{"unit_type_id": foreignType})
	requireError(t, r, 422, "TYPE_FOREIGN", "чужой тип юнита при правке")

	// Менеджер удаляет чужой юнит.
	r = tasksAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/units/%d", unit2), manager.Token, nil)
	requireStatus(t, r, 200, "менеджер удалил чужой юнит")

	// unit-types: гейт менеджера, дубликат, каскад удаления юнитов.
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/unit-types", employee.Token, map[string]any{"name": "x"})
	requireStatus(t, r, 403, "тип юнита сотрудником")
	dupName := uniq("Дубль ")
	dupID := createUnitType(t, manager, dupName)
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/unit-types", manager.Token, map[string]any{"name": dupName})
	requireError(t, r, 409, "DUPLICATE", "дубликат типа юнита")
	r = tasksAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/unit-types/%d", dupID), manager.Token,
		map[string]any{"name": dupName + " v2"})
	requireStatus(t, r, 200, "переименование типа")

	// Каскад: юнит с типом dupID исчезает вместе с типом.
	cascade := startUnit(t, employee, taskID, dupID, "обречённый")
	stopUnit(t, employee, cascade)
	r = tasksAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/unit-types/%d", dupID), manager.Token, nil)
	requireStatus(t, r, 200, "удаление типа")
	r = tasksAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/tasks/%d/units", taskID), employee.Token, nil)
	units = nil
	_ = jsonUnmarshal(r.Raw, &units)
	for _, u := range units {
		if int64(u["id"].(float64)) == cascade {
			t.Fatalf("юнит пережил каскадное удаление типа: %s", r.Raw)
		}
	}

	// Отделы и этапы: ролевые гейты + дубликаты + reorder.
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/departments", employee.Token, map[string]any{"name": "x"})
	requireStatus(t, r, 403, "отдел сотрудником")
	deptName := uniq("Отдел дубль ")
	d2 := createDept(t, manager, deptName)
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/departments", manager.Token, map[string]any{"name": deptName})
	requireError(t, r, 409, "DUPLICATE", "дубликат отдела")
	r = tasksAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/departments/%d", d2), manager.Token, nil)
	requireStatus(t, r, 200, "удаление отдела")

	r = tasksAPI.doJSON(t, http.MethodPost, "/api/stages", employee.Token, map[string]any{"name": "x"})
	requireStatus(t, r, 403, "этап сотрудником")
	s1 := tasksAPI.doJSON(t, http.MethodPost, "/api/stages", manager.Token, map[string]any{"name": uniq("С1 ")})
	requireStatus(t, s1, 201, "этап 1")
	s2 := tasksAPI.doJSON(t, http.MethodPost, "/api/stages", manager.Token,
		map[string]any{"name": uniq("С2 "), "color": "violet"})
	requireStatus(t, s2, 201, "этап 2")
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/stages", manager.Token,
		map[string]any{"name": uniq("С3 "), "color": "#ff0000"})
	requireError(t, r, 400, "VALIDATION_ERROR", "этап с hex-цветом")
	r = tasksAPI.doJSON(t, http.MethodPatch, "/api/stages/reorder", manager.Token,
		map[string]any{"ids": []int64{int64(s2.Num("id")), int64(s1.Num("id"))}})
	requireStatus(t, r, 200, "reorder этапов")
	var stages []map[string]any
	if err := jsonUnmarshal(r.Raw, &stages); err != nil || len(stages) < 2 {
		t.Fatalf("reorder: %v %s", err, r.Raw)
	}
	if int64(stages[0]["id"].(float64)) != int64(s2.Num("id")) {
		t.Fatalf("порядок этапов после reorder: %s", r.Raw)
	}
}

// ── Статистика ───────────────────────────────────────────────────

func TestStatsConsistencyAndExport(t *testing.T) {
	admin, companyID, deptID := newTaskCompany(t)
	employee := newMember(t, admin, companyID, roleEmployee)
	manager := newMember(t, admin, companyID, roleManager)
	typeID := createUnitType(t, admin, uniq("Аналитика "))

	t1 := createTask(t, admin, deptID, "Статистика 1", nil)
	t2 := createTask(t, admin, deptID, "Статистика 2", nil)

	// Юниты сотрудника с детерминированными часами: 2ч по t1, 1ч по t2.
	u1 := startUnit(t, employee, t1, typeID, "работа 1")
	stopUnit(t, employee, u1)
	setUnitTimes(t, employee, u1, "2026-07-01T10:00:00", "2026-07-01T12:00:00")
	u2 := startUnit(t, employee, t2, typeID, "работа 2")
	stopUnit(t, employee, u2)
	setUnitTimes(t, employee, u2, "2026-07-01T13:00:00", "2026-07-01T14:00:00")

	// Закрываем обе задачи.
	for _, id := range []int64{t1, t2} {
		r := tasksAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/tasks/%d/archive", id), admin.Token, nil)
		requireStatus(t, r, 200, "архив для статистики")
	}

	period := "?from=2020-01-01&to=2030-01-01"

	// Common: received=2, closed=2, remaining=0; часы по задачам и сотрудникам.
	r := tasksAPI.doJSON(t, http.MethodGet, "/api/stats/common"+period, admin.Token, nil)
	requireStatus(t, r, 200, "stats common")
	tasksM, _ := r.JSON["tasks"].(map[string]any)
	if tasksM["received"].(float64) != 2 || tasksM["closed"].(float64) != 2 ||
		tasksM["remaining"].(float64) != 0 || tasksM["debt"].(float64) != 0 {
		t.Fatalf("common metrics: %s", r.Raw)
	}
	byHours := r.List("tasks_by_hours")
	hoursByTask := map[int64]float64{}
	for _, it := range byHours {
		m := it.(map[string]any)
		hoursByTask[int64(m["task_id"].(float64))] = m["total_hours"].(float64)
	}
	if hoursByTask[t1] != 2 || hoursByTask[t2] != 1 {
		t.Fatalf("tasks_by_hours: %v", hoursByTask)
	}
	byEmp := r.List("tasks_by_employees")
	if len(byEmp) != 1 {
		t.Fatalf("tasks_by_employees: %s", r.Raw)
	}
	emp := byEmp[0].(map[string]any)
	if int64(emp["user_id"].(float64)) != employee.ID ||
		emp["total_hours"].(float64) != 3 || emp["tasks_count"].(float64) != 2 {
		t.Fatalf("часы сотрудника: %v", emp)
	}

	// Extended: по типам юнитов и отделам + календарь.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/extended"+period, admin.Token, nil)
	requireStatus(t, r, 200, "stats extended")
	byTypes := r.List("by_unit_types")
	if len(byTypes) != 1 {
		t.Fatalf("by_unit_types: %s", r.Raw)
	}
	bt := byTypes[0].(map[string]any)
	if bt["total_hours"].(float64) != 3 || bt["tasks_count"].(float64) != 2 {
		t.Fatalf("by_unit_types значения: %v", bt)
	}
	byDepts := r.List("by_departments")
	if len(byDepts) != 1 || byDepts[0].(map[string]any)["tasks_count"].(float64) != 2 {
		t.Fatalf("by_departments: %s", r.Raw)
	}
	calOK := false
	for _, it := range r.List("calendar") {
		m := it.(map[string]any)
		if m["date"] == "2026-07-01" && m["total_hours"].(float64) == 3 {
			calOK = true
		}
	}
	if !calOK {
		t.Fatalf("календарь без дня юнитов: %s", r.Raw)
	}

	// Profile сотрудника: 3 часа, 2 задачи.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/profile"+period, employee.Token, nil)
	requireStatus(t, r, 200, "stats profile")
	if r.Num("total_hours") != 3 || r.Num("tasks_count") != 2 {
		t.Fatalf("profile: %s", r.Raw)
	}

	// user-tasks: себя — можно; чужие часы сотруднику — 403; менеджеру — можно.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/user-tasks"+period, employee.Token, nil)
	requireStatus(t, r, 200, "user-tasks self")
	if r.Num("tasks_count") != 2 {
		t.Fatalf("user-tasks self: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/stats/user-tasks%s&user_id=%d", period, admin.ID), employee.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "чужие часы сотруднику")
	r = tasksAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/stats/user-tasks%s&user_id=%d", period, employee.ID), manager.Token, nil)
	requireStatus(t, r, 200, "чужие часы менеджеру")
	if r.Num("tasks_count") != 2 {
		t.Fatalf("user-tasks менеджером: %s", r.Raw)
	}

	// responsibles (закрытые у ответственного) и employees (менеджер+).
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/responsibles", admin.Token, nil)
	requireStatus(t, r, 200, "responsibles")
	respOK := false
	var responsibles []map[string]any
	_ = jsonUnmarshal(r.Raw, &responsibles)
	for _, m := range responsibles {
		if int64(m["user_id"].(float64)) == admin.ID && m["closed_count"].(float64) == 2 {
			respOK = true
		}
	}
	if !respOK {
		t.Fatalf("responsibles: %s", r.Raw)
	}
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/employees", employee.Token, nil)
	requireStatus(t, r, 403, "employees сотрудником")
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/employees", manager.Token, nil)
	requireStatus(t, r, 200, "employees менеджером")

	// Экспорт xlsx: права (менеджер+) + сигнатура PK.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/common/export"+period, employee.Token, nil)
	requireStatus(t, r, 403, "экспорт сотрудником")
	for _, path := range []string{"/api/stats/common/export", "/api/stats/extended/export"} {
		r = tasksAPI.doJSON(t, http.MethodGet, path+period, manager.Token, nil)
		requireStatus(t, r, 200, "экспорт "+path)
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, xlsxMime) {
			t.Fatalf("%s: content-type %q", path, ct)
		}
		if len(r.Raw) < 500 || string(r.Raw[:2]) != "PK" {
			t.Fatalf("%s: не xlsx", path)
		}
	}

	// Мусорный период → 400.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/common?from=не-дата", admin.Token, nil)
	requireError(t, r, 400, "VALIDATION_ERROR", "мусорный период")
}

// Регрессия: один аккаунт в двух компаниях — часы юнита, отработанного в
// компании A, не должны протекать в статистику компании B (скоуп по
// units.company_id, а не по членству автора).
func TestStatsNoCrossCompanyLeak(t *testing.T) {
	adminA, companyA, deptA := newTaskCompany(t)
	employee := newMember(t, adminA, companyA, roleEmployee)
	typeA := createUnitType(t, adminA, uniq("Аналитика A "))

	// Сотрудник отрабатывает 2 часа по задаче компании A.
	taskA := createTask(t, adminA, deptA, "Задача A", nil)
	employee.switchCompany(t, companyA)
	uA := startUnit(t, employee, taskA, typeA, "работа A")
	stopUnit(t, employee, uA)
	setUnitTimes(t, employee, uA, "2026-07-01T10:00:00", "2026-07-01T12:00:00")

	// Второй компанией того же сотрудника делаем членом (юнитов в ней нет).
	adminB, companyB, _ := newTaskCompany(t)
	addToCompany(t, adminB, companyB, employee, roleEmployee)

	period := "?from=2020-01-01&to=2030-01-01"

	// Статистика компании B: сотрудник в списке часов появляться не должен —
	// его 2 часа принадлежат компании A.
	r := tasksAPI.doJSON(t, http.MethodGet, "/api/stats/common"+period, adminB.Token, nil)
	requireStatus(t, r, 200, "stats common B")
	for _, it := range r.List("tasks_by_employees") {
		m := it.(map[string]any)
		if int64(m["user_id"].(float64)) == employee.ID {
			t.Fatalf("часы сотрудника из компании A протекли в компанию B: %s", r.Raw)
		}
	}

	// А в компании A его 2 часа на месте.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/stats/common"+period, adminA.Token, nil)
	requireStatus(t, r, 200, "stats common A")
	found := false
	for _, it := range r.List("tasks_by_employees") {
		m := it.(map[string]any)
		if int64(m["user_id"].(float64)) == employee.ID {
			found = true
			if m["total_hours"].(float64) != 2 {
				t.Fatalf("часы сотрудника в компании A: %v", m)
			}
		}
	}
	if !found {
		t.Fatalf("сотрудник пропал из статистики своей компании A: %s", r.Raw)
	}
}

// ── YouGile: статус и вебхук без внешних вызовов ─────────────────

func TestYougileStatusAndWebhook(t *testing.T) {
	admin, companyID, _ := newTaskCompany(t)
	employee := newMember(t, admin, companyID, roleEmployee)

	// Не настроен: company_enabled=false, аккаунт не подключён.
	r := tasksAPI.doJSON(t, http.MethodGet, "/api/yougile/status", employee.Token, nil)
	requireStatus(t, r, 200, "yougile status")
	if r.Bool("company_enabled") || r.Bool("connected") {
		t.Fatalf("yougile не должен быть настроен: %s", r.Raw)
	}

	// Настройки компании — только администратор.
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/yougile/company-settings", employee.Token, nil)
	requireStatus(t, r, 403, "company-settings сотрудником")
	r = tasksAPI.doJSON(t, http.MethodGet, "/api/yougile/company-settings", admin.Token, nil)
	requireStatus(t, r, 200, "company-settings админом")
	if r.Bool("enabled") {
		t.Fatalf("интеграция не должна быть включена: %s", r.Raw)
	}

	// Вебхук: неверный секрет → 404 (у компании секрет вообще не настроен).
	payload := map[string]any{"event": "task-updated", "payload": map[string]any{"id": "x"}}
	r = tasksAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/yougile/webhook/%d/wrong-secret", companyID), "", payload)
	requireError(t, r, 404, "NOT_FOUND", "вебхук с неверным секретом")

	// Вебхук: валидная форма, но несуществующая компания → 404.
	r = tasksAPI.doJSON(t, http.MethodPost, "/api/yougile/webhook/99999999/secret", "", payload)
	requireError(t, r, 404, "NOT_FOUND", "вебхук несуществующей компании")
}

package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// ── CRUD ежедневников ────────────────────────────────────────────

func TestDiaryCRUD(t *testing.T) {
	owner := newVerifiedUser(t)

	// Валидация имени.
	r := diaryAPI.doJSON(t, http.MethodPost, "/api/diaries", owner.Token, map[string]any{"name": "   "})
	requireError(t, r, 400, "VALIDATION", "создание без имени")
	r = diaryAPI.doJSON(t, http.MethodPost, "/api/diaries", owner.Token,
		map[string]any{"name": strings.Repeat("ъ", 121)})
	requireError(t, r, 400, "VALIDATION", "слишком длинное имя")

	// Создание/чтение/переименование.
	id := createDiary(t, owner, "Мой список")
	r = diaryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/diaries/%d", id), owner.Token, nil)
	requireStatus(t, r, 200, "карточка ежедневника")
	if r.Str("name") != "Мой список" || r.Bool("shared") || !r.Bool("can_check") {
		t.Fatalf("карточка ежедневника: %s", r.Raw)
	}
	r = diaryAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/diaries/%d", id), owner.Token,
		map[string]any{"name": "Переименован"})
	requireStatus(t, r, 200, "переименование")

	// Список «Мои» содержит ежедневник.
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries", owner.Token, nil)
	requireStatus(t, r, 200, "список моих")
	if !diaryListHas(r, id) {
		t.Fatalf("список моих не содержит %d: %s", id, r.Raw)
	}

	// Чужой пользователь не видит и не правит (404, не 403 — не раскрываем).
	other := newVerifiedUser(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, fmt.Sprintf("/api/diaries/%d", id)},
		{http.MethodPatch, fmt.Sprintf("/api/diaries/%d", id)},
		{http.MethodDelete, fmt.Sprintf("/api/diaries/%d", id)},
		{http.MethodGet, fmt.Sprintf("/api/diaries/%d/records", id)},
		{http.MethodPost, fmt.Sprintf("/api/diaries/%d/records", id)},
		{http.MethodGet, fmt.Sprintf("/api/diaries/%d/shares", id)},
		{http.MethodPost, fmt.Sprintf("/api/diaries/%d/shares", id)},
		{http.MethodGet, fmt.Sprintf("/api/diaries/%d/members", id)},
		{http.MethodGet, fmt.Sprintf("/api/diaries/%d/export", id)},
	} {
		body := map[string]any{"name": "x", "entry_date": "2026-07-01", "title": "x"}
		rr := diaryAPI.doJSON(t, tc.method, tc.path, other.Token, body)
		if rr.Status != 404 {
			t.Fatalf("%s %s чужим: ожидался 404, получен %d: %s", tc.method, tc.path, rr.Status, rr.Raw)
		}
	}

	// Без токена → 401.
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries", "", nil)
	requireError(t, r, 401, "UNAUTHORIZED", "diaries без токена")

	// Удаление владельцем.
	r = diaryAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/diaries/%d", id), owner.Token, nil)
	requireStatus(t, r, 200, "удаление ежедневника")
	r = diaryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/diaries/%d", id), owner.Token, nil)
	requireStatus(t, r, 404, "карточка удалённого")
}

func diaryListHas(r apiResp, id int64) bool {
	for _, it := range r.List("diaries") {
		m, _ := it.(map[string]any)
		if v, _ := m["id"].(float64); int64(v) == id {
			return true
		}
	}
	return false
}

func diaryFromList(r apiResp, id int64) map[string]any {
	for _, it := range r.List("diaries") {
		m, _ := it.(map[string]any)
		if v, _ := m["id"].(float64); int64(v) == id {
			return m
		}
	}
	return nil
}

// ── Записи: валидация, диапазоны, архив, поиск ───────────────────

func TestDiaryEntries(t *testing.T) {
	owner := newVerifiedUser(t)
	diaryID := createDiary(t, owner, "Записи")

	// Валидация: без даты, без названия, длинное название.
	r := diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records", diaryID),
		owner.Token, map[string]any{"title": "Без даты"})
	requireError(t, r, 400, "VALIDATION", "запись без даты")
	r = diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records", diaryID),
		owner.Token, map[string]any{"entry_date": "2026-07-01", "title": "  "})
	requireError(t, r, 400, "VALIDATION", "запись без названия")
	r = diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records", diaryID),
		owner.Token, map[string]any{"entry_date": "2026-07-01", "title": strings.Repeat("я", 201)})
	requireError(t, r, 400, "VALIDATION", "слишком длинное название")

	// Время вне диапазона приводится к 0..1439.
	eClamped := createEntry(t, owner, diaryID, "2026-07-01", "Клэмп времени",
		map[string]any{"start_min": -25, "end_min": 5000})
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records/%d", diaryID, eClamped), owner.Token, nil)
	requireStatus(t, r, 200, "чтение записи")
	if r.Num("start_min") != 0 || r.Num("end_min") != 1439 {
		t.Fatalf("клэмп времени 0..1439: %s", r.Raw)
	}

	// Три дня: 1, 2 и 10 июля.
	e1 := eClamped
	e2 := createEntry(t, owner, diaryID, "2026-07-02", "Вторая", map[string]any{"description": "уникальное-слово-абракадабра"})
	e3 := createEntry(t, owner, diaryID, "2026-07-10", "Третья", nil)

	// Диапазон недели 29.06–06.07 (to не включается) — только e1 и e2.
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records?from=2026-06-29&to=2026-07-06", diaryID), owner.Token, nil)
	requireStatus(t, r, 200, "записи за диапазон")
	got := entryIDs(r)
	if len(got) != 2 || got[0] != e1 || got[1] != e2 {
		t.Fatalf("диапазон: ожидались [%d %d], получено %v: %s", e1, e2, got, r.Raw)
	}

	// Правка записи: смена дня переносит в другой диапазон.
	r = diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d", diaryID, e2), owner.Token,
		map[string]any{"entry_date": "2026-07-11", "title": "Вторая (перенесена)", "description": "уникальное-слово-абракадабра"})
	requireStatus(t, r, 200, "правка записи")
	if r.Str("entry_date") != "2026-07-11" {
		t.Fatalf("правка: entry_date не сменился: %s", r.Raw)
	}

	// Поиск по описанию (сквозной ILIKE).
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records?search=абракадабра", diaryID), owner.Token, nil)
	requireStatus(t, r, 200, "поиск")
	if got := entryIDs(r); len(got) != 1 || got[0] != e2 {
		t.Fatalf("поиск: ожидалась запись %d, получено %v", e2, got)
	}

	// done → запись уходит из активных во вкладку архива.
	r = diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d/done", diaryID, e3), owner.Token,
		map[string]any{"done": true})
	requireStatus(t, r, 200, "отметка done")
	if !r.Bool("done") {
		t.Fatalf("done не проставился: %s", r.Raw)
	}
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records?from=2026-07-06&to=2026-07-20", diaryID), owner.Token, nil)
	if got := entryIDs(r); len(got) != 1 || got[0] != e2 {
		t.Fatalf("активные после done: ожидалась только %d, получено %v", e2, got)
	}
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records?archived=1", diaryID), owner.Token, nil)
	if got := entryIDs(r); len(got) != 1 || got[0] != e3 {
		t.Fatalf("архив: ожидалась %d, получено %v", e3, got)
	}

	// undone → возвращается в активные.
	r = diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d/done", diaryID, e3), owner.Token,
		map[string]any{"done": false})
	requireStatus(t, r, 200, "снятие done")
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records?archived=1", diaryID), owner.Token, nil)
	if got := entryIDs(r); len(got) != 0 {
		t.Fatalf("архив после undone должен быть пуст, получено %v", got)
	}

	// Счётчики в списке ежедневников.
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries", owner.Token, nil)
	d := diaryFromList(r, diaryID)
	if d == nil || d["active_count"].(float64) != 3 || d["done_count"].(float64) != 0 {
		t.Fatalf("счётчики списка: ожидалось 3/0: %v", d)
	}

	// Несуществующая запись → 404; запись чужого ежедневника → 404.
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records/99999999", diaryID), owner.Token, nil)
	requireStatus(t, r, 404, "несуществующая запись")
	otherDiary := createDiary(t, owner, "Другой")
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records/%d", otherDiary, e1), owner.Token, nil)
	requireStatus(t, r, 404, "запись не из этого ежедневника")

	// Удаление записи.
	r = diaryAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/diaries/%d/records/%d", diaryID, e1), owner.Token, nil)
	requireStatus(t, r, 200, "удаление записи")
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records/%d", diaryID, e1), owner.Token, nil)
	requireStatus(t, r, 404, "чтение удалённой записи")
}

// TestDiaryEntryDateTimezone — день записи в RFC3339 с не-UTC смещением не
// должен «уплывать» через границу суток: клиент прислал 5 июля своей зоны —
// запись обязана лечь на 5 июля.
func TestDiaryEntryDateTimezone(t *testing.T) {
	owner := newVerifiedUser(t)
	diaryID := createDiary(t, owner, "Часовые пояса")

	cases := []struct {
		in   string
		want string
	}{
		{"2026-07-05", "2026-07-05"},                // плоская дата
		{"2026-07-05T00:30:00+05:00", "2026-07-05"}, // раннее утро восточной зоны
		{"2026-07-05T23:30:00-03:00", "2026-07-05"}, // поздний вечер западной зоны
		{"2026-07-05T12:00:00Z", "2026-07-05"},      // UTC-полдень
	}
	for _, tc := range cases {
		r := diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records", diaryID),
			owner.Token, map[string]any{"entry_date": tc.in, "title": "TZ " + tc.in})
		requireStatus(t, r, 201, "создание записи "+tc.in)
		if got := r.Str("entry_date"); got != tc.want {
			t.Errorf("entry_date %q: сохранился день %q, ожидался %q", tc.in, got, tc.want)
		}
	}

	// То же при переносе на день в RFC3339.
	id := createEntry(t, owner, diaryID, "2026-07-01", "Переносимая", nil)
	r := diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d/move", diaryID, id), owner.Token,
		map[string]any{"entry_date": "2026-07-09T00:15:00+03:00"})
	requireStatus(t, r, 200, "move с RFC3339-датой")
	if got := r.Str("entry_date"); got != "2026-07-09" {
		t.Errorf("move: сохранился день %q, ожидался 2026-07-09", got)
	}
}

// ── Адресный шаринг и can_check ──────────────────────────────────

func TestDiaryMemberSharingCanCheck(t *testing.T) {
	owner := newVerifiedUser(t)
	reader := newVerifiedUser(t)
	diaryID := createDiary(t, owner, "Расшаренный")
	entryID := createEntry(t, owner, diaryID, "2026-07-03", "Общая задача", nil)

	memberPath := fmt.Sprintf("/api/diaries/%d/members", diaryID)

	// Нельзя поделиться с собой и с несуществующим пользователем.
	r := diaryAPI.doJSON(t, http.MethodPost, memberPath, owner.Token,
		map[string]any{"user_id": owner.ID})
	requireError(t, r, 400, "VALIDATION", "шаринг самому себе")
	r = diaryAPI.doJSON(t, http.MethodPost, memberPath, owner.Token,
		map[string]any{"user_id": 99999999})
	requireError(t, r, 404, "NOT_FOUND", "шаринг несуществующему")
	r = diaryAPI.doJSON(t, http.MethodPost, memberPath, owner.Token, map[string]any{})
	requireError(t, r, 400, "VALIDATION", "шаринг без user_id")

	// Шаринг без права отметки.
	r = diaryAPI.doJSON(t, http.MethodPost, memberPath, owner.Token,
		map[string]any{"user_id": reader.ID, "can_check": false})
	requireStatus(t, r, 201, "адресный шаринг")

	// Адресат видит ежедневник: shared=true, can_check=false.
	r = diaryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/diaries/%d", diaryID), reader.Token, nil)
	requireStatus(t, r, 200, "чужой ежедневник у адресата")
	if !r.Bool("shared") || r.Bool("can_check") {
		t.Fatalf("у адресата ожидалось shared=true can_check=false: %s", r.Raw)
	}

	// Вкладка «Поделились» со счётчиками.
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries?tab=shared", reader.Token, nil)
	requireStatus(t, r, 200, "вкладка Поделились")
	d := diaryFromList(r, diaryID)
	if d == nil {
		t.Fatalf("ежедневник не появился во вкладке Поделились: %s", r.Raw)
	}
	if d["can_check"].(bool) || d["active_count"].(float64) != 1 || d["done_count"].(float64) != 0 {
		t.Fatalf("вкладка Поделились: ожидалось can_check=false, 1/0: %v", d)
	}
	// Во вкладке «Мои» адресата его нет.
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries", reader.Token, nil)
	if diaryListHas(r, diaryID) {
		t.Fatalf("чужой ежедневник не должен попадать во вкладку Мои")
	}

	// Чтение записей и экспорт адресату доступны.
	r = diaryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/diaries/%d/records", diaryID), reader.Token, nil)
	requireStatus(t, r, 200, "записи у адресата")
	r = diaryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/diaries/%d/export", diaryID), reader.Token, nil)
	requireStatus(t, r, 200, "экспорт у адресата")

	// Одна запись тоже читается адресатом.
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records/%d", diaryID, entryID), reader.Token, nil)
	requireStatus(t, r, 200, "одна запись у адресата")

	// Без can_check отметить done нельзя → 403 FORBIDDEN.
	donePath := fmt.Sprintf("/api/diaries/%d/records/%d/done", diaryID, entryID)
	r = diaryAPI.doJSON(t, http.MethodPatch, donePath, reader.Token, map[string]any{"done": true})
	requireError(t, r, 403, "FORBIDDEN", "done без can_check")

	// Редактировать не может никто, кроме владельца (в т.ч. адресат с can_check).
	r = diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d", diaryID, entryID), reader.Token,
		map[string]any{"entry_date": "2026-07-03", "title": "Взлом"})
	requireStatus(t, r, 404, "правка записи адресатом")
	r = diaryAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/diaries/%d/records/%d", diaryID, entryID), reader.Token, nil)
	requireStatus(t, r, 404, "удаление записи адресатом")
	r = diaryAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/diaries/%d/records", diaryID), reader.Token,
		map[string]any{"entry_date": "2026-07-03", "title": "Чужая запись"})
	requireStatus(t, r, 404, "создание записи адресатом")
	// Управлять шарингом тоже не может.
	r = diaryAPI.doJSON(t, http.MethodPost, memberPath, reader.Token,
		map[string]any{"user_id": owner.ID})
	requireStatus(t, r, 404, "шаринг адресатом")

	// Upsert: повторный POST с can_check=true обновляет право.
	r = diaryAPI.doJSON(t, http.MethodPost, memberPath, owner.Token,
		map[string]any{"user_id": reader.ID, "can_check": true})
	requireStatus(t, r, 201, "upsert can_check")
	r = diaryAPI.doJSON(t, http.MethodGet, memberPath, owner.Token, nil)
	requireStatus(t, r, 200, "список адресатов")
	members := r.List("members")
	if len(members) != 1 {
		t.Fatalf("ожидался один адресат: %s", r.Raw)
	}
	if m := members[0].(map[string]any); !m["can_check"].(bool) {
		t.Fatalf("после upsert ожидался can_check=true: %v", m)
	}

	// Теперь адресат может закрывать и переоткрывать записи.
	r = diaryAPI.doJSON(t, http.MethodPatch, donePath, reader.Token, map[string]any{"done": true})
	requireStatus(t, r, 200, "done с can_check")
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries?tab=shared", reader.Token, nil)
	d = diaryFromList(r, diaryID)
	if d == nil || d["done_count"].(float64) != 1 || d["active_count"].(float64) != 0 {
		t.Fatalf("счётчики после done: ожидалось 0/1: %v", d)
	}

	// Отзыв доступа: адресат теряет ежедневник.
	r = diaryAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/diaries/%d/members/%d", diaryID, reader.ID), owner.Token, nil)
	requireStatus(t, r, 200, "отзыв адресного доступа")
	r = diaryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/diaries/%d", diaryID), reader.Token, nil)
	requireStatus(t, r, 404, "доступ после отзыва")
	r = diaryAPI.doJSON(t, http.MethodPatch, donePath, reader.Token, map[string]any{"done": false})
	requireStatus(t, r, 404, "done после отзыва")

	// Повторный шаринг и удаление ежедневника владельцем: вкладка «Поделились»
	// адресата пустеет (каскад чистит связи).
	r = diaryAPI.doJSON(t, http.MethodPost, memberPath, owner.Token,
		map[string]any{"user_id": reader.ID, "can_check": true})
	requireStatus(t, r, 201, "повторный шаринг")
	r = diaryAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/diaries/%d", diaryID), owner.Token, nil)
	requireStatus(t, r, 200, "удаление расшаренного ежедневника")
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries?tab=shared", reader.Token, nil)
	requireStatus(t, r, 200, "вкладка Поделились после удаления")
	if diaryListHas(r, diaryID) {
		t.Fatalf("удалённый ежедневник остался во вкладке Поделились: %s", r.Raw)
	}
}

// ── Перенос и ручной порядок ─────────────────────────────────────

func TestDiaryMoveAndReorder(t *testing.T) {
	owner := newVerifiedUser(t)
	diaryA := createDiary(t, owner, "Список А")
	diaryB := createDiary(t, owner, "Список Б")

	e1 := createEntry(t, owner, diaryA, "2026-07-06", "Первая", map[string]any{"start_min": 600})
	e2 := createEntry(t, owner, diaryA, "2026-07-06", "Вторая", map[string]any{"start_min": 540})
	e3 := createEntry(t, owner, diaryA, "2026-07-06", "Третья", nil)

	listDay := func(diaryID int64) []int64 {
		r := diaryAPI.doJSON(t, http.MethodGet,
			fmt.Sprintf("/api/diaries/%d/records?from=2026-07-06&to=2026-07-07", diaryID), owner.Token, nil)
		requireStatus(t, r, 200, "записи дня")
		return entryIDs(r)
	}

	// Без ручного порядка: без времени — первыми, затем по времени начала.
	if got := listDay(diaryA); !equalIDs(got, []int64{e3, e2, e1}) {
		t.Fatalf("сортировка по умолчанию: ожидалось [%d %d %d], получено %v", e3, e2, e1, got)
	}

	// Reorder задаёт позиции 1..N — порядок выдачи меняется.
	r := diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records/reorder", diaryA),
		owner.Token, map[string]any{"entry_date": "2026-07-06", "ids": []int64{e1, e3, e2}})
	requireStatus(t, r, 200, "reorder")
	if got := listDay(diaryA); !equalIDs(got, []int64{e1, e3, e2}) {
		t.Fatalf("после reorder: ожидалось [%d %d %d], получено %v", e1, e3, e2, got)
	}

	// Reorder без даты → 400; не-владельцем → 404.
	r = diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records/reorder", diaryA),
		owner.Token, map[string]any{"ids": []int64{e1}})
	requireError(t, r, 400, "VALIDATION", "reorder без даты")
	other := newVerifiedUser(t)
	r = diaryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/diaries/%d/records/reorder", diaryA),
		other.Token, map[string]any{"entry_date": "2026-07-06", "ids": []int64{e1}})
	requireStatus(t, r, 404, "reorder не-владельцем")

	// Move: на другой день внутри списка.
	r = diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d/move", diaryA, e1), owner.Token,
		map[string]any{"entry_date": "2026-07-07"})
	requireStatus(t, r, 200, "move на другой день")
	if r.Str("entry_date") != "2026-07-07" {
		t.Fatalf("move: entry_date не сменился: %s", r.Raw)
	}

	// Move: в другой свой ежедневник.
	r = diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d/move", diaryA, e2), owner.Token,
		map[string]any{"diary_id": diaryB})
	requireStatus(t, r, 200, "move в другой ежедневник")
	if int64(r.Num("diary_id")) != diaryB {
		t.Fatalf("move: diary_id не сменился: %s", r.Raw)
	}
	if got := listDay(diaryB); !equalIDs(got, []int64{e2}) {
		t.Fatalf("запись не появилась в списке Б: %v", got)
	}

	// Move в чужой ежедневник → 404.
	foreign := createDiary(t, other, "Чужой список")
	r = diaryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/diaries/%d/records/%d/move", diaryA, e3), owner.Token,
		map[string]any{"diary_id": foreign})
	requireStatus(t, r, 404, "move в чужой ежедневник")
}

func equalIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ── Публичные ссылки ─────────────────────────────────────────────

func TestDiaryPublicShares(t *testing.T) {
	owner := newVerifiedUser(t)
	diaryID := createDiary(t, owner, "Публичный")
	createEntry(t, owner, diaryID, "2026-07-04", "Видимая запись", nil)

	sharePath := fmt.Sprintf("/api/diaries/%d/shares", diaryID)

	// Создание ссылки.
	r := diaryAPI.doJSON(t, http.MethodPost, sharePath, owner.Token, nil)
	requireStatus(t, r, 201, "создание публичной ссылки")
	code := r.Str("code")
	shareID := int64(r.Num("id"))
	if code == "" || shareID == 0 {
		t.Fatalf("ссылка без code/id: %s", r.Raw)
	}

	// Список ссылок владельца.
	r = diaryAPI.doJSON(t, http.MethodGet, sharePath, owner.Token, nil)
	requireStatus(t, r, 200, "список ссылок")
	if len(r.List("shares")) != 1 {
		t.Fatalf("ожидалась одна ссылка: %s", r.Raw)
	}

	// Публичный просмотр БЕЗ авторизации.
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries/shared/"+code, "", nil)
	requireStatus(t, r, 200, "публичная карточка")
	if !r.Bool("shared") {
		t.Fatalf("публичная карточка должна быть shared: %s", r.Raw)
	}
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries/shared/"+code+"/records", "", nil)
	requireStatus(t, r, 200, "публичные записи")
	if len(entryIDs(r)) != 1 {
		t.Fatalf("публичные записи: ожидалась 1: %s", r.Raw)
	}
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries/shared/"+code+"/export", "", nil)
	requireStatus(t, r, 200, "публичный экспорт")
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, xlsxMime) {
		t.Fatalf("публичный экспорт: content-type %q", ct)
	}

	// Мусорный код → 404.
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries/shared/deadbeef", "", nil)
	requireStatus(t, r, 404, "мусорный код")

	// Отзыв ссылки: код перестаёт работать.
	r = diaryAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("%s/%d", sharePath, shareID), owner.Token, nil)
	requireStatus(t, r, 200, "отзыв ссылки")
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries/shared/"+code, "", nil)
	requireStatus(t, r, 404, "отозванный код")
	r = diaryAPI.doJSON(t, http.MethodGet, "/api/diaries/shared/"+code+"/records", "", nil)
	requireStatus(t, r, 404, "записи по отозванному коду")
}

// ── Bulk-delete и экспорт ────────────────────────────────────────

func TestDiaryBulkDeleteAndExport(t *testing.T) {
	owner := newVerifiedUser(t)
	diaryID := createDiary(t, owner, "Массовые операции")
	e1 := createEntry(t, owner, diaryID, "2026-07-08", "Раз", nil)
	e2 := createEntry(t, owner, diaryID, "2026-07-08", "Два", nil)
	e3 := createEntry(t, owner, diaryID, "2026-07-08", "Три", nil)

	// Чужая запись в ids не удаляется (guard по diary_id).
	stranger := newVerifiedUser(t)
	foreignDiary := createDiary(t, stranger, "Чужое")
	foreignEntry := createEntry(t, stranger, foreignDiary, "2026-07-08", "Не трожь", nil)

	r := diaryAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/diaries/%d/records/bulk-delete", diaryID), owner.Token,
		map[string]any{"ids": []int64{e1, e2, foreignEntry}})
	requireStatus(t, r, 200, "bulk-delete")
	if r.Num("deleted") != 2 {
		t.Fatalf("bulk-delete: ожидалось 2 удалённых, тело: %s", r.Raw)
	}
	// Чужая запись жива.
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/records/%d", foreignDiary, foreignEntry), stranger.Token, nil)
	requireStatus(t, r, 200, "чужая запись после bulk-delete")

	// Экспорт владельца: 200 + xlsx content-type (+ ids-фильтр).
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/export?from=2026-07-08&to=2026-07-09", diaryID), owner.Token, nil)
	requireStatus(t, r, 200, "экспорт")
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, xlsxMime) {
		t.Fatalf("экспорт: content-type %q", ct)
	}
	if len(r.Raw) < 500 || string(r.Raw[:2]) != "PK" {
		t.Fatalf("экспорт: тело не похоже на xlsx (%d байт)", len(r.Raw))
	}
	r = diaryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/diaries/%d/export?ids=%d", diaryID, e3), owner.Token, nil)
	requireStatus(t, r, 200, "экспорт по ids")
}

package apitest

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Хелперы calendarsvc ──────────────────────────────────────────

func createCalendar(t *testing.T, a *actor, name string) int64 {
	t.Helper()
	r := calendarAPI.doJSON(t, http.MethodPost, "/api/calendars", a.Token, map[string]any{"name": name})
	requireStatus(t, r, 201, "создание календаря "+name)
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("создание календаря: нет id: %s", r.Raw)
	}
	return id
}

// createEvent — запись календаря с обязательным event_at.
func createEvent(t *testing.T, a *actor, calID int64, eventAt string, data map[string]any) int64 {
	t.Helper()
	if data == nil {
		data = map[string]any{}
	}
	r := calendarAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/calendars/%d/records", calID),
		a.Token, map[string]any{"event_at": eventAt, "data": data})
	requireStatus(t, r, 201, "создание события "+eventAt)
	return int64(r.Num("id"))
}

// ── Структура: близнец реестров + флаги карточки и условная видимость ──

func TestCalendarStructureAndFieldFlags(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Календари "))
	employee := newMember(t, admin, companyID, roleEmployee)

	// Структура — только администратор.
	r := calendarAPI.doJSON(t, http.MethodPost, "/api/calendars", employee.Token,
		map[string]any{"name": "x"})
	requireError(t, r, 403, "FORBIDDEN", "создание календаря сотрудником")
	r = calendarAPI.doJSON(t, http.MethodPost, "/api/calendars", admin.Token,
		map[string]any{"name": " "})
	requireError(t, r, 400, "VALIDATION", "календарь без имени")

	calID := createCalendar(t, admin, "Мероприятия")

	// Неизвестный тип поля → 400.
	r = calendarAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/calendars/%d/fields", calID),
		admin.Token, map[string]any{"fields": []map[string]any{{"label": "x", "type": "wat"}}})
	requireError(t, r, 400, "VALIDATION", "неизвестный тип поля календаря")

	// Флаги show_in_table/show_in_card и условная видимость сохраняются.
	fields := putFields(t, calendarAPI, "/api/calendars", admin, calID, []map[string]any{
		{"label": "Онлайн", "type": "checkbox", "show_in_table": true, "show_in_card": true},
		{"label": "Тема", "type": "text", "show_in_table": true, "show_in_card": false},
	})
	var onlineID int64
	for _, f := range fields {
		if f["label"] == "Онлайн" {
			onlineID = int64(f["id"].(float64))
			if f["show_in_table"] != true || f["show_in_card"] != true {
				t.Fatalf("флаги видимости не сохранились: %v", f)
			}
		}
		if f["label"] == "Тема" && f["show_in_card"] != false {
			t.Fatalf("show_in_card=false не сохранился: %v", f)
		}
	}
	// Поле «Ссылка» видно только при Онлайн=true (условная видимость — фронтовая,
	// сервер обязан хранить visible_field_id/visible_value).
	fields = putFields(t, calendarAPI, "/api/calendars", admin, calID, []map[string]any{
		{"id": onlineID, "label": "Онлайн", "type": "checkbox", "show_in_table": true, "show_in_card": true},
		{"label": "Ссылка", "type": "link", "show_in_card": true,
			"visible_field_id": onlineID, "visible_value": "true"},
	})
	found := false
	for _, f := range fields {
		if f["label"] == "Ссылка" {
			found = true
			if int64(f["visible_field_id"].(float64)) != onlineID || f["visible_value"] != "true" {
				t.Fatalf("условная видимость не сохранилась: %v", f)
			}
		}
	}
	if !found {
		t.Fatalf("поле со ссылкой не вернулось: %v", fields)
	}
	// Перечитанный календарь хранит те же флаги.
	r = calendarAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/calendars/%d", calID), employee.Token, nil)
	requireStatus(t, r, 200, "карточка календаря")
	for _, fv := range r.List("fields") {
		f := fv.(map[string]any)
		if f["label"] == "Ссылка" && f["visible_value"] != "true" {
			t.Fatalf("visible_value потерялся при чтении: %v", f)
		}
	}

	// Скоуп: чужая компания → 404; супер-админ → 403.
	adminB := newVerifiedUser(t)
	adminB.createCompany(t, uniq("Другая "))
	r = calendarAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/calendars/%d", calID), adminB.Token, nil)
	requireStatus(t, r, 404, "чужой календарь")
	root := newSuperAdmin(t)
	r = calendarAPI.doJSON(t, http.MethodGet, "/api/calendars", root.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "календари супер-админом")
}

// ── Записи: event_at обязателен и без секунд, диапазоны, поиск ───

func TestCalendarEntries(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("События "))
	employee := newMember(t, admin, companyID, roleEmployee)
	calID := createCalendar(t, admin, "План")
	fields := putFields(t, calendarAPI, "/api/calendars", admin, calID, []map[string]any{
		{"label": "Тема", "type": "text"},
		{"label": "Зал", "type": "select", "config": map[string]any{"options": []string{"большой", "малый"}}},
	})
	themeK := fieldKey(t, fields, "Тема")
	hallK := fieldKey(t, fields, "Зал")

	// event_at обязателен; мусорный формат — тоже 400.
	r := calendarAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/calendars/%d/records", calID),
		employee.Token, map[string]any{"data": map[string]any{themeK: "без даты"}})
	requireError(t, r, 400, "VALIDATION", "событие без event_at")
	r = calendarAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/calendars/%d/records", calID),
		employee.Token, map[string]any{"event_at": "31.12.2026 10:00", "data": map[string]any{}})
	requireError(t, r, 400, "VALIDATION", "мусорный event_at")

	// Секунды нормализуются (обрезаются до минуты).
	r = calendarAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/calendars/%d/records", calID),
		employee.Token, map[string]any{"event_at": "2026-07-05T10:00:45Z",
			"data": map[string]any{themeK: "Планёрка"}})
	requireStatus(t, r, 201, "событие с секундами")
	e1 := int64(r.Num("id"))
	if got := r.Str("event_at"); !strings.HasPrefix(got, "2026-07-05T10:00:00") {
		t.Fatalf("секунды не нормализованы: %q", got)
	}

	// Валидация значений полей — как в реестрах.
	r = calendarAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/calendars/%d/records", calID),
		employee.Token, map[string]any{"event_at": "2026-07-05T12:00:00Z",
			"data": map[string]any{hallK: "несуществующий"}})
	requireError(t, r, 400, "VALIDATION", "select вне вариантов")

	e2 := createEvent(t, employee, calID, "2026-07-06T09:30:00Z",
		map[string]any{themeK: "Ретроспектива", hallK: "большой"})
	e3 := createEvent(t, employee, calID, "2026-07-20T15:00:00Z",
		map[string]any{themeK: "Демо уникальное-слово-квартал"})

	// Диапазон недели: from включительно, to не включается.
	r = calendarAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/calendars/%d/records?from=%s&to=%s", calID,
			urlQuery("2026-07-05T00:00:00Z"), urlQuery("2026-07-12T00:00:00Z")),
		employee.Token, nil)
	requireStatus(t, r, 200, "события за неделю")
	if got := recordIDs(r); !equalIDs(got, []int64{e1, e2}) {
		t.Fatalf("диапазон недели: ожидались [%d %d], получено %v", e1, e2, got)
	}

	// Поиск сквозной.
	r = calendarAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/calendars/%d/records?search=%s", calID, urlQuery("уникальное-слово")),
		employee.Token, nil)
	if got := recordIDs(r); !equalIDs(got, []int64{e3}) {
		t.Fatalf("поиск: %v", got)
	}

	// Правка: перенос даты (тоже без секунд) и смена данных.
	r = calendarAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/calendars/%d/records/%d", calID, e1), employee.Token,
		map[string]any{"event_at": "2026-07-07T08:15:30Z", "data": map[string]any{themeK: "Планёрка (перенос)"}})
	requireStatus(t, r, 200, "перенос события")
	if got := r.Str("event_at"); !strings.HasPrefix(got, "2026-07-07T08:15:00") {
		t.Fatalf("перенос: %q", got)
	}

	// Чужая компания записи не видит и не правит.
	adminB := newVerifiedUser(t)
	adminB.createCompany(t, uniq("Б "))
	r = calendarAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/calendars/%d/records", calID), adminB.Token, nil)
	requireStatus(t, r, 404, "чужие события")
	r = calendarAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/calendars/%d/records/%d", calID, e1), adminB.Token,
		map[string]any{"event_at": "2026-07-07T08:15:00Z"})
	requireStatus(t, r, 404, "правка чужого события")

	// Запись не из этого календаря → 404.
	otherCal := createCalendar(t, admin, "Другой план")
	r = calendarAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/calendars/%d/records/%d", otherCal, e1), employee.Token, nil)
	requireStatus(t, r, 404, "событие не из этого календаря")

	// Экспорт за период: только события периода; image/file не выгружаются
	// (регресс общего ядра — проверено в реестрах).
	r = calendarAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/calendars/%d/export?from=%s&to=%s", calID,
			urlQuery("2026-07-05T00:00:00Z"), urlQuery("2026-07-12T00:00:00Z")),
		employee.Token, nil)
	requireStatus(t, r, 200, "экспорт за период")
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, xlsxMime) {
		t.Fatalf("экспорт: content-type %q", ct)
	}
	if string(r.Raw[:2]) != "PK" {
		t.Fatalf("экспорт: не xlsx")
	}

	// bulk-delete с чужим id — не задевает другой календарь.
	foreign := createEvent(t, employee, otherCal, "2026-07-06T10:00:00Z", nil)
	r = calendarAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/calendars/%d/records/bulk-delete", calID), employee.Token,
		map[string]any{"ids": []int64{e2, e3, foreign}})
	requireStatus(t, r, 200, "bulk-delete событий")
	if r.Num("deleted") != 2 {
		t.Fatalf("bulk-delete: %s", r.Raw)
	}
	r = calendarAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/calendars/%d/records/%d", otherCal, foreign), employee.Token, nil)
	requireStatus(t, r, 200, "чужое событие живо")
}

// ── Загрузки и публичные ссылки ──────────────────────────────────

func TestCalendarUploadsAndSharing(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Афиша "))
	employee := newMember(t, admin, companyID, roleEmployee)
	calID := createCalendar(t, admin, "Афиша")
	fields := putFields(t, calendarAPI, "/api/calendars", admin, calID, []map[string]any{
		{"label": "Постер", "type": "image"},
		{"label": "Название", "type": "text"},
	})
	posterK := fieldKey(t, fields, "Постер")
	nameK := fieldKey(t, fields, "Название")

	// Загрузка в префикс calendar/ + чистка файла при удалении записи.
	up := calendarAPI.doMultipart(t, "/api/calendars/uploads", employee.Token, "постер.png",
		[]byte{0x89, 'P', 'N', 'G', 0, 0, 0, 0})
	requireStatus(t, up, 201, "загрузка постера")
	path := up.Str("path")
	if !strings.HasPrefix(path, "calendar/") {
		t.Fatalf("путь загрузки: %q", path)
	}
	if _, err := os.Stat(filepath.Join(uploadsDir, path)); err != nil {
		t.Fatalf("файл не появился: %v", err)
	}
	evID := createEvent(t, employee, calID, "2026-08-01T18:00:00Z", map[string]any{
		posterK: map[string]any{"path": path, "name": "постер.png", "mime": "image/png", "size": 8},
		nameK:   "Концерт",
	})
	r := calendarAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/calendars/%d/records/%d", calID, evID), employee.Token, nil)
	requireStatus(t, r, 200, "удаление события с файлом")
	if _, err := os.Stat(filepath.Join(uploadsDir, path)); !os.IsNotExist(err) {
		t.Fatalf("файл события не удалён: %s", path)
	}

	// Публичные ссылки.
	createEvent(t, employee, calID, "2026-08-02T12:00:00Z", map[string]any{nameK: "Открытая лекция"})
	sharePath := fmt.Sprintf("/api/calendars/%d/shares", calID)
	r = calendarAPI.doJSON(t, http.MethodPost, sharePath, employee.Token, nil)
	requireStatus(t, r, 201, "создание ссылки календаря")
	code := r.Str("code")
	shareID := int64(r.Num("id"))

	r = calendarAPI.doJSON(t, http.MethodGet, "/api/calendars/shared/"+code, "", nil)
	requireStatus(t, r, 200, "публичный календарь")
	r = calendarAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/calendars/shared/%s/records?from=%s&to=%s", code,
			urlQuery("2026-08-01T00:00:00Z"), urlQuery("2026-09-01T00:00:00Z")), "", nil)
	requireStatus(t, r, 200, "публичные события")
	if len(recordIDs(r)) != 1 {
		t.Fatalf("публичные события: %s", r.Raw)
	}
	r = calendarAPI.doJSON(t, http.MethodGet, "/api/calendars/shared/"+code+"/export", "", nil)
	requireStatus(t, r, 200, "публичный экспорт календаря")
	if string(r.Raw[:2]) != "PK" {
		t.Fatalf("публичный экспорт: не xlsx")
	}

	// Мутации — только с токеном; мусорный и отозванный коды → 404.
	r = calendarAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/calendars/%d/records", calID), "", map[string]any{"event_at": "2026-08-03T10:00:00Z"})
	requireError(t, r, 401, "UNAUTHORIZED", "создание события без токена")
	r = calendarAPI.doJSON(t, http.MethodGet, "/api/calendars/shared/deadbeef", "", nil)
	requireStatus(t, r, 404, "мусорный код календаря")
	r = calendarAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("%s/%d", sharePath, shareID),
		employee.Token, nil)
	requireStatus(t, r, 200, "отзыв ссылки календаря")
	r = calendarAPI.doJSON(t, http.MethodGet, "/api/calendars/shared/"+code, "", nil)
	requireStatus(t, r, 404, "отозванный код календаря")
}

package apitest

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ── Хелперы registrysvc ──────────────────────────────────────────

func createRegistry(t *testing.T, a *actor, name string) int64 {
	t.Helper()
	r := registryAPI.doJSON(t, http.MethodPost, "/api/registries", a.Token, map[string]any{"name": name})
	requireStatus(t, r, 201, "создание реестра "+name)
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("создание реестра: нет id: %s", r.Raw)
	}
	return id
}

// putFields — полная замена полей; возвращает поля ответа (с назначенными id).
func putFields(t *testing.T, api *svcClient, base string, a *actor, id int64, fields []map[string]any) []map[string]any {
	t.Helper()
	r := api.doJSON(t, http.MethodPut, fmt.Sprintf("%s/%d/fields", base, id), a.Token,
		map[string]any{"fields": fields})
	requireStatus(t, r, 200, "замена полей")
	out := []map[string]any{}
	for _, f := range r.List("fields") {
		out = append(out, f.(map[string]any))
	}
	return out
}

// fieldKey — строковый ключ поля (id) по label из ответа PUT fields.
func fieldKey(t *testing.T, fields []map[string]any, label string) string {
	t.Helper()
	for _, f := range fields {
		if f["label"] == label {
			return strconv.FormatInt(int64(f["id"].(float64)), 10)
		}
	}
	t.Fatalf("поле %q не найдено: %v", label, fields)
	return ""
}

func createRecord(t *testing.T, a *actor, regID int64, data map[string]any) int64 {
	t.Helper()
	r := registryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/registries/%d/records", regID),
		a.Token, map[string]any{"data": data})
	requireStatus(t, r, 201, "создание записи реестра")
	return int64(r.Num("id"))
}

func recordIDs(r apiResp) []int64 {
	items := r.List("items")
	out := make([]int64, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		id, _ := m["id"].(float64)
		out = append(out, int64(id))
	}
	return out
}

// стандартный набор полей реестра для тестов записей.
func regBaseFields() []map[string]any {
	return []map[string]any{
		{"label": "Название", "type": "text", "col_span": 2, "show_in_table": true},
		{"label": "Код", "type": "number", "config": map[string]any{"pattern": `^\d+$`}, "show_in_table": true},
		{"label": "Статус", "type": "select", "config": map[string]any{"options": []string{"новый", "в работе"}}},
		{"label": "Сайт", "type": "link"},
		{"label": "Дата", "type": "datetime"},
		{"label": "Готово", "type": "checkbox"},
	}
}

// ── Структура: роли, валидация, чистка данных удалённого поля ────

func TestRegistryStructureAndRoles(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Реестры "))
	employee := newMember(t, admin, companyID, roleEmployee)

	// Создание/правка структуры — только администратор компании.
	r := registryAPI.doJSON(t, http.MethodPost, "/api/registries", employee.Token,
		map[string]any{"name": "x"})
	requireError(t, r, 403, "FORBIDDEN", "создание реестра сотрудником")

	// Валидация имени.
	r = registryAPI.doJSON(t, http.MethodPost, "/api/registries", admin.Token,
		map[string]any{"name": "  "})
	requireError(t, r, 400, "VALIDATION", "реестр без имени")
	r = registryAPI.doJSON(t, http.MethodPost, "/api/registries", admin.Token,
		map[string]any{"name": strings.Repeat("я", 121)})
	requireError(t, r, 400, "VALIDATION", "слишком длинное имя реестра")

	regID := createRegistry(t, admin, "Справочник поставщиков")

	// Поля: неизвестный тип и пустой label → 400.
	r = registryAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/registries/%d/fields", regID),
		admin.Token, map[string]any{"fields": []map[string]any{{"label": "x", "type": "magic"}}})
	requireError(t, r, 400, "VALIDATION", "неизвестный тип поля")
	r = registryAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/registries/%d/fields", regID),
		admin.Token, map[string]any{"fields": []map[string]any{{"label": " ", "type": "text"}}})
	requireError(t, r, 400, "VALIDATION", "поле без названия")

	// Нормализация span'ов: col 1..3, row ≥1.
	fields := putFields(t, registryAPI, "/api/registries", admin, regID, []map[string]any{
		{"label": "Гигант", "type": "text", "col_span": 7, "row_span": 0},
	})
	if fields[0]["col_span"].(float64) != 3 || fields[0]["row_span"].(float64) != 1 {
		t.Fatalf("нормализация span'ов: %v", fields[0])
	}

	// Сотрудник структуру не правит, но реестр видит.
	r = registryAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/registries/%d/fields", regID),
		employee.Token, map[string]any{"fields": []map[string]any{}})
	requireError(t, r, 403, "FORBIDDEN", "PUT fields сотрудником")
	r = registryAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/registries/%d", regID),
		employee.Token, map[string]any{"name": "x"})
	requireError(t, r, 403, "FORBIDDEN", "PATCH реестра сотрудником")
	r = registryAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/registries/%d", regID),
		employee.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "DELETE реестра сотрудником")
	r = registryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/registries/%d", regID), employee.Token, nil)
	requireStatus(t, r, 200, "чтение реестра сотрудником")

	// Скоуп: админ другой компании реестр не видит (404), список пуст.
	adminB := newVerifiedUser(t)
	adminB.createCompany(t, uniq("Другая "))
	r = registryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/registries/%d", regID), adminB.Token, nil)
	requireStatus(t, r, 404, "чужой реестр")
	r = registryAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/registries/%d", regID),
		adminB.Token, map[string]any{"name": "hack"})
	requireStatus(t, r, 404, "правка чужого реестра")
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries", adminB.Token, nil)
	requireStatus(t, r, 200, "список реестров B")
	if len(r.List("registries")) != 0 {
		t.Fatalf("чужие реестры в списке: %s", r.Raw)
	}

	// Супер-админ без активной компании в company-scoped роуты не проходит.
	root := newSuperAdmin(t)
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries", root.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "реестры супер-админом")

	// Удаление поля чистит его значения в записях.
	fields = putFields(t, registryAPI, "/api/registries", admin, regID, []map[string]any{
		{"label": "Имя", "type": "text"},
		{"label": "Заметка", "type": "text"},
	})
	nameKey := fieldKey(t, fields, "Имя")
	noteKey := fieldKey(t, fields, "Заметка")
	recID := createRecord(t, employee, regID, map[string]any{
		nameKey: "Ромашка", noteKey: "уникальная-заметка-хвост",
	})
	// Убираем поле «Заметка» (передаём только «Имя» с его id).
	var keepID int64
	for _, f := range fields {
		if f["label"] == "Имя" {
			keepID = int64(f["id"].(float64))
		}
	}
	putFields(t, registryAPI, "/api/registries", admin, regID, []map[string]any{
		{"id": keepID, "label": "Имя", "type": "text"},
	})
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records/%d", regID, recID), employee.Token, nil)
	requireStatus(t, r, 200, "запись после удаления поля")
	data, _ := r.JSON["data"].(map[string]any)
	if _, ok := data[noteKey]; ok {
		t.Fatalf("значение удалённого поля осталось: %s", r.Raw)
	}
	if data[nameKey] != "Ромашка" {
		t.Fatalf("живое поле пострадало: %s", r.Raw)
	}
	// Поиск по вычищенному значению больше не находит (search_text пересчитан).
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records?search=%s", regID, urlQuery("уникальная-заметка")),
		employee.Token, nil)
	requireStatus(t, r, 200, "поиск по удалённому полю")
	if len(recordIDs(r)) != 0 {
		t.Fatalf("search_text не пересчитан: %s", r.Raw)
	}

	// Удаление реестра.
	r = registryAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/registries/%d", regID), admin.Token, nil)
	requireStatus(t, r, 200, "удаление реестра")
	r = registryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/registries/%d", regID), admin.Token, nil)
	requireStatus(t, r, 404, "удалённый реестр")
}

// ── Записи: coerce, поиск, сортировка, пагинация, bulk-delete ────

func TestRegistryRecords(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Записи "))
	employee := newMember(t, admin, companyID, roleEmployee)
	regID := createRegistry(t, admin, "Каталог")
	fields := putFields(t, registryAPI, "/api/registries", admin, regID, regBaseFields())
	nameK := fieldKey(t, fields, "Название")
	codeK := fieldKey(t, fields, "Код")
	statusK := fieldKey(t, fields, "Статус")
	siteK := fieldKey(t, fields, "Сайт")
	dateK := fieldKey(t, fields, "Дата")

	// Валидация значений: number-маска и select-варианты.
	r := registryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/registries/%d/records", regID),
		employee.Token, map[string]any{"data": map[string]any{codeK: "12a"}})
	requireError(t, r, 400, "VALIDATION", "число мимо маски")
	r = registryAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/registries/%d/records", regID),
		employee.Token, map[string]any{"data": map[string]any{statusK: "закрыт"}})
	requireError(t, r, 400, "VALIDATION", "select вне вариантов")

	// Неизвестные ключи данных отбрасываются.
	rec1 := createRecord(t, employee, regID, map[string]any{
		nameK: "Альфа", codeK: "9", statusK: "новый", "999999": "мусор",
	})
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records/%d", regID, rec1), employee.Token, nil)
	requireStatus(t, r, 200, "чтение записи")
	data, _ := r.JSON["data"].(map[string]any)
	if _, ok := data["999999"]; ok {
		t.Fatalf("неизвестный ключ не отброшен: %s", r.Raw)
	}

	rec2 := createRecord(t, employee, regID, map[string]any{
		nameK: "Бета", codeK: "21", statusK: "в работе", siteK: "https://example.org/beta",
	})
	rec3 := createRecord(t, employee, regID, map[string]any{
		nameK: "Гамма", codeK: "70", dateK: "2026-12-31T10:00:00Z",
	})

	// Правка записи любым участником (в т.ч. админом).
	r = registryAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/registries/%d/records/%d", regID, rec1), admin.Token,
		map[string]any{"data": map[string]any{nameK: "Альфа v2", codeK: "9", statusK: "новый"}})
	requireStatus(t, r, 200, "правка записи админом")

	// Поиск: по тексту, числу, ссылке, дате и варианту списка.
	searches := map[string]int64{
		"альфа":       rec1, // ILIKE без регистра
		"21":          rec2,
		"example.org": rec2,
		"2026-12-31":  rec3,
	}
	for q, want := range searches {
		r = registryAPI.doJSON(t, http.MethodGet,
			fmt.Sprintf("/api/registries/%d/records?search=%s", regID, urlQuery(q)), employee.Token, nil)
		requireStatus(t, r, 200, "поиск "+q)
		ids := recordIDs(r)
		if len(ids) != 1 || ids[0] != want {
			t.Fatalf("поиск %q: ожидалась %d, получено %v", q, want, ids)
		}
	}
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records?search=%s", regID, urlQuery("в работе")), employee.Token, nil)
	if ids := recordIDs(r); len(ids) != 1 || ids[0] != rec2 {
		t.Fatalf("поиск по select: %v", recordIDs(r))
	}

	// Сортировка по числовому полю — численно (9 < 10 < 70), asc и desc.
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records?sort=%s&order=asc", regID, codeK), employee.Token, nil)
	if ids := recordIDs(r); !equalIDs(ids, []int64{rec1, rec2, rec3}) {
		t.Fatalf("числовая сортировка asc: %v", ids)
	}
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records?sort=%s&order=desc", regID, codeK), employee.Token, nil)
	if ids := recordIDs(r); !equalIDs(ids, []int64{rec3, rec2, rec1}) {
		t.Fatalf("числовая сортировка desc: %v", ids)
	}
	// По текстовому полю: Альфа < Бета < Гамма.
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records?sort=%s&order=asc", regID, nameK), employee.Token, nil)
	if ids := recordIDs(r); !equalIDs(ids, []int64{rec1, rec2, rec3}) {
		t.Fatalf("текстовая сортировка: %v", ids)
	}
	// По created_at desc — последняя созданная первой.
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records?sort=created_at&order=desc", regID), employee.Token, nil)
	if ids := recordIDs(r); !equalIDs(ids, []int64{rec3, rec2, rec1}) {
		t.Fatalf("сортировка created_at desc: %v", ids)
	}

	// Пагинация: total/page/per_page.
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records?per_page=2&page=2&sort=created_at&order=asc", regID),
		employee.Token, nil)
	requireStatus(t, r, 200, "пагинация")
	if r.Num("total") != 3 || r.Num("page") != 2 || r.Num("per_page") != 2 || len(r.List("items")) != 1 {
		t.Fatalf("пагинация: %s", r.Raw)
	}

	// Запись из чужого реестра → 404 (и по чужой компании тоже).
	otherReg := createRegistry(t, admin, "Другой реестр")
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records/%d", otherReg, rec1), employee.Token, nil)
	requireStatus(t, r, 404, "запись не из этого реестра")
	adminB := newVerifiedUser(t)
	adminB.createCompany(t, uniq("Б "))
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records", regID), adminB.Token, nil)
	requireStatus(t, r, 404, "записи чужой компании")

	// Экспорт: все поля; по фильтру; по ids; image/file не выгружаются.
	r = registryAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/registries/%d/export", regID),
		employee.Token, nil)
	requireStatus(t, r, 200, "экспорт всех")
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, xlsxMime) {
		t.Fatalf("экспорт: content-type %q", ct)
	}
	if string(r.Raw[:2]) != "PK" {
		t.Fatalf("экспорт: не xlsx")
	}
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/export?search=%s", regID, urlQuery("Гамма")), employee.Token, nil)
	requireStatus(t, r, 200, "экспорт по фильтру")
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/export?ids=%d,%d", regID, rec1, rec2), employee.Token, nil)
	requireStatus(t, r, 200, "экспорт по ids")

	// bulk-delete: чужой реестр в ids не задевается.
	foreign := createRecord(t, employee, otherReg, map[string]any{})
	r = registryAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/registries/%d/records/bulk-delete", regID), employee.Token,
		map[string]any{"ids": []int64{rec1, rec2, foreign}})
	requireStatus(t, r, 200, "bulk-delete")
	if r.Num("deleted") != 2 {
		t.Fatalf("bulk-delete: ожидалось 2, тело: %s", r.Raw)
	}
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/%d/records/%d", otherReg, foreign), employee.Token, nil)
	requireStatus(t, r, 200, "чужая запись жива")
}

// ── Загрузки файлов и чистка при удалении ────────────────────────

func TestRegistryUploadsAndFileCleanup(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Файлы "))
	employee := newMember(t, admin, companyID, roleEmployee)
	regID := createRegistry(t, admin, "С картинками")
	fields := putFields(t, registryAPI, "/api/registries", admin, regID, []map[string]any{
		{"label": "Фото", "type": "image"},
		{"label": "Документ", "type": "file"},
	})
	photoK := fieldKey(t, fields, "Фото")
	docK := fieldKey(t, fields, "Документ")

	// Мини-PNG (валидная сигнатура достаточно — сервис контент не проверяет).
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}

	up := registryAPI.doMultipart(t, "/api/registries/uploads", employee.Token, "фото.png", png)
	requireStatus(t, up, 201, "загрузка картинки")
	photoPath := up.Str("path")
	if !strings.HasPrefix(photoPath, "registry/") || up.Str("name") != "фото.png" {
		t.Fatalf("метаданные загрузки: %s", up.Raw)
	}
	if _, err := os.Stat(filepath.Join(uploadsDir, photoPath)); err != nil {
		t.Fatalf("файл не появился на диске: %v", err)
	}
	// Без токена загрузка запрещена.
	r := registryAPI.doMultipart(t, "/api/registries/uploads", "", "x.bin", []byte("x"))
	requireStatus(t, r, 401, "загрузка без токена")

	up2 := registryAPI.doMultipart(t, "/api/registries/uploads", employee.Token, "договор.pdf", []byte("%PDF-1.4"))
	requireStatus(t, up2, 201, "загрузка файла")
	docPath := up2.Str("path")

	fileVal := func(u apiResp) map[string]any {
		return map[string]any{"path": u.Str("path"), "name": u.Str("name"),
			"mime": u.Str("mime"), "size": u.Num("size")}
	}
	recID := createRecord(t, employee, regID, map[string]any{
		photoK: fileVal(up), docK: fileVal(up2),
	})

	// Удаление записи чистит оба файла с диска.
	r = registryAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/registries/%d/records/%d", regID, recID), employee.Token, nil)
	requireStatus(t, r, 200, "удаление записи с файлами")
	for _, p := range []string{photoPath, docPath} {
		if _, err := os.Stat(filepath.Join(uploadsDir, p)); !os.IsNotExist(err) {
			t.Fatalf("файл %s не удалён вместе с записью", p)
		}
	}

	// Удаление ПОЛЯ чистит файлы его значений во всех записях.
	up3 := registryAPI.doMultipart(t, "/api/registries/uploads", employee.Token, "ещё.png", png)
	requireStatus(t, up3, 201, "загрузка для теста поля")
	path3 := up3.Str("path")
	createRecord(t, employee, regID, map[string]any{photoK: fileVal(up3)})
	var docFieldID int64
	for _, f := range fields {
		if f["label"] == "Документ" {
			docFieldID = int64(f["id"].(float64))
		}
	}
	putFields(t, registryAPI, "/api/registries", admin, regID, []map[string]any{
		{"id": docFieldID, "label": "Документ", "type": "file"},
	})
	if _, err := os.Stat(filepath.Join(uploadsDir, path3)); !os.IsNotExist(err) {
		t.Fatalf("файл удалённого поля не вычищен: %s", path3)
	}
}

// ── Публичные ссылки ─────────────────────────────────────────────

func TestRegistrySharing(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Шаринг "))
	employee := newMember(t, admin, companyID, roleEmployee)
	regID := createRegistry(t, admin, "Публичный реестр")
	fields := putFields(t, registryAPI, "/api/registries", admin, regID, []map[string]any{
		{"label": "Товар", "type": "text"},
		{"label": "Обложка", "type": "image"},
	})
	nameK := fieldKey(t, fields, "Товар")
	createRecord(t, employee, regID, map[string]any{nameK: "Секретный товар"})

	sharePath := fmt.Sprintf("/api/registries/%d/shares", regID)

	// Создание и список — любой участник.
	r := registryAPI.doJSON(t, http.MethodPost, sharePath, employee.Token, nil)
	requireStatus(t, r, 201, "создание ссылки")
	code := r.Str("code")
	shareID := int64(r.Num("id"))
	if code == "" || shareID == 0 {
		t.Fatalf("ссылка без code/id: %s", r.Raw)
	}
	r = registryAPI.doJSON(t, http.MethodGet, sharePath, employee.Token, nil)
	requireStatus(t, r, 200, "список ссылок")
	if len(r.List("shares")) != 1 {
		t.Fatalf("ожидалась одна ссылка: %s", r.Raw)
	}

	// Публичный просмотр без авторизации: структура, записи, экспорт.
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries/shared/"+code, "", nil)
	requireStatus(t, r, 200, "публичная структура")
	if len(r.List("fields")) != 2 {
		t.Fatalf("публичная структура без полей: %s", r.Raw)
	}
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries/shared/"+code+"/records", "", nil)
	requireStatus(t, r, 200, "публичные записи")
	if len(recordIDs(r)) != 1 {
		t.Fatalf("публичные записи: %s", r.Raw)
	}
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries/shared/"+code+"/export", "", nil)
	requireStatus(t, r, 200, "публичный экспорт")
	if string(r.Raw[:2]) != "PK" {
		t.Fatalf("публичный экспорт: не xlsx")
	}

	// Экспорт только по неэкспортируемому полю (image) → 400: картинки и файлы
	// в xlsx не выгружаются.
	var imgID int64
	for _, f := range fields {
		if f["label"] == "Обложка" {
			imgID = int64(f["id"].(float64))
		}
	}
	r = registryAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/registries/shared/%s/export?fields=%d", code, imgID), "", nil)
	requireError(t, r, 400, "VALIDATION", "экспорт только image-поля")

	// Мутационные ручки по коду отсутствуют: любые записи требуют токен.
	r = registryAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/registries/%d/records", regID), "", map[string]any{"data": map[string]any{}})
	requireError(t, r, 401, "UNAUTHORIZED", "создание записи без токена")
	r = registryAPI.doJSON(t, http.MethodPost, "/api/registries", "", map[string]any{"name": "x"})
	requireError(t, r, 401, "UNAUTHORIZED", "создание реестра без токена")

	// Мусорный код → 404; отзыв — код умирает.
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries/shared/deadbeef", "", nil)
	requireStatus(t, r, 404, "мусорный код")
	r = registryAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("%s/%d", sharePath, shareID),
		employee.Token, nil)
	requireStatus(t, r, 200, "отзыв ссылки")
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries/shared/"+code, "", nil)
	requireStatus(t, r, 404, "отозванный код")
	r = registryAPI.doJSON(t, http.MethodGet, "/api/registries/shared/"+code+"/records", "", nil)
	requireStatus(t, r, 404, "записи по отозванному коду")
}

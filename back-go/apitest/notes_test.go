package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// tiptapDoc — минимальный валидный документ TipTap с одним параграфом текста.
func tiptapDoc(text string) map[string]any {
	return map[string]any{
		"type": "doc",
		"content": []map[string]any{
			{"type": "paragraph", "content": []map[string]any{{"type": "text", "text": text}}},
		},
	}
}

// Сквозной сценарий заметок: создание → edit-ссылка → анонимная правка по коду
// → правка видна владельцу; view-ссылка писать не может; чужой пользователь
// заметку не видит (404).
func TestNotesSharedEditFlow(t *testing.T) {
	owner := newVerifiedUser(t)

	// Создание заметки и наполнение владельцем.
	r := notesAPI.doJSON(t, http.MethodPost, "/api/notes", owner.Token, map[string]any{"title": "Регламент"})
	requireStatus(t, r, 201, "создание заметки")
	noteID := int64(r.Num("id"))

	r = notesAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/notes/%d", noteID), owner.Token,
		map[string]any{"doc": tiptapDoc("первая версия текста")})
	requireStatus(t, r, 200, "правка владельцем")

	// Скоуп по владельцу: чужой пользователь получает 404, не 403.
	other := newVerifiedUser(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, fmt.Sprintf("/api/notes/%d", noteID)},
		{http.MethodPatch, fmt.Sprintf("/api/notes/%d", noteID)},
		{http.MethodDelete, fmt.Sprintf("/api/notes/%d", noteID)},
		{http.MethodGet, fmt.Sprintf("/api/notes/%d/shares", noteID)},
		{http.MethodGet, fmt.Sprintf("/api/notes/%d/export", noteID)},
	} {
		rr := notesAPI.doJSON(t, tc.method, tc.path, other.Token, map[string]any{"title": "x"})
		if rr.Status != 404 {
			t.Fatalf("%s %s чужим: ожидался 404, получен %d: %s", tc.method, tc.path, rr.Status, rr.Raw)
		}
	}

	// Ссылки: view и edit.
	r = notesAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/notes/%d/shares", noteID), owner.Token,
		map[string]any{"access": "view"})
	requireStatus(t, r, 201, "view-ссылка")
	viewCode := r.Str("code")
	r = notesAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/notes/%d/shares", noteID), owner.Token,
		map[string]any{"access": "edit"})
	requireStatus(t, r, 201, "edit-ссылка")
	editCode := r.Str("code")
	editShareID := int64(r.Num("id"))

	// Некорректный режим доступа.
	r = notesAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/notes/%d/shares", noteID), owner.Token,
		map[string]any{"access": "admin"})
	requireError(t, r, 400, "VALIDATION", "ссылка с неизвестным access")

	// Анонимное чтение по коду (без токена).
	r = notesAPI.doJSON(t, http.MethodGet, "/api/notes/shared/"+viewCode, "", nil)
	requireStatus(t, r, 200, "чтение по view-ссылке")
	if r.Str("access") != "view" {
		t.Fatalf("access view-ссылки: %s", r.Raw)
	}

	// view-ссылка не пишет.
	r = notesAPI.doJSON(t, http.MethodPut, "/api/notes/shared/"+viewCode, "",
		map[string]any{"title": "вандализм"})
	requireError(t, r, 403, "FORBIDDEN", "запись по view-ссылке")

	// edit-ссылка пишет анонимно.
	r = notesAPI.doJSON(t, http.MethodPut, "/api/notes/shared/"+editCode, "",
		map[string]any{"title": "Регламент v2", "doc": tiptapDoc("поправлено по ссылке")})
	requireStatus(t, r, 200, "правка по edit-ссылке")

	// Правка видна владельцу; text_content пересчитан сервером (excerpt).
	r = notesAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/notes/%d", noteID), owner.Token, nil)
	requireStatus(t, r, 200, "заметка после анонимной правки")
	if r.Str("title") != "Регламент v2" || !strings.Contains(r.Str("excerpt"), "поправлено по ссылке") {
		t.Fatalf("анонимная правка не дошла до владельца: %s", r.Raw)
	}

	// Экспорт в txt — заголовок + плоский текст.
	r = notesAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/notes/%d/export", noteID), owner.Token, nil)
	requireStatus(t, r, 200, "экспорт txt")
	if body := string(r.Raw); !strings.HasPrefix(body, "Регламент v2") || !strings.Contains(body, "поправлено по ссылке") {
		t.Fatalf("экспорт txt: %q", body)
	}

	// Отзыв edit-ссылки: код перестаёт работать.
	r = notesAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/notes/%d/shares/%d", noteID, editShareID), owner.Token, nil)
	requireStatus(t, r, 200, "отзыв ссылки")
	r = notesAPI.doJSON(t, http.MethodPut, "/api/notes/shared/"+editCode, "",
		map[string]any{"title": "после отзыва"})
	requireError(t, r, 404, "NOT_FOUND", "запись по отозванной ссылке")
}

// Теги-метки (бывшие «группы», миграция 00046): заметка с несколькими тегами,
// фильтр списка, удаление тега не удаляет заметки; импорт txt создаёт заметку
// с заголовком из первой строки.
func TestNotesGroupsAndImport(t *testing.T) {
	owner := newVerifiedUser(t)

	r := notesAPI.doJSON(t, http.MethodPost, "/api/notes/tags", owner.Token, map[string]any{"name": "Работа"})
	requireStatus(t, r, 201, "тег 1")
	workID := int64(r.Num("id"))
	r = notesAPI.doJSON(t, http.MethodPost, "/api/notes/tags", owner.Token, map[string]any{"name": "Личное"})
	requireStatus(t, r, 201, "тег 2")
	homeID := int64(r.Num("id"))

	r = notesAPI.doJSON(t, http.MethodPost, "/api/notes", owner.Token, map[string]any{"title": "В двух тегах"})
	requireStatus(t, r, 201, "заметка")
	noteID := int64(r.Num("id"))
	r = notesAPI.doJSON(t, http.MethodPut, fmt.Sprintf("/api/notes/%d/tags", noteID), owner.Token,
		map[string]any{"tag_ids": []int64{workID, homeID}})
	requireStatus(t, r, 200, "назначение тегов")

	// Фильтр по тегу.
	r = notesAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/notes?tag_ids=%d", workID), owner.Token, nil)
	requireStatus(t, r, 200, "список по тегу")
	if len(r.List("notes")) != 1 {
		t.Fatalf("фильтр по тегу: %s", r.Raw)
	}

	// Удаление тега не трогает заметку — только связь.
	r = notesAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/notes/tags/%d", workID), owner.Token, nil)
	requireStatus(t, r, 200, "удаление тега")
	r = notesAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/notes/%d", noteID), owner.Token, nil)
	requireStatus(t, r, 200, "заметка после удаления тега")

	// Импорт txt: первая строка → заголовок, остальное → текст документа.
	r = notesAPI.doMultipart(t, "/api/notes/import", owner.Token, "list.txt",
		[]byte("Импортированный список\n\nкупить хлеб\nпозвонить в банк"))
	requireStatus(t, r, 201, "импорт txt")
	if r.Str("title") != "Импортированный список" {
		t.Fatalf("заголовок импорта: %s", r.Raw)
	}
	imported := int64(r.Num("id"))
	r = notesAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/notes/%d", imported), owner.Token, nil)
	requireStatus(t, r, 200, "импортированная заметка")
	if !strings.Contains(r.Str("excerpt"), "купить хлеб") {
		t.Fatalf("текст импорта: %s", r.Raw)
	}

	// Поиск по тексту (серверный ILIKE).
	r = notesAPI.doJSON(t, http.MethodGet, "/api/notes?search="+urlQuery("позвонить"), owner.Token, nil)
	requireStatus(t, r, 200, "поиск")
	if len(r.List("notes")) != 1 {
		t.Fatalf("поиск по тексту: %s", r.Raw)
	}
}

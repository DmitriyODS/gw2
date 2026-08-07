package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// scene — минимальная валидная сцена холста с одной надписью.
func scene(text string) map[string]any {
	return map[string]any{
		"version":    1,
		"background": "grid",
		"objects": []map[string]any{
			{"id": "t1", "type": "text", "x": 40, "y": 40, "text": text, "size": 18, "color": "ink"},
		},
	}
}

// Сквозной сценарий досок: создание → рисование → edit-ссылка правит холст
// анонимно → правка видна владельцу; view-ссылка писать не может; чужой
// пользователь доски не видит (404).
func TestBoardsSharedEditFlow(t *testing.T) {
	owner := newVerifiedUser(t)

	r := boardAPI.doJSON(t, http.MethodPost, "/api/boards", owner.Token, map[string]any{"title": "Схема релиза"})
	requireStatus(t, r, 201, "создание доски")
	boardID := int64(r.Num("id"))

	r = boardAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/boards/%d", boardID), owner.Token,
		map[string]any{"scene": scene("первый набросок")})
	requireStatus(t, r, 200, "правка владельцем")

	// Скоуп по владельцу: чужой получает 404, не 403.
	other := newVerifiedUser(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, fmt.Sprintf("/api/boards/%d", boardID)},
		{http.MethodPatch, fmt.Sprintf("/api/boards/%d", boardID)},
		{http.MethodDelete, fmt.Sprintf("/api/boards/%d", boardID)},
		{http.MethodGet, fmt.Sprintf("/api/boards/%d/shares", boardID)},
	} {
		rr := boardAPI.doJSON(t, tc.method, tc.path, other.Token, map[string]any{"title": "x"})
		if rr.Status != 404 {
			t.Fatalf("%s %s чужим: ожидался 404, получен %d: %s", tc.method, tc.path, rr.Status, rr.Raw)
		}
	}

	// Публичные ссылки: view только читает, edit — правит.
	r = boardAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/boards/%d/shares", boardID), owner.Token,
		map[string]any{"access": "view"})
	requireStatus(t, r, 201, "view-ссылка")
	viewCode := r.Str("code")

	r = boardAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/boards/%d/shares", boardID), owner.Token,
		map[string]any{"access": "edit"})
	requireStatus(t, r, 201, "edit-ссылка")
	editCode := r.Str("code")

	// Без токена — по коду ссылки.
	r = boardAPI.doJSON(t, http.MethodGet, "/api/boards/shared/"+viewCode, "", nil)
	requireStatus(t, r, 200, "чтение по view-ссылке")

	r = boardAPI.doJSON(t, http.MethodPut, "/api/boards/shared/"+viewCode, "",
		map[string]any{"scene": scene("вандализм")})
	if r.Status != 403 {
		t.Fatalf("view-ссылка не должна править: статус %d: %s", r.Status, r.Raw)
	}

	r = boardAPI.doJSON(t, http.MethodPut, "/api/boards/shared/"+editCode, "",
		map[string]any{"scene": scene("правка по ссылке")})
	requireStatus(t, r, 200, "правка по edit-ссылке")

	// Владелец видит чужую правку, а поиск находит доску по тексту надписи:
	// text_content пересчитывает сервер из сцены.
	r = boardAPI.doJSON(t, http.MethodGet, "/api/boards?search=правка", owner.Token, nil)
	requireStatus(t, r, 200, "поиск по надписям")
	if !strings.Contains(string(r.Raw), "Схема релиза") {
		t.Fatalf("доска должна находиться по тексту надписи: %s", r.Raw)
	}
}

// Адресный шаринг: доска, открытая пользователю на чтение, видна ему в
// «поделились со мной», но правку он получает только с can_edit.
func TestBoardsUserShareAccess(t *testing.T) {
	owner := newVerifiedUser(t)
	mate := newVerifiedUser(t)

	r := boardAPI.doJSON(t, http.MethodPost, "/api/boards", owner.Token, map[string]any{"title": "Общая доска"})
	requireStatus(t, r, 201, "создание доски")
	boardID := int64(r.Num("id"))

	r = boardAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/boards/%d/members", boardID), owner.Token,
		map[string]any{"target": "user", "user_id": mate.ID, "can_edit": false})
	requireStatus(t, r, 201, "поделиться с пользователем")

	r = boardAPI.doJSON(t, http.MethodGet, "/api/boards?shared=1", mate.Token, nil)
	requireStatus(t, r, 200, "список «поделились со мной»")
	if !strings.Contains(string(r.Raw), "Общая доска") {
		t.Fatalf("расшаренная доска должна быть в списке адресата: %s", r.Raw)
	}

	// Только чтение: правка сцены запрещена.
	r = boardAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/boards/%d", boardID), mate.Token,
		map[string]any{"scene": scene("чужая правка")})
	if r.Status != 403 {
		t.Fatalf("адресат без can_edit не должен править: статус %d: %s", r.Status, r.Raw)
	}

	// Выдали право — правка проходит.
	r = boardAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/boards/%d/members", boardID), owner.Token,
		map[string]any{"target": "user", "user_id": mate.ID, "can_edit": true})
	requireStatus(t, r, 201, "повысить право адресата")

	r = boardAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/boards/%d", boardID), mate.Token,
		map[string]any{"scene": scene("правка соавтора")})
	requireStatus(t, r, 200, "правка адресатом с can_edit")
}

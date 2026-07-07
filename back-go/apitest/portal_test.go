package apitest

import (
	"fmt"
	"net/http"
	"testing"
)

// ── Хелперы portalsvc ──────────────────────────────────────────────

func createTopic(t *testing.T, a *actor, name string) int64 {
	t.Helper()
	r := portalAPI.doJSON(t, http.MethodPost, "/api/portal/topics", a.Token, map[string]any{"name": name})
	requireStatus(t, r, 201, "создание раздела "+name)
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("создание раздела: нет id: %s", r.Raw)
	}
	return id
}

func createPost(t *testing.T, a *actor, body string) int64 {
	t.Helper()
	r := portalAPI.doJSON(t, http.MethodPost, "/api/portal/posts", a.Token, map[string]any{"body": body})
	requireStatus(t, r, 201, "создание поста")
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("создание поста: нет id: %s", r.Raw)
	}
	return id
}

// ── Топики: только администратор ────────────────────────────────

func TestPortalTopics_AdminOnlyAndValidation(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал "))
	employee := newMember(t, admin, companyID, roleEmployee)

	r := portalAPI.doJSON(t, http.MethodPost, "/api/portal/topics", employee.Token,
		map[string]any{"name": "Новости"})
	requireError(t, r, 403, "FORBIDDEN", "создание раздела сотрудником")

	r = portalAPI.doJSON(t, http.MethodPost, "/api/portal/topics", admin.Token,
		map[string]any{"name": "   "})
	requireError(t, r, 400, "VALIDATION", "раздел без имени")

	topicID := createTopic(t, admin, "Новости")

	r = portalAPI.doJSON(t, http.MethodGet, "/api/portal/topics", employee.Token, nil)
	requireStatus(t, r, 200, "список разделов")
	found := false
	for _, tv := range r.List("topics") {
		if int64(tv.(map[string]any)["id"].(float64)) == topicID {
			found = true
		}
	}
	if !found {
		t.Fatalf("созданный раздел не в списке: %s", r.Raw)
	}

	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/topics/%d", topicID), employee.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "удаление раздела сотрудником")
	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/topics/%d", topicID), admin.Token, nil)
	requireStatus(t, r, 200, "удаление раздела администратором")
}

// ── Посты: CRUD, скоуп по компании, комментарии, реакции ─────────

func TestPortalPosts_CRUDScopeCommentsReactions(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-посты "))
	author := newMember(t, admin, companyID, roleEmployee)
	stranger := newMember(t, admin, companyID, roleEmployee)

	r := portalAPI.doJSON(t, http.MethodPost, "/api/portal/posts", author.Token, map[string]any{"body": "   "})
	requireError(t, r, 400, "VALIDATION", "пост без текста")

	postID := createPost(t, author, "Добро пожаловать в портал!")

	// Список постов компании.
	r = portalAPI.doJSON(t, http.MethodGet, "/api/portal/posts", author.Token, nil)
	requireStatus(t, r, 200, "список постов")
	if len(r.List("posts")) == 0 {
		t.Fatalf("список постов пуст: %s", r.Raw)
	}

	// Правка чужого поста — запрещена; автором — можно.
	r = portalAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/portal/posts/%d", postID), stranger.Token,
		map[string]any{"body": "чужая правка"})
	requireError(t, r, 403, "FORBIDDEN", "правка чужого поста")
	r = portalAPI.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/portal/posts/%d", postID), author.Token,
		map[string]any{"body": "обновлённый текст"})
	requireStatus(t, r, 200, "правка своего поста")

	// Комментарии: пустой текст → 400; создание/удаление по правам.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID), stranger.Token,
		map[string]any{"text": "  "})
	requireError(t, r, 400, "VALIDATION", "комментарий без текста")
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID), stranger.Token,
		map[string]any{"text": "интересно!"})
	requireStatus(t, r, 201, "создание комментария")
	commentID := int64(r.Num("id"))

	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/comments/%d", commentID), author.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "удаление чужого комментария не-автором/не-админом")
	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/comments/%d", commentID), stranger.Token, nil)
	requireStatus(t, r, 200, "автор удаляет свой комментарий")

	// Реакции: добавление идемпотентно, чтение через карточку поста.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/reactions", postID), stranger.Token,
		map[string]any{"emoji": "👍"})
	requireStatus(t, r, 201, "добавление реакции")
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/reactions", postID), stranger.Token,
		map[string]any{"emoji": "👍"})
	requireStatus(t, r, 201, "повторная реакция идемпотентна")

	r = portalAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/portal/posts/%d", postID), author.Token, nil)
	requireStatus(t, r, 200, "карточка поста")
	counts, _ := r.JSON["reaction_counts"].(map[string]any)
	if counts == nil || counts["👍"] != float64(1) {
		t.Fatalf("ожидалась 1 реакция 👍, получено: %v", r.JSON["reaction_counts"])
	}

	// Скоуп по компании: чужая компания → 404.
	other := newVerifiedUser(t)
	other.createCompany(t, uniq("Другая портал "))
	r = portalAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/portal/posts/%d", postID), other.Token, nil)
	requireStatus(t, r, 404, "чужой пост")

	// Удаление поста — только автор/администратор; чистит вложения (см. Test ниже).
	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/posts/%d", postID), stranger.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "удаление поста посторонним")
	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/posts/%d", postID), admin.Token, nil)
	requireStatus(t, r, 200, "удаление поста администратором")
}

// ── Закрепление: лимит 10 на компанию ────────────────────────────

func TestPortalPin_LimitAndPermissions(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-пины "))
	author := newMember(t, admin, companyID, roleEmployee)
	stranger := newMember(t, admin, companyID, roleEmployee)

	first := createPost(t, author, "первый пост")

	r := portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/pin", first), stranger.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "закрепление чужого поста не-администратором")
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/pin", first), author.Token, nil)
	requireStatus(t, r, 200, "закрепление автором")

	var ids []int64
	for i := 0; i < 9; i++ {
		id := createPost(t, author, fmt.Sprintf("пост #%d", i))
		r := portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/pin", id), author.Token, nil)
		requireStatus(t, r, 200, fmt.Sprintf("закрепление #%d", i))
		ids = append(ids, id)
	}
	// Уже 10 закреплённых (first + 9) — следующий должен упасть.
	extra := createPost(t, author, "лишний пост")
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/pin", extra), author.Token, nil)
	requireError(t, r, 422, "TOO_MANY_PINNED", "превышение лимита закреплённых")

	// Открепление освобождает слот.
	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/posts/%d/pin", first), author.Token, nil)
	requireStatus(t, r, 200, "открепление")
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/pin", extra), author.Token, nil)
	requireStatus(t, r, 200, "закрепление после освобождения слота")

	// Список ?pinned=true отдаёт только закреплённые, сверху.
	r = portalAPI.doJSON(t, http.MethodGet, "/api/portal/posts?pinned=true", author.Token, nil)
	requireStatus(t, r, 200, "список закреплённых")
	if len(r.List("posts")) != 10 {
		t.Fatalf("ожидалось 10 закреплённых постов, получено %d", len(r.List("posts")))
	}
}

// ── Лента: keyset-пагинация + пин с автоистечением ───────────────

func TestPortalFeed_KeysetPaginationAndPinDays(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-лента "))
	author := newMember(t, admin, companyID, roleEmployee)

	var ids []int64
	for i := 0; i < 5; i++ {
		ids = append(ids, createPost(t, author, fmt.Sprintf("пост ленты #%d", i)))
	}

	// Пин с автоистечением: тело {days: 7} → pinned_until проставлен.
	r := portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/pin", ids[0]), author.Token,
		map[string]any{"days": 7})
	requireStatus(t, r, 200, "закрепление с days")
	if r.JSON["pinned_until"] == nil {
		t.Fatalf("ожидался pinned_until при days=7: %s", r.Raw)
	}

	// Первая страница: секция pinned + хронология без закреплённого + курсор.
	r = portalAPI.doJSON(t, http.MethodGet, "/api/portal/posts?limit=2", author.Token, nil)
	requireStatus(t, r, 200, "первая страница ленты")
	if len(r.List("pinned")) != 1 {
		t.Fatalf("ожидался 1 закреплённый в секции pinned: %s", r.Raw)
	}
	page1 := r.List("posts")
	if len(page1) != 2 {
		t.Fatalf("ожидалось 2 поста на первой странице: %s", r.Raw)
	}
	if int64(page1[0].(map[string]any)["id"].(float64)) != ids[4] ||
		int64(page1[1].(map[string]any)["id"].(float64)) != ids[3] {
		t.Fatalf("хронология первой страницы не DESC: %s", r.Raw)
	}
	cursor := r.Str("next_cursor")
	if cursor == "" {
		t.Fatalf("ожидался next_cursor: %s", r.Raw)
	}

	// Вставка нового поста между страницами не ломает курсор (нет дублей/пропусков).
	createPost(t, author, "свежий пост между страницами")

	r = portalAPI.doJSON(t, http.MethodGet, "/api/portal/posts?limit=2&cursor="+cursor, author.Token, nil)
	requireStatus(t, r, 200, "вторая страница ленты")
	page2 := r.List("posts")
	if len(page2) != 2 ||
		int64(page2[0].(map[string]any)["id"].(float64)) != ids[2] ||
		int64(page2[1].(map[string]any)["id"].(float64)) != ids[1] {
		t.Fatalf("вторая страница должна отдать посты %d,%d: %s", ids[2], ids[1], r.Raw)
	}
	// Хвоста больше нет (ids[0] закреплён и в хронологию не входит).
	if r.JSON["next_cursor"] != nil {
		t.Fatalf("ожидался next_cursor=null в конце ленты: %s", r.Raw)
	}
	if len(r.List("pinned")) != 0 {
		t.Fatalf("pinned-секция только на первой странице: %s", r.Raw)
	}
}

// ── Вложения: аплоад + чистка при удалении поста ─────────────────

func TestPortalAttachment_UploadAndCleanupOnDelete(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-файлы "))
	author := newMember(t, admin, companyID, roleEmployee)

	postID := createPost(t, author, "пост с картинкой")
	r := portalAPI.doMultipart(t, fmt.Sprintf("/api/portal/posts/%d/attachments", postID), author.Token,
		"cover.png", []byte("fake-png-bytes"))
	requireStatus(t, r, 201, "загрузка вложения")
	if r.Str("url") == "" {
		t.Fatalf("ожидался url вложения: %s", r.Raw)
	}

	r = portalAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/portal/posts/%d", postID), author.Token, nil)
	requireStatus(t, r, 200, "карточка поста с вложением")
	if len(r.List("attachments")) != 1 {
		t.Fatalf("ожидалось 1 вложение, получено: %v", r.JSON["attachments"])
	}

	// Удаление поста не должно падать даже с вложениями (чистка файлов — best-effort).
	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/posts/%d", postID), author.Token, nil)
	requireStatus(t, r, 200, "удаление поста с вложением")
}

// ── Непрочитанные посты: серверный бейдж ─────────────────────────

func TestPortalUnread_CountAndSeen(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-бейдж "))
	author := newMember(t, admin, companyID, roleEmployee)
	reader := newMember(t, admin, companyID, roleEmployee)

	createPost(t, author, "новость для бейджа")

	// Второй сотрудник видит 1 непрочитанный; свой пост автору не считается.
	r := portalAPI.doJSON(t, http.MethodGet, "/api/portal/unread", reader.Token, nil)
	requireStatus(t, r, 200, "счётчик непрочитанных")
	if r.Num("count") != 1 {
		t.Fatalf("ожидался 1 непрочитанный пост, получено: %s", r.Raw)
	}
	r = portalAPI.doJSON(t, http.MethodGet, "/api/portal/unread", author.Token, nil)
	requireStatus(t, r, 200, "счётчик непрочитанных у автора")
	if r.Num("count") != 0 {
		t.Fatalf("свой пост не должен считаться непрочитанным: %s", r.Raw)
	}

	// Отметка просмотра сбрасывает счётчик.
	r = portalAPI.doJSON(t, http.MethodPost, "/api/portal/seen", reader.Token, nil)
	requireStatus(t, r, 200, "отметка просмотра")
	r = portalAPI.doJSON(t, http.MethodGet, "/api/portal/unread", reader.Token, nil)
	requireStatus(t, r, 200, "счётчик после отметки")
	if r.Num("count") != 0 {
		t.Fatalf("после /seen ожидалось 0, получено: %s", r.Raw)
	}
}

// ── Пересылка поста в мессенджер (реальный gRPC portalsvc↔msgsvc) ─

func TestPortalForward_ToMessengerConversation(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-форвард "))
	author := newMember(t, admin, companyID, roleEmployee)
	colleague := newMember(t, admin, companyID, roleEmployee)

	postID := createPost(t, author, "Важная новость для всей компании! Не пропустите годовое собрание в пятницу.")

	// По conversation_id: сперва открываем диалог автора с коллегой.
	convID := openConv(t, author, colleague.ID)
	r := portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/forward", postID), author.Token,
		map[string]any{"conversation_ids": []int64{convID}})
	requireStatus(t, r, 200, "пересылка поста по conversation_id")
	if r.Num("forwarded") != 1 || r.Num("failed") != 0 {
		t.Fatalf("ожидалось forwarded=1 failed=0, получено: %s", r.Raw)
	}

	msgs := listMsgs(t, colleague, convID, "")
	if len(msgs) == 0 {
		t.Fatalf("плашка поста не пришла в диалог")
	}
	last := msgs[len(msgs)-1]
	if last["kind"] != "post" {
		t.Fatalf("ожидался kind=post, получено: %v", last["kind"])
	}
	post, _ := last["post"].(map[string]any)
	if post == nil || post["title"] == "" {
		t.Fatalf("плашка поста без превью: %v", last)
	}

	// Пересылка плашки поста ВНУТРИ мессенджера сохраняет kind и превью.
	fw := messengerAPI.doJSON(t, http.MethodPost, "/api/messenger/forward", colleague.Token, map[string]any{
		"message_id": int64(last["id"].(float64)), "user_ids": []int64{admin.ID},
	})
	requireStatus(t, fw, 201, "пересылка плашки поста внутри мессенджера")
	fwList := fw.List("forwarded")
	if len(fwList) != 1 {
		t.Fatalf("ожидалась 1 доставка пересылки: %s", fw.Raw)
	}
	fwMsg := fwList[0].(map[string]any)["message"].(map[string]any)
	if fwMsg["kind"] != "post" {
		t.Fatalf("пересланная плашка потеряла kind=post: %v", fwMsg["kind"])
	}
	fwPost, _ := fwMsg["post"].(map[string]any)
	if fwPost == nil || fwPost["title"] != post["title"] {
		t.Fatalf("пересланная плашка потеряла превью поста: %v", fwMsg)
	}

	// По user_ids (без явного conversation_id — резолвится через EnsureDialog).
	stranger := newMember(t, admin, companyID, roleEmployee)
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/forward", postID), author.Token,
		map[string]any{"user_ids": []int64{stranger.ID}})
	requireStatus(t, r, 200, "пересылка поста по user_id")
	if r.Num("forwarded") != 1 {
		t.Fatalf("ожидалось forwarded=1 при резолве по user_id, получено: %s", r.Raw)
	}
}

// Пересылка в диалог, в котором отправитель НЕ участвует, отвергается
// msgsvc (проверка участия в CreatePostMessage) и честно считается failed.
func TestPortalForward_ForeignConversationRejected(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-чужой-диалог "))
	author := newMember(t, admin, companyID, roleEmployee)
	memberA := newMember(t, admin, companyID, roleEmployee)
	memberB := newMember(t, admin, companyID, roleEmployee)

	postID := createPost(t, author, "Пост, который нельзя подбросить в чужую переписку.")

	// Диалог между A и B — автор поста в нём не участник.
	foreignConv := openConv(t, memberA, memberB.ID)

	r := portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/forward", postID), author.Token,
		map[string]any{"conversation_ids": []int64{foreignConv}})
	requireStatus(t, r, 200, "пересылка в чужой диалог")
	if r.Num("forwarded") != 0 || r.Num("failed") != 1 {
		t.Fatalf("ожидалось forwarded=0 failed=1, получено: %s", r.Raw)
	}

	// Плашка в чужом диалоге не появилась.
	for _, m := range listMsgs(t, memberA, foreignConv, "") {
		if m["kind"] == "post" {
			t.Fatalf("плашка поста попала в чужой диалог: %v", m)
		}
	}
}

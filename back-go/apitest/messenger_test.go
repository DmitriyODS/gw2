package apitest

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// ── Хелперы мессенджера ──────────────────────────────────────────

// openConv — открыть/найти диалог с пользователем, вернуть его id.
func openConv(t *testing.T, a *actor, otherID int64) int64 {
	t.Helper()
	r := messengerAPI.doJSON(t, http.MethodPost, "/api/messenger/conversations", a.Token,
		map[string]any{"user_id": otherID})
	requireStatus(t, r, 200, fmt.Sprintf("открытие диалога с %d", otherID))
	id := int64(r.Num("id"))
	if id == 0 {
		t.Fatalf("открытие диалога: нет id: %s", r.Raw)
	}
	return id
}

// sendMsg — отправить сообщение (body поверх дефолта), вернуть ответ.
func sendMsg(t *testing.T, a *actor, convID int64, body map[string]any) apiResp {
	t.Helper()
	return messengerAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/messenger/conversations/%d/messages", convID), a.Token, body)
}

// listMsgs — список сообщений диалога (top-level массив) как []map.
func listMsgs(t *testing.T, a *actor, convID int64, query string) []map[string]any {
	t.Helper()
	r := messengerAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/messenger/conversations/%d/messages%s", convID, query), a.Token, nil)
	requireStatus(t, r, 200, "список сообщений")
	var out []map[string]any
	if err := jsonUnmarshal(r.Raw, &out); err != nil {
		t.Fatalf("разбор списка сообщений: %v; тело: %s", err, r.Raw)
	}
	return out
}

// convItem — элемент списка /conversations по id ("" — не найден).
func convItem(t *testing.T, a *actor, convID int64) map[string]any {
	t.Helper()
	r := messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/conversations", a.Token, nil)
	requireStatus(t, r, 200, "список диалогов")
	var items []map[string]any
	if err := jsonUnmarshal(r.Raw, &items); err != nil {
		t.Fatalf("разбор списка диалогов: %v; тело: %s", err, r.Raw)
	}
	for _, it := range items {
		if int64(it["id"].(float64)) == convID {
			return it
		}
	}
	return nil
}

// ── Диалоги 1:1 ──────────────────────────────────────────────────

func TestMessengerDialogLifecycle(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Мессенджер "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newMember(t, admin, companyID, roleEmployee)

	// Идемпотентность: повторное открытие возвращает тот же диалог (пара a<b).
	conv1 := openConv(t, a, b.ID)
	conv2 := openConv(t, a, b.ID)
	if conv1 != conv2 {
		t.Fatalf("повторное открытие дало другой диалог: %d != %d", conv1, conv2)
	}
	// С другой стороны — та же пара.
	conv3 := openConv(t, b, a.ID)
	if conv3 != conv1 {
		t.Fatalf("диалог со стороны B отличается: %d != %d", conv3, conv1)
	}

	// Нельзя открыть диалог с самим собой.
	r := messengerAPI.doJSON(t, http.MethodPost, "/api/messenger/conversations", a.Token,
		map[string]any{"user_id": a.ID})
	requireError(t, r, 400, "SELF_CONVERSATION", "диалог с самим собой")

	// Без токена — 401.
	r = messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/conversations", "", nil)
	requireStatus(t, r, 401, "список диалогов без токена")

	// Отправка текста.
	m := sendMsg(t, a, conv1, map[string]any{"text": "привет"})
	requireStatus(t, m, 201, "отправка сообщения")
	if m.Str("text") != "привет" {
		t.Fatalf("текст сообщения не совпал: %s", m.Raw)
	}

	// Пустое сообщение отклоняется.
	r = sendMsg(t, a, conv1, map[string]any{"text": "   "})
	requireError(t, r, 400, "EMPTY_MESSAGE", "пустое сообщение")

	// Чужой (не участник) не видит переписку.
	c := newMember(t, admin, companyID, roleEmployee)
	r = messengerAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/messenger/conversations/%d/messages", conv1), c.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "доступ чужого к диалогу")
}

// Регрессия: отправка сообщения в ГРУППУ (user_a_id диалога NULL) не должна
// падать 500 при сканировании снапшота; проверяем и «кто прочитал».
func TestMessengerGroupSendAndReadBy(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Группа "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newMember(t, admin, companyID, roleEmployee)

	// a создаёт группу с участником b.
	r := messengerAPI.doJSON(t, http.MethodPost, "/api/messenger/groups", a.Token,
		map[string]any{"title": "Команда", "member_ids": []int64{b.ID}})
	requireStatus(t, r, 201, "создание группы")
	groupID := int64(r.Num("id"))
	if groupID == 0 {
		t.Fatalf("создание группы: нет id: %s", r.Raw)
	}

	// Отправка текста в группу — раньше падало 500 (NULL user_a_id → int64).
	m := sendMsg(t, a, groupID, map[string]any{"text": "всем привет"})
	requireStatus(t, m, 201, "отправка сообщения в группу")
	if m.Str("text") != "всем привет" {
		t.Fatalf("текст группового сообщения не совпал: %s", m.Raw)
	}
	msgID := int64(m.Num("id"))

	// b видит сообщение (плюс системная плашка создания группы).
	msgs := listMsgs(t, b, groupID, "")
	found := false
	for _, mm := range msgs {
		if s, _ := mm["text"].(string); s == "всем привет" {
			found = true
		}
	}
	if !found {
		t.Fatalf("b не видит групповое сообщение: %v", msgs)
	}

	// До прочтения b — читателей нет.
	rb := messengerAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/messenger/messages/%d/read-by", msgID), a.Token, nil)
	requireStatus(t, rb, 200, "read-by до прочтения")

	// b читает группу → у a в read-by появляется b.
	rr := messengerAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/messenger/conversations/%d/read", groupID), b.Token, nil)
	requireStatus(t, rr, 200, "отметка прочтения b")

	rb = messengerAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/messenger/messages/%d/read-by", msgID), a.Token, nil)
	requireStatus(t, rb, 200, "read-by после прочтения")
	var readBy struct {
		Readers []map[string]any `json:"readers"`
	}
	if err := jsonUnmarshal(rb.Raw, &readBy); err != nil {
		t.Fatalf("разбор read-by: %v; тело: %s", err, rb.Raw)
	}
	if len(readBy.Readers) != 1 || int64(readBy.Readers[0]["id"].(float64)) != b.ID {
		t.Fatalf("read-by ожидал [b], получил: %s", rb.Raw)
	}
}

func TestMessengerPagination(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Пагинация "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newMember(t, admin, companyID, roleEmployee)
	conv := openConv(t, a, b.ID)

	var ids []int64
	for i := 0; i < 3; i++ {
		m := sendMsg(t, a, conv, map[string]any{"text": fmt.Sprintf("m%d", i)})
		requireStatus(t, m, 201, "отправка")
		ids = append(ids, int64(m.Num("id")))
	}

	all := listMsgs(t, a, conv, "")
	if len(all) != 3 {
		t.Fatalf("ожидалось 3 сообщения, получено %d", len(all))
	}
	// limit=2 → два последних (по возрастанию id).
	last2 := listMsgs(t, a, conv, "?limit=2")
	if len(last2) != 2 {
		t.Fatalf("limit=2: ожидалось 2, получено %d", len(last2))
	}
	if int64(last2[1]["id"].(float64)) != ids[2] {
		t.Fatalf("limit=2: последнее сообщение не самое свежее: %v", last2)
	}
	// before_id первого из last2 → более старые.
	older := listMsgs(t, a, conv, fmt.Sprintf("?before_id=%d", ids[1]))
	if len(older) != 1 || int64(older[0]["id"].(float64)) != ids[0] {
		t.Fatalf("before_id: ожидался только первый, получено %v", older)
	}
}

func TestMessengerAttachmentsAndForward(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Вложения "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newMember(t, admin, companyID, roleEmployee)
	c := newMember(t, admin, companyID, roleEmployee)
	convAB := openConv(t, a, b.ID)

	// Пустой файл отклоняется.
	r := messengerAPI.doMultipart(t, "/api/messenger/uploads", a.Token, "x.png", nil)
	requireError(t, r, 400, "EMPTY_FILE", "пустой файл")

	// Загрузка вложения и отправка с ним.
	up := messengerAPI.doMultipart(t, "/api/messenger/uploads", a.Token, "photo.png",
		[]byte("\x89PNG\r\n\x1a\nfake-image-bytes"))
	requireStatus(t, up, 201, "загрузка вложения")
	attID := int64(up.Num("id"))
	srcURL := up.Str("url")
	if attID == 0 || srcURL == "" {
		t.Fatalf("загрузка: нет id/url: %s", up.Raw)
	}
	// Файл действительно лёг на диск.
	srcPath := filepath.Join(uploadsDir, srcURL[len("/uploads/"):])
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("файл вложения не найден на диске: %v", err)
	}

	m := sendMsg(t, a, convAB, map[string]any{"text": "смотри", "attachment_ids": []int64{attID}})
	requireStatus(t, m, 201, "сообщение с вложением")
	srcMsgID := int64(m.Num("id"))

	// Пересылка сообщения пользователю C: файл копируется физически.
	fw := messengerAPI.doJSON(t, http.MethodPost, "/api/messenger/forward", a.Token, map[string]any{
		"message_id": srcMsgID, "user_ids": []int64{c.ID},
	})
	requireStatus(t, fw, 201, "пересылка")
	fwList := fw.List("forwarded")
	if len(fwList) != 1 {
		t.Fatalf("пересылка: ожидалась 1 доставка: %s", fw.Raw)
	}
	fwMsg := fwList[0].(map[string]any)["message"].(map[string]any)
	fwAtt := fwMsg["attachments"].([]any)
	if len(fwAtt) != 1 {
		t.Fatalf("у пересланного сообщения нет вложения: %s", fw.Raw)
	}
	fwURL := fwAtt[0].(map[string]any)["url"].(string)
	if fwURL == srcURL {
		t.Fatalf("пересланный файл делит путь с оригиналом (не скопирован): %s", fwURL)
	}
	fwPath := filepath.Join(uploadsDir, fwURL[len("/uploads/"):])
	if _, err := os.Stat(fwPath); err != nil {
		t.Fatalf("скопированный файл не найден на диске: %v", err)
	}
}

func TestMessengerReplyEditDelete(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Ответы "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newMember(t, admin, companyID, roleEmployee)
	conv := openConv(t, a, b.ID)

	m1 := sendMsg(t, a, conv, map[string]any{"text": "вопрос"})
	requireStatus(t, m1, 201, "m1")
	m1ID := int64(m1.Num("id"))

	m2 := sendMsg(t, b, conv, map[string]any{"text": "ответ", "reply_to_id": m1ID})
	requireStatus(t, m2, 201, "m2")
	m2ID := int64(m2.Num("id"))
	if m2.JSON["reply_to"] == nil {
		t.Fatalf("reply_to не проставился: %s", m2.Raw)
	}

	// Правка чужого сообщения запрещена (a правит сообщение b).
	r := messengerAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/messenger/messages/%d", m2ID), a.Token, map[string]any{"text": "взлом"})
	requireError(t, r, 403, "FORBIDDEN", "правка чужого сообщения")

	// Автор редактирует своё → edited_at выставляется.
	r = messengerAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/messenger/messages/%d", m2ID), b.Token, map[string]any{"text": "ответ*"})
	requireStatus(t, r, 200, "правка своего сообщения")
	if r.JSON["edited_at"] == nil || r.Str("text") != "ответ*" {
		t.Fatalf("правка не отразилась: %s", r.Raw)
	}

	// Удаление цели ответа «у всех» → reply_to у m2 сбрасывается (FK SET NULL).
	r = messengerAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/messenger/messages/%d?scope=all", m1ID), a.Token, nil)
	requireStatus(t, r, 200, "удаление m1 у всех")
	if !r.Bool("for_all") {
		t.Fatalf("ожидалось for_all=true: %s", r.Raw)
	}
	msgs := listMsgs(t, b, conv, "")
	for _, mm := range msgs {
		if int64(mm["id"].(float64)) == m2ID && mm["reply_to"] != nil {
			t.Fatalf("reply_to не сброшен после удаления цели: %v", mm)
		}
	}
}

func TestMessengerSoftDeleteBothSides(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Удаление "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newMember(t, admin, companyID, roleEmployee)
	conv := openConv(t, a, b.ID)

	m := sendMsg(t, a, conv, map[string]any{"text": "к удалению"})
	requireStatus(t, m, 201, "сообщение")
	mID := int64(m.Num("id"))

	// B скрывает у себя — у A ещё видно.
	r := messengerAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/messenger/messages/%d?scope=me", mID), b.Token, nil)
	requireStatus(t, r, 200, "B скрывает у себя")
	if r.Bool("for_all") {
		t.Fatalf("одно сокрытие не должно давать for_all: %s", r.Raw)
	}
	if len(listMsgs(t, b, conv, "")) != 0 {
		t.Fatalf("сообщение всё ещё видно у B")
	}
	if len(listMsgs(t, a, conv, "")) != 1 {
		t.Fatalf("сообщение пропало у A после сокрытия у B")
	}

	// A тоже скрывает — обе стороны скрыли → физическое удаление.
	r = messengerAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/messenger/messages/%d?scope=me", mID), a.Token, nil)
	requireStatus(t, r, 200, "A скрывает у себя")
	if !r.Bool("for_all") {
		t.Fatalf("после сокрытия обеими сторонами ожидалось физическое удаление: %s", r.Raw)
	}
	var cnt int
	if err := db.QueryRow(dbCtx(t), `SELECT count(*) FROM messages WHERE id=$1`, mID).Scan(&cnt); err != nil {
		t.Fatalf("проверка физического удаления: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("сообщение не удалено физически: осталось %d строк", cnt)
	}
}

func TestMessengerPinning(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Закрепление "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newMember(t, admin, companyID, roleEmployee)
	conv := openConv(t, a, b.ID)

	m := sendMsg(t, a, conv, map[string]any{"text": "важное"})
	requireStatus(t, m, 201, "сообщение")
	mID := int64(m.Num("id"))

	// Закрепление сообщения — общее (видят оба).
	r := messengerAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/messenger/messages/%d/pin", mID), a.Token, nil)
	requireStatus(t, r, 200, "закрепление сообщения")
	if !r.Bool("pinned") {
		t.Fatalf("сообщение не закрепилось: %s", r.Raw)
	}
	r = messengerAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/messenger/conversations/%d/pinned", conv), b.Token, nil)
	requireStatus(t, r, 200, "закреплённые у B")
	var pinned []map[string]any
	_ = jsonUnmarshal(r.Raw, &pinned)
	if len(pinned) != 1 {
		t.Fatalf("B не видит закреплённое сообщение: %s", r.Raw)
	}

	// Закрепление диалога — личное.
	r = messengerAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/messenger/conversations/%d/pin", conv), a.Token, nil)
	requireStatus(t, r, 200, "закрепление диалога")
	if !r.Bool("is_pinned") {
		t.Fatalf("диалог не закрепился: %s", r.Raw)
	}
	if it := convItem(t, a, conv); it == nil || it["is_pinned"] != true {
		t.Fatalf("диалог не отмечен закреплённым у A: %v", it)
	}
	// У B закрепление личное — не проставлено.
	if it := convItem(t, b, conv); it == nil || it["is_pinned"] != false {
		t.Fatalf("личное закрепление протекло к B: %v", it)
	}
}

// TestMessengerSupportChatAutoAppears — личный чат техподдержки (dev-чат)
// должен появляться в списке диалогов сотрудника компании сам, ещё до первой
// переписки (см. комментарий в ListConversations: «должен существовать всегда»).
// Активная компания живёт только в токене, поэтому её нужно донести до
// ListConversations — иначе dev-чат не создаётся.
func TestMessengerSupportChatAutoAppears(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Поддержка "))
	a := newMember(t, admin, companyID, roleEmployee)

	r := messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/conversations", a.Token, nil)
	requireStatus(t, r, 200, "список диалогов сотрудника")
	var items []map[string]any
	if err := jsonUnmarshal(r.Raw, &items); err != nil {
		t.Fatalf("разбор списка диалогов: %v", err)
	}
	found := false
	for _, it := range items {
		if it["is_dev_chat"] == true {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("dev-чат техподдержки не появился в списке сам: %s", r.Raw)
	}
}

// TestMessengerCrossCompany — диалог 1:1 разрешён между человеком с компанией
// и человеком без общей/любой компании (company_id диалога — NULL).
func TestMessengerCrossCompany(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Кросс "))
	a := newMember(t, admin, companyID, roleEmployee)
	b := newVerifiedUser(t) // без компании вовсе

	conv := openConv(t, a, b.ID)
	m := sendMsg(t, a, conv, map[string]any{"text": "кросс-компанийный привет"})
	requireStatus(t, m, 201, "сообщение без общей компании")

	var companyDB *int64
	if err := db.QueryRow(dbCtx(t),
		`SELECT company_id FROM conversations WHERE id=$1`, conv).Scan(&companyDB); err != nil {
		t.Fatalf("чтение company_id диалога: %v", err)
	}
	if companyDB != nil {
		t.Fatalf("ожидался company_id=NULL для кросс-компанийного диалога, получено %d", *companyDB)
	}
	// B (без компании) читает и отвечает.
	if len(listMsgs(t, b, conv, "")) != 1 {
		t.Fatalf("B не видит сообщение кросс-компанийного диалога")
	}
	requireStatus(t, sendMsg(t, b, conv, map[string]any{"text": "и тебе"}), 201, "ответ B")
}

package apitest

import (
	"fmt"
	"net/http"
	"testing"
)

// ── Dev-чат (техподдержка) и inbox супер-админа ──────────────────

func TestMessengerDevChatAndSupportInbox(t *testing.T) {
	root := newSuperAdmin(t)

	// Без активной компании dev-чата нет.
	solo := newVerifiedUser(t)
	r := messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/dev-chat", solo.Token, nil)
	requireError(t, r, 400, "NO_ACTIVE_COMPANY", "dev-чат без компании")

	owner := newVerifiedUser(t)
	owner.createCompany(t, uniq("Поддержка "))
	r = messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/dev-chat", owner.Token, nil)
	requireStatus(t, r, 200, "открытие dev-чата")
	devConv := int64(r.Num("id"))
	if r.JSON["is_dev_chat"] != true {
		t.Fatalf("это не dev-чат: %s", r.Raw)
	}

	// Первое обращение → синхронный автоответ бота (kind=dev_reply, is_bot).
	requireStatus(t, sendMsg(t, owner, devConv, map[string]any{"text": "всё сломалось"}),
		201, "обращение в поддержку")
	botCount := 0
	for _, m := range listMsgs(t, owner, devConv, "") {
		if m["is_bot"] == true {
			botCount++
			if m["kind"] != "system_dev_reply" {
				t.Fatalf("автоответ не system_dev_reply: %v", m)
			}
		}
	}
	if botCount != 1 {
		t.Fatalf("ожидался ровно один автоответ, получено %d", botCount)
	}
	// Автоответ (sender NULL) — непрочитанное владельца (инвариант фильтра
	// sender_id IS NULL OR sender_id != me).
	if it := convItem(t, owner, devConv); it == nil || it["unread_count"].(float64) != 1 {
		t.Fatalf("автоответ не попал в unread владельца: %v", it)
	}
	// Второе сообщение в те же сутки — без нового автоответа.
	requireStatus(t, sendMsg(t, owner, devConv, map[string]any{"text": "и ещё вопрос"}),
		201, "второе обращение")
	botCount = 0
	for _, m := range listMsgs(t, owner, devConv, "") {
		if m["is_bot"] == true {
			botCount++
		}
	}
	if botCount != 1 {
		t.Fatalf("автоответ продублировался: %d за сутки", botCount)
	}

	// Support inbox: не супер-админу — 403; чат владельца в списке, unread —
	// только человеческие сообщения владельца (2), бот не считается.
	r = messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/support-inbox", owner.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "inbox не супер-админом")
	r = messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/support-inbox", root.Token, nil)
	requireStatus(t, r, 200, "support-inbox")
	var inbox []map[string]any
	_ = jsonUnmarshal(r.Raw, &inbox)
	var entry map[string]any
	for _, it := range inbox {
		if int64(it["id"].(float64)) == devConv {
			entry = it
		}
	}
	if entry == nil {
		t.Fatalf("dev-чат %d не попал в support-inbox", devConv)
	}
	if ou, _ := entry["owner_user"].(map[string]any); ou == nil || int64(ou["id"].(float64)) != owner.ID {
		t.Fatalf("owner_user в inbox: %v", entry)
	}
	if entry["unread_count"].(float64) != 2 {
		t.Fatalf("unread для админа: %v, ожидалось 2 (без бота)", entry["unread_count"])
	}

	// Супер-админ отвечает в чужом dev-чате — kind=dev_reply; посторонний —
	// не имеет доступа вовсе.
	ra := sendMsg(t, root, devConv, map[string]any{"text": "чиним"})
	requireStatus(t, ra, 201, "ответ супер-админа")
	if ra.Str("kind") != "system_dev_reply" {
		t.Fatalf("ответ супер-админа должен быть system_dev_reply: %s", ra.Raw)
	}
	stranger := newVerifiedUser(t)
	r = messengerAPI.doJSON(t, http.MethodGet,
		fmt.Sprintf("/api/messenger/conversations/%d/messages", devConv), stranger.Token, nil)
	requireError(t, r, 403, "FORBIDDEN", "dev-чат посторонним")

	// Чат техподдержки удалить нельзя.
	r = messengerAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/messenger/conversations/%d?scope=all", devConv), owner.Token, nil)
	requireError(t, r, 400, "DEV_CHAT_UNDELETABLE", "удаление dev-чата")
}

// TestMessengerPetChatDeleteRecreates — удаление pet-чата всегда физическое;
// следующий открывший запрос создаёт его заново пустым.
func TestMessengerPetChatDeleteRecreates(t *testing.T) {
	owner := newVerifiedUser(t)
	owner.createCompany(t, uniq("Грувик-чат "))

	r := messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/pet-chat", owner.Token, nil)
	requireStatus(t, r, 200, "открытие pet-чата")
	petConv := int64(r.Num("id"))
	requireStatus(t, sendMsg(t, owner, petConv, map[string]any{"text": "история"}),
		201, "сообщение в pet-чат")

	r = messengerAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/messenger/conversations/%d?scope=me", petConv), owner.Token, nil)
	requireStatus(t, r, 200, "удаление pet-чата")
	if !r.Bool("physical") {
		t.Fatalf("удаление pet-чата должно быть физическим: %s", r.Raw)
	}

	r = messengerAPI.doJSON(t, http.MethodGet, "/api/messenger/pet-chat", owner.Token, nil)
	requireStatus(t, r, 200, "пересоздание pet-чата")
	newConv := int64(r.Num("id"))
	if newConv == petConv {
		t.Fatalf("pet-чат не пересоздался (тот же id %d)", petConv)
	}
	if msgs := listMsgs(t, owner, newConv, ""); len(msgs) != 0 {
		t.Fatalf("пересозданный pet-чат не пуст: %v", msgs)
	}
}

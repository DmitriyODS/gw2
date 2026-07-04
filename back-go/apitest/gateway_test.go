package apitest

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// gatewayAPI — REST шлюза (exact /api/messenger/presence живёт в нём).
var gatewayAPI = &svcClient{base: gatewayBase}

// presenceOnline — снимок GET /api/messenger/presence.
func presenceOnline(t *testing.T, a *actor) map[int64]bool {
	t.Helper()
	r := gatewayAPI.doJSON(t, http.MethodGet, "/api/messenger/presence", a.Token, nil)
	requireStatus(t, r, 200, "presence")
	out := map[int64]bool{}
	for _, v := range r.List("online") {
		if id, ok := v.(float64); ok {
			out[int64(id)] = true
		}
	}
	return out
}

// waitPresence — дождаться нужного онлайн-состояния пользователя (переходы
// asynchronous: beat/drop и sweeper).
func waitPresence(t *testing.T, viewer *actor, userID int64, online bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if presenceOnline(t, viewer)[userID] == online {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("presence: пользователь %d не перешёл в online=%v за %s", userID, online, timeout)
}

// ── Handshake ────────────────────────────────────────────────────

func TestGatewayHandshake(t *testing.T) {
	// Невалидный токен → _error и закрытие.
	c := dialWS(t)
	c.emit(t, "auth", map[string]any{"token": "не-paseto"})
	f := c.waitFrame(t, "_error", 5*time.Second)
	if f.Obj()["code"] != "AUTH_FAILED" {
		t.Fatalf("ожидался AUTH_FAILED: %s", f.Data)
	}

	// Валидный токен → _connected с нашим user_id.
	a := newVerifiedUser(t)
	ws := connectWS(t, a.Token)
	_ = ws

	// Не-WS запрос на /ws → 426 Upgrade Required.
	r := gatewayAPI.doJSON(t, http.MethodGet, "/ws", "", nil)
	requireStatus(t, r, 426, "GET /ws без upgrade")
}

// ── Presence ─────────────────────────────────────────────────────

func TestGatewayPresence(t *testing.T) {
	a := newVerifiedUser(t)
	viewer := newVerifiedUser(t)

	// REST presence без токена — 401.
	r := gatewayAPI.doJSON(t, http.MethodGet, "/api/messenger/presence", "", nil)
	requireStatus(t, r, 401, "presence без токена")

	ws := connectWS(t, a.Token)
	waitPresence(t, viewer, a.ID, true, 5*time.Second)

	// Вкладка скрыта → офлайн (соединение живо!), видима → снова онлайн.
	ws.emit(t, "presence:visibility", map[string]any{"visible": false})
	waitPresence(t, viewer, a.ID, false, 5*time.Second)
	ws.emit(t, "presence:visibility", map[string]any{"visible": true})
	waitPresence(t, viewer, a.ID, true, 5*time.Second)

	// Отключение → офлайн + last_seen_at записан в users.
	ws.close()
	waitPresence(t, viewer, a.ID, false, 5*time.Second)
	var lastSeen *time.Time
	if err := db.QueryRow(dbCtx(t),
		`SELECT last_seen_at FROM users WHERE id=$1`, a.ID).Scan(&lastSeen); err != nil {
		t.Fatalf("чтение last_seen_at: %v", err)
	}
	if lastSeen == nil {
		t.Fatalf("last_seen_at не записан при уходе в офлайн")
	}
}

// ── Доставка событий и typing ────────────────────────────────────

func TestGatewayMessageDeliveryAndTyping(t *testing.T) {
	a := newVerifiedUser(t)
	b := newVerifiedUser(t)
	c := newVerifiedUser(t) // посторонний: адресные события к нему не текут

	convID := openConv(t, a, b.ID)

	wsA := connectWS(t, a.Token)
	wsB := connectWS(t, b.Token)
	wsC := connectWS(t, c.Token)

	// REST-отправка msgsvc → Redis gw2:messenger:events → мост шлюза → WS.
	m := sendMsg(t, b, convID, map[string]any{"text": "через шлюз"})
	requireStatus(t, m, 201, "сообщение B")
	sameConv := func(f wsFrame) bool {
		id, _ := f.Obj()["conversation_id"].(float64)
		return int64(id) == convID
	}
	got := wsA.waitFrameMatch(t, "message:new", sameConv, 10*time.Second)
	msg, _ := got.Obj()["message"].(map[string]any)
	if msg == nil || msg["text"] != "через шлюз" {
		t.Fatalf("message:new без текста: %s", got.Data)
	}
	// Эхо отправителю (другие вкладки B).
	wsB.waitFrameMatch(t, "message:new", sameConv, 10*time.Second)
	// Постороннему адресное событие не доставляется.
	if f, err := wsC.tryWaitFrame("message:new", sameConv, 1500*time.Millisecond); err == nil {
		t.Fatalf("message:new чужого диалога протёк постороннему: %s", f.Data)
	}

	// Эфемерный typing: A → B без БД (релей шлюза).
	wsA.emit(t, "typing", map[string]any{
		"conversation_id": convID, "to_user_id": b.ID, "typing": true,
	})
	f := wsB.waitFrameMatch(t, "typing", sameConv, 10*time.Second)
	if int64(f.Obj()["user_id"].(float64)) != a.ID || f.Obj()["typing"] != true {
		t.Fatalf("typing-кадр: %s", f.Data)
	}

	// message:updated при правке доезжает обоим.
	msgID := int64(m.Num("id"))
	r := messengerAPI.doJSON(t, http.MethodPatch,
		fmt.Sprintf("/api/messenger/messages/%d", msgID), b.Token,
		map[string]any{"text": "поправлено"})
	requireStatus(t, r, 200, "правка")
	wsA.waitFrameMatch(t, "message:updated", sameConv, 10*time.Second)

	// message:deleted при удалении «для всех».
	r = messengerAPI.doJSON(t, http.MethodDelete,
		fmt.Sprintf("/api/messenger/messages/%d?scope=all", msgID), b.Token, nil)
	requireStatus(t, r, 200, "удаление для всех")
	f = wsA.waitFrameMatch(t, "message:deleted", sameConv, 10*time.Second)
	if int64(f.Obj()["message_id"].(float64)) != msgID {
		t.Fatalf("message:deleted не про то сообщение: %s", f.Data)
	}
}

// TestGatewayCallsUnavailable — callsvc не поднят: команда call:* обязана
// отвечать call:error CALLS_UNAVAILABLE инициатору, а не молчать/падать.
func TestGatewayCallsUnavailable(t *testing.T) {
	a := newVerifiedUser(t)
	b := newVerifiedUser(t)
	ws := connectWS(t, a.Token)

	ws.emit(t, "call:start", map[string]any{"user_ids": []int64{b.ID}, "media": "audio"})
	f := ws.waitFrame(t, "call:error", 15*time.Second)
	if f.Obj()["code"] != "CALLS_UNAVAILABLE" {
		t.Fatalf("ожидался CALLS_UNAVAILABLE: %s", f.Data)
	}
}

// TestGatewayBridgesAllServiceChannels — мост шлюза подписан на каналы ВСЕХ
// сервисов: события diarysvc (личная комната) и calendarsvc (комната all)
// доезжают до WS-клиентов. Регресс: calendar/diary отсутствовали в списке
// каналов моста — их realtime молча терялся.
func TestGatewayBridgesAllServiceChannels(t *testing.T) {
	// Diary: событие адресовано владельцу (user_{id}).
	owner := newVerifiedUser(t)
	ws := connectWS(t, owner.Token)
	name := uniq("Ежедневник WS ")
	diaryID := createDiary(t, owner, name)
	f := ws.waitFrameMatch(t, "diary:created", func(f wsFrame) bool {
		id, _ := f.Obj()["id"].(float64)
		return int64(id) == diaryID
	}, 10*time.Second)
	if f.Obj()["name"] != name {
		t.Fatalf("diary:created без имени: %s", f.Data)
	}
	// Запись дня — diary_entry:created (имя с префиксом, чтобы не пересекаться
	// с entry:* календаря); payload плоский.
	entryID := createEntry(t, owner, diaryID, "2026-07-04", "Запись WS", nil)
	ws.waitFrameMatch(t, "diary_entry:created", func(f wsFrame) bool {
		id, _ := f.Obj()["id"].(float64)
		return int64(id) == entryID
	}, 10*time.Second)

	// Calendar: события в комнату all с company_id в payload.
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Календарь WS "))
	wsAdmin := connectWS(t, admin.Token)
	calID := createCalendar(t, admin, uniq("Календарь "))
	wsAdmin.waitFrameMatch(t, "calendar:created", func(f wsFrame) bool {
		id, _ := f.Obj()["id"].(float64)
		return int64(id) == calID
	}, 10*time.Second)
	eventID := createEvent(t, admin, calID, "2026-07-04T12:30:00Z", nil)
	f = wsAdmin.waitFrameMatch(t, "entry:created", func(f wsFrame) bool {
		id, _ := f.Obj()["id"].(float64)
		return int64(id) == eventID
	}, 10*time.Second)
	if int64(f.Obj()["company_id"].(float64)) != companyID {
		t.Fatalf("entry:created без company_id: %s", f.Data)
	}
}

package apitest

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// Звонки сквозь шлюз: WS-команды call:* → gatewaysvc → gRPC callsvc → БД и
// события обратно. Отдельный экземпляр шлюза (callsGatewayWS) ходит в
// поднятый callsvc; основной шлюз проверяет CALLS_UNAVAILABLE.

const callWait = 15 * time.Second

func frameCallID(f wsFrame) int64 {
	obj := f.Obj()
	if call, ok := obj["call"].(map[string]any); ok {
		if id, ok := call["id"].(float64); ok {
			return int64(id)
		}
	}
	if id, ok := obj["call_id"].(float64); ok {
		return int64(id)
	}
	if id, ok := obj["id"].(float64); ok {
		return int64(id)
	}
	return 0
}

func forCall(callID int64) func(wsFrame) bool {
	return func(f wsFrame) bool { return frameCallID(f) == callID }
}

// startCall — a звонит b; возвращает id звонка, у b уже звонит.
func startCall(t *testing.T, wsA, wsB *wsClient, b *actor) int64 {
	t.Helper()
	wsA.emit(t, "call:start", map[string]any{"user_ids": []int64{b.ID}, "media": "audio"})
	started := wsA.waitFrame(t, "call:started", callWait)
	callID := frameCallID(started)
	if callID == 0 {
		t.Fatalf("call:started без id: %s", started.Data)
	}
	if lk, _ := started.Obj()["livekit"].(map[string]any); lk == nil || lk["token"] == "" {
		t.Fatalf("инициатор не получил токен LiveKit: %s", started.Data)
	}
	wsB.waitFrameMatch(t, "call:incoming", forCall(callID), callWait)
	return callID
}

type callRow struct {
	Status     string
	AnsweredAt *time.Time
	EndedAt    *time.Time
	OpenParts  int
}

func loadCall(t *testing.T, callID int64) callRow {
	t.Helper()
	var r callRow
	err := db.QueryRow(dbCtx(t), `
		SELECT status, answered_at, ended_at,
		       (SELECT count(*) FROM call_participants WHERE call_id = c.id AND left_at IS NULL)
		FROM calls c WHERE id = $1`, callID).Scan(&r.Status, &r.AnsweredAt, &r.EndedAt, &r.OpenParts)
	if err != nil {
		t.Fatalf("звонок %d: %v", callID, err)
	}
	return r
}

// Разговор состоялся и закончился: длительность — от ответа, все участники
// вышли, история отдаёт звонок с участниками.
func TestCallsP2PAnsweredAndFinished(t *testing.T) {
	a, b := newVerifiedUser(t), newVerifiedUser(t)
	wsA := connectWSAt(t, callsGatewayWS, a.Token)
	wsB := connectWSAt(t, callsGatewayWS, b.Token)
	callID := startCall(t, wsA, wsB, b)

	wsB.emit(t, "call:accept", map[string]any{"call_id": callID})
	wsB.waitFrameMatch(t, "call:accepted", forCall(callID), callWait)
	if r := loadCall(t, callID); r.Status != "active" || r.AnsweredAt == nil {
		t.Fatalf("после ответа: %+v", r)
	}

	wsB.emit(t, "call:leave", map[string]any{"call_id": callID})
	ended := wsA.waitFrameMatch(t, "call:ended", forCall(callID), callWait)
	if ended.Obj()["status"] != "ended" {
		t.Fatalf("call:ended: %s", ended.Data)
	}
	r := loadCall(t, callID)
	if r.Status != "ended" || r.EndedAt == nil {
		t.Fatalf("звонок не закрыт: %+v", r)
	}
	if r.OpenParts != 0 {
		t.Errorf("участников «в звонке» после завершения: %d", r.OpenParts)
	}

	resp := callsAPI.doJSON(t, http.MethodGet, "/api/calls/history", a.Token, nil)
	requireStatus(t, resp, 200, "история звонков")
	var history []map[string]any
	if err := json.Unmarshal(resp.Raw, &history); err != nil {
		t.Fatalf("история: %v; %s", err, resp.Raw)
	}
	var found map[string]any
	for _, c := range history {
		if int64(c["id"].(float64)) == callID {
			found = c
		}
	}
	if found == nil {
		t.Fatalf("звонка %d нет в истории: %s", callID, resp.Raw)
	}
	if found["duration_sec"] == nil {
		t.Error("у состоявшегося звонка нет длительности")
	}
	if parts, _ := found["participants"].([]any); len(parts) != 2 {
		t.Errorf("участников в истории: %d, ожидалось 2", len(parts))
	}
}

// «Принять» и сразу «положить трубку»: шлюз обязан исполнить команды по
// порядку — иначе сервер оставлял человека в звонке, и он был «занят».
func TestCallsCommandsKeepOrder(t *testing.T) {
	a, b := newVerifiedUser(t), newVerifiedUser(t)
	wsA := connectWSAt(t, callsGatewayWS, a.Token)
	wsB := connectWSAt(t, callsGatewayWS, b.Token)

	for i := 0; i < 3; i++ {
		callID := startCall(t, wsA, wsB, b)
		wsB.emit(t, "call:accept", map[string]any{"call_id": callID})
		wsB.emit(t, "call:leave", map[string]any{"call_id": callID})
		wsA.waitFrameMatch(t, "call:ended", forCall(callID), callWait)

		active := callsAPI.doJSON(t, http.MethodGet, "/api/calls/active", b.Token, nil)
		requireStatus(t, active, 200, "активный звонок")
		if active.JSON["call"] != nil || active.JSON["incoming"] != nil {
			t.Fatalf("круг %d: собеседник остался в звонке: %s", i, active.Raw)
		}
	}
}

// Отказ делает p2p пропущенным; запоздавшее «принять» не воскрешает его.
func TestCallsDeclinedStaysMissed(t *testing.T) {
	a, b := newVerifiedUser(t), newVerifiedUser(t)
	wsA := connectWSAt(t, callsGatewayWS, a.Token)
	wsB := connectWSAt(t, callsGatewayWS, b.Token)
	callID := startCall(t, wsA, wsB, b)

	wsB.emit(t, "call:decline", map[string]any{"call_id": callID})
	ended := wsA.waitFrameMatch(t, "call:ended", forCall(callID), callWait)
	if ended.Obj()["status"] != "missed" {
		t.Fatalf("после отказа: %s", ended.Data)
	}

	wsB.emit(t, "call:accept", map[string]any{"call_id": callID})
	errFrame := wsB.waitFrame(t, "call:error", callWait)
	if errFrame.Obj()["code"] != "NOT_INVITED" {
		t.Fatalf("запоздавшее принятие: %s", errFrame.Data)
	}
	r := loadCall(t, callID)
	if r.Status != "missed" || r.AnsweredAt != nil || r.OpenParts != 0 {
		t.Fatalf("пропущенный звонок изменился: %+v", r)
	}
}

// Повторная доставка вебхука LiveKit не рассылает завершение дважды.
func TestCallsWebhookDeliveredTwice(t *testing.T) {
	a, b := newVerifiedUser(t), newVerifiedUser(t)
	wsA := connectWSAt(t, callsGatewayWS, a.Token)
	wsB := connectWSAt(t, callsGatewayWS, b.Token)
	callID := startCall(t, wsA, wsB, b)

	event := map[string]any{
		"id":    uniq("EV_"),
		"event": "room_finished",
		"room":  map[string]any{"name": "call-" + strconv.FormatInt(callID, 10)},
	}
	for i := 0; i < 2; i++ {
		if status := postLivekitWebhook(t, event); status != 200 {
			t.Fatalf("вебхук #%d: статус %d", i+1, status)
		}
	}
	wsA.waitFrameMatch(t, "call:ended", forCall(callID), callWait)
	if f, err := wsA.tryWaitFrame("call:ended", forCall(callID), 1500*time.Millisecond); err == nil {
		t.Fatalf("завершение разослано повторно: %s", f.Data)
	}
	if r := loadCall(t, callID); r.Status != "missed" {
		t.Fatalf("комната закрылась до ответа — ожидался missed: %+v", r)
	}
}

// postLivekitWebhook — вебхук, подписанный как это делает LiveKit: JWT HS256
// ключом API с хешем тела в клейме sha256.
func postLivekitWebhook(t *testing.T, event map[string]any) int {
	t.Helper()
	body, _ := json.Marshal(event)
	digest := sha256.Sum256(body)
	now := time.Now().Unix()
	claims, _ := json.Marshal(map[string]any{
		"iss": "devkey", "nbf": now - 10, "exp": now + 60,
		"sha256": base64.StdEncoding.EncodeToString(digest[:]),
	})
	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString(claims)
	mac := hmac.New(sha256.New, []byte(livekitSecret))
	mac.Write([]byte(unsigned))
	token := unsigned + "." + enc.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, callsBase+"/api/calls/livekit-webhook", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/webhook+json")
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("вебхук: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// Package ring — ринг-сигналинг звонков: WS-команды call:* → gRPC callsvc.
//
// Порт прежнего back/app/sockets/call_events.py. Шлюз только резолвит
// пользователя по соединению, зовёт callsvc и эмитит сокет-события по
// данным из ответов (списки адресатов считает callsvc); вся бизнес-логика,
// ринг-state и LiveKit — в callsvc. Оркестрация системной плашки звонка
// в чате тоже в callsvc (он сам ходит в msgsvc и рассылает
// message:new/message:updated через свой Redis-канал).
//
// События публикуются в gw2:gateway:events (общий envelope) — их доставляет
// мост, как и события остальных сервисов.
package ring

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/callspb"
)

const grpcTimeout = 10 * time.Second

// Bus — публикация сокет-событий (events.Publisher gateway-канала).
type Bus interface {
	Publish(ctx context.Context, event string, rooms []string, payload any)
}

// CallsClient — срез gRPC-клиента callsvc (в тестах — фейк).
type CallsClient interface {
	StartCall(ctx context.Context, in *callspb.StartCallRequest, opts ...grpc.CallOption) (*callspb.StartCallResponse, error)
	InviteToCall(ctx context.Context, in *callspb.InviteToCallRequest, opts ...grpc.CallOption) (*callspb.InviteToCallResponse, error)
	AcceptCall(ctx context.Context, in *callspb.AcceptCallRequest, opts ...grpc.CallOption) (*callspb.AcceptCallResponse, error)
	DeclineCall(ctx context.Context, in *callspb.DeclineCallRequest, opts ...grpc.CallOption) (*callspb.DeclineCallResponse, error)
	LeaveCall(ctx context.Context, in *callspb.LeaveCallRequest, opts ...grpc.CallOption) (*callspb.LeaveCallResponse, error)
	EndCall(ctx context.Context, in *callspb.EndCallRequest, opts ...grpc.CallOption) (*callspb.EndCallResponse, error)
}

type Ring struct {
	calls CallsClient
	bus   Bus
	log   *slog.Logger
}

func New(calls CallsClient, bus Bus, log *slog.Logger) *Ring {
	return &Ring{calls: calls, bus: bus, log: log}
}

// Dial — gRPC-подключение к callsvc.
func Dial(addr string) (callspb.CallServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return callspb.NewCallServiceClient(conn), conn, nil
}

// ── парсинг payload'ов команд ────────────────────────────────────

func intOrZero(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

func intList(v any) []int64 {
	raw, _ := v.([]any)
	out := make([]int64, 0, len(raw))
	for _, item := range raw {
		if id := intOrZero(item); id != 0 {
			out = append(out, id)
		}
	}
	return out
}

func userRoom(id int64) string { return "user_" + strconv.FormatInt(id, 10) }

func userRooms(ids []int64) []string {
	rooms := make([]string, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, userRoom(id))
	}
	return rooms
}

// ── конвертация pb → форма CallSchema ────────────────────────────
// Та же форма, что отдавал calls_client.call_to_dict во Flask (REST/сокеты
// фронта на неё опираются): пустые строки → null.

func optStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func optInt(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func callToDict(call *callspb.Call) map[string]any {
	parts := make([]map[string]any, 0, len(call.GetParticipants()))
	for _, p := range call.GetParticipants() {
		parts = append(parts, map[string]any{
			"user_id":     p.GetUserId(),
			"fio":         optStr(p.GetFio()),
			"avatar_path": optStr(p.GetAvatarPath()),
			"role":        p.GetRole(),
			"joined_at":   optStr(p.GetJoinedAt()),
			"left_at":     optStr(p.GetLeftAt()),
			"declined":    p.GetDeclined(),
		})
	}
	var duration any
	if call.DurationSec != nil {
		duration = call.GetDurationSec()
	}
	return map[string]any{
		"id":              call.GetId(),
		"kind":            call.GetKind(),
		"status":          call.GetStatus(),
		"media":           call.GetMedia(),
		"started_at":      optStr(call.GetStartedAt()),
		"ended_at":        optStr(call.GetEndedAt()),
		"initiator_id":    call.GetInitiatorId(),
		"initiator_fio":   optStr(call.GetInitiatorFio()),
		"conversation_id": optInt(call.GetConversationId()),
		"share_code":      optStr(call.GetShareCode()),
		"duration_sec":    duration,
		"participants":    parts,
	}
}

func livekitToDict(info *callspb.LivekitInfo) map[string]any {
	return map[string]any{"token": info.GetToken(), "url": info.GetUrl()}
}

// ── обработка команд ─────────────────────────────────────────────

// Dispatch — обработать WS-команду call:* от пользователя.
func (r *Ring) Dispatch(userID int64, event string, data json.RawMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	var payload map[string]any
	_ = json.Unmarshal(data, &payload)

	switch event {
	case "call:start":
		r.start(ctx, userID, payload)
	case "call:invite":
		r.invite(ctx, userID, payload)
	case "call:accept":
		r.accept(ctx, userID, payload)
	case "call:decline":
		r.decline(ctx, userID, payload)
	case "call:leave":
		r.leave(ctx, userID, payload)
	case "call:end":
		r.end(ctx, userID, payload)
	}
}

// emitError — call:error инициатору команды. Транспортная недоступность
// callsvc — код CALLS_UNAVAILABLE, как прежний Flask-шлюз.
func (r *Ring) emitError(ctx context.Context, userID int64, err *callspb.Error) {
	r.bus.Publish(ctx, "call:error", []string{userRoom(userID)}, map[string]any{
		"code": err.GetCode(), "message": err.GetMessage(),
	})
}

func (r *Ring) emitUnavailable(ctx context.Context, userID int64, method string, err error) {
	r.log.Error("calls_grpc.unavailable", "method", method, "error", err)
	r.bus.Publish(ctx, "call:error", []string{userRoom(userID)}, map[string]any{
		"code": "CALLS_UNAVAILABLE", "message": "Сервис звонков временно недоступен",
	})
}

func (r *Ring) start(ctx context.Context, me int64, data map[string]any) {
	userIDs := intList(data["user_ids"])
	media, _ := data["media"].(string)
	if media == "" {
		media = "video"
	}
	r.log.Info("call.start", "initiator_id", me, "user_ids", userIDs, "media", media)

	resp, err := r.calls.StartCall(ctx, &callspb.StartCallRequest{
		InitiatorId: me, InviteeIds: userIDs, Media: media,
	})
	if err != nil {
		r.emitUnavailable(ctx, me, "StartCall", err)
		return
	}
	if resp.GetError() != nil {
		r.log.Warn("call.start_failed", "initiator_id", me,
			"code", resp.GetError().GetCode(), "message", resp.GetError().GetMessage())
		r.emitError(ctx, me, resp.GetError())
		return
	}

	payload := callToDict(resp.GetCall())

	// Инициатору — подтверждение + токен LiveKit (он входит в комнату сразу
	// и «ждёт» там остальных).
	r.bus.Publish(ctx, "call:started", []string{userRoom(me)}, map[string]any{
		"call":    payload,
		"livekit": livekitToDict(resp.GetLivekit()),
	})

	// Приглашённым — входящий звонок.
	for _, p := range resp.GetCall().GetParticipants() {
		if p.GetRole() == "invitee" {
			r.bus.Publish(ctx, "call:incoming", []string{userRoom(p.GetUserId())}, payload)
		}
	}
	// Системную плашку звонка в чате создаёт callsvc (message:new придёт
	// его каналом).
}

func (r *Ring) invite(ctx context.Context, me int64, data map[string]any) {
	callID := intOrZero(data["call_id"])
	inviteeIDs := intList(data["user_ids"])
	if callID == 0 || len(inviteeIDs) == 0 {
		return
	}

	resp, err := r.calls.InviteToCall(ctx, &callspb.InviteToCallRequest{
		CallId: callID, InviterId: me, InviteeIds: inviteeIDs,
	})
	if err != nil {
		r.emitUnavailable(ctx, me, "InviteToCall", err)
		return
	}
	if resp.GetError() != nil {
		r.emitError(ctx, me, resp.GetError())
		return
	}

	payload := callToDict(resp.GetCall())
	newIDs := resp.GetNewInviteeIds()
	r.log.Info("call.invite", "call_id", callID, "inviter_id", me, "new_ids", newIDs)
	for _, uid := range newIDs {
		r.bus.Publish(ctx, "call:incoming", []string{userRoom(uid)}, payload)
	}
	// Уже находящимся в звонке — обновить метаданные.
	r.bus.Publish(ctx, "call:invited", userRooms(resp.GetNotifyUserIds()), map[string]any{
		"call_id":  callID,
		"user_ids": newIDs,
		"call":     payload,
	})
}

func (r *Ring) accept(ctx context.Context, me int64, data map[string]any) {
	callID := intOrZero(data["call_id"])
	if callID == 0 {
		return
	}

	resp, err := r.calls.AcceptCall(ctx, &callspb.AcceptCallRequest{CallId: callID, UserId: me})
	if err != nil {
		r.emitUnavailable(ctx, me, "AcceptCall", err)
		return
	}
	if resp.GetError() != nil {
		r.emitError(ctx, me, resp.GetError())
		return
	}

	// Принявшему — токен; дальше он подключается к комнате LiveKit, и
	// остальные узнают о нём от самого LiveKit (ParticipantConnected).
	r.bus.Publish(ctx, "call:accepted", []string{userRoom(me)}, map[string]any{
		"call_id": callID,
		"call":    callToDict(resp.GetCall()),
		"livekit": livekitToDict(resp.GetLivekit()),
	})
}

func (r *Ring) decline(ctx context.Context, me int64, data map[string]any) {
	callID := intOrZero(data["call_id"])
	if callID == 0 {
		return
	}

	resp, err := r.calls.DeclineCall(ctx, &callspb.DeclineCallRequest{CallId: callID, UserId: me})
	if err != nil {
		r.emitUnavailable(ctx, me, "DeclineCall", err)
		return
	}
	if resp.GetError() != nil {
		r.emitError(ctx, me, resp.GetError())
		return
	}
	if resp.GetCall() == nil {
		return // звонка уже нет — no-op
	}

	targets := resp.GetNotifyUserIds()
	r.bus.Publish(ctx, "call:participant-declined", userRooms(targets), map[string]any{
		"call_id": callID, "user_id": me,
	})
	if resp.GetEnded() {
		r.bus.Publish(ctx, "call:ended", userRooms(union(targets, me)), map[string]any{
			"call_id": callID, "status": resp.GetCall().GetStatus(),
		})
	}
}

func (r *Ring) leave(ctx context.Context, me int64, data map[string]any) {
	callID := intOrZero(data["call_id"])
	if callID == 0 {
		return
	}

	resp, err := r.calls.LeaveCall(ctx, &callspb.LeaveCallRequest{CallId: callID, UserId: me})
	if err != nil {
		r.emitUnavailable(ctx, me, "LeaveCall", err)
		return
	}
	if resp.GetError() != nil {
		r.emitError(ctx, me, resp.GetError())
		return
	}
	if resp.GetCall() == nil {
		return
	}
	if resp.GetEnded() {
		r.bus.Publish(ctx, "call:ended", userRooms(union(resp.GetNotifyUserIds(), me)), map[string]any{
			"call_id": callID, "status": resp.GetCall().GetStatus(),
		})
	}
}

func (r *Ring) end(ctx context.Context, me int64, data map[string]any) {
	callID := intOrZero(data["call_id"])
	if callID == 0 {
		return
	}

	resp, err := r.calls.EndCall(ctx, &callspb.EndCallRequest{CallId: callID, UserId: me})
	if err != nil {
		r.emitUnavailable(ctx, me, "EndCall", err)
		return
	}
	if resp.GetError() != nil {
		r.emitError(ctx, me, resp.GetError())
		return
	}
	if resp.GetCall() == nil {
		return
	}
	r.bus.Publish(ctx, "call:ended", userRooms(resp.GetNotifyUserIds()), map[string]any{
		"call_id": callID, "status": resp.GetCall().GetStatus(),
	})
}

// union — ids + extra без дублей (порядок сохраняется).
func union(ids []int64, extra int64) []int64 {
	for _, id := range ids {
		if id == extra {
			return ids
		}
	}
	return append(append([]int64{}, ids...), extra)
}

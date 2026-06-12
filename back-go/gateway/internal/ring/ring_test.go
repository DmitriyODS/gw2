package ring

// Тесты ринг-обработчиков: фейковый gRPC-клиент callsvc + фейковая шина.
// Сценарии повторяют прежний Flask-шлюз (sockets/call_events.py).

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	"google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/callspb"
)

type busEvent struct {
	Event   string
	Rooms   []string
	Payload any
}

type fakeBus struct{ events []busEvent }

func (b *fakeBus) Publish(_ context.Context, event string, rooms []string, payload any) {
	b.events = append(b.events, busEvent{event, rooms, payload})
}

func (b *fakeBus) byEvent(event string) []busEvent {
	var out []busEvent
	for _, e := range b.events {
		if e.Event == event {
			out = append(out, e)
		}
	}
	return out
}

type fakeCalls struct {
	startResp   *callspb.StartCallResponse
	acceptResp  *callspb.AcceptCallResponse
	declineResp *callspb.DeclineCallResponse
	err         error
}

func (f *fakeCalls) StartCall(_ context.Context, _ *callspb.StartCallRequest, _ ...grpc.CallOption) (*callspb.StartCallResponse, error) {
	return f.startResp, f.err
}
func (f *fakeCalls) InviteToCall(_ context.Context, _ *callspb.InviteToCallRequest, _ ...grpc.CallOption) (*callspb.InviteToCallResponse, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeCalls) AcceptCall(_ context.Context, _ *callspb.AcceptCallRequest, _ ...grpc.CallOption) (*callspb.AcceptCallResponse, error) {
	return f.acceptResp, f.err
}
func (f *fakeCalls) DeclineCall(_ context.Context, _ *callspb.DeclineCallRequest, _ ...grpc.CallOption) (*callspb.DeclineCallResponse, error) {
	return f.declineResp, f.err
}
func (f *fakeCalls) LeaveCall(_ context.Context, _ *callspb.LeaveCallRequest, _ ...grpc.CallOption) (*callspb.LeaveCallResponse, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeCalls) EndCall(_ context.Context, _ *callspb.EndCallRequest, _ ...grpc.CallOption) (*callspb.EndCallResponse, error) {
	return nil, errors.New("not implemented")
}

func sampleCall() *callspb.Call {
	return &callspb.Call{
		Id: 5, Kind: "p2p", Status: "ringing", Media: "video",
		StartedAt: "2026-06-12T10:00:00Z", InitiatorId: 10,
		InitiatorFio: "Инициатор", ConversationId: 70, ShareCode: "abc",
		Participants: []*callspb.Participant{
			{UserId: 10, Fio: "Инициатор", Role: "initiator", JoinedAt: "2026-06-12T10:00:00Z"},
			{UserId: 20, Fio: "Собеседник", Role: "invitee"},
		},
	}
}

func newRing(calls CallsClient) (*Ring, *fakeBus) {
	bus := &fakeBus{}
	return New(calls, bus, slog.New(slog.DiscardHandler)), bus
}

func TestStartEmitsStartedAndIncoming(t *testing.T) {
	r, bus := newRing(&fakeCalls{startResp: &callspb.StartCallResponse{
		Call:    sampleCall(),
		Livekit: &callspb.LivekitInfo{Token: "T", Url: "/livekit"},
	}})

	r.Dispatch(10, "call:start", json.RawMessage(`{"user_ids": [20], "media": "video"}`))

	started := bus.byEvent("call:started")
	if len(started) != 1 || !reflect.DeepEqual(started[0].Rooms, []string{"user_10"}) {
		t.Fatalf("call:started = %+v", started)
	}
	payload := started[0].Payload.(map[string]any)
	call := payload["call"].(map[string]any)
	if call["id"] != int64(5) || call["initiator_fio"] != "Инициатор" {
		t.Fatalf("call dict = %v", call)
	}
	if call["ended_at"] != nil || call["duration_sec"] != nil {
		t.Fatalf("пустые поля должны быть null: %v", call)
	}
	lk := payload["livekit"].(map[string]any)
	if lk["token"] != "T" || lk["url"] != "/livekit" {
		t.Fatalf("livekit = %v", lk)
	}

	incoming := bus.byEvent("call:incoming")
	if len(incoming) != 1 || !reflect.DeepEqual(incoming[0].Rooms, []string{"user_20"}) {
		t.Fatalf("call:incoming = %+v", incoming)
	}
}

func TestStartBusinessErrorEmitsCallError(t *testing.T) {
	r, bus := newRing(&fakeCalls{startResp: &callspb.StartCallResponse{
		Error: &callspb.Error{Code: "BUSY", Message: "Вы уже в звонке", HttpStatus: 409},
	}})
	r.Dispatch(10, "call:start", json.RawMessage(`{"user_ids": [20]}`))

	errs := bus.byEvent("call:error")
	if len(errs) != 1 || !reflect.DeepEqual(errs[0].Rooms, []string{"user_10"}) {
		t.Fatalf("call:error = %+v", errs)
	}
	payload := errs[0].Payload.(map[string]any)
	if payload["code"] != "BUSY" {
		t.Fatalf("payload = %v", payload)
	}
}

func TestStartTransportErrorEmitsUnavailable(t *testing.T) {
	r, bus := newRing(&fakeCalls{err: errors.New("connection refused")})
	r.Dispatch(10, "call:start", json.RawMessage(`{"user_ids": [20]}`))

	errs := bus.byEvent("call:error")
	if len(errs) != 1 {
		t.Fatalf("call:error = %+v", errs)
	}
	if errs[0].Payload.(map[string]any)["code"] != "CALLS_UNAVAILABLE" {
		t.Fatalf("payload = %v", errs[0].Payload)
	}
}

func TestDeclineEndedNotifiesEveryone(t *testing.T) {
	call := sampleCall()
	call.Status = "missed"
	r, bus := newRing(&fakeCalls{declineResp: &callspb.DeclineCallResponse{
		Call: call, Ended: true, NotifyUserIds: []int64{10},
	}})
	r.Dispatch(20, "call:decline", json.RawMessage(`{"call_id": 5}`))

	declined := bus.byEvent("call:participant-declined")
	if len(declined) != 1 || !reflect.DeepEqual(declined[0].Rooms, []string{"user_10"}) {
		t.Fatalf("participant-declined = %+v", declined)
	}
	ended := bus.byEvent("call:ended")
	if len(ended) != 1 || !reflect.DeepEqual(ended[0].Rooms, []string{"user_10", "user_20"}) {
		t.Fatalf("call:ended = %+v", ended)
	}
	payload := ended[0].Payload.(map[string]any)
	if payload["status"] != "missed" || payload["call_id"] != int64(5) {
		t.Fatalf("payload = %v", payload)
	}
}

func TestDeclineNoCallIsNoop(t *testing.T) {
	r, bus := newRing(&fakeCalls{declineResp: &callspb.DeclineCallResponse{}})
	r.Dispatch(20, "call:decline", json.RawMessage(`{"call_id": 5}`))
	if len(bus.events) != 0 {
		t.Fatalf("no-op ожидался: %+v", bus.events)
	}
}

func TestAcceptEmitsAcceptedToMe(t *testing.T) {
	r, bus := newRing(&fakeCalls{acceptResp: &callspb.AcceptCallResponse{
		Call:    sampleCall(),
		Livekit: &callspb.LivekitInfo{Token: "T2", Url: "/livekit"},
	}})
	r.Dispatch(20, "call:accept", json.RawMessage(`{"call_id": 5}`))

	accepted := bus.byEvent("call:accepted")
	if len(accepted) != 1 || !reflect.DeepEqual(accepted[0].Rooms, []string{"user_20"}) {
		t.Fatalf("call:accepted = %+v", accepted)
	}
	payload := accepted[0].Payload.(map[string]any)
	if payload["call_id"] != int64(5) {
		t.Fatalf("payload = %v", payload)
	}
}

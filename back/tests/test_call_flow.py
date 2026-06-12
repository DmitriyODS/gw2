"""Тесты Socket.IO-шлюза звонков (sockets/call_events.py).

Бизнес-логика звонков живёт в Go-микросервисе callsvc, домен мессенджера
(парный диалог, плашка звонка) — в msgsvc; оба покрыты своими Go-тестами.
Здесь проверяем именно шлюз: хендлеры зовут gRPC обоих сервисов и
раскладывают ответы по сокет-событиям. Вместо callsvc — in-process fake
gRPC-сервер с canned-ответами (ниже), вместо msgsvc — фикстура
fake_messenger из conftest; БД-записи не создаются, LiveKit не нужен.
Нужны dev-БД и Redis (фикстура app пропустит тесты, если их нет).
"""
import json
from concurrent import futures

import grpc
import pytest

from app.extensions import socketio as sio
from app.grpc import calls_pb2, calls_pb2_grpc, messenger_pb2
from tests.conftest import free_port, grpc_direct_execute, make_token

# id, которых нет в dev-БД: шлюз ходит только в фейковые gRPC-сервисы,
# реальные строки звонка/диалога не нужны.
TEST_CALL_ID = 2_000_000_000
TEST_CONV_ID = 2_000_000_001


class FakeCallService(calls_pb2_grpc.CallServiceServicer):
    """Canned-ответы CallService + запись входящих запросов для ассертов."""

    def __init__(self):
        self.requests = []
        self.responses = {}  # имя RPC -> ответ или callable(request)

    def _respond(self, name, request, default):
        self.requests.append((name, request))
        resp = self.responses.get(name, default)
        return resp(request) if callable(resp) else resp

    def StartCall(self, request, context):
        return self._respond("StartCall", request, calls_pb2.StartCallResponse())

    def InviteToCall(self, request, context):
        return self._respond("InviteToCall", request, calls_pb2.InviteToCallResponse())

    def AcceptCall(self, request, context):
        return self._respond("AcceptCall", request, calls_pb2.AcceptCallResponse())

    def DeclineCall(self, request, context):
        return self._respond("DeclineCall", request, calls_pb2.DeclineCallResponse())

    def LeaveCall(self, request, context):
        return self._respond("LeaveCall", request, calls_pb2.LeaveCallResponse())

    def EndCall(self, request, context):
        return self._respond("EndCall", request, calls_pb2.EndCallResponse())


def _reset_grpc_stub():
    # calls_client кэширует channel/stub в module-globals — сбрасываем, чтобы
    # клиент пересоздал их на подменённый CALLS_GRPC_ADDR.
    from app.services import calls_client
    if calls_client._channel is not None:
        calls_client._channel.close()
    calls_client._channel = None
    calls_client._stub = None


@pytest.fixture
def fake_calls(monkeypatch):
    from app.services import calls_client

    servicer = FakeCallService()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    calls_pb2_grpc.add_CallServiceServicer_to_server(servicer, server)
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()

    monkeypatch.setenv("CALLS_GRPC_ADDR", f"127.0.0.1:{port}")
    monkeypatch.setattr(calls_client, "_execute", grpc_direct_execute)
    _reset_grpc_stub()
    yield servicer
    server.stop(None)
    _reset_grpc_stub()


@pytest.fixture
def grpc_down(monkeypatch, messenger_down):
    """CALLS_GRPC_ADDR (и msgsvc — через messenger_down) указывают на порты,
    где никто не слушает."""
    from app.services import calls_client

    monkeypatch.setenv("CALLS_GRPC_ADDR", f"127.0.0.1:{free_port()}")
    monkeypatch.setattr(calls_client, "_execute", grpc_direct_execute)
    _reset_grpc_stub()
    yield
    _reset_grpc_stub()


def _connect(app, user_id):
    client = sio.test_client(
        app,
        auth={"token": make_token(app, user_id)},
        flask_test_client=app.test_client(),
    )
    assert client.is_connected()
    client.get_received()  # сбросить стартовые события (presence и т. п.)
    return client


def _events(received):
    """Список get_received() -> dict {event_name: args[0]}."""
    out = {}
    for r in received:
        out[r["name"]] = r["args"][0] if r.get("args") else None
    return out


def _pb_call(call_id, caller, callee, conversation_id=0, status="ringing",
             media="video"):
    return calls_pb2.Call(
        id=call_id, kind="p2p", status=status, media=media,
        started_at="2026-01-01T10:00:00+00:00",
        initiator_id=caller, conversation_id=conversation_id,
        share_code="sh-test",
        participants=[
            calls_pb2.Participant(user_id=caller, role="initiator"),
            calls_pb2.Participant(user_id=callee, role="invitee"),
        ],
    )


def _pill_json(req):
    """Снапшот плашки звонка, как его отдаёт msgsvc в message_json."""
    return json.dumps({
        "id": 9001,
        "conversation_id": req.conversation_id,
        "kind": "call",
        "call": {"id": req.call_id, "status": "ringing"},
    })


def test_start_call_emits_started_incoming_and_chat_pill(
        app, two_users, fake_calls, fake_messenger):
    caller, callee = two_users
    fake_messenger.responses["EnsureDialog"] = messenger_pb2.EnsureDialogResponse(
        conversation_id=TEST_CONV_ID)
    fake_messenger.responses["CreateCallMessage"] = (
        lambda req: messenger_pb2.CreateCallMessageResponse(
            message_json=_pill_json(req),
            notify_user_ids=[callee, caller],
        ))
    fake_calls.responses["StartCall"] = lambda req: calls_pb2.StartCallResponse(
        call=_pb_call(TEST_CALL_ID, caller, callee,
                      conversation_id=req.conversation_id),
        livekit=calls_pb2.LivekitInfo(token="tok-caller", url="ws://livekit.test"),
    )

    c_caller = _connect(app, caller)
    c_callee = _connect(app, callee)
    try:
        c_caller.emit("call:start", {"user_ids": [callee], "media": "video"})

        # Парный диалог Flask запросил у msgsvc и передал его id в StartCall.
        ens_name, ens_req = fake_messenger.requests[0]
        assert ens_name == "EnsureDialog"
        assert {ens_req.user_a_id, ens_req.user_b_id} == {caller, callee}
        name, req = fake_calls.requests[0]
        assert name == "StartCall"
        assert req.initiator_id == caller
        assert list(req.invitee_ids) == [callee]
        assert req.media == "video"
        assert req.conversation_id == TEST_CONV_ID

        got_caller = _events(c_caller.get_received())
        started = got_caller["call:started"]
        assert started["call"]["id"] == TEST_CALL_ID
        assert started["call"]["conversation_id"] == TEST_CONV_ID
        assert started["call"]["share_code"] == "sh-test"
        assert started["livekit"] == {"token": "tok-caller", "url": "ws://livekit.test"}

        # Плашку создал msgsvc — Flask разослал готовый message_json
        # адресатам из notify_user_ids.
        pill_name, pill_req = fake_messenger.requests[1]
        assert pill_name == "CreateCallMessage"
        assert (pill_req.conversation_id, pill_req.sender_id, pill_req.call_id) == \
            (TEST_CONV_ID, caller, TEST_CALL_ID)

        pill = got_caller["message:new"]
        assert pill["conversation_id"] == TEST_CONV_ID
        assert pill["from_user_id"] == caller
        assert pill["message"]["kind"] == "call"
        assert pill["message"]["call"]["id"] == TEST_CALL_ID

        got_callee = _events(c_callee.get_received())
        assert got_callee["call:incoming"]["id"] == TEST_CALL_ID
        assert got_callee["message:new"]["message"]["kind"] == "call"
    finally:
        for c in (c_caller, c_callee):
            if c.is_connected():
                c.disconnect()


def test_accept_returns_livekit_token_only_to_acceptor(
        app, two_users, fake_calls, fake_messenger):
    caller, callee = two_users
    fake_calls.responses["AcceptCall"] = calls_pb2.AcceptCallResponse(
        call=_pb_call(TEST_CALL_ID, caller, callee, status="active"),
        livekit=calls_pb2.LivekitInfo(token="tok-callee", url="ws://livekit.test"),
    )

    c_caller = _connect(app, caller)
    c_callee = _connect(app, callee)
    try:
        c_callee.emit("call:accept", {"call_id": TEST_CALL_ID})

        name, req = fake_calls.requests[0]
        assert (name, req.call_id, req.user_id) == ("AcceptCall", TEST_CALL_ID, callee)

        got_callee = _events(c_callee.get_received())
        accepted = got_callee["call:accepted"]
        assert accepted["call_id"] == TEST_CALL_ID
        assert accepted["call"]["status"] == "active"
        assert accepted["livekit"]["token"] == "tok-callee"
        # Остальные узнают о принявшем от LiveKit (ParticipantConnected),
        # сокет-событие — только ему самому.
        assert "call:accepted" not in _events(c_caller.get_received())

        # Плашка перечитана у msgsvc; пустой message_json (дефолт фейка) —
        # message:updated никому не уходит.
        assert ("GetCallMessage", TEST_CALL_ID) in [
            (n, r.call_id) for n, r in fake_messenger.requests]
        assert "message:updated" not in got_callee
    finally:
        for c in (c_caller, c_callee):
            if c.is_connected():
                c.disconnect()


def test_decline_notifies_initiator_and_ends_p2p(
        app, two_users, fake_calls, fake_messenger):
    caller, callee = two_users
    fake_calls.responses["DeclineCall"] = calls_pb2.DeclineCallResponse(
        call=_pb_call(TEST_CALL_ID, caller, callee, status="missed"),
        ended=True,
        notify_user_ids=[caller],
    )

    c_caller = _connect(app, caller)
    c_callee = _connect(app, callee)
    try:
        c_callee.emit("call:decline", {"call_id": TEST_CALL_ID})

        got_caller = _events(c_caller.get_received())
        assert got_caller["call:participant-declined"] == {
            "call_id": TEST_CALL_ID, "user_id": callee,
        }
        assert got_caller["call:ended"] == {
            "call_id": TEST_CALL_ID, "status": "missed",
        }
        # Отказавшийся тоже получает call:ended (закрыть свои вкладки).
        assert _events(c_callee.get_received())["call:ended"]["status"] == "missed"
    finally:
        for c in (c_caller, c_callee):
            if c.is_connected():
                c.disconnect()


def test_business_error_reaches_initiator_as_call_error(
        app, two_users, fake_calls, fake_messenger):
    caller, callee = two_users
    fake_calls.responses["StartCall"] = calls_pb2.StartCallResponse(
        error=calls_pb2.Error(code="BUSY", message="Пользователь уже в звонке",
                              http_status=409),
    )

    c_caller = _connect(app, caller)
    try:
        c_caller.emit("call:start", {"user_ids": [callee], "media": "audio"})

        got = _events(c_caller.get_received())
        assert got["call:error"]["code"] == "BUSY"
        assert "call:started" not in got
        assert "message:new" not in got
    finally:
        if c_caller.is_connected():
            c_caller.disconnect()


def test_messenger_down_does_not_break_call(
        app, two_users, fake_calls, messenger_down):
    """msgsvc недоступен: звонок проходит без привязки к чату и без плашки."""
    caller, callee = two_users
    fake_calls.responses["StartCall"] = lambda req: calls_pb2.StartCallResponse(
        call=_pb_call(TEST_CALL_ID, caller, callee,
                      conversation_id=req.conversation_id),
        livekit=calls_pb2.LivekitInfo(token="tok-caller", url="ws://livekit.test"),
    )

    c_caller = _connect(app, caller)
    try:
        c_caller.emit("call:start", {"user_ids": [callee], "media": "video"})

        name, req = fake_calls.requests[0]
        assert name == "StartCall"
        assert req.conversation_id == 0  # диалог не создан — msgsvc лежит

        got = _events(c_caller.get_received())
        assert got["call:started"]["call"]["id"] == TEST_CALL_ID
        assert "call:error" not in got
        assert "message:new" not in got
    finally:
        if c_caller.is_connected():
            c_caller.disconnect()


def test_grpc_unavailable_emits_calls_unavailable(app, two_users, grpc_down):
    caller, callee = two_users

    c_caller = _connect(app, caller)
    try:
        c_caller.emit("call:start", {"user_ids": [callee], "media": "video"})

        got = _events(c_caller.get_received())
        assert got["call:error"]["code"] == "CALLS_UNAVAILABLE"
        assert "call:started" not in got
    finally:
        if c_caller.is_connected():
            c_caller.disconnect()

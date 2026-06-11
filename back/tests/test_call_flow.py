"""Тесты Socket.IO-шлюза звонков (sockets/call_events.py).

Бизнес-логика звонков живёт в Go-микросервисе и покрыта его тестами
(back-go/calls/internal/...). Здесь проверяем именно шлюз: хендлеры зовут
gRPC и раскладывают ответ по сокет-событиям и плашке звонка в чате. Вместо
callsvc — in-process fake gRPC-сервер с canned-ответами; LiveKit не нужен.
Нужны dev-БД и Redis (фикстура app пропустит тесты, если их нет).
"""
import socket
from concurrent import futures

import grpc
import pytest

from app.extensions import socketio as sio
from app.grpc import calls_pb2, calls_pb2_grpc
from tests.conftest import cleanup_call_artifacts, make_token

# Заведомо отсутствующий в dev-БД id звонка: emit_call_system_message_update
# не найдёт плашку и должен тихо выйти.
MISSING_CALL_ID = 2_000_000_000


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


def _direct_execute(method, request):
    # В pytest eventlet не monkey-patch'ится: tpool-путь calls_client отдал бы
    # управление hub'у, и фоновые гринлеты create_app повисли бы на не-зелёном
    # time.sleep. Транспорт для шлюза прозрачен — зовём gRPC напрямую.
    return method(request, timeout=5)


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
    monkeypatch.setattr(calls_client, "_execute", _direct_execute)
    _reset_grpc_stub()
    yield servicer
    server.stop(None)
    _reset_grpc_stub()


@pytest.fixture
def grpc_down(monkeypatch):
    """CALLS_GRPC_ADDR указывает на порт, где никто не слушает."""
    from app.services import calls_client

    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()

    monkeypatch.setenv("CALLS_GRPC_ADDR", f"127.0.0.1:{port}")
    monkeypatch.setattr(calls_client, "_execute", _direct_execute)
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


def _create_call_row(app, initiator_id):
    """Запись звонка в БД — в проде её создаёт Go-сервис; здесь нужна, чтобы
    плашке в чате (messages.call_id FK) было на что ссылаться."""
    from app.extensions import db
    from app.models import Call, User
    with app.app_context():
        company_id = db.session.get(User, initiator_id).company_id
        call = Call(initiator_id=initiator_id, company_id=company_id,
                    kind="p2p", status="ringing", media="video")
        db.session.add(call)
        db.session.commit()
        return call.id


def _conv_id_between(app, a, b):
    from app.repositories import message_repo
    with app.app_context():
        conv = message_repo.get_conversation_between(a, b)
        return conv.id if conv else None


def test_start_call_emits_started_incoming_and_chat_pill(app, two_users, fake_calls):
    caller, callee = two_users
    conv_before = _conv_id_between(app, caller, callee)
    call_id = _create_call_row(app, caller)
    fake_calls.responses["StartCall"] = lambda req: calls_pb2.StartCallResponse(
        call=_pb_call(call_id, caller, callee, conversation_id=req.conversation_id),
        livekit=calls_pb2.LivekitInfo(token="tok-caller", url="ws://livekit.test"),
    )

    c_caller = _connect(app, caller)
    c_callee = _connect(app, callee)
    try:
        c_caller.emit("call:start", {"user_ids": [callee], "media": "video"})

        # Парный диалог Flask создал сам и передал его id в gRPC-запрос.
        conv_id = _conv_id_between(app, caller, callee)
        assert conv_id, "Flask не создал парный диалог"
        name, req = fake_calls.requests[0]
        assert name == "StartCall"
        assert req.initiator_id == caller
        assert list(req.invitee_ids) == [callee]
        assert req.media == "video"
        assert req.conversation_id == conv_id

        got_caller = _events(c_caller.get_received())
        started = got_caller["call:started"]
        assert started["call"]["id"] == call_id
        assert started["call"]["conversation_id"] == conv_id
        assert started["call"]["share_code"] == "sh-test"
        assert started["livekit"] == {"token": "tok-caller", "url": "ws://livekit.test"}

        pill = got_caller["message:new"]
        assert pill["conversation_id"] == conv_id
        assert pill["message"]["kind"] == "call"
        assert pill["message"]["call"]["id"] == call_id

        got_callee = _events(c_callee.get_received())
        assert got_callee["call:incoming"]["id"] == call_id
        assert got_callee["message:new"]["message"]["kind"] == "call"
    finally:
        for c in (c_caller, c_callee):
            if c.is_connected():
                c.disconnect()
        cleanup_call_artifacts(
            app, call_id=call_id,
            conversation_id=None if conv_before else _conv_id_between(app, caller, callee),
        )


def test_accept_returns_livekit_token_only_to_acceptor(app, two_users, fake_calls):
    caller, callee = two_users
    fake_calls.responses["AcceptCall"] = calls_pb2.AcceptCallResponse(
        call=_pb_call(MISSING_CALL_ID, caller, callee, status="active"),
        livekit=calls_pb2.LivekitInfo(token="tok-callee", url="ws://livekit.test"),
    )

    c_caller = _connect(app, caller)
    c_callee = _connect(app, callee)
    try:
        c_callee.emit("call:accept", {"call_id": MISSING_CALL_ID})

        name, req = fake_calls.requests[0]
        assert (name, req.call_id, req.user_id) == ("AcceptCall", MISSING_CALL_ID, callee)

        got_callee = _events(c_callee.get_received())
        accepted = got_callee["call:accepted"]
        assert accepted["call_id"] == MISSING_CALL_ID
        assert accepted["call"]["status"] == "active"
        assert accepted["livekit"]["token"] == "tok-callee"
        # Остальные узнают о принявшем от LiveKit (ParticipantConnected),
        # сокет-событие — только ему самому.
        assert "call:accepted" not in _events(c_caller.get_received())
    finally:
        for c in (c_caller, c_callee):
            if c.is_connected():
                c.disconnect()


def test_decline_notifies_initiator_and_ends_p2p(app, two_users, fake_calls):
    caller, callee = two_users
    fake_calls.responses["DeclineCall"] = calls_pb2.DeclineCallResponse(
        call=_pb_call(MISSING_CALL_ID, caller, callee, status="missed"),
        ended=True,
        notify_user_ids=[caller],
    )

    c_caller = _connect(app, caller)
    c_callee = _connect(app, callee)
    try:
        c_callee.emit("call:decline", {"call_id": MISSING_CALL_ID})

        got_caller = _events(c_caller.get_received())
        assert got_caller["call:participant-declined"] == {
            "call_id": MISSING_CALL_ID, "user_id": callee,
        }
        assert got_caller["call:ended"] == {
            "call_id": MISSING_CALL_ID, "status": "missed",
        }
        # Отказавшийся тоже получает call:ended (закрыть свои вкладки).
        assert _events(c_callee.get_received())["call:ended"]["status"] == "missed"
    finally:
        for c in (c_caller, c_callee):
            if c.is_connected():
                c.disconnect()


def test_business_error_reaches_initiator_as_call_error(app, two_users, fake_calls):
    caller, callee = two_users
    conv_before = _conv_id_between(app, caller, callee)
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
        if conv_before is None:
            cleanup_call_artifacts(
                app, conversation_id=_conv_id_between(app, caller, callee))


def test_grpc_unavailable_emits_calls_unavailable(app, two_users, grpc_down):
    caller, callee = two_users
    conv_before = _conv_id_between(app, caller, callee)

    c_caller = _connect(app, caller)
    try:
        c_caller.emit("call:start", {"user_ids": [callee], "media": "video"})

        got = _events(c_caller.get_received())
        assert got["call:error"]["code"] == "CALLS_UNAVAILABLE"
        assert "call:started" not in got
    finally:
        if c_caller.is_connected():
            c_caller.disconnect()
        if conv_before is None:
            cleanup_call_artifacts(
                app, conversation_id=_conv_id_between(app, caller, callee))

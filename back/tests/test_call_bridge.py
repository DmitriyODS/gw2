"""Юнит-тесты моста событий звонков (sockets/call_bridge._handle_event).

Go-сервис звонков публикует события в Redis-канал gw2:calls:events; сам
pub/sub-слушатель здесь не поднимаем — скармливаем события хендлеру напрямую
и проверяем, во что они превращаются на стороне Socket.IO (вместо настоящего
SocketIO — рекордер вызовов emit). Плашка звонка читается у msgsvc по gRPC —
вместо него in-process fake (фикстура fake_messenger из conftest).
"""
import json

from app.grpc import messenger_pb2
from app.sockets.call_bridge import _handle_event

CALL_ID = 2_000_000_000
CONV_ID = 2_000_000_001


class EmitRecorder:
    def __init__(self):
        self.calls = []

    def emit(self, event, payload, room=None):
        self.calls.append((event, payload, room))


def test_call_ended_broadcasts_to_notify_users(app, fake_messenger):
    # Плашки у msgsvc нет (бизнес-ошибка) — уходит только call:ended адресатам.
    fake_messenger.responses["GetCallMessage"] = messenger_pb2.GetCallMessageResponse(
        error=messenger_pb2.Error(code="MSG_NOT_FOUND", message="нет плашки",
                                  http_status=404),
    )
    rec = EmitRecorder()
    _handle_event(app, rec, {
        "type": "call_ended", "call_id": CALL_ID,
        "status": "ended", "notify_user_ids": [7, 9],
    })
    assert rec.calls == [
        ("call:ended", {"call_id": CALL_ID, "status": "ended"}, "user_7"),
        ("call:ended", {"call_id": CALL_ID, "status": "ended"}, "user_9"),
    ]


def test_event_without_call_id_is_ignored(app):
    rec = EmitRecorder()
    _handle_event(app, rec, {"type": "call_ended", "notify_user_ids": [7]})
    assert rec.calls == []


def test_status_changed_refreshes_chat_pill(app, fake_messenger):
    pill = {"id": 9001, "conversation_id": CONV_ID, "kind": "call",
            "call": {"id": CALL_ID, "status": "active"}}
    fake_messenger.responses["GetCallMessage"] = messenger_pb2.GetCallMessageResponse(
        conversation_id=CONV_ID,
        message_json=json.dumps(pill),
        notify_user_ids=[7, 9],
    )

    rec = EmitRecorder()
    _handle_event(app, rec, {"type": "call_status_changed", "call_id": CALL_ID})

    name, req = fake_messenger.requests[0]
    assert (name, req.call_id) == ("GetCallMessage", CALL_ID)
    event = {"conversation_id": CONV_ID, "message": pill}
    assert rec.calls == [
        ("message:updated", event, "user_7"),
        ("message:updated", event, "user_9"),
    ]


def test_status_changed_without_pill_is_noop(app, fake_messenger):
    """Звонок есть, а плашки в чате нет (пустой message_json — дефолт фейка) —
    хендлер должен тихо выйти."""
    rec = EmitRecorder()
    _handle_event(app, rec, {"type": "call_status_changed", "call_id": CALL_ID})
    assert rec.calls == []


def test_messenger_unavailable_is_swallowed(app, messenger_down):
    """msgsvc лежит: call:ended всё равно доставляется, плашка пропускается."""
    rec = EmitRecorder()
    _handle_event(app, rec, {
        "type": "call_ended", "call_id": CALL_ID,
        "status": "ended", "notify_user_ids": [7],
    })
    assert rec.calls == [
        ("call:ended", {"call_id": CALL_ID, "status": "ended"}, "user_7"),
    ]

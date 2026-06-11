"""Юнит-тесты моста событий звонков (sockets/call_bridge._handle_event).

Go-сервис публикует события в Redis-канал gw2:calls:events; сам pub/sub-
слушатель здесь не поднимаем — скармливаем события хендлеру напрямую и
проверяем, во что они превращаются на стороне Socket.IO (вместо настоящего
SocketIO — рекордер вызовов emit). Нужна dev-БД (плашка читается из неё).
"""
from app.sockets.call_bridge import _handle_event
from tests.conftest import cleanup_call_artifacts

# Заведомо отсутствующий в dev-БД id звонка.
MISSING_CALL_ID = 2_000_000_000


class EmitRecorder:
    def __init__(self):
        self.calls = []

    def emit(self, event, payload, room=None):
        self.calls.append((event, payload, room))


def _make_call_in_chat(app, a, b, with_pill):
    """Диалог пары + запись звонка с conversation_id (и плашка, если нужна).
    Возвращает (call_id, conv_id, conv_existed, side_a, side_b, msg_id)."""
    from app.extensions import db
    from app.models import Call, User
    from app.repositories import message_repo
    with app.app_context():
        conv_existed = message_repo.get_conversation_between(a, b) is not None
        conv = message_repo.get_or_create_conversation(a, b)
        call = Call(initiator_id=a, company_id=db.session.get(User, a).company_id,
                    kind="p2p", status="active", media="video",
                    conversation_id=conv.id)
        db.session.add(call)
        db.session.flush()
        msg_id = None
        if with_pill:
            msg_id = message_repo.create_call_message(conv.id, a, call.id).id
        db.session.commit()
        return call.id, conv.id, conv_existed, conv.user_a_id, conv.user_b_id, msg_id


def test_call_ended_broadcasts_to_notify_users(app):
    rec = EmitRecorder()
    _handle_event(app, rec, {
        "type": "call_ended", "call_id": MISSING_CALL_ID,
        "status": "ended", "notify_user_ids": [7, 9],
    })
    # Звонка в БД нет — плашка не обновляется, только call:ended адресатам.
    assert rec.calls == [
        ("call:ended", {"call_id": MISSING_CALL_ID, "status": "ended"}, "user_7"),
        ("call:ended", {"call_id": MISSING_CALL_ID, "status": "ended"}, "user_9"),
    ]


def test_event_without_call_id_is_ignored(app):
    rec = EmitRecorder()
    _handle_event(app, rec, {"type": "call_ended", "notify_user_ids": [7]})
    assert rec.calls == []


def test_status_changed_refreshes_chat_pill(app, two_users):
    a, b = two_users
    call_id, conv_id, conv_existed, side_a, side_b, msg_id = \
        _make_call_in_chat(app, a, b, with_pill=True)

    rec = EmitRecorder()
    try:
        _handle_event(app, rec, {"type": "call_status_changed", "call_id": call_id})

        assert {(name, room) for name, _, room in rec.calls} == {
            ("message:updated", f"user_{side_a}"),
            ("message:updated", f"user_{side_b}"),
        }
        payload = rec.calls[0][1]
        assert payload["conversation_id"] == conv_id
        assert payload["message"]["id"] == msg_id
        assert payload["message"]["kind"] == "call"
        assert payload["message"]["call"]["id"] == call_id
    finally:
        cleanup_call_artifacts(app, call_id=call_id,
                               conversation_id=None if conv_existed else conv_id)


def test_status_changed_without_pill_is_noop(app, two_users):
    """Звонок есть, а плашки в чате нет — хендлер должен тихо выйти."""
    a, b = two_users
    call_id, conv_id, conv_existed, *_ = _make_call_in_chat(app, a, b, with_pill=False)

    rec = EmitRecorder()
    try:
        _handle_event(app, rec, {"type": "call_status_changed", "call_id": call_id})
        assert rec.calls == []
    finally:
        cleanup_call_artifacts(app, call_id=call_id,
                               conversation_id=None if conv_existed else conv_id)

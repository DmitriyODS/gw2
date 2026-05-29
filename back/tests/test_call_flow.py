"""Интеграционный E2E сокет-флоу звонка через SocketIOTestClient.

Проверяем главное, что чинили: полный путь start → incoming → accept →
accepted/participant-joined, и — самое важное — call:rejoin после «перезагрузки
вкладки» собеседника (новый сокет того же пользователя возвращается в звонок).

Требует поднятых dev-БД и Redis (фикстура app сама пропустит тест, если их нет).
"""
from app.extensions import socketio as sio
from app.sockets import call_state
from tests.conftest import make_token


def _events(received):
    """Преобразует список get_received() в dict {event_name: args[0]}."""
    out = {}
    for r in received:
        out[r["name"]] = r["args"][0] if r.get("args") else None
    return out


def _connect(app, user_id):
    token = make_token(app, user_id)
    return sio.test_client(
        app,
        auth={"token": token},
        flask_test_client=app.test_client(),
    )


def test_full_call_and_rejoin_flow(app, two_users):
    caller, callee = two_users
    created_call_id = None

    c_caller = _connect(app, caller)
    c_callee = _connect(app, callee)
    assert c_caller.is_connected()
    assert c_callee.is_connected()
    # Сбросим стартовые события (presence и т. п.).
    c_caller.get_received()
    c_callee.get_received()

    try:
        # 1. Инициатор звонит.
        c_caller.emit("call:start", {"user_ids": [callee], "media": "video"})

        started = _events(c_caller.get_received())
        assert "call:started" in started, "инициатор не получил call:started"
        created_call_id = started["call:started"]["id"]

        incoming = _events(c_callee.get_received())
        assert "call:incoming" in incoming, "получатель не получил call:incoming"
        assert incoming["call:incoming"]["id"] == created_call_id

        # 2. Получатель принимает.
        c_callee.emit("call:accept", {"call_id": created_call_id})

        accepted = _events(c_callee.get_received())
        assert "call:accepted" in accepted
        assert accepted["call:accepted"]["existing_participants"] == [caller]

        joined = _events(c_caller.get_received())
        assert "call:participant-joined" in joined
        assert joined["call:participant-joined"]["user_id"] == callee

        # Оба сейчас в звонке.
        assert sorted(call_state.get_participants(created_call_id)) == sorted([caller, callee])

        # 3. «Перезагрузка вкладки» получателя: старый сокет отваливается,
        #    новый сокет того же пользователя возвращается в звонок. Благодаря
        #    grace-окну пользователь всё ещё числится в звонке.
        c_callee.disconnect()
        # Пользователь по-прежнему в звонке (его не убрали мгновенно).
        assert call_state.get_user_active_call(callee) == created_call_id

        c_callee2 = _connect(app, callee)
        c_callee2.get_received()
        c_caller.get_received()

        c_callee2.emit("call:rejoin", {"call_id": created_call_id})

        rejoined = _events(c_callee2.get_received())
        assert "call:accepted" in rejoined, "rejoin не вернул call:accepted"
        assert rejoined["call:accepted"]["existing_participants"] == [caller]

        caller_evt = _events(c_caller.get_received())
        assert "call:participant-joined" in caller_evt
        assert caller_evt["call:participant-joined"]["rejoin"] is True

        if c_callee2.is_connected():
            c_callee2.disconnect()
    finally:
        if c_caller.is_connected():
            c_caller.disconnect()
        if c_callee.is_connected():
            c_callee.disconnect()
        # Чистим созданные тестом строки, чтобы не копить в dev-БД.
        if created_call_id is not None:
            _cleanup_call(app, created_call_id)


def _cleanup_call(app, call_id):
    from app.extensions import db
    from app.models import Call, CallParticipant, Message
    try:
        with app.app_context():
            db.session.execute(db.delete(Message).where(Message.call_id == call_id))
            db.session.execute(db.delete(CallParticipant).where(CallParticipant.call_id == call_id))
            db.session.execute(db.delete(Call).where(Call.id == call_id))
            db.session.commit()
    except Exception:
        db.session.rollback()
    # И из in-memory, если осталось.
    call_state.end_call(call_id)

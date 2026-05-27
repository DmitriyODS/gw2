"""WebRTC сигналинг через Socket.IO.

Что делает сервер: маршрутизирует invite/accept/decline/leave и пересылает
offer/answer/ice candidate от одного участника другому. Сами медиа-потоки
идут peer-to-peer через WebRTC, сервер их не видит.

Схема mesh (для 1:1 и групп ≤ ~5): у каждого участника отдельный
RTCPeerConnection с каждым другим участником. Когда новый участник
вступает в звонок (accept), он получает список уже подключённых и сам
инициирует offer к каждому из них.
"""
from flask_socketio import SocketIO
from app.utils.logger import get_logger
from app.schemas import CallSchema, CallParticipantBriefSchema
from app.services import call_service
from app.services.call_service import CallServiceError
from app.sockets import call_state

logger = get_logger(__name__)
_call_schema = CallSchema()
_part_schema = CallParticipantBriefSchema()


def _resolve_user_id_from_sid(sid: str) -> int | None:
    from app.sockets.presence import _sid_user
    return _sid_user.get(sid)


def register_call_events(socketio: SocketIO) -> None:

    @socketio.on("call:start")
    def on_start(data):
        """Клиент инициирует звонок. data = {user_ids: [...], media: 'audio'|'video'}."""
        from flask import request as flask_request
        from flask import current_app

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        user_ids = (data or {}).get("user_ids") or []
        media = (data or {}).get("media") or "video"

        try:
            with current_app.app_context():
                call = call_service.start_call(me, user_ids, media=media)
        except CallServiceError as e:
            socketio.emit("call:error",
                          {"code": e.code, "message": e.message},
                          room=f"user_{me}")
            return

        payload = _call_schema.dump(call)
        # Инициатору — подтверждение
        socketio.emit("call:started", payload, room=f"user_{me}")
        # Приглашённым — входящий звонок
        for part in call.participants:
            if part.role == "invitee":
                socketio.emit("call:incoming", payload, room=f"user_{part.user_id}")

    @socketio.on("call:accept")
    def on_accept(data):
        from flask import request as flask_request
        from flask import current_app

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = (data or {}).get("call_id")
        if not call_id:
            return

        try:
            with current_app.app_context():
                call = call_service.accept_call(call_id, me)
        except CallServiceError as e:
            socketio.emit("call:error",
                          {"code": e.code, "message": e.message},
                          room=f"user_{me}")
            return

        existing = [uid for uid in call_state.get_participants(call_id) if uid != me]

        # Самому принявшему — кто уже в звонке (он будет инициировать offer'ы к ним).
        socketio.emit("call:accepted", {
            "call_id": call_id,
            "existing_participants": existing,
            "call": _call_schema.dump(call),
        }, room=f"user_{me}")

        # Остальным — кто к ним присоединился (они должны принять offer от него).
        for uid in existing:
            socketio.emit("call:participant-joined", {
                "call_id": call_id,
                "user_id": me,
            }, room=f"user_{uid}")

    @socketio.on("call:decline")
    def on_decline(data):
        from flask import request as flask_request
        from flask import current_app

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = (data or {}).get("call_id")
        if not call_id:
            return

        # До вызова service запомним, кого нужно уведомить (тех, кто ещё в state).
        targets = list({*call_state.get_participants(call_id), *call_state.get_invited(call_id)})
        targets = [t for t in targets if t != me]

        with current_app.app_context():
            call = call_service.decline_call(call_id, me)

        if call is None:
            return

        payload = {"call_id": call_id, "user_id": me}
        for uid in targets:
            socketio.emit("call:participant-declined", payload, room=f"user_{uid}")
        # Если звонок завершён (p2p отказ или последний отказался) — сообщим всем
        if call_state.get_call(call_id) is None:
            ended_payload = {"call_id": call_id, "status": call.status}
            for uid in (*targets, me):
                socketio.emit("call:ended", ended_payload, room=f"user_{uid}")

    @socketio.on("call:leave")
    def on_leave(data):
        from flask import request as flask_request
        from flask import current_app

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = (data or {}).get("call_id")
        if not call_id:
            return

        targets = [uid for uid in call_state.get_participants(call_id) if uid != me]

        with current_app.app_context():
            call = call_service.leave_call(call_id, me)

        if call is None:
            return

        for uid in targets:
            socketio.emit("call:participant-left",
                          {"call_id": call_id, "user_id": me},
                          room=f"user_{uid}")

        if call_state.get_call(call_id) is None:
            ended_payload = {"call_id": call_id, "status": call.status}
            for uid in (*targets, me):
                socketio.emit("call:ended", ended_payload, room=f"user_{uid}")

    @socketio.on("call:end")
    def on_end(data):
        """Инициатор завершает звонок целиком (для всех)."""
        from flask import request as flask_request
        from flask import current_app

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = (data or {}).get("call_id")
        if not call_id:
            return

        targets = list({*call_state.get_participants(call_id), *call_state.get_invited(call_id)})

        with current_app.app_context():
            call = call_service.end_call_by_initiator(call_id, me)
        if call is None:
            return

        for uid in targets:
            socketio.emit("call:ended",
                          {"call_id": call_id, "status": call.status},
                          room=f"user_{uid}")

    # ── WebRTC сигналинг ─────────────────────────────────────────
    # Просто маршрутизация offer/answer/ice от одного участника к другому.
    # Сервер не парсит SDP, лишь проверяет, что оба сейчас в одном звонке.

    @socketio.on("webrtc:signal")
    def on_signal(data):
        """data = {call_id, to_user_id, kind: 'offer'|'answer'|'ice', payload: {...}}"""
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = (data or {}).get("call_id")
        to_user_id = (data or {}).get("to_user_id")
        if not call_id or not to_user_id:
            return
        # Проверяем, что оба в одном звонке (или приглашены — для случая первого
        # offer'а до accept в редком race).
        state = call_state.get_call(call_id)
        if not state or me not in state["invited"] or to_user_id not in state["invited"]:
            return

        socketio.emit("webrtc:signal", {
            "call_id": call_id,
            "from_user_id": me,
            "kind": data.get("kind"),
            "payload": data.get("payload"),
        }, room=f"user_{to_user_id}")

    # Минимальный сигнал mute/camera — чтобы UI у других участников отражал
    # состояние без задержки на media renegotiation. Сами треки управляются
    # локально через MediaStreamTrack.enabled.
    @socketio.on("call:media-state")
    def on_media_state(data):
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = (data or {}).get("call_id")
        if not call_id:
            return
        targets = [uid for uid in call_state.get_participants(call_id) if uid != me]
        for uid in targets:
            socketio.emit("call:media-state", {
                "call_id": call_id,
                "user_id": me,
                "audio": bool((data or {}).get("audio", True)),
                "video": bool((data or {}).get("video", True)),
            }, room=f"user_{uid}")


def cleanup_call_on_disconnect(socketio: SocketIO, user_id: int) -> None:
    """Дёргается из presence.on_disconnect, когда у пользователя не осталось
    активных видимых соединений: убираем его из звонка."""
    from flask import current_app
    with current_app.app_context():
        result = call_service.cleanup_user_on_disconnect(user_id)
    if not result:
        return
    call_id, notify_targets = result
    for uid in notify_targets:
        socketio.emit("call:participant-left",
                      {"call_id": call_id, "user_id": user_id},
                      room=f"user_{uid}")
    if call_state.get_call(call_id) is None:
        for uid in (*notify_targets, user_id):
            socketio.emit("call:ended",
                          {"call_id": call_id, "status": "ended"},
                          room=f"user_{uid}")

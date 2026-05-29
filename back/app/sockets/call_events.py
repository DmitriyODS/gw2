"""WebRTC сигналинг через Socket.IO.

Что делает сервер: маршрутизирует invite/accept/decline/leave и пересылает
offer/answer/ice candidate от одного участника другому. Сами медиа-потоки
идут peer-to-peer через WebRTC, сервер их не видит.

Схема mesh (для 1:1 и групп ≤ ~5): у каждого участника отдельный
RTCPeerConnection с каждым другим участником. Когда новый участник
вступает в звонок (accept), он получает список уже подключённых и сам
инициирует offer к каждому из них.
"""
import os

from flask_socketio import SocketIO
from app.utils.logger import get_logger
from app.schemas import CallSchema, CallParticipantBriefSchema, MessageSchema
from app.services import call_service
from app.services.call_service import CallServiceError
from app.sockets import call_state

logger = get_logger(__name__)
_call_schema = CallSchema()
_part_schema = CallParticipantBriefSchema()
_msg_schema = MessageSchema()

# Сколько секунд держать пользователя в звонке после обрыва сокета, давая ему
# шанс переподключиться (перезагрузка вкладки, кратковременная потеря сети).
CALL_REJOIN_GRACE_SEC = int(os.getenv("CALL_REJOIN_GRACE_SEC", "15"))


def _resolve_user_id_from_sid(sid: str) -> int | None:
    from app.sockets.presence import _sid_user
    return _sid_user.get(sid)


def _emit_call_system_message_update(socketio: SocketIO, call_id: int) -> None:
    """Перечитать системное сообщение о звонке и эмитить message:updated
    в комнаты обоих участников парного диалога. Вызывается после изменения
    статуса звонка (ringing → active → ended / missed). Безопасно если
    `system_message_id` нет (group-звонок) — просто выходит."""
    from flask import current_app
    from app.extensions import db
    from app.models import Conversation

    state = call_state.get_call(call_id)
    msg_id = state.get("system_message_id") if state else None
    conv_id = state.get("conversation_id") if state else None
    if not msg_id or not conv_id:
        # state мог уже почиститься (звонок завершён) — попробуем найти
        # через БД: для каждого Call.conversation_id берём последнее
        # сообщение kind='call' с этим call_id.
        from app.models import Message, Call
        with current_app.app_context():
            call = db.session.get(Call, call_id)
            if not call or not call.conversation_id:
                return
            conv_id = call.conversation_id
            msg = db.session.execute(
                db.select(Message).where(
                    Message.call_id == call_id, Message.kind == "call",
                ).order_by(Message.id.desc()).limit(1)
            ).scalar_one_or_none()
            if not msg:
                return
            conv = db.session.get(Conversation, conv_id)
            payload = _msg_schema.dump(msg)
            user_a = conv.user_a_id
            user_b = conv.user_b_id
    else:
        with current_app.app_context():
            msg = call_service.get_system_call_message(msg_id)
            if not msg:
                return
            conv = db.session.get(Conversation, conv_id)
            if not conv:
                return
            payload = _msg_schema.dump(msg)
            user_a = conv.user_a_id
            user_b = conv.user_b_id

    event = {
        "conversation_id": conv_id,
        "message": payload,
    }
    for uid in (user_a, user_b):
        socketio.emit("message:updated", event, room=f"user_{uid}")


def register_call_events(socketio: SocketIO) -> None:

    @socketio.on("call:start")
    def on_start(data):
        """Клиент инициирует звонок. data = {user_ids: [...], media: 'audio'|'video'}."""
        from flask import request as flask_request
        from flask import current_app

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        raw_ids = (data or {}).get("user_ids") or []
        try:
            user_ids = [int(uid) for uid in raw_ids if uid is not None]
        except (TypeError, ValueError):
            user_ids = []
        media = (data or {}).get("media") or "video"
        logger.info("call.start", extra={"extra": {
            "initiator_id": me, "user_ids": user_ids, "media": media,
        }})

        # ВАЖНО: marshmallow dump() и доступ к call.participants/initiator —
        # это lazy-load relationships, нужна активная SQLAlchemy session.
        # Делаем всё внутри app_context, чтобы не словить DetachedInstanceError.
        try:
            with current_app.app_context():
                call = call_service.start_call(me, user_ids, media=media)
                payload = _call_schema.dump(call)
                invitees = [
                    (p.user_id, p.role) for p in call.participants
                ]
                call_id = call.id
                # Системное сообщение о звонке (для p2p) — нужно эмитить
                # как обычное message:new в комнаты обеих сторон чата,
                # чтобы плашка тут же появилась в открытой переписке.
                sys_state = call_state.get_call(call_id) or {}
                sys_msg_id = sys_state.get("system_message_id")
                sys_conv_id = sys_state.get("conversation_id")
                sys_payload = None
                if sys_msg_id and sys_conv_id:
                    sys_msg = call_service.get_system_call_message(sys_msg_id)
                    if sys_msg:
                        sys_payload = _msg_schema.dump(sys_msg)
        except CallServiceError as e:
            logger.warning("call.start_failed", extra={"extra": {
                "initiator_id": me, "code": e.code, "message": e.message,
            }})
            socketio.emit("call:error",
                          {"code": e.code, "message": e.message},
                          room=f"user_{me}")
            return

        # Инициатору — подтверждение
        socketio.emit("call:started", payload, room=f"user_{me}")
        # Приглашённым — входящий звонок. Диагностика «звонок ушёл, но
        # получатель ничего не видит»: логируем, сколько сокетов сейчас
        # в комнате user_{id}. Если 0 — получатель не онлайн (или его
        # вкладка не сделала connect), пакет потеряется без push-сервера.
        from app.sockets.presence import _sid_user
        for user_id, role in invitees:
            if role == "invitee":
                sockets_in_room = sum(1 for uid in _sid_user.values() if uid == user_id)
                logger.info("call.incoming_emit", extra={"extra": {
                    "call_id": call_id, "to_user_id": user_id,
                    "sockets_in_room": sockets_in_room,
                }})
                socketio.emit("call:incoming", payload, room=f"user_{user_id}")

        # Плашка о звонке в чате — эмитим как message:new (как обычное
        # сообщение). Фронт сам отрендерит её специально по kind='call'.
        if sys_payload and sys_conv_id:
            for user_id, _role in invitees:
                socketio.emit("message:new", {
                    "conversation_id": sys_conv_id,
                    "message": sys_payload,
                    "from_user_id": me,
                }, room=f"user_{user_id}")
            # И инициатору тоже (тогда плашка появится и в его открытом чате).
            socketio.emit("message:new", {
                "conversation_id": sys_conv_id,
                "message": sys_payload,
                "from_user_id": me,
            }, room=f"user_{me}")

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
                # dump внутри app_context — иначе DetachedInstanceError при
                # lazy-load initiator/participants
                call_payload = _call_schema.dump(call)
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
            "call": call_payload,
        }, room=f"user_{me}")

        # Остальным — кто к ним присоединился (они должны принять offer от него).
        for uid in existing:
            socketio.emit("call:participant-joined", {
                "call_id": call_id,
                "user_id": me,
            }, room=f"user_{uid}")

        # Обновим плашку в чате: status ringing → active.
        _emit_call_system_message_update(socketio, call_id)

    @socketio.on("call:rejoin")
    def on_rejoin(data):
        """Возврат к звонку после перезагрузки вкладки. Пользователь всё ещё
        числится в звонке (благодаря grace-окну на disconnect). Заново отдаём
        ему список уже подключённых (он переинициирует offer'ы), а остальным
        шлём participant-joined с флагом rejoin — чтобы они сбросили устаревший
        peer и приняли свежий offer."""
        from flask import request as flask_request
        from flask import current_app

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        try:
            call_id = int((data or {}).get("call_id"))
        except (TypeError, ValueError):
            return

        state = call_state.get_call(call_id)
        if not state or me not in state["invited"]:
            socketio.emit("call:error",
                          {"code": "NOT_IN_CALL", "message": "Звонок уже завершён"},
                          room=f"user_{me}")
            return

        try:
            with current_app.app_context():
                call = call_service.accept_call(call_id, me)  # идемпотентно
                call_payload = _call_schema.dump(call)
        except CallServiceError as e:
            socketio.emit("call:error",
                          {"code": e.code, "message": e.message},
                          room=f"user_{me}")
            return

        existing = [uid for uid in call_state.get_participants(call_id) if uid != me]
        # Сначала просим существующих сбросить устаревший peer (participant-joined
        # с rejoin), и только потом отдаём вернувшемуся список — чтобы его свежий
        # offer не пришёл к ним раньше, чем они дропнули мёртвое соединение.
        for uid in existing:
            socketio.emit("call:participant-joined", {
                "call_id": call_id,
                "user_id": me,
                "rejoin": True,
            }, room=f"user_{uid}")
        socketio.emit("call:accepted", {
            "call_id": call_id,
            "existing_participants": existing,
            "call": call_payload,
        }, room=f"user_{me}")
        _emit_call_system_message_update(socketio, call_id)

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
            # Берём поля, пока сессия жива (избегаем DetachedInstanceError).
            call_status = call.status if call is not None else None

        if call is None:
            return

        payload = {"call_id": call_id, "user_id": me}
        for uid in targets:
            socketio.emit("call:participant-declined", payload, room=f"user_{uid}")
        # Если звонок завершён (p2p отказ или последний отказался) — сообщим всем
        if call_state.get_call(call_id) is None:
            ended_payload = {"call_id": call_id, "status": call_status}
            for uid in (*targets, me):
                socketio.emit("call:ended", ended_payload, room=f"user_{uid}")
        # Плашка в чате обновится: status → missed/ended.
        _emit_call_system_message_update(socketio, call_id)

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
            call_status = call.status if call is not None else None

        if call is None:
            return

        for uid in targets:
            socketio.emit("call:participant-left",
                          {"call_id": call_id, "user_id": me},
                          room=f"user_{uid}")

        if call_state.get_call(call_id) is None:
            ended_payload = {"call_id": call_id, "status": call_status}
            for uid in (*targets, me):
                socketio.emit("call:ended", ended_payload, room=f"user_{uid}")
        _emit_call_system_message_update(socketio, call_id)

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
            call_status = call.status if call is not None else None
        if call is None:
            return

        for uid in targets:
            socketio.emit("call:ended",
                          {"call_id": call_id, "status": call_status},
                          room=f"user_{uid}")
        _emit_call_system_message_update(socketio, call_id)

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
        try:
            call_id = int((data or {}).get("call_id"))
            to_user_id = int((data or {}).get("to_user_id"))
        except (TypeError, ValueError):
            return
        # Проверяем, что оба в одном звонке (или приглашены — для случая первого
        # offer'а до accept в редком race).
        state = call_state.get_call(call_id)
        if not state or me not in state["invited"] or to_user_id not in state["invited"]:
            logger.debug("webrtc.signal_rejected", extra={"extra": {
                "from": me, "to": to_user_id, "call_id": call_id,
                "kind": (data or {}).get("kind"),
            }})
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
    """Дёргается из presence.on_disconnect. НЕ убираем пользователя из звонка
    сразу: при перезагрузке вкладки/смене сети сокет рвётся на пару секунд и
    тут же переустанавливается. Планируем отложенную проверку — и только если
    за grace-окно пользователь так и не вернулся (нет ни одного соединения),
    окончательно выводим его из звонка и уведомляем остальных."""
    call_id = call_state.get_user_active_call(user_id)
    if call_id is None:
        return

    def _delayed_cleanup():
        socketio.sleep(CALL_REJOIN_GRACE_SEC)
        from app.sockets import presence
        # Вернулся (переподключился) — оставляем в звонке как есть.
        if presence.has_any_connection(user_id):
            return
        # За время ожидания мог сменить/завершить звонок.
        if call_state.get_user_active_call(user_id) != call_id:
            return
        _finalize_disconnect(socketio, user_id, call_id)

    socketio.start_background_task(_delayed_cleanup)


def _finalize_disconnect(socketio: SocketIO, user_id: int, call_id: int) -> None:
    """Окончательно убрать отвалившегося пользователя из звонка."""
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
    _emit_call_system_message_update(socketio, call_id)

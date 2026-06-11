"""Ринг-сигналинг звонков через Socket.IO — тонкий шлюз к Go-микросервису.

Бизнес-логика звонков (валидация, БД, ринг-state, LiveKit) переехала в
callsvc (back-go/calls); Flask здесь только:
  - резолвит пользователя по сокету и зовёт gRPC (services/calls_client.py);
  - эмитит сокет-события по данным из ответа (списки адресатов считает Go);
  - ведёт системную плашку звонка в чате (домен мессенджера остался тут).

События, инициированные самим сервисом (вебхуки LiveKit → call:ended),
прилетают отдельным каналом Redis — см. sockets/call_bridge.py.
"""
from flask_socketio import SocketIO

from app.schemas import MessageSchema
from app.services import calls_client
from app.services.calls_client import CallsServiceError
from app.utils.logger import get_logger

logger = get_logger(__name__)
_msg_schema = MessageSchema()


def _resolve_user_id_from_sid(sid: str) -> int | None:
    from app.sockets.presence import _sid_user
    return _sid_user.get(sid)


def _int_or_none(value) -> int | None:
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _int_list(raw) -> list[int]:
    out = []
    for v in raw or []:
        iv = _int_or_none(v)
        if iv is not None:
            out.append(iv)
    return out


def _emit_error(socketio: SocketIO, user_id: int, e: CallsServiceError) -> None:
    socketio.emit("call:error", {"code": e.code, "message": e.message},
                  room=f"user_{user_id}")


def emit_call_system_message_update(socketio: SocketIO, call_id: int) -> None:
    """Перечитать системное сообщение о звонке (kind='call') и эмитить
    message:updated обеим сторонам парного диалога. Вызывается после смены
    статуса звонка (ringing → active → ended/missed). Для group-звонков
    плашки нет — тихо выходим."""
    from flask import current_app
    from app.extensions import db
    from app.models import Call, Conversation, Message

    with current_app.app_context():
        call = db.session.get(Call, call_id)
        if not call or not call.conversation_id:
            return
        # Фильтр по диалогу звонка обязателен: пересланные плашки ссылаются
        # на тот же call_id, но живут в других диалогах.
        msg = db.session.execute(
            db.select(Message).where(
                Message.call_id == call_id, Message.kind == "call",
                Message.conversation_id == call.conversation_id,
            ).order_by(Message.id.desc()).limit(1)
        ).scalar_one_or_none()
        if not msg:
            return
        conv = db.session.get(Conversation, call.conversation_id)
        if not conv:
            return
        event = {
            "conversation_id": call.conversation_id,
            "message": _msg_schema.dump(msg),
        }
        for uid in (conv.user_a_id, conv.user_b_id):
            if uid:
                socketio.emit("message:updated", event, room=f"user_{uid}")


def register_call_events(socketio: SocketIO) -> None:

    @socketio.on("call:start")
    def on_start(data):
        """Клиент инициирует звонок. data = {user_ids: [...], media}."""
        from flask import request as flask_request
        from flask import current_app
        from app.extensions import db
        from app.repositories import message_repo

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        user_ids = _int_list((data or {}).get("user_ids"))
        media = (data or {}).get("media") or "video"
        logger.info("call.start", extra={"extra": {
            "initiator_id": me, "user_ids": user_ids, "media": media,
        }})

        # Парный диалог — домен мессенджера, остаётся на Flask: создаём до
        # звонка (Go пишет conversation_id в запись звонка) и коммитим,
        # чтобы FK в БД уже существовал к моменту INSERT на стороне Go.
        conv_id = 0
        others = {uid for uid in user_ids if uid != me}
        if len(others) == 1:
            with current_app.app_context():
                conv = message_repo.get_or_create_conversation(me, next(iter(others)))
                db.session.commit()
                conv_id = conv.id

        try:
            resp = calls_client.start_call(me, user_ids, media, conv_id)
        except CallsServiceError as e:
            logger.warning("call.start_failed", extra={"extra": {
                "initiator_id": me, "code": e.code, "message": e.message,
            }})
            _emit_error(socketio, me, e)
            return

        payload = calls_client.call_to_dict(resp.call)
        call_id = payload["id"]

        # Инициатору — подтверждение + токен LiveKit (он входит в комнату
        # сразу и «ждёт» там остальных).
        socketio.emit("call:started", {
            "call": payload,
            "livekit": calls_client.livekit_to_dict(resp.livekit),
        }, room=f"user_{me}")

        # Приглашённым — входящий звонок.
        invitee_ids = [p["user_id"] for p in payload["participants"]
                       if p["role"] == "invitee"]
        for uid in invitee_ids:
            socketio.emit("call:incoming", payload, room=f"user_{uid}")

        # Системная плашка звонка в чате (только p2p) — обычное message:new,
        # фронт рендерит её специально по kind='call'.
        if conv_id and payload["kind"] == "p2p":
            with current_app.app_context():
                sys_msg = message_repo.create_call_message(conv_id, me, call_id)
                db.session.commit()
                sys_payload = _msg_schema.dump(sys_msg)
            for uid in (*invitee_ids, me):
                socketio.emit("message:new", {
                    "conversation_id": conv_id,
                    "message": sys_payload,
                    "from_user_id": me,
                }, room=f"user_{uid}")

    @socketio.on("call:invite")
    def on_invite(data):
        """Любой участник зовёт ещё людей. data = {call_id, user_ids}."""
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = _int_or_none((data or {}).get("call_id"))
        invitee_ids = _int_list((data or {}).get("user_ids"))
        if call_id is None or not invitee_ids:
            return

        try:
            resp = calls_client.invite_to_call(call_id, me, invitee_ids)
        except CallsServiceError as e:
            _emit_error(socketio, me, e)
            return

        payload = calls_client.call_to_dict(resp.call)
        new_ids = list(resp.new_invitee_ids)
        logger.info("call.invite", extra={"extra": {
            "call_id": call_id, "inviter_id": me, "new_ids": new_ids,
        }})
        for uid in new_ids:
            socketio.emit("call:incoming", payload, room=f"user_{uid}")
        # Уже находящимся в звонке — обновить метаданные.
        for uid in resp.notify_user_ids:
            socketio.emit("call:invited", {
                "call_id": call_id,
                "user_ids": new_ids,
                "call": payload,
            }, room=f"user_{uid}")

    @socketio.on("call:accept")
    def on_accept(data):
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = _int_or_none((data or {}).get("call_id"))
        if not call_id:
            return

        try:
            resp = calls_client.accept_call(call_id, me)
        except CallsServiceError as e:
            _emit_error(socketio, me, e)
            return

        # Принявшему — токен; дальше он подключается к комнате LiveKit, и
        # остальные узнают о нём от самого LiveKit (ParticipantConnected).
        socketio.emit("call:accepted", {
            "call_id": call_id,
            "call": calls_client.call_to_dict(resp.call),
            "livekit": calls_client.livekit_to_dict(resp.livekit),
        }, room=f"user_{me}")

        # Плашка в чате: status ringing → active.
        emit_call_system_message_update(socketio, call_id)

    @socketio.on("call:decline")
    def on_decline(data):
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = _int_or_none((data or {}).get("call_id"))
        if not call_id:
            return

        try:
            resp = calls_client.decline_call(call_id, me)
        except CallsServiceError as e:
            _emit_error(socketio, me, e)
            return
        if not resp.HasField("call"):
            return  # звонка уже нет — no-op

        payload = {"call_id": call_id, "user_id": me}
        targets = list(resp.notify_user_ids)
        for uid in targets:
            socketio.emit("call:participant-declined", payload, room=f"user_{uid}")
        if resp.ended:
            ended_payload = {"call_id": call_id, "status": resp.call.status}
            for uid in {*targets, me}:
                socketio.emit("call:ended", ended_payload, room=f"user_{uid}")
        emit_call_system_message_update(socketio, call_id)

    @socketio.on("call:leave")
    def on_leave(data):
        """Явный выход (повесить трубку / «не возвращаюсь»). Отключение от
        комнаты придёт и вебхуком, но сокет-путь даёт мгновенную реакцию."""
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = _int_or_none((data or {}).get("call_id"))
        if not call_id:
            return

        try:
            resp = calls_client.leave_call(call_id, me)
        except CallsServiceError as e:
            _emit_error(socketio, me, e)
            return
        if not resp.HasField("call"):
            return

        if resp.ended:
            ended_payload = {"call_id": call_id, "status": resp.call.status}
            for uid in {*resp.notify_user_ids, me}:
                socketio.emit("call:ended", ended_payload, room=f"user_{uid}")
        emit_call_system_message_update(socketio, call_id)

    @socketio.on("call:end")
    def on_end(data):
        """Инициатор завершает звонок целиком (для всех)."""
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        call_id = _int_or_none((data or {}).get("call_id"))
        if not call_id:
            return

        try:
            resp = calls_client.end_call(call_id, me)
        except CallsServiceError as e:
            _emit_error(socketio, me, e)
            return
        if not resp.HasField("call"):
            return

        for uid in resp.notify_user_ids:
            socketio.emit("call:ended",
                          {"call_id": call_id, "status": resp.call.status},
                          room=f"user_{uid}")
        emit_call_system_message_update(socketio, call_id)

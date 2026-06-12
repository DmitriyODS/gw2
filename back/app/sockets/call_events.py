"""Ринг-сигналинг звонков через Socket.IO — тонкий шлюз к Go-микросервисам.

Бизнес-логика звонков (валидация, БД, ринг-state, LiveKit) живёт в callsvc
(back-go/calls), домен мессенджера — в msgsvc (back-go/messenger); Flask
здесь только:
  - резолвит пользователя по сокету и зовёт gRPC (services/calls_client.py,
    services/messenger_client.py);
  - эмитит сокет-события по данным из ответов (списки адресатов считает Go);
  - связывает домены: парный диалог и системная плашка звонка в чате —
    через gRPC msgsvc, ошибки мессенджера звонок не роняют.

События, инициированные самим сервисом (вебхуки LiveKit → call:ended),
прилетают отдельным каналом Redis — см. sockets/call_bridge.py.
"""
import json

from flask_socketio import SocketIO

from app.services import calls_client, messenger_client
from app.services.calls_client import CallsServiceError
from app.services.messenger_client import MessengerServiceError
from app.utils.logger import get_logger

logger = get_logger(__name__)


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
    """Перечитать у msgsvc системное сообщение о звонке (kind='call') и
    эмитить message:updated адресатам. Вызывается после смены статуса звонка
    (ringing → active → ended/missed). Плашки нет (group-звонок) или msgsvc
    недоступен — пропускаем: плашка вторична, звонок она не роняет."""
    try:
        resp = messenger_client.get_call_message(call_id)
    except MessengerServiceError as e:
        if e.code == "MESSENGER_UNAVAILABLE":
            logger.warning("call.pill_update_skipped", extra={"extra": {
                "call_id": call_id, "code": e.code,
            }})
        return
    if not resp.message_json:
        return
    event = {
        "conversation_id": resp.conversation_id,
        "message": json.loads(resp.message_json),
    }
    for uid in resp.notify_user_ids:
        socketio.emit("message:updated", event, room=f"user_{uid}")


def register_call_events(socketio: SocketIO) -> None:

    @socketio.on("call:start")
    def on_start(data):
        """Клиент инициирует звонок. data = {user_ids: [...], media}."""
        from flask import request as flask_request

        me = _resolve_user_id_from_sid(flask_request.sid)
        if me is None:
            return
        user_ids = _int_list((data or {}).get("user_ids"))
        media = (data or {}).get("media") or "video"
        logger.info("call.start", extra={"extra": {
            "initiator_id": me, "user_ids": user_ids, "media": media,
        }})

        # Парный диалог — домен мессенджера (msgsvc): создаём до звонка,
        # чтобы FK в БД уже существовал к моменту INSERT звонка в callsvc.
        # msgsvc недоступен — звонок не блокируем, просто пройдёт без
        # привязки к чату и без плашки (она вторична).
        conv_id = 0
        others = {uid for uid in user_ids if uid != me}
        if len(others) == 1:
            try:
                conv_id = messenger_client.ensure_dialog(me, next(iter(others)))
            except MessengerServiceError as e:
                logger.warning("call.ensure_dialog_failed", extra={"extra": {
                    "initiator_id": me, "code": e.code, "message": e.message,
                }})

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
        # фронт рендерит её специально по kind='call'. Создаёт msgsvc; его
        # ошибка звонок не роняет — плашка вторична.
        if conv_id and payload["kind"] == "p2p":
            try:
                pill = messenger_client.create_call_message(conv_id, me, call_id)
            except MessengerServiceError as e:
                logger.warning("call.pill_create_failed", extra={"extra": {
                    "call_id": call_id, "code": e.code, "message": e.message,
                }})
            else:
                event = {
                    "conversation_id": conv_id,
                    "message": json.loads(pill.message_json),
                    "from_user_id": me,
                }
                for uid in pill.notify_user_ids:
                    socketio.emit("message:new", event, room=f"user_{uid}")

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

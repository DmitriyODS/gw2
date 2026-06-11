"""Мост событий звонков: Redis pub/sub (Go-сервис) → Socket.IO (клиенты).

Go-микросервис звонков сам применяет вебхуки LiveKit (joined/left/
room_finished) и публикует результат в канал gw2:calls:events. Здесь —
фоновый слушатель, который транслирует эти события в комнаты пользователей:

  {"type": "call_ended", "call_id", "status", "notify_user_ids": [...]}
      → call:ended каждому адресату + обновление плашки в чате;
  {"type": "call_status_changed", "call_id"}
      → message:updated (плашка ringing → active).

Соединение с Redis самовосстанавливается: обрыв — пауза и переподключение.
"""
import json

from flask import Flask
from flask_socketio import SocketIO

from app.utils.logger import get_logger

logger = get_logger(__name__)

CHANNEL = "gw2:calls:events"
_RECONNECT_DELAY_SEC = 3


def start_call_bridge(app: Flask, socketio: SocketIO) -> None:
    socketio.start_background_task(_listen_loop, app, socketio)


def _listen_loop(app: Flask, socketio: SocketIO) -> None:
    import redis as redis_lib

    redis_url = app.config["REDIS_URL"]
    while True:
        try:
            client = redis_lib.from_url(redis_url, decode_responses=True)
            pubsub = client.pubsub(ignore_subscribe_messages=True)
            pubsub.subscribe(CHANNEL)
            logger.info("call_bridge.subscribed", extra={"extra": {"channel": CHANNEL}})
            for message in pubsub.listen():
                if message.get("type") != "message":
                    continue
                try:
                    _handle_event(app, socketio, json.loads(message["data"]))
                except Exception:
                    logger.exception("call_bridge.handle_failed")
        except Exception as e:
            logger.warning("call_bridge.connection_lost", extra={"extra": {"error": str(e)}})
            socketio.sleep(_RECONNECT_DELAY_SEC)


def _handle_event(app: Flask, socketio: SocketIO, event: dict) -> None:
    from app.sockets.call_events import emit_call_system_message_update

    kind = event.get("type")
    call_id = event.get("call_id")
    if not call_id:
        return

    if kind == "call_ended":
        payload = {"call_id": call_id, "status": event.get("status")}
        for uid in event.get("notify_user_ids") or []:
            socketio.emit("call:ended", payload, room=f"user_{uid}")

    if kind in ("call_ended", "call_status_changed"):
        with app.app_context():
            emit_call_system_message_update(socketio, call_id)

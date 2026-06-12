"""Обобщённый Redis-мост: события Go-микросервисов → Socket.IO (клиенты).

Socket.IO остаётся во Flask, поэтому каждый вынесенный сервис (мессенджер,
groove) публикует свои сокет-события в Redis-канал `gw2:<svc>:events`
в едином формате:

  {"event": "message:new", "rooms": ["user_12", "all"], "payload": {...}}

Обычные события транслируются вербатим в каждую комнату из `rooms`.
События с именем на "_" — служебные хуки моста: наружу не эмитятся,
а диспатчатся в зарегистрированные python-обработчики.

Канал звонков (gw2:calls:events, исторический формат) живёт отдельно —
см. sockets/call_bridge.py. Соединение с Redis самовосстанавливается:
обрыв — пауза и переподключение.
"""
import json

from flask import Flask
from flask_socketio import SocketIO

from app.utils.logger import get_logger

logger = get_logger(__name__)

CHANNELS = [
    "gw2:messenger:events",
    "gw2:groove:events",
    "gw2:tasks:events",
]
_RECONNECT_DELAY_SEC = 3


from app.integrations.yougile.bridge_hooks import (  # noqa: E402
    handle_task_archived, handle_task_updated,
)

# Служебные события (имя на "_") → обработчик(app, socketio, payload).
# tasksvc дёргает ими исходящий YouGile-пуш, пока интеграция живёт во Flask.
_INTERNAL_HANDLERS = {
    "_yg_task_updated": handle_task_updated,
    "_yg_task_archived": handle_task_archived,
}


def start_service_bridge(app: Flask, socketio: SocketIO) -> None:
    socketio.start_background_task(_listen_loop, app, socketio)


def _listen_loop(app: Flask, socketio: SocketIO) -> None:
    import redis as redis_lib

    redis_url = app.config["REDIS_URL"]
    while True:
        try:
            client = redis_lib.from_url(redis_url, decode_responses=True)
            pubsub = client.pubsub(ignore_subscribe_messages=True)
            pubsub.subscribe(*CHANNELS)
            logger.info("service_bridge.subscribed",
                        extra={"extra": {"channels": CHANNELS}})
            for message in pubsub.listen():
                if message.get("type") != "message":
                    continue
                try:
                    _handle_event(app, socketio, json.loads(message["data"]))
                except Exception:
                    logger.exception("service_bridge.handle_failed")
        except Exception as e:
            logger.warning("service_bridge.connection_lost",
                           extra={"extra": {"error": str(e)}})
            socketio.sleep(_RECONNECT_DELAY_SEC)


def _handle_event(app: Flask, socketio: SocketIO, event: dict) -> None:
    name = event.get("event")
    if not name:
        return
    payload = event.get("payload") or {}

    if name.startswith("_"):
        handler = _INTERNAL_HANDLERS.get(name)
        if handler is None:
            logger.warning("service_bridge.unknown_internal",
                           extra={"extra": {"event": name}})
            return
        handler(app, socketio, payload)
        return

    for room in event.get("rooms") or []:
        socketio.emit(name, payload, room=room)

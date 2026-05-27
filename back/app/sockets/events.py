from flask_socketio import SocketIO, join_room, disconnect
from flask_jwt_extended import decode_token
from app.utils.logger import get_logger

logger = get_logger(__name__)


def register_events(socketio: SocketIO) -> None:
    from app.sockets.call_events import register_call_events
    register_call_events(socketio)

    @socketio.on("connect")
    def on_connect(auth):
        """Верифицировать JWT из auth-payload (Socket.IO v4) и присоединить к комнатам."""
        from flask import request as flask_request
        token = (auth or {}).get("token") or flask_request.args.get("token")
        if not token:
            logger.warning("ws.connect_rejected: no token")
            disconnect()
            return False

        try:
            decoded = decode_token(token)
            user_id = decoded["sub"]
        except Exception as e:
            logger.warning("ws.connect_rejected: invalid token", extra={"extra": {"error": str(e)}})
            disconnect()
            return False

        join_room("all")
        join_room(f"user_{user_id}")
        from app.sockets import presence
        presence.on_connect(flask_request.sid, user_id)
        logger.info("ws.connect", extra={"extra": {"user_id": user_id, "event": "ws.connect"}})

    @socketio.on("disconnect")
    def on_disconnect():
        from flask import request as flask_request
        from app.sockets import presence
        from app.sockets.presence import _sid_user
        user_id = _sid_user.get(flask_request.sid)
        presence.on_disconnect(flask_request.sid)
        # Если у пользователя не осталось активных видимых соединений и он был
        # в звонке — выкинуть его оттуда и уведомить остальных.
        if user_id is not None and not presence._has_visible_connection(user_id):
            from app.sockets.call_events import cleanup_call_on_disconnect
            cleanup_call_on_disconnect(socketio, user_id)
        logger.info("ws.disconnect", extra={"extra": {"event": "ws.disconnect"}})

    @socketio.on("presence:visibility")
    def on_visibility(data):
        """Клиент сообщает, видима ли его вкладка. На мобильных это
        единственный надёжный сигнал «ушёл/вернулся» — дисконнект при
        сворачивании приходит с большой задержкой или не приходит вовсе."""
        from flask import request as flask_request
        from app.sockets import presence
        visible = bool((data or {}).get("visible", True))
        presence.on_visibility(flask_request.sid, visible)

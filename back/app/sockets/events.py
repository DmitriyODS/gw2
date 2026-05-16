from flask_socketio import SocketIO, join_room, disconnect
from flask_jwt_extended import decode_token
from jwt.exceptions import PyJWTError
from app.utils.logger import get_logger

logger = get_logger(__name__)


def register_events(socketio: SocketIO) -> None:

    @socketio.on("connect")
    def on_connect(auth):
        """Верифицировать JWT из query-param token и присоединить к комнатам."""
        from flask import request as flask_request
        token = flask_request.args.get("token")
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
        logger.info("ws.connect", extra={"extra": {"user_id": user_id, "event": "ws.connect"}})

    @socketio.on("disconnect")
    def on_disconnect():
        logger.info("ws.disconnect", extra={"extra": {"event": "ws.disconnect"}})

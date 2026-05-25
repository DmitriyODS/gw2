"""Учёт присутствия (онлайн-статус) пользователей.

Держим в памяти процесса: сколько активных сокет-соединений у каждого
пользователя (несколько вкладок/устройств — несколько соединений). Пользователь
«онлайн», пока есть хотя бы одно соединение. При обрыве последнего — пишем
last_seen_at в БД и рассылаем presence:update.

Развёртывание — один app-контейнер с eventlet, поэтому in-memory достаточно.
Если когда-нибудь появится несколько процессов socketio, presence надо будет
вынести в Redis.
"""
from datetime import datetime, timezone

from app.extensions import db, socketio
from app.models import User
from app.utils.logger import get_logger

logger = get_logger(__name__)

# user_id -> число активных соединений
_counts: dict[int, int] = {}
# sid -> user_id (чтобы на disconnect знать, кто отключился)
_sid_user: dict[str, int] = {}


def on_connect(sid: str, user_id) -> None:
    user_id = int(user_id)
    _sid_user[sid] = user_id
    was = _counts.get(user_id, 0)
    _counts[user_id] = was + 1
    if was == 0:
        socketio.emit("presence:update",
                      {"user_id": user_id, "online": True, "last_seen_at": None},
                      room="all")


def on_disconnect(sid: str) -> None:
    user_id = _sid_user.pop(sid, None)
    if user_id is None:
        return
    remaining = _counts.get(user_id, 0) - 1
    if remaining > 0:
        _counts[user_id] = remaining
        return

    _counts.pop(user_id, None)
    now = datetime.now(timezone.utc)
    try:
        db.session.execute(
            db.update(User).where(User.id == user_id).values(last_seen_at=now)
        )
        db.session.commit()
    except Exception as e:  # noqa: BLE001
        db.session.rollback()
        logger.warning("presence.last_seen_failed",
                       extra={"extra": {"user_id": user_id, "error": str(e)}})

    socketio.emit("presence:update",
                  {"user_id": user_id, "online": False, "last_seen_at": now.isoformat()},
                  room="all")


def online_user_ids() -> list[int]:
    return list(_counts.keys())

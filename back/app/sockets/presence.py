"""Учёт присутствия (онлайн-статус) пользователей.

Держим в памяти процесса состояние каждого сокет-соединения: к какому
пользователю относится и видна ли сейчас его вкладка. Пользователь «онлайн»,
пока у него есть хотя бы одно соединение с видимой вкладкой.

Почему именно видимость, а не просто наличие соединения: на мобильных
(особенно iOS Safari) при сворачивании приложения/блокировке экрана сокет
не рвётся сразу — он «замораживается», и сервер ещё долго (до ping-timeout)
считает пользователя онлайн, а last_seen потом проставляется с большим
запозданием. Поэтому клиент явно шлёт presence:visibility при уходе вкладки
в фон (visibilitychange/pagehide) — и мы сразу помечаем «не в сети» с точным
временем последней активности. На десктопе это тоже корректнее: «в сети» =
вкладка реально открыта и видима.

Развёртывание — один app-контейнер с eventlet, поэтому in-memory достаточно.
Если когда-нибудь появится несколько процессов socketio, presence надо будет
вынести в Redis.
"""
from datetime import datetime, timezone

from app.extensions import db, socketio
from app.models import User
from app.utils.logger import get_logger

logger = get_logger(__name__)

# sid -> user_id
_sid_user: dict[str, int] = {}
# sid -> видна ли вкладка этого соединения (по умолчанию True на connect)
_sid_visible: dict[str, bool] = {}
# user_id'ы, считающиеся сейчас онлайн (есть хотя бы одна видимая вкладка)
_online: set[int] = set()


def _has_visible_connection(user_id: int) -> bool:
    return any(
        _sid_visible.get(sid, False)
        for sid, uid in _sid_user.items()
        if uid == user_id
    )


def _persist_last_seen(user_id: int) -> datetime:
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
    return now


def _set_online(user_id: int, online: bool) -> None:
    """Меняем онлайн-статус пользователя и рассылаем событие только на переходе,
    чтобы не спамить presence:update и не «двигать» last_seen вперёд при поздних
    дисконнектах уже ушедшего в фон пользователя."""
    if online:
        if user_id in _online:
            return
        _online.add(user_id)
        socketio.emit("presence:update",
                      {"user_id": user_id, "online": True, "last_seen_at": None},
                      room="all")
    else:
        if user_id not in _online:
            return
        _online.discard(user_id)
        now = _persist_last_seen(user_id)
        socketio.emit("presence:update",
                      {"user_id": user_id, "online": False, "last_seen_at": now.isoformat()},
                      room="all")


def on_connect(sid: str, user_id) -> None:
    user_id = int(user_id)
    _sid_user[sid] = user_id
    _sid_visible[sid] = True
    _set_online(user_id, True)


def on_disconnect(sid: str) -> None:
    user_id = _sid_user.pop(sid, None)
    _sid_visible.pop(sid, None)
    if user_id is None:
        return
    if not _has_visible_connection(user_id):
        _set_online(user_id, False)


def on_visibility(sid: str, visible: bool) -> None:
    """Клиент сообщил, что его вкладка стала видимой/скрытой."""
    if sid not in _sid_user:
        return
    _sid_visible[sid] = bool(visible)
    user_id = _sid_user[sid]
    _set_online(user_id, _has_visible_connection(user_id))


def online_user_ids() -> list[int]:
    return list(_online)

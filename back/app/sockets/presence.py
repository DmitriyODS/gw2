"""Учёт присутствия (онлайн-статус) пользователей.

Держим в памяти процесса состояние каждого сокет-соединения: к какому
пользователю относится, видна ли вкладка и когда мы последний раз слышали
от клиента heartbeat. Пользователь «онлайн», пока у него есть хотя бы одно
соединение с видимой вкладкой И недавним heartbeat'ом.

Почему heartbeat, а не одна только видимость: на мобильных (особенно iOS
Safari) при сворачивании приложения/блокировке экрана сокет «замораживается»
— дисконнект приходит с большой задержкой или теряется, и сервер долго
считает пользователя онлайн. Клиент шлёт `presence:heartbeat` каждые ~25с,
пока вкладка видима. Фоновая задача `sweep_stale` ходит раз в `SWEEP_INTERVAL`
секунд и помечает «не в сети» тех, от кого heartbeat'а не было дольше
`STALE_AFTER` секунд. Так last_seen всегда близок к реальному уходу.

Развёртывание — один app-контейнер с eventlet, поэтому in-memory достаточно.
Если когда-нибудь появится несколько процессов socketio, presence надо будет
вынести в Redis.
"""
from datetime import datetime, timezone
from time import monotonic

from app.extensions import db, socketio
from app.models import User
from app.utils.logger import get_logger

logger = get_logger(__name__)

# Интервал между прогонами sweep_stale (сек).
SWEEP_INTERVAL = 15.0
# Если от sid нет heartbeat'а дольше — считаем «вкладка ушла в фон».
STALE_AFTER = 60.0

# sid -> user_id
_sid_user: dict[str, int] = {}
# sid -> видна ли вкладка этого соединения (по умолчанию True на connect)
_sid_visible: dict[str, bool] = {}
# sid -> monotonic time последнего heartbeat / события активности
_sid_last_beat: dict[str, float] = {}
# user_id'ы, считающиеся сейчас онлайн (есть хотя бы одна видимая вкладка)
_online: set[int] = set()

_sweep_started = False


def _has_visible_connection(user_id: int) -> bool:
    return any(
        _sid_visible.get(sid, False)
        for sid, uid in _sid_user.items()
        if uid == user_id
    )


def has_any_connection(user_id: int) -> bool:
    """Есть ли у пользователя хоть одно живое сокет-соединение (видимое или
    нет). Для звонков важно именно это: при перезагрузке вкладки/смене сети
    соединение на мгновение рвётся и переустанавливается, но в фоне сокет
    тоже валиден — звонок не должен завершаться, пока есть любой коннект."""
    return any(uid == user_id for uid in _sid_user.values())


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
    _sid_last_beat[sid] = monotonic()
    _set_online(user_id, True)


def on_disconnect(sid: str) -> None:
    user_id = _sid_user.pop(sid, None)
    _sid_visible.pop(sid, None)
    _sid_last_beat.pop(sid, None)
    if user_id is None:
        return
    if not _has_visible_connection(user_id):
        _set_online(user_id, False)


def on_visibility(sid: str, visible: bool) -> None:
    """Клиент сообщил, что его вкладка стала видимой/скрытой."""
    if sid not in _sid_user:
        return
    _sid_visible[sid] = bool(visible)
    if visible:
        _sid_last_beat[sid] = monotonic()
    user_id = _sid_user[sid]
    _set_online(user_id, _has_visible_connection(user_id))


def on_heartbeat(sid: str) -> None:
    """Регулярный пинг от клиента. Подтверждает, что вкладка жива и видима."""
    if sid not in _sid_user:
        return
    _sid_last_beat[sid] = monotonic()
    # Если sid когда-то был помечен как невидимый из-за просрочки heartbeat'а,
    # heartbeat возвращает его в строй.
    if not _sid_visible.get(sid, False):
        _sid_visible[sid] = True
        user_id = _sid_user[sid]
        _set_online(user_id, True)


def _sweep_once(app) -> None:
    """Один прогон: помечаем все sid'ы, от которых давно не было heartbeat'а,
    как невидимые. Когда у пользователя не остаётся видимых соединений —
    выставляем offline с актуальным last_seen."""
    now = monotonic()
    affected_users: set[int] = set()
    for sid, last in list(_sid_last_beat.items()):
        if not _sid_visible.get(sid, False):
            continue
        if now - last > STALE_AFTER:
            _sid_visible[sid] = False
            uid = _sid_user.get(sid)
            if uid is not None:
                affected_users.add(uid)
    if not affected_users:
        return
    with app.app_context():
        for uid in affected_users:
            if not _has_visible_connection(uid):
                _set_online(uid, False)


def start_sweeper(app) -> None:
    """Запускаем фоновую задачу один раз на процесс. Использует socketio
    background-task — eventlet-совместимый sleep."""
    global _sweep_started
    if _sweep_started:
        return
    _sweep_started = True

    def _loop():
        while True:
            try:
                _sweep_once(app)
            except Exception as e:  # noqa: BLE001
                logger.warning("presence.sweep_failed", extra={"extra": {"error": str(e)}})
            socketio.sleep(SWEEP_INTERVAL)

    socketio.start_background_task(_loop)


def online_user_ids() -> list[int]:
    return list(_online)

"""Защита от подбора пароля.

После каждых 5 подряд неудачных попыток входа аккаунт блокируется
на экспоненциально растущее время: 10с, 20с, 40с, 80с, 160с…
Удачный вход сбрасывает счётчик. Учёт ведётся в Redis (TTL — сутки),
ключ — нормализованный логин, чтобы не зависеть от IP NAT.
"""
import math
import time
from flask import current_app
from redis import Redis

_INITIAL_DELAY_SEC = 10
_LOCK_EVERY_N_FAILS = 5
_TTL_SEC = 24 * 3600


_redis_client: Redis | None = None


def _redis() -> Redis:
    global _redis_client
    if _redis_client is None:
        url = current_app.config["REDIS_URL"]
        _redis_client = Redis.from_url(url, decode_responses=True)
    return _redis_client


def _attempts_key(login: str) -> str:
    return f"gw2:bf:attempts:{login.lower().strip()}"


def _lock_key(login: str) -> str:
    return f"gw2:bf:locked_until:{login.lower().strip()}"


def get_lock_remaining(login: str) -> int:
    """Сколько секунд ещё длится блокировка для логина. 0 — не заблокирован."""
    if not login:
        return 0
    raw = _redis().get(_lock_key(login))
    if not raw:
        return 0
    try:
        until = float(raw)
    except (TypeError, ValueError):
        return 0
    remaining = int(math.ceil(until - time.time()))
    return remaining if remaining > 0 else 0


def register_failure(login: str) -> int:
    """Учесть неудачную попытку входа. Если кратно 5 — выставить блокировку.
    Возвращает количество секунд блокировки (0 если ещё не заблокирован)."""
    if not login:
        return 0
    r = _redis()
    akey = _attempts_key(login)
    attempts = r.incr(akey)
    r.expire(akey, _TTL_SEC)

    if attempts % _LOCK_EVERY_N_FAILS == 0:
        # Сколько раз уже срабатывала блокировка — для экспоненты.
        steps = attempts // _LOCK_EVERY_N_FAILS
        delay = _INITIAL_DELAY_SEC * (2 ** (steps - 1))
        until = time.time() + delay
        r.set(_lock_key(login), str(until), ex=delay + 5)
        return delay
    return 0


def register_success(login: str) -> None:
    if not login:
        return
    r = _redis()
    r.delete(_attempts_key(login), _lock_key(login))

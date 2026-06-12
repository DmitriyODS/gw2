"""gRPC-клиент groovesvc (back-go/groove) — хуки геймификации «Мой Groove».

Домен Groove (лента, питомцы, рейды, AI-механики) живёт в groovesvc;
Flask лишь сообщает ему о доменных событиях задач/юнитов. Интерфейс хуков
повторяет прежний feed_service.on_*: fire-and-forget из фонового greenlet'а,
любая ошибка только логируется — геймификация НИКОГДА не роняет основной
запрос и не задерживает его (вызов уходит после коммита, ответа не ждём).

Особенности — те же, что у calls_client/messenger_client:
  - gRPC-вызовы блокирующие (C-core), приложение живёт под eventlet —
    каждый вызов уходит через eventlet.tpool в настоящий OS-поток;
  - транспорт всегда OK; бизнес-ошибка приходит полем error в ответе.
"""
from __future__ import annotations

import os
import threading

import grpc

from app.grpc import groove_pb2, groove_pb2_grpc
from app.utils.logger import get_logger

logger = get_logger(__name__)

_HOOK_TIMEOUT = 10.0

_lock = threading.Lock()
_channel: grpc.Channel | None = None
_stub: groove_pb2_grpc.GrooveServiceStub | None = None


def _addr() -> str:
    return os.getenv("GROOVE_GRPC_ADDR", "localhost:9094")


def _get_stub() -> groove_pb2_grpc.GrooveServiceStub:
    global _channel, _stub
    if _stub is None:
        with _lock:
            if _stub is None:
                _channel = grpc.insecure_channel(_addr())
                _stub = groove_pb2_grpc.GrooveServiceStub(_channel)
    return _stub


def _execute(method, request):
    """Блокирующий вызов уводим в OS-поток через tpool (eventlet);
    без eventlet (pytest и т. п.) зовём напрямую."""
    try:
        from eventlet import tpool
    except ImportError:
        return method(request, timeout=_HOOK_TIMEOUT)
    return tpool.execute(method, request, timeout=_HOOK_TIMEOUT)


def _fire(method_name: str, request) -> None:
    """Fire-and-forget: хук уезжает в фоне, ошибки только в лог."""
    def _job():
        try:
            resp = _execute(getattr(_get_stub(), method_name), request)
            if resp.HasField("error"):
                logger.warning("groove_grpc.hook_rejected", extra={"extra": {
                    "method": method_name, "code": resp.error.code,
                    "message": resp.error.message,
                }})
        except Exception as e:
            logger.warning("groove_grpc.hook_failed", extra={"extra": {
                "method": method_name, "err": str(e),
            }})

    try:
        from app.extensions import socketio
        socketio.start_background_task(_job)
    except Exception as e:
        logger.warning("groove_grpc.spawn_failed",
                       extra={"extra": {"method": method_name, "err": str(e)}})


# ── Хуки (интерфейс прежнего feed_service.on_*) ────────────────────
# Как и прежний `_safe`: сбор полей из ORM-объектов тоже не должен ронять
# вызывающий код — любые ошибки только в лог.

def _safe(fn) -> None:
    try:
        fn()
    except Exception as e:
        logger.warning("groove_grpc.hook_failed", extra={"extra": {"err": str(e)}})


def on_unit_started(unit) -> None:
    def _job():
        _fire("OnUnitStarted", groove_pb2.UnitStartedRequest(
            company_id=unit.company_id,
            user_id=unit.user_id,
            unit_id=unit.id,
            unit_name=unit.name or "",
            task_id=unit.task_id or 0,
            task_name=(unit.task.name if unit.task else "") or "",
        ))
    _safe(_job)


def on_unit_stopped(unit) -> None:
    def _job():
        minutes = 0
        if unit.datetime_end and unit.datetime_start:
            minutes = max(0, int((unit.datetime_end - unit.datetime_start)
                                 .total_seconds() // 60))
        _fire("OnUnitStopped", groove_pb2.UnitStoppedRequest(
            company_id=unit.company_id,
            user_id=unit.user_id,
            unit_id=unit.id,
            unit_name=unit.name or "",
            task_id=unit.task_id or 0,
            task_name=(unit.task.name if unit.task else "") or "",
            minutes=minutes,
        ))
    _safe(_job)


def on_task_closed(task, actor_id=None) -> None:
    def _job():
        # «Герой» закрытия: actor → responsible → author (как в прежнем хуке).
        hero_id = actor_id or getattr(task, "responsible_user_id", None) \
            or getattr(task, "author_id", None) or 0
        _fire("OnTaskClosed", groove_pb2.TaskClosedRequest(
            company_id=task.company_id,
            hero_user_id=hero_id,
            task_id=task.id,
            task_name=task.name or "",
        ))
    _safe(_job)

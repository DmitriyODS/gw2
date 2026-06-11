"""gRPC-клиент микросервиса звонков (back-go/calls).

Вся бизнес-логика звонков живёт в Go-сервисе (callsvc): валидация, БД,
ринг-state, LiveKit-токены/комнаты, вебхуки. Flask остался транспортом
Socket.IO — сокет-хендлеры (sockets/call_events.py) зовут эти функции.

Особенности:
  - gRPC-вызовы блокирующие (C-core), а приложение живёт под eventlet —
    поэтому каждый вызов уходит через eventlet.tpool в настоящий OS-поток,
    чтобы не замораживать event-loop;
  - бизнес-ошибки сервис возвращает полем `error` в ответе (transport всегда
    OK) — здесь они конвертируются в CallsServiceError, как прежний
    CallServiceError;
  - недоступность сервиса (grpc.RpcError) — CallsServiceError с кодом
    CALLS_UNAVAILABLE, чтобы фронт показал внятную ошибку в call:error.
"""
import os
import threading
from typing import Optional

import grpc

from app.grpc import calls_pb2, calls_pb2_grpc
from app.utils.logger import get_logger

logger = get_logger(__name__)

_GRPC_TIMEOUT_SEC = 10

_lock = threading.Lock()
_channel: Optional[grpc.Channel] = None
_stub: Optional[calls_pb2_grpc.CallServiceStub] = None


class CallsServiceError(Exception):
    """Бизнес-ошибка звонков (коды совпадают с прежним CallServiceError)."""

    def __init__(self, code: str, message: str, http_status: int = 400):
        super().__init__(f"{code}: {message}")
        self.code = code
        self.message = message
        self.http_status = http_status


def _addr() -> str:
    return os.getenv("CALLS_GRPC_ADDR", "localhost:9090")


def _get_stub() -> calls_pb2_grpc.CallServiceStub:
    global _channel, _stub
    if _stub is None:
        with _lock:
            if _stub is None:
                _channel = grpc.insecure_channel(_addr())
                _stub = calls_pb2_grpc.CallServiceStub(_channel)
    return _stub


def _execute(method, request):
    """В проде приложение живёт под eventlet — блокирующий вызов уводим в
    OS-поток через tpool. Без eventlet (pytest и т. п.) зовём напрямую."""
    try:
        from eventlet import tpool
    except ImportError:
        return method(request, timeout=_GRPC_TIMEOUT_SEC)
    return tpool.execute(method, request, timeout=_GRPC_TIMEOUT_SEC)


def _call(method_name: str, request):
    method = getattr(_get_stub(), method_name)
    try:
        response = _execute(method, request)
    except grpc.RpcError as e:
        logger.error("calls_grpc.unavailable", extra={"extra": {
            "method": method_name, "code": str(e.code()), "details": e.details(),
        }})
        raise CallsServiceError(
            "CALLS_UNAVAILABLE", "Сервис звонков временно недоступен", 503)
    if response.HasField("error"):
        err = response.error
        raise CallsServiceError(err.code, err.message, err.http_status or 400)
    return response


def start_call(initiator_id: int, invitee_ids: list[int], media: str,
               conversation_id: int = 0):
    return _call("StartCall", calls_pb2.StartCallRequest(
        initiator_id=initiator_id, invitee_ids=invitee_ids,
        media=media or "video", conversation_id=conversation_id or 0,
    ))


def invite_to_call(call_id: int, inviter_id: int, invitee_ids: list[int]):
    return _call("InviteToCall", calls_pb2.InviteToCallRequest(
        call_id=call_id, inviter_id=inviter_id, invitee_ids=invitee_ids,
    ))


def accept_call(call_id: int, user_id: int):
    return _call("AcceptCall", calls_pb2.AcceptCallRequest(
        call_id=call_id, user_id=user_id,
    ))


def decline_call(call_id: int, user_id: int):
    return _call("DeclineCall", calls_pb2.DeclineCallRequest(
        call_id=call_id, user_id=user_id,
    ))


def leave_call(call_id: int, user_id: int):
    return _call("LeaveCall", calls_pb2.LeaveCallRequest(
        call_id=call_id, user_id=user_id,
    ))


def end_call(call_id: int, user_id: int):
    return _call("EndCall", calls_pb2.EndCallRequest(
        call_id=call_id, user_id=user_id,
    ))


# ── Конвертация pb → dict (форма прежних ответов REST/сокетов) ───

def call_to_dict(call: calls_pb2.Call) -> dict:
    return {
        "id": call.id,
        "kind": call.kind,
        "status": call.status,
        "media": call.media,
        "started_at": call.started_at or None,
        "ended_at": call.ended_at or None,
        "initiator_id": call.initiator_id,
        "initiator_fio": call.initiator_fio or None,
        "conversation_id": call.conversation_id or None,
        "share_code": call.share_code or None,
        "duration_sec": call.duration_sec if call.HasField("duration_sec") else None,
        "participants": [{
            "user_id": p.user_id,
            "fio": p.fio or None,
            "avatar_path": p.avatar_path or None,
            "role": p.role,
            "joined_at": p.joined_at or None,
            "left_at": p.left_at or None,
            "declined": p.declined,
        } for p in call.participants],
    }


def livekit_to_dict(info: calls_pb2.LivekitInfo) -> dict:
    return {"token": info.token, "url": info.url}

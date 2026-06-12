"""gRPC-клиент микросервиса мессенджера (back-go/messenger).

REST /api/messenger/* и вся бизнес-логика переписки живут в Go-сервисе
(msgsvc); Flask дёргает его только из смежных доменов:
  - звонки (sockets/call_events.py, call_bridge.py) — парный диалог и
    системная плашка звонка в чате;
  - pet-чат (исторически; ответы Грувика теперь генерирует groovesvc) — история диалога и ответ
    Грувика от лица бота.

Особенности — те же, что у calls_client:
  - gRPC-вызовы блокирующие (C-core), приложение живёт под eventlet —
    каждый вызов уходит через eventlet.tpool в настоящий OS-поток;
  - бизнес-ошибки сервис возвращает полем `error` в ответе (transport
    всегда OK) — конвертируются в MessengerServiceError;
  - недоступность сервиса (grpc.RpcError) — MessengerServiceError с кодом
    MESSENGER_UNAVAILABLE (503).
"""
import os
import threading
from typing import Optional

import grpc

from app.grpc import messenger_pb2, messenger_pb2_grpc
from app.utils.logger import get_logger

logger = get_logger(__name__)

_GRPC_TIMEOUT_SEC = 10

_lock = threading.Lock()
_channel: Optional[grpc.Channel] = None
_stub: Optional[messenger_pb2_grpc.MessengerServiceStub] = None


class MessengerServiceError(Exception):
    """Бизнес-ошибка мессенджера (коды совпадают с REST msgsvc)."""

    def __init__(self, code: str, message: str, http_status: int = 400):
        super().__init__(f"{code}: {message}")
        self.code = code
        self.message = message
        self.http_status = http_status


def _addr() -> str:
    return os.getenv("MESSENGER_GRPC_ADDR", "localhost:9092")


def _get_stub() -> messenger_pb2_grpc.MessengerServiceStub:
    global _channel, _stub
    if _stub is None:
        with _lock:
            if _stub is None:
                _channel = grpc.insecure_channel(_addr())
                _stub = messenger_pb2_grpc.MessengerServiceStub(_channel)
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
        logger.error("messenger_grpc.unavailable", extra={"extra": {
            "method": method_name, "code": str(e.code()), "details": e.details(),
        }})
        raise MessengerServiceError(
            "MESSENGER_UNAVAILABLE", "Сервис мессенджера временно недоступен", 503)
    if response.HasField("error"):
        err = response.error
        raise MessengerServiceError(err.code, err.message, err.http_status or 400)
    return response


def ensure_dialog(user_a_id: int, user_b_id: int) -> int:
    """Найти или создать парный диалог. Возвращает conversation_id."""
    resp = _call("EnsureDialog", messenger_pb2.EnsureDialogRequest(
        user_a_id=user_a_id, user_b_id=user_b_id,
    ))
    return resp.conversation_id


def create_call_message(conversation_id: int, sender_id: int, call_id: int):
    """Системная плашка звонка (kind='call'). Ответ: message_json — готовый
    JSON-снапшот сообщения в форме REST, notify_user_ids — адресаты."""
    return _call("CreateCallMessage", messenger_pb2.CreateCallMessageRequest(
        conversation_id=conversation_id, sender_id=sender_id, call_id=call_id,
    ))


def get_call_message(call_id: int):
    """Актуальный снапшот плашки звонка (для message:updated)."""
    return _call("GetCallMessage", messenger_pb2.GetCallMessageRequest(
        call_id=call_id,
    ))


def post_bot_message(conversation_id: int, text: str) -> int:
    """Сообщение бота (sender NULL + is_bot). msgsvc сам эмитит message:new
    через Redis-мост — на стороне Flask эмитить ничего не нужно."""
    resp = _call("PostBotMessage", messenger_pb2.PostBotMessageRequest(
        conversation_id=conversation_id, text=text,
    ))
    return resp.message_id


def list_recent_messages(conversation_id: int, limit: int):
    """Последние сообщения диалога в хронологическом порядке —
    контекст для AI-ответа Грувика. Элементы: {id, is_bot, sender_id,
    text, created_at}."""
    resp = _call("ListRecentMessages", messenger_pb2.ListRecentMessagesRequest(
        conversation_id=conversation_id, limit=limit,
    ))
    return list(resp.messages)

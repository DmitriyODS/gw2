"""gRPC-клиент LLM-шлюза (back-go/ai, aisvc).

aisvc владеет ключами компаний, вызовами ProxyAPI/OpenAI и эмбеддингами
(task_embeddings + pgvector). Flask дёргает его из groove/tv/поиска задач.

`get_ai_client(company_id)` — публичная точка входа с ПРЕЖНИМ контрактом:
возвращает клиент или `None`, если ИИ у компании выключен ИЛИ aisvc
недоступен (fail-open: ИИ-фичи опциональны, потребители уходят в свои
статические фолбэки и никогда не падают из-за ИИ).

Особенности — те же, что у calls_client/messenger_client:
  - gRPC-вызовы блокирующие (C-core), приложение живёт под eventlet —
    каждый вызов уходит через eventlet.tpool в настоящий OS-поток;
  - бизнес-ошибки сервис возвращает полем `error` в ответе (transport
    всегда OK) — конвертируются в AiServiceError;
  - методы клиента (chat/chat_with_tools/embed) при ошибках БРОСАЮТ
    AiServiceError — ровно как раньше бросал OpenAIError; вызывающий код
    ловит Exception и подставляет фолбэк.

Status кэшируется per-company на 60 секунд (как прежний кэш AIClient):
после смены настроек в aisvc worst-case ждать минуту — приемлемо.
"""
from __future__ import annotations

import json
import os
import threading
import time
from dataclasses import dataclass
from typing import Callable, Iterable, Optional

import grpc

from app.grpc import ai_pb2, ai_pb2_grpc
from app.utils.logger import get_logger

logger = get_logger(__name__)

# Дефолты на случай, если aisvc не прислал модель (БД это не допускает).
DEFAULT_MODEL_CHAT = "gpt-4o-mini"
DEFAULT_MODEL_EMBEDDING = "text-embedding-3-small"

_REQUEST_TIMEOUT = 30.0   # дефолтный таймаут LLM-вызова (как раньше)
_STATUS_TIMEOUT = 5.0     # Status/ReindexTask — быстрые RPC без похода в LLM
_GRPC_SLACK_SEC = 5.0     # запас gRPC-дедлайна сверх timeout_sec самого LLM
_CACHE_TTL_SEC = 60.0

_lock = threading.Lock()
_channel: Optional[grpc.Channel] = None
_stub: Optional[ai_pb2_grpc.AiServiceStub] = None

# Status-кэш: company_id → (клиент, истекает_в). Негатив не кэшируем —
# включение ИИ подхватывается сразу (как и раньше).
_cache: dict[int, tuple["AIClient", float]] = {}


class AiServiceError(Exception):
    """Бизнес/транспортная ошибка aisvc (код AI_UNAVAILABLE — сервис лёг)."""

    def __init__(self, code: str, message: str, http_status: int = 400):
        super().__init__(f"{code}: {message}")
        self.code = code
        self.message = message
        self.http_status = http_status


def _addr() -> str:
    return os.getenv("AI_GRPC_ADDR", "localhost:9093")


def _get_stub() -> ai_pb2_grpc.AiServiceStub:
    global _channel, _stub
    if _stub is None:
        with _lock:
            if _stub is None:
                _channel = grpc.insecure_channel(_addr())
                _stub = ai_pb2_grpc.AiServiceStub(_channel)
    return _stub


def _execute(method, request, timeout: float):
    """В проде приложение живёт под eventlet — блокирующий вызов уводим в
    OS-поток через tpool. Без eventlet (pytest и т. п.) зовём напрямую."""
    try:
        from eventlet import tpool
    except ImportError:
        return method(request, timeout=timeout)
    return tpool.execute(method, request, timeout=timeout)


def _call(method_name: str, request, timeout: float = _STATUS_TIMEOUT):
    method = getattr(_get_stub(), method_name)
    try:
        response = _execute(method, request, timeout)
    except grpc.RpcError as e:
        logger.warning("ai_grpc.unavailable", extra={"extra": {
            "method": method_name, "code": str(e.code()), "details": e.details(),
        }})
        raise AiServiceError(
            "AI_UNAVAILABLE", "Сервис ИИ временно недоступен", 503)
    if response.HasField("error"):
        err = response.error
        raise AiServiceError(err.code, err.message, err.http_status or 400)
    return response


@dataclass(frozen=True)
class AIClient:
    """Лёгкий хэндл «ИИ компании включён»: знает модели, ходит в aisvc."""

    company_id: int
    model_chat: str
    model_embedding: str

    def _chat_once(self, messages: list[dict], tools_json: str,
                   max_tokens: int, temperature: float, timeout: float):
        return _call("Chat", ai_pb2.ChatRequest(
            company_id=self.company_id,
            messages_json=json.dumps(messages, ensure_ascii=False, default=str),
            tools_json=tools_json,
            max_tokens=max_tokens,
            temperature=temperature,
            timeout_sec=timeout,
        ), timeout=timeout + _GRPC_SLACK_SEC)

    def chat(self, messages: list[dict], *, model: str | None = None,
             max_tokens: int = 400, temperature: float = 0.7,
             timeout: float = _REQUEST_TIMEOUT) -> str:
        # `model` сохранён для совместимости сигнатуры; модель выбирает aisvc
        # по настройкам компании (никто из потребителей её и не передавал).
        resp = self._chat_once(messages, "", max_tokens, temperature, timeout)
        return (resp.content or "").strip()

    def chat_with_tools(
        self,
        messages: list[dict],
        *,
        tools: list[dict],
        on_tool: Callable[[str, dict], object],
        model: str | None = None,
        max_tokens: int = 400,
        temperature: float = 0.7,
        timeout: float = _REQUEST_TIMEOUT,
        max_iterations: int = 4,
    ) -> str:
        """Чат с OpenAI function-calling; цикл крутится здесь, во Flask.

        aisvc.Chat выполняет РОВНО ОДИН ход (messages → content | tool_calls).
        `on_tool(name, args_dict)` вызывается синхронно для каждого tool_call
        и должен вернуть JSON-сериализуемый результат. Цикл останавливается,
        когда модель ответила текстом или достигнут `max_iterations`
        (страховка от бесконечных циклов).

        Список `messages` не мутируем — копируем внутри.
        """
        convo = list(messages)
        tools_json = json.dumps(tools, ensure_ascii=False)
        for _ in range(max_iterations):
            resp = self._chat_once(convo, tools_json,
                                   max_tokens, temperature, timeout)
            tool_calls = _parse_tool_calls(resp.tool_calls_json)
            if not tool_calls:
                return (resp.content or "").strip()

            # Сохраняем assistant-сообщение с tool_calls в истории.
            convo.append({
                "role": "assistant",
                "content": resp.content or "",
                "tool_calls": tool_calls,
            })
            for tc in tool_calls:
                fn = tc.get("function") or {}
                try:
                    args = json.loads(fn.get("arguments") or "{}")
                except json.JSONDecodeError:
                    args = {}
                try:
                    result = on_tool(fn.get("name") or "", args)
                except Exception as e:
                    result = {"error": f"tool_handler_failed: {e}"}
                convo.append({
                    "role": "tool",
                    "tool_call_id": tc.get("id") or "",
                    "content": json.dumps(result, ensure_ascii=False, default=str),
                })

        # Достигли лимита итераций — финальный заход без tools, чтобы
        # модель точно ответила текстом, а не очередным tool_call.
        resp = self._chat_once(convo, "", max_tokens, temperature, timeout)
        return (resp.content or "").strip()

    def embed(self, texts: str | Iterable[str], *, model: str | None = None,
              timeout: float = _REQUEST_TIMEOUT) -> list[list[float]]:
        """Возвращает список векторов в том же порядке, что входные тексты."""
        items = [texts] if isinstance(texts, str) else list(texts)
        return [
            list(_call("Embed", ai_pb2.EmbedRequest(
                company_id=self.company_id, text=item,
            ), timeout=timeout + _GRPC_SLACK_SEC).vector)
            for item in items
        ]


def _parse_tool_calls(raw: str) -> list[dict]:
    if not raw:
        return []
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        logger.warning("ai.tool_calls.bad_json", extra={"extra": {"raw": raw[:200]}})
        return []
    return parsed if isinstance(parsed, list) else []


def get_ai_client(company_id: int | None) -> AIClient | None:
    if company_id is None:
        return None
    now = time.monotonic()
    cached = _cache.get(company_id)
    if cached and cached[1] > now:
        return cached[0]
    try:
        resp = _call("Status", ai_pb2.StatusRequest(company_id=company_id))
    except AiServiceError:
        # fail-open: aisvc недоступен / ошибка — ИИ-фичи тихо выключаются.
        _cache.pop(company_id, None)
        return None
    if not resp.enabled:
        # явно затираем кэш, чтобы при выключении подхватилось сразу.
        _cache.pop(company_id, None)
        return None
    client = AIClient(
        company_id=company_id,
        model_chat=resp.model_chat or DEFAULT_MODEL_CHAT,
        model_embedding=resp.model_embedding or DEFAULT_MODEL_EMBEDDING,
    )
    _cache[company_id] = (client, now + _CACHE_TTL_SEC)
    return client


# ─────────────── эмбеддинги задач (владелец — aisvc) ───────────────

def schedule_reindex(task_id: int) -> None:
    """Fire-and-forget переиндексация задачи в aisvc. Безопасен в любом
    сервисе: тихо игнорирует ошибки и НИКОГДА не валит вызывающий запрос."""
    def _job():
        try:
            _call("ReindexTask", ai_pb2.ReindexTaskRequest(task_id=task_id))
        except Exception as e:
            logger.warning("ai.reindex.failed",
                           extra={"extra": {"task_id": task_id, "err": str(e)}})

    try:
        from app.extensions import socketio
        socketio.start_background_task(_job)
    except Exception as e:
        logger.warning("ai.reindex.spawn_failed",
                       extra={"extra": {"task_id": task_id, "err": str(e)}})


def semantic_search(company_id: int, query: str) -> list[tuple[int, float]]:
    """Список (task_id, score) по убыванию релевантности из aisvc.

    Fail-open: ошибка/недоступность → пустой список (поиск задач при
    включённом ИИ честно отдаёт пустую семантическую выдачу, как раньше).
    """
    if not query.strip():
        return []
    try:
        resp = _call("SemanticSearch", ai_pb2.SemanticSearchRequest(
            company_id=company_id, query=query,
        ), timeout=_REQUEST_TIMEOUT)
    except AiServiceError as e:
        logger.warning("ai.search.failed", extra={"extra": {
            "company_id": company_id, "err": str(e)}})
        return []
    return [(h.task_id, float(h.score)) for h in resp.hits]

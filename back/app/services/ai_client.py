"""Тонкая обёртка вокруг openai SDK, настроенная на ProxyAPI.

`get_ai_client(company_id)` — единственная публичная точка входа. Возвращает
готовый `AIClient` для компании или `None`, если AI выключен / ключ не задан /
ключ нерасшифровываемый (например, сменили AI_KEY_ENCRYPTION_KEY).

Кэш в памяти процесса: один app-контейнер с eventlet, поэтому простой dict с
TTL 60 секунд достаточен. После сохранения новых настроек суперадмином через
API в worst-case ждать минуту до подхвата — приемлемо.
"""
from __future__ import annotations

import json
import time
from dataclasses import dataclass
from typing import Callable, Iterable

from openai import OpenAI, OpenAIError

from app.extensions import db
from app.models.company import Company
from app.utils.ai_secret import decrypt_api_key
from app.utils.logger import get_logger

logger = get_logger(__name__)


PROXYAPI_BASE_URL = "https://api.proxyapi.ru/openai/v1"

# Дефолты на случай, если в БД пусто (миграция этого не допускает, но всё же).
DEFAULT_MODEL_CHAT = "gpt-4o-mini"
DEFAULT_MODEL_EMBEDDING = "text-embedding-3-small"

_REQUEST_TIMEOUT = 30.0   # фактическим эндпоинтам можно переопределять
_CACHE_TTL_SEC = 60.0

# user_id-agnostic кэш: ключ — company_id, значение — (client, expires_at).
_cache: dict[int, tuple["AIClient", float]] = {}


@dataclass(frozen=True)
class AIClient:
    """Минимальный обёрнутый клиент. Реальный openai-объект — _raw."""

    company_id: int
    model_chat: str
    model_embedding: str
    _raw: OpenAI

    def chat(self, messages: list[dict], *, model: str | None = None,
             max_tokens: int = 400, temperature: float = 0.7,
             timeout: float = _REQUEST_TIMEOUT) -> str:
        resp = self._raw.chat.completions.create(
            model=model or self.model_chat,
            messages=messages,
            max_tokens=max_tokens,
            temperature=temperature,
            timeout=timeout,
        )
        return (resp.choices[0].message.content or "").strip()

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
        """Чат с поддержкой OpenAI function-calling.

        `on_tool(name, args_dict)` вызывается синхронно для каждого
        tool_call и должен вернуть JSON-сериализуемый результат. Цикл
        останавливается, когда модель вернула обычный ответ без tool_calls
        или достигнут `max_iterations` (страховка от бесконечных циклов).

        Список `messages` мутируется — копируем внутри, чтобы не нагадить
        вызывающему коду.
        """
        convo = list(messages)
        for _ in range(max_iterations):
            resp = self._raw.chat.completions.create(
                model=model or self.model_chat,
                messages=convo,
                tools=tools,
                max_tokens=max_tokens,
                temperature=temperature,
                timeout=timeout,
            )
            msg = resp.choices[0].message
            tool_calls = getattr(msg, "tool_calls", None) or []
            if not tool_calls:
                return (msg.content or "").strip()

            # Сохраняем assistant-сообщение с tool_calls в истории.
            convo.append({
                "role": "assistant",
                "content": msg.content or "",
                "tool_calls": [
                    {
                        "id": tc.id,
                        "type": "function",
                        "function": {
                            "name": tc.function.name,
                            "arguments": tc.function.arguments or "{}",
                        },
                    }
                    for tc in tool_calls
                ],
            })
            for tc in tool_calls:
                try:
                    args = json.loads(tc.function.arguments or "{}")
                except json.JSONDecodeError:
                    args = {}
                try:
                    result = on_tool(tc.function.name, args)
                except Exception as e:
                    result = {"error": f"tool_handler_failed: {e}"}
                convo.append({
                    "role": "tool",
                    "tool_call_id": tc.id,
                    "content": json.dumps(result, ensure_ascii=False, default=str),
                })

        # Достигли лимита итераций — финальный заход без tools, чтобы
        # модель точно ответила текстом, а не очередным tool_call.
        resp = self._raw.chat.completions.create(
            model=model or self.model_chat,
            messages=convo,
            max_tokens=max_tokens,
            temperature=temperature,
            timeout=timeout,
        )
        return (resp.choices[0].message.content or "").strip()

    def embed(self, texts: str | Iterable[str], *, model: str | None = None,
              timeout: float = _REQUEST_TIMEOUT) -> list[list[float]]:
        """Возвращает список векторов в том же порядке, что входные тексты."""
        items = [texts] if isinstance(texts, str) else list(texts)
        if not items:
            return []
        resp = self._raw.embeddings.create(
            model=model or self.model_embedding,
            input=items,
            timeout=timeout,
        )
        # API возвращает items с полем index — на всякий случай сортируем.
        return [d.embedding for d in sorted(resp.data, key=lambda d: d.index)]

    def test(self) -> dict:
        """Дёшево проверяем, что и chat, и embedding модели отвечают.

        Не падаем при ошибке — возвращаем структуру с флагами для UI.
        """
        result = {"chat": False, "embedding": False, "error": None, "latency_ms": 0}
        t0 = time.monotonic()
        try:
            self.chat(
                [{"role": "user", "content": "ping"}],
                max_tokens=2, temperature=0, timeout=10.0,
            )
            result["chat"] = True
        except OpenAIError as e:
            result["error"] = f"chat: {e}"
        except Exception as e:
            result["error"] = f"chat: {e}"
        try:
            self.embed("ping", timeout=10.0)
            result["embedding"] = True
        except OpenAIError as e:
            result["error"] = (result["error"] or "") + f" embedding: {e}"
        except Exception as e:
            result["error"] = (result["error"] or "") + f" embedding: {e}"
        result["latency_ms"] = int((time.monotonic() - t0) * 1000)
        return result


def _build_client(company: Company) -> AIClient | None:
    if not company.ai_enabled or not company.ai_api_key_enc:
        return None
    api_key = decrypt_api_key(company.ai_api_key_enc)
    if not api_key:
        logger.warning("ai.decrypt_failed", extra={"company_id": company.id})
        return None
    raw = OpenAI(api_key=api_key, base_url=PROXYAPI_BASE_URL,
                 timeout=_REQUEST_TIMEOUT)
    return AIClient(
        company_id=company.id,
        model_chat=company.ai_model_chat or DEFAULT_MODEL_CHAT,
        model_embedding=company.ai_model_embedding or DEFAULT_MODEL_EMBEDDING,
        _raw=raw,
    )


def get_ai_client(company_id: int | None) -> AIClient | None:
    if company_id is None:
        return None
    now = time.monotonic()
    cached = _cache.get(company_id)
    if cached and cached[1] > now:
        return cached[0]
    company = db.session.get(Company, company_id)
    if company is None:
        return None
    client = _build_client(company)
    if client is not None:
        _cache[company_id] = (client, now + _CACHE_TTL_SEC)
    else:
        # явно затираем кэш, чтобы при выключении подхватилось сразу.
        _cache.pop(company_id, None)
    return client


def invalidate_ai_client(company_id: int) -> None:
    """Вызывать сразу после изменения AI-настроек компании."""
    _cache.pop(company_id, None)

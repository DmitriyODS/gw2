"""HTTP-клиент YouGile REST API v2 (`https://ru.yougile.com/api-v2`).

Намеренно тонкая обёртка: только то, что нужно нашей интеграции (auth,
проекты/доски/колонки, задачи, чат задачи, файлы, webhooks). Никаких
abstract-фабрик и DI — простой объект с методами.

Аутентификация. На большинство методов идёт `Authorization: Bearer <key>`,
кроме трёх auth-эндпоинтов (`/auth/companies`, `/auth/keys`, `/auth/keys/get`)
и `DELETE /auth/keys/{key}` — там Bearer тоже не нужен.

Rate-limit. Сервер отдаёт 429 без `Retry-After` (по нашим наблюдениям). Делаем
до трёх попыток с экспоненциальным backoff'ом (1с/2с/4с). На 4xx (кроме 429)
сразу бросаем YougileError, ретрай ничего не даст. 5xx — ретраим столько же.
"""
from __future__ import annotations

import logging
import time
from typing import Any

import requests


BASE_URL = "https://ru.yougile.com/api-v2"
DEFAULT_TIMEOUT = 15  # секунд. YG обычно отвечает <1с, но webhook-эндпоинт долгий.
MAX_RETRIES = 3
RETRY_STATUSES = {429, 500, 502, 503, 504}

logger = logging.getLogger(__name__)


class YougileError(RuntimeError):
    """Любая ошибка от YG, которая не вписывается в auth/rate-limit."""
    def __init__(self, message: str, status: int | None = None, payload: Any = None):
        super().__init__(message)
        self.status = status
        self.payload = payload


class YougileAuthError(YougileError):
    """401/403 — ключ невалиден или прав не хватает."""


class YougileRateLimited(YougileError):
    """429 после всех ретраев. Вызывающий код решает, что делать."""


class YougileClient:
    """Тонкий клиент. Создаётся либо с `key` (Bearer), либо без — для auth-флоу."""

    def __init__(self, key: str | None = None, *, timeout: int = DEFAULT_TIMEOUT,
                 session: requests.Session | None = None):
        self.key = key
        self.timeout = timeout
        self._session = session or requests.Session()
        self._session.headers.setdefault("User-Agent", "GrooveWork-Yougile/1.0")
        self._session.headers.setdefault("Accept", "application/json")

    # ── низкоуровневое ───────────────────────────────────────────────────

    def _headers(self, anonymous: bool = False) -> dict[str, str]:
        h = {"Content-Type": "application/json"}
        if not anonymous and self.key:
            h["Authorization"] = f"Bearer {self.key}"
        return h

    def _request(self, method: str, path: str, *,
                 json: Any = None, params: dict | None = None,
                 anonymous: bool = False) -> Any:
        url = f"{BASE_URL}{path}"
        for attempt in range(MAX_RETRIES):
            try:
                resp = self._session.request(
                    method, url,
                    headers=self._headers(anonymous=anonymous),
                    json=json, params=params,
                    timeout=self.timeout,
                )
            except requests.RequestException as e:
                # Сетевые/таймауты — ретраим как 5xx.
                if attempt == MAX_RETRIES - 1:
                    raise YougileError(f"YouGile недоступен: {e}") from e
                time.sleep(2 ** attempt)
                continue

            if resp.status_code in RETRY_STATUSES and attempt < MAX_RETRIES - 1:
                time.sleep(2 ** attempt)
                continue

            return self._parse(resp)

        # До сюда дойти не должны — последний return уже отработал.
        raise YougileError("retry-loop exhausted")  # pragma: no cover

    @staticmethod
    def _parse(resp: requests.Response) -> Any:
        if resp.status_code == 401:
            raise YougileAuthError("Неверный логин/пароль или ключ", status=401)
        if resp.status_code == 403:
            raise YougileAuthError("Нет доступа", status=403)
        if resp.status_code == 429:
            raise YougileRateLimited("Превышен лимит запросов YouGile", status=429)
        if resp.status_code >= 400:
            try:
                payload = resp.json()
            except ValueError:
                payload = resp.text
            raise YougileError(f"YouGile {resp.status_code}",
                               status=resp.status_code, payload=payload)
        if not resp.content:
            return None
        try:
            return resp.json()
        except ValueError:
            return resp.text

    # ── auth ─────────────────────────────────────────────────────────────

    def list_companies(self, login: str, password: str, name: str | None = None) -> list[dict]:
        """`POST /auth/companies`. Без Bearer'а."""
        body: dict[str, Any] = {"login": login, "password": password}
        if name:
            body["name"] = name
        data = self._request("POST", "/auth/companies", json=body, anonymous=True)
        # API возвращает обёртку `{content: [...], paging: {...}}`.
        if isinstance(data, dict) and "content" in data:
            return list(data["content"])
        return data or []

    def create_key(self, login: str, password: str, company_id: str) -> str:
        """`POST /auth/keys`. Возвращает строку-ключ."""
        body = {"login": login, "password": password, "companyId": company_id}
        data = self._request("POST", "/auth/keys", json=body, anonymous=True)
        if not isinstance(data, dict) or "key" not in data:
            raise YougileError("Неожиданный ответ /auth/keys", payload=data)
        return str(data["key"])

    def delete_key(self, key: str) -> None:
        """`DELETE /auth/keys/{key}`. Анонимно — кто знает ключ, тот его и удалит."""
        self._request("DELETE", f"/auth/keys/{key}", anonymous=True)

    # ── профиль / структура ──────────────────────────────────────────────

    def me(self) -> dict:
        return self._request("GET", "/users/me")

    def list_projects(self, limit: int = 1000) -> list[dict]:
        return self._page("/projects", limit=limit)

    def list_boards(self, project_id: str | None = None, limit: int = 1000) -> list[dict]:
        params = {"projectId": project_id} if project_id else {}
        return self._page("/boards", params=params, limit=limit)

    def list_columns(self, board_id: str, limit: int = 1000) -> list[dict]:
        return self._page("/columns", params={"boardId": board_id}, limit=limit)

    # ── задачи ───────────────────────────────────────────────────────────

    def get_task(self, task_id: str) -> dict:
        return self._request("GET", f"/tasks/{task_id}")

    def create_task(self, body: dict) -> dict:
        return self._request("POST", "/tasks", json=body)

    def update_task(self, task_id: str, body: dict) -> dict:
        return self._request("PUT", f"/tasks/{task_id}", json=body)

    # ── чат задачи ───────────────────────────────────────────────────────

    def post_chat_message(self, chat_id: str, body: dict) -> dict:
        return self._request("POST", f"/chats/{chat_id}/messages", json=body)

    # ── webhooks ─────────────────────────────────────────────────────────

    def create_webhook(self, url: str, event: str, filters: list[dict] | None = None) -> dict:
        body = {"url": url, "event": event, "filters": filters or []}
        return self._request("POST", "/webhooks", json=body)

    def update_webhook(self, webhook_id: str, body: dict) -> dict:
        return self._request("PUT", f"/webhooks/{webhook_id}", json=body)

    def list_webhooks(self, limit: int = 1000) -> list[dict]:
        return self._page("/webhooks", limit=limit)

    # ── пагинация (внутреннее) ───────────────────────────────────────────

    def _page(self, path: str, *, params: dict | None = None, limit: int = 1000) -> list[dict]:
        """Собрать все страницы list-эндпоинта.

        В нашем сценарии (проекты/доски/колонки одной компании) элементов
        десятки/сотни, лимит 1000 покрывает почти всё с одного запроса;
        полноценная пагинация на будущее, без unbounded-цикла.
        """
        out: list[dict] = []
        offset = 0
        page_size = min(limit, 1000)
        while True:
            p = dict(params or {})
            p["limit"] = page_size
            p["offset"] = offset
            data = self._request("GET", path, params=p)
            content: list = []
            if isinstance(data, dict) and "content" in data:
                content = data["content"]
            elif isinstance(data, list):
                content = data
            out.extend(content)
            if len(content) < page_size or len(out) >= limit:
                break
            offset += page_size
        return out[:limit]

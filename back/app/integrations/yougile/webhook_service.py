"""Регистрация и обработка webhook'ов YouGile.

Два маршрута:

  1. Исходящая регистрация: при включении интеграции в настройках компании
     (uses_yougile=True + есть проект/доска) мы дёргаем `POST /webhooks`
     в YouGile один раз. Возвращённый id храним в `companies.yg_webhook_id`,
     а сгенерированный secret — в `yg_webhook_secret` (он будет в URL,
     по которому YG нам стучится).

  2. Ингресс: эндпоинт `POST /api/yougile/webhook/<company_id>/<secret>`
     принимает payload, проверяет secret, и применяет изменения к нашей
     задаче через `apply_event(...)`. Применение — в `task_apply.py`
     (вынесено, чтобы webhook_service не разрастался).

Антицикл: см. `task_apply.py` — при сравнении `sync_hash` пропускаем своё эхо.

Прод-настройки:
  - `YOUGILE_WEBHOOK_PUBLIC_BASE` — публичный URL приложения. Без него YG
    некуда стучаться. Не задан → регистрация падает с понятным сообщением.
  - Nginx уже проксирует `/api/*` в Flask, отдельного route не нужно.
"""
from __future__ import annotations

import os
import secrets
import re

from app.extensions import db
from app.integrations.yougile.account_service import build_client_for_user
from app.integrations.yougile.client import (
    YougileAuthError, YougileError,
)
from app.models.company import Company
from app.models.user import User
from app.utils.logger import get_logger

logger = get_logger(__name__)


class YougileWebhookError(RuntimeError):
    def __init__(self, code: str, message: str):
        super().__init__(message)
        self.code = code
        self.message = message


# Подписываемся одной строкой — `task-*` покрывает created/updated/moved/
# deleted/restored/renamed/completed. `chat_message-*` НЕ подписываем — чат
# мы намеренно не зеркалим (см. план интеграции, п.1 из ответов пользователя).
_EVENT_PATTERN = "task-.*"


def _public_base() -> str:
    base = (os.environ.get("YOUGILE_WEBHOOK_PUBLIC_BASE") or "").strip().rstrip("/")
    if not base:
        raise YougileWebhookError(
            "PUBLIC_BASE_MISSING",
            "Не задан YOUGILE_WEBHOOK_PUBLIC_BASE — невозможно зарегистрировать webhook",
        )
    return base


def _ingress_url(company: Company) -> str:
    return (
        f"{_public_base()}/api/yougile/webhook/"
        f"{company.id}/{company.yg_webhook_secret}"
    )


def ensure_registered(actor: User, company: Company) -> None:
    """Зарегистрировать webhook на стороне YouGile (или обновить, если уже есть).

    Идемпотентно: если в `companies.yg_webhook_id` уже что-то есть, делаем PUT
    на тот же id. Это покрывает «сменили доску — надо переподписаться».
    """
    if not (company.yg_board_id and company.yg_company_id):
        raise YougileWebhookError(
            "NOT_CONFIGURED",
            "Сначала выберите проект и доску",
        )

    client = build_client_for_user(actor)
    if client is None:
        raise YougileWebhookError(
            "NOT_CONNECTED",
            "Подключите свой YouGile-аккаунт",
        )

    if not company.yg_webhook_secret:
        company.yg_webhook_secret = secrets.token_urlsafe(24)

    payload = {
        "url": _ingress_url(company),
        "event": _EVENT_PATTERN,
        # Фильтр по location — id доски: YG прислёт события только по нашим
        # карточкам, чужие доски игнор. Имя фильтра у YG — `location`,
        # значение — массив id (project/board/column тоже подходят).
        "filters": [{"name": "location", "value": [company.yg_board_id]}],
    }
    try:
        if company.yg_webhook_id:
            client.update_webhook(company.yg_webhook_id, payload)
        else:
            data = client.create_webhook(**payload)
            wid = data.get("id") if isinstance(data, dict) else None
            if not wid:
                raise YougileWebhookError(
                    "BAD_RESPONSE",
                    "YouGile не вернул id webhook'а",
                )
            company.yg_webhook_id = wid
    except YougileAuthError:
        raise YougileWebhookError("BAD_KEY",
                                  "Ключ YouGile недействителен, переподключите аккаунт")
    except YougileError as e:
        raise YougileWebhookError("YOUGILE_ERROR", f"YouGile: {e}")

    db.session.commit()
    logger.info("yougile.webhook_registered",
                extra={"company_id": company.id, "webhook_id": company.yg_webhook_id})


def deregister(actor: User, company: Company) -> None:
    """Отписаться от webhook'а.

    YouGile не даёт DELETE на /webhooks/{id}, только PUT (отключение). Делаем
    PUT с url=пустая строка и event=`disabled-*` — YG примет как «обновили,
    но всё равно никуда не доставлять». Параллельно чистим локальные поля
    yg_webhook_id/yg_webhook_secret, чтобы при следующем enable создать
    новый webhook с чистого листа.
    """
    if not company.yg_webhook_id:
        return
    client = build_client_for_user(actor)
    if client is not None:
        try:
            client.update_webhook(company.yg_webhook_id, {
                "url": "https://example.invalid/disabled",
                "event": "disabled-event",
                "filters": [],
            })
        except YougileError as e:
            logger.warning("yougile.webhook_deregister_failed",
                           extra={"company_id": company.id, "err": str(e)})
    company.yg_webhook_id = None
    company.yg_webhook_secret = None
    db.session.commit()
    logger.info("yougile.webhook_deregistered", extra={"company_id": company.id})


# ── проверка ingress secret'а ─────────────────────────────────────────────

def verify_secret(company: Company, secret: str) -> bool:
    """Constant-time сравнение, чтобы не утечь secret по таймингу."""
    expected = company.yg_webhook_secret or ""
    if not expected or not secret:
        return False
    return secrets.compare_digest(expected, secret)

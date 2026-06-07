"""Настройки YouGile-интеграции на уровне компании.

Только DIRECTOR+ (см. require_role в blueprint'е). Сам сервис не проверяет
права — это делает декоратор, сервис принимает уже валидированного
пользователя и работает с его компанией.

Этап 2 — сохраняем выбор: yg_company / project / board / completed_column,
резолвим первую колонку доски (yg_first_column_id).
Webhook регистрируется на этапе 4.
"""
from __future__ import annotations

from dataclasses import dataclass

from app.extensions import db
from app.integrations.yougile.account_service import build_client_for_user
from app.integrations.yougile.client import (
    YougileAuthError, YougileError,
)
from app.integrations.yougile.webhook_service import (
    YougileWebhookError, deregister as wh_deregister, ensure_registered as wh_register,
)
from app.models.company import Company
from app.models.user import User
from app.utils.logger import get_logger

logger = get_logger(__name__)


class YougileCompanyError(RuntimeError):
    def __init__(self, code: str, message: str):
        super().__init__(message)
        self.code = code
        self.message = message


@dataclass
class CompanyYougileSettings:
    enabled: bool
    yg_company_id: str | None
    yg_company_name: str | None
    yg_project_id: str | None
    yg_project_title: str | None
    yg_board_id: str | None
    yg_board_title: str | None
    yg_first_column_id: str | None
    yg_completed_column_id: str | None
    webhook_registered: bool


def get_settings(company: Company) -> CompanyYougileSettings:
    return CompanyYougileSettings(
        enabled=bool((company.settings or {}).get("uses_yougile")),
        yg_company_id=company.yg_company_id,
        yg_company_name=company.yg_company_name,
        yg_project_id=company.yg_project_id,
        yg_project_title=company.yg_project_title,
        yg_board_id=company.yg_board_id,
        yg_board_title=company.yg_board_title,
        yg_first_column_id=company.yg_first_column_id,
        yg_completed_column_id=company.yg_completed_column_id,
        webhook_registered=bool(company.yg_webhook_id),
    )


# ── каталоги (для админ-визарда) ──────────────────────────────────────────

def list_projects(actor: User) -> list[dict]:
    client = build_client_for_user(actor)
    if client is None:
        raise YougileCompanyError("NOT_CONNECTED",
                                  "Сначала подключите свой YouGile-аккаунт")
    try:
        items = client.list_projects()
    except YougileAuthError:
        raise YougileCompanyError("BAD_KEY", "Ключ YouGile недействителен, переподключите аккаунт")
    except YougileError as e:
        raise YougileCompanyError("YOUGILE_ERROR", f"YouGile: {e}")
    return [{"id": i.get("id"), "title": i.get("title")} for i in items if i.get("id")]


def list_boards(actor: User, project_id: str) -> list[dict]:
    client = build_client_for_user(actor)
    if client is None:
        raise YougileCompanyError("NOT_CONNECTED", "Сначала подключите свой YouGile-аккаунт")
    try:
        items = client.list_boards(project_id=project_id)
    except YougileAuthError:
        raise YougileCompanyError("BAD_KEY", "Ключ YouGile недействителен, переподключите аккаунт")
    except YougileError as e:
        raise YougileCompanyError("YOUGILE_ERROR", f"YouGile: {e}")
    return [{"id": i.get("id"), "title": i.get("title"),
             "projectId": i.get("projectId")} for i in items if i.get("id")]


def list_columns(actor: User, board_id: str) -> list[dict]:
    client = build_client_for_user(actor)
    if client is None:
        raise YougileCompanyError("NOT_CONNECTED", "Сначала подключите свой YouGile-аккаунт")
    try:
        items = client.list_columns(board_id=board_id)
    except YougileAuthError:
        raise YougileCompanyError("BAD_KEY", "Ключ YouGile недействителен, переподключите аккаунт")
    except YougileError as e:
        raise YougileCompanyError("YOUGILE_ERROR", f"YouGile: {e}")
    return [{"id": i.get("id"), "title": i.get("title"),
             "boardId": i.get("boardId")} for i in items if i.get("id")]


def _resolve_first_column(actor: User, board_id: str) -> tuple[str | None, list[dict]]:
    """Вернуть id первой колонки доски + сам список (для UI completed-селекта).

    «Первая» — это та, которая идёт первой в ответе `/columns?boardId`. У YG
    нет явного поля `order`, но в практике колонки возвращаются в порядке
    создания/отображения. Для нашего сценария (одна доска, новые задачи в
    левой колонке) — годится.
    """
    cols = list_columns(actor, board_id)
    return (cols[0]["id"] if cols else None), cols


# ── сохранение настроек ───────────────────────────────────────────────────

def update_settings(actor: User, company: Company, payload: dict) -> CompanyYougileSettings:
    """Принимает уже валидированный payload (YougileCompanySettingsUpdateSchema).

    При смене доски автоматически перерезолвит `yg_first_column_id`. При
    очистке `yg_company_id=None` — сбрасывает связанные поля (проект, доска,
    колонки), потому что они становятся бессмысленными.
    """
    changed_board = ("yg_board_id" in payload
                     and payload["yg_board_id"] != company.yg_board_id)
    cleared_company = ("yg_company_id" in payload
                       and not payload["yg_company_id"])

    if "yg_company_id" in payload:
        company.yg_company_id = payload["yg_company_id"] or None
    if "yg_company_name" in payload:
        company.yg_company_name = payload["yg_company_name"] or None
    if "yg_project_id" in payload:
        company.yg_project_id = payload["yg_project_id"] or None
    if "yg_project_title" in payload:
        company.yg_project_title = payload["yg_project_title"] or None
    if "yg_board_id" in payload:
        company.yg_board_id = payload["yg_board_id"] or None
    if "yg_board_title" in payload:
        company.yg_board_title = payload["yg_board_title"] or None
    if "yg_completed_column_id" in payload:
        company.yg_completed_column_id = payload["yg_completed_column_id"] or None

    if cleared_company:
        company.yg_project_id = None
        company.yg_project_title = None
        company.yg_board_id = None
        company.yg_board_title = None
        company.yg_first_column_id = None
        company.yg_completed_column_id = None

    # Резолв первой колонки происходит при выборе доски. Если board очищен —
    # сбрасываем; если изменён — перерезолвим, не дожидаясь отдельной кнопки.
    if changed_board:
        if company.yg_board_id:
            try:
                first_id, _ = _resolve_first_column(actor, company.yg_board_id)
                company.yg_first_column_id = first_id
            except YougileCompanyError as e:
                logger.warning("yougile.resolve_first_col_failed",
                               extra={"company_id": company.id, "err": e.message})
                company.yg_first_column_id = None
        else:
            company.yg_first_column_id = None

    # Флаг включения в settings JSONB (используется UI как «фичу видно»).
    enabled_changed_to = None
    if "enabled" in payload:
        s = dict(company.settings or {})
        new_val = bool(payload["enabled"])
        if s.get("uses_yougile") != new_val:
            enabled_changed_to = new_val
        s["uses_yougile"] = new_val
        company.settings = s

    db.session.commit()

    # Webhook'и регистрируем/снимаем синхронно с переключением флага. Это
    # ОК для нашего объёма (одно действие в минуту максимум); сетевые ошибки
    # YG логируем, но не блокируем сохранение настроек.
    if enabled_changed_to is True:
        try:
            wh_register(actor, company)
        except YougileWebhookError as e:
            logger.warning("yougile.webhook_register_failed",
                           extra={"company_id": company.id, "err": e.message})
    elif enabled_changed_to is False:
        try:
            wh_deregister(actor, company)
        except YougileWebhookError as e:
            logger.warning("yougile.webhook_deregister_failed_top",
                           extra={"company_id": company.id, "err": e.message})

    logger.info("yougile.company_settings_updated",
                extra={"company_id": company.id, "actor_id": actor.id})
    return get_settings(company)

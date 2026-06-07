"""Импорт/экспорт одиночной задачи между Groove Work и YouGile.

Этап 3. Никакой автосинхронизации — только ручные действия пользователя
(«создать из YouGile», «создать в YouGile», «отвязать»). Полноценные webhook'и
и фоновая синхронизация — этап 4.

Обе операции пишут системные сообщения:

  - В чат GW-задачи (comments) — короткая запись от лица инициатора:
    «🔗 Связано с YouGile: <yg_url>».
  - В чат YG-карточки — `«🔗 Карточка в Groove Work: <gw_url>».

Антицикл (для будущего webhook'а): после каждой исходящей правки/создания
пишем `yougile_sync_hash` от полей, которые сами отправили; входящие
webhook'и сравнят и проигнорят свой же эхо-апдейт.
"""
from __future__ import annotations

import hashlib
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any

from app.extensions import db
from app.integrations.yougile.account_service import build_client_for_user
from app.integrations.yougile.client import (
    YougileAuthError, YougileError,
)
from app.integrations.yougile.parser import parse_task_url
from app.models.company import Company
from app.models.task import Task
from app.models.user import User
from app.models.user_yougile_account import UserYougileAccount
from app.repositories import comment_repo, task_repo
from app.services import task_service
from app.utils.logger import get_logger

logger = get_logger(__name__)


class YougileTaskError(RuntimeError):
    def __init__(self, code: str, message: str, http_status: int = 400):
        super().__init__(message)
        self.code = code
        self.message = message
        self.http_status = http_status


# ── общие хелперы ─────────────────────────────────────────────────────────

def _require_company_enabled(company: Company) -> None:
    """Проверить, что admin включил YG в компании. Иначе создавать/импортить
    карточки нельзя."""
    s = company.settings or {}
    if not s.get("uses_yougile"):
        raise YougileTaskError("COMPANY_DISABLED",
                               "YouGile-интеграция выключена в настройках компании")
    if not (company.yg_company_id and company.yg_project_id
            and company.yg_board_id and company.yg_first_column_id):
        raise YougileTaskError("COMPANY_NOT_CONFIGURED",
                               "В настройках компании не выбраны проект/доска")


def _require_user_connected(user: User):
    client = build_client_for_user(user)
    if client is None:
        raise YougileTaskError("USER_NOT_CONNECTED",
                               "Подключите свой YouGile-аккаунт в настройках",
                               http_status=412)
    return client


def _yg_task_url(yg_company_id: str, yg_task_id: str) -> str:
    """Канонический URL карточки в YouGile.

    Формат `team/<companyId>/#tasks?task=<taskId>` — у YG несколько вариантов,
    но этот точно работает и совпадает с тем, который сам YG показывает при
    открытой карточке.
    """
    return f"https://ru.yougile.com/team/{yg_company_id}/#tasks?task={yg_task_id}"


def _gw_task_url(task: Task, *, origin: str | None = None) -> str:
    """Canonical-ссылка на задачу в GW. origin берётся из request.url_root,
    но передаётся параметром, чтобы сервис не зависел от Flask-контекста."""
    base = (origin or "").rstrip("/")
    return f"{base}/tasks/{task.id}" if base else f"/tasks/{task.id}"


def _ts_to_dt(ms: int | None) -> datetime | None:
    if not ms:
        return None
    try:
        return datetime.fromtimestamp(int(ms) / 1000.0, tz=timezone.utc)
    except (TypeError, ValueError, OSError):
        return None


def _dt_to_ms(dt: datetime | None) -> int | None:
    if dt is None:
        return None
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return int(dt.timestamp() * 1000)


def _sync_hash(*, title: str | None, description: str | None,
               deadline_ms: int | None, completed: bool) -> str:
    """Хеш «состояния, которое мы только что отправили в YG».

    Если webhook вернёт payload с тем же хешем — игнор. Поля выбраны такие,
    которыми реально обмениваемся (тайм-трекинг/чек-листы синкать не будем).
    """
    parts = "|".join([
        (title or "").strip(),
        (description or "").strip(),
        str(deadline_ms or ""),
        "1" if completed else "0",
    ])
    return hashlib.sha1(parts.encode("utf-8")).hexdigest()


def _post_system_comment(task: Task, author: User, text: str) -> None:
    """Системное сообщение в чат GW-задачи.

    Пишем от лица инициатора (не вводим отдельного «system»-юзера, чтобы
    избежать миграций). Префикс 🔗 помогает фронту понять «это служебное».
    """
    try:
        comment_repo.create(task_id=task.id, author_id=author.id, text=text)
        db.session.commit()
    except Exception as e:  # noqa: BLE001
        logger.warning("yougile.system_comment_failed",
                       extra={"task_id": task.id, "err": str(e)})
        db.session.rollback()


def _post_yg_link_back(client, yg_task_id: str, gw_url: str) -> None:
    """Системное сообщение в чат YG-карточки: «Карточка в GW: <url>».

    Чат задачи в YG: id чата == id задачи. text+textHtml+label обязательны
    (см. CreateChatMessageDto).
    """
    try:
        client.post_chat_message(yg_task_id, {
            "text": f"🔗 Карточка в Groove Work: {gw_url}",
            "textHtml": (f"<p>🔗 Карточка в Groove Work: "
                         f"<a href=\"{gw_url}\">{gw_url}</a></p>"),
            "label": "Groove Work",
        })
    except YougileError as e:
        logger.warning("yougile.post_link_back_failed",
                       extra={"yg_task_id": yg_task_id, "err": str(e)})


# ── импорт из YouGile ─────────────────────────────────────────────────────

@dataclass
class ImportPayload:
    url: str
    department_id: int
    responsible_user_id: int | None = None
    stage_id: int | None = None
    pull_deadline: bool = True


def import_from_url(user: User, payload: ImportPayload, *,
                    origin: str | None = None) -> Task:
    company: Company | None = user.company
    if company is None:
        raise YougileTaskError("NO_COMPANY", "Пользователь без компании")
    _require_company_enabled(company)
    client = _require_user_connected(user)

    parsed = parse_task_url(payload.url)
    if parsed is None:
        raise YougileTaskError("BAD_URL",
                               "Не удалось разобрать ссылку на YouGile-карточку")

    # Проверим, что карточка действительно в той же компании YG.
    # parser достаёт company_id только из team-URL — на ?task= формат не
    # обязан попасть, поэтому если parsed.company_id is None — просто
    # верим тому, что вернёт API; собственно `client` уже привязан к нужной
    # компании ключом.
    if parsed.company_id and parsed.company_id != company.yg_company_id:
        raise YougileTaskError(
            "FOREIGN_COMPANY",
            "Эта карточка из другой компании YouGile",
        )

    # Уже привязана? Тогда возвращаем существующую GW-задачу.
    existing = Task.query.filter_by(
        company_id=company.id,
        yougile_task_id=parsed.task_id,
    ).first()
    if existing is not None:
        return existing

    try:
        yg = client.get_task(parsed.task_id)
    except YougileAuthError:
        raise YougileTaskError("BAD_KEY", "Ключ YouGile недействителен, переподключите аккаунт")
    except YougileError as e:
        raise YougileTaskError("YOUGILE_ERROR", f"YouGile: {e}")

    title = (yg.get("title") or "").strip() or "Без названия"
    yg_deadline = None
    if payload.pull_deadline:
        dl = yg.get("deadline") or {}
        yg_deadline = _ts_to_dt(dl.get("deadline"))

    # Сначала создаём GW-задачу.
    task = task_service.create_task(
        name=title,
        author_id=user.id,
        department_id=payload.department_id,
        company_id=company.id,
        link_yougile=_yg_task_url(company.yg_company_id, parsed.task_id),
        deadline=yg_deadline,
        responsible_user_id=payload.responsible_user_id,
        stage_id=payload.stage_id,
    )

    # И заполняем структурные YG-поля + sync_hash.
    task_repo.update(
        task,
        yougile_task_id=parsed.task_id,
        yougile_column_id=yg.get("columnId"),
        # project/board id напрямую YG не отдаёт в /tasks/{id}; кешируем то,
        # что задано в компании (обычно карточка живёт в нашей же доске).
        yougile_project_id=company.yg_project_id,
        yougile_board_id=company.yg_board_id,
        yougile_synced_at=datetime.now(timezone.utc),
        yougile_sync_hash=_sync_hash(
            title=title, description=yg.get("description"),
            deadline_ms=_dt_to_ms(yg_deadline),
            completed=bool(yg.get("completed")),
        ),
    )
    db.session.commit()

    # Системные сообщения: в YG-карточке — ссылка на GW; в GW-чате —
    # ссылка на YG.
    _post_yg_link_back(client, parsed.task_id, _gw_task_url(task, origin=origin))
    _post_system_comment(
        task, user,
        f"🔗 Связано с YouGile: {task.link_yougile}",
    )

    logger.info("yougile.imported",
                extra={"task_id": task.id, "yougile_task_id": parsed.task_id})
    return task


# ── экспорт в YouGile ─────────────────────────────────────────────────────

def export_to_yougile(user: User, gw_task_id: int, *,
                      origin: str | None = None) -> Task:
    company: Company | None = user.company
    if company is None:
        raise YougileTaskError("NO_COMPANY", "Пользователь без компании")
    _require_company_enabled(company)
    client = _require_user_connected(user)

    task = task_repo.get_by_id(gw_task_id)
    if task is None or task.company_id != company.id:
        raise YougileTaskError("NOT_FOUND", "Задача не найдена", http_status=404)
    if task.yougile_task_id:
        raise YougileTaskError("ALREADY_LINKED", "Задача уже связана с YouGile")

    # Берём yg_user_id из аккаунта — это «себя в YG», нам нужно для assigned.
    acc: UserYougileAccount | None = UserYougileAccount.query.filter_by(user_id=user.id).first()
    assigned = [acc.yg_user_id] if (acc and acc.yg_user_id) else []

    body: dict[str, Any] = {
        "title": task.name,
        "columnId": company.yg_first_column_id,
        "assigned": assigned,
    }
    if task.deadline:
        body["deadline"] = {
            "deadline": _dt_to_ms(task.deadline),
            "startDate": _dt_to_ms(task.created_at),
            "withTime": False,
        }

    try:
        yg = client.create_task(body)
    except YougileAuthError:
        raise YougileTaskError("BAD_KEY", "Ключ YouGile недействителен, переподключите аккаунт")
    except YougileError as e:
        raise YougileTaskError("YOUGILE_ERROR", f"YouGile: {e}")

    yg_task_id = yg.get("id")
    if not yg_task_id:
        raise YougileTaskError("YOUGILE_ERROR", "YouGile не вернул id новой карточки")

    yg_url = _yg_task_url(company.yg_company_id, yg_task_id)
    task_repo.update(
        task,
        link_yougile=yg_url,
        yougile_task_id=yg_task_id,
        yougile_project_id=company.yg_project_id,
        yougile_board_id=company.yg_board_id,
        yougile_column_id=company.yg_first_column_id,
        yougile_synced_at=datetime.now(timezone.utc),
        yougile_sync_hash=_sync_hash(
            title=task.name, description=None,
            deadline_ms=_dt_to_ms(task.deadline),
            completed=False,
        ),
    )
    db.session.commit()

    _post_yg_link_back(client, yg_task_id, _gw_task_url(task, origin=origin))
    _post_system_comment(
        task, user, f"🔗 Карточка создана в YouGile: {yg_url}",
    )
    logger.info("yougile.exported",
                extra={"task_id": task.id, "yougile_task_id": yg_task_id})
    return task


# ── отвязка ───────────────────────────────────────────────────────────────

def unlink_task(user: User, gw_task_id: int) -> Task:
    task = task_repo.get_by_id(gw_task_id)
    if task is None or (user.company_id and task.company_id != user.company_id):
        raise YougileTaskError("NOT_FOUND", "Задача не найдена", http_status=404)
    if not task.yougile_task_id:
        return task  # уже отвязана — идемпотентно

    yg_url = task.link_yougile
    task_repo.update(
        task,
        link_yougile=None,
        yougile_task_id=None,
        yougile_project_id=None,
        yougile_board_id=None,
        yougile_column_id=None,
        yougile_synced_at=None,
        yougile_sync_hash=None,
    )
    db.session.commit()

    _post_system_comment(
        task, user,
        f"🔗 Связь с YouGile разорвана (была: {yg_url})" if yg_url
        else "🔗 Связь с YouGile разорвана",
    )
    logger.info("yougile.unlinked", extra={"task_id": task.id})
    return task

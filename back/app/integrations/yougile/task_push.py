"""Исходящий push изменений GW-задачи в YouGile.

Вызывается из `api/tasks.py` post-action: после успешного update / archive /
restore. Внутри — best-effort: если интеграция не включена, юзер не
подключён или YG лежит — молча логируем и идём дальше; пользовательский
запрос на GW-задачу НЕ должен ломаться из-за внешнего сервиса.

Антицикл: пишем `yougile_sync_hash` от состояния, которое только что
отправили — webhook отбросит своё же эхо.
"""
from __future__ import annotations

from datetime import datetime, timezone

from app.extensions import db
from app.integrations.yougile.account_service import build_client_for_user
from app.integrations.yougile.client import YougileError
from app.integrations.yougile.task_service import _dt_to_ms, _sync_hash
from app.models.task import Task
from app.models.user import User
from app.repositories import task_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)


def _maybe_client(user: User):
    """Тихо вернуть клиент или None — этот модуль вызывается «попутно», без
    падений в пользовательских action'ах."""
    try:
        return build_client_for_user(user)
    except Exception as e:  # noqa: BLE001
        logger.warning("yougile.push_client_unavailable",
                       extra={"user_id": user.id, "err": str(e)})
        return None


def push_after_update(actor: User, task: Task, changed_fields: set[str]) -> None:
    """Отправить в YG только то, что реально поменялось у нас и интересует YG.

    Маппинг GW→YG:
      name      → title
      deadline  → deadline.deadline (ms)
    Остальные поля (department/responsible/stage) — внутренние, в YG не
    шлём.
    """
    if not task.yougile_task_id:
        return
    if not (changed_fields & {"name", "deadline"}):
        return

    client = _maybe_client(actor)
    if client is None:
        return

    body: dict = {}
    if "name" in changed_fields:
        body["title"] = task.name
    if "deadline" in changed_fields:
        body["deadline"] = (
            {"deadline": _dt_to_ms(task.deadline), "startDate": None, "withTime": False}
            if task.deadline else None
        )

    new_hash = _sync_hash(
        title=task.name, description=None,
        deadline_ms=_dt_to_ms(task.deadline) if task.deadline else None,
        completed=bool(task.is_archived),
    )

    try:
        client.update_task(task.yougile_task_id, body)
    except YougileError as e:
        logger.warning("yougile.push_update_failed",
                       extra={"task_id": task.id, "err": str(e)})
        return

    task_repo.update(
        task,
        yougile_synced_at=datetime.now(timezone.utc),
        yougile_sync_hash=new_hash,
    )
    db.session.commit()
    logger.info("yougile.pushed_update",
                extra={"task_id": task.id, "fields": sorted(changed_fields)})


def push_after_archive(actor: User, task: Task, *, archived: bool) -> None:
    """Архивация/восстановление в GW → completed/columnId в YG.

    Если у компании задана `yg_completed_column_id` — двигаем карточку
    туда; иначе ставим `completed=true/false`.
    """
    if not task.yougile_task_id:
        return
    client = _maybe_client(actor)
    if client is None:
        return

    company = task.company
    body: dict = {}
    target_col: str | None = None

    if archived:
        if company and company.yg_completed_column_id:
            target_col = company.yg_completed_column_id
        else:
            body["completed"] = True
    else:
        # Восстанавливаем: completed=false. Если задача была перенесена в
        # «выполнено»-колонку — возвращаем её в первую колонку (это самое
        # ожидаемое поведение, пользователь только что нажал «Восстановить»).
        body["completed"] = False
        if company and company.yg_first_column_id \
                and task.yougile_column_id == company.yg_completed_column_id:
            target_col = company.yg_first_column_id

    if target_col:
        body["columnId"] = target_col

    if not body:
        return

    try:
        client.update_task(task.yougile_task_id, body)
    except YougileError as e:
        logger.warning("yougile.push_archive_failed",
                       extra={"task_id": task.id, "err": str(e)})
        return

    upd = {
        "yougile_synced_at": datetime.now(timezone.utc),
        "yougile_sync_hash": _sync_hash(
            title=task.name, description=None,
            deadline_ms=_dt_to_ms(task.deadline) if task.deadline else None,
            completed=archived,
        ),
    }
    if target_col:
        upd["yougile_column_id"] = target_col
    task_repo.update(task, **upd)
    db.session.commit()
    logger.info("yougile.pushed_archive",
                extra={"task_id": task.id, "archived": archived})

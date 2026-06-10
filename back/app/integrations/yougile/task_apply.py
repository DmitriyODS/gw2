"""Применение событий webhook'а YouGile к нашим GW-задачам.

Один публичный вход — `apply_event(company, payload)`. Внутри маршрутизация
по `payload.event` и приведение полей.

Поддерживаемые события (см. CreateWebhookDto в `/api-json`):

  task-created   — игнор. Импорт «новых задач из YG» — отдельный сценарий,
                   автомассового создания мы НЕ хотим (см. план).
  task-updated   — поменялось одно из полей. Маппим diff.
  task-moved     — поменялся columnId. Кладём в yougile_column_id; если
                   columnId == yg_completed_column_id — архивируем у себя.
  task-deleted   — пользовательский выбор: разрыв связи + системный
                   комментарий + toast (через task:updated).
  task-restored  — отменяем archive, если делали по deleted.
  task-renamed   — `title` сменилось. То же что task-updated, но YG отдаёт
                   отдельным event'ом — нормализуем к одному и тому же
                   обработчику.

Антицикл. Перед апдейтом считаем тот же `sync_hash`, что в task_service:
если входящий хеш совпадает с сохранённым у задачи `yougile_sync_hash` —
значит, это наше эхо, пропускаем.
"""
from __future__ import annotations

from datetime import datetime, timezone
from typing import Any

from app.extensions import db, socketio
from app.integrations.yougile.task_service import (
    _dt_to_ms, _sync_hash, _ts_to_dt, _post_system_comment,
)
from app.models.company import Company
from app.models.task import Task
from app.models.user import User
from app.repositories import task_repo, user_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)


# ── маршрутизация ─────────────────────────────────────────────────────────

def apply_event(company: Company, payload: dict[str, Any]) -> dict[str, Any]:
    """Главная точка входа для webhook-ингресса.

    Возвращает `{status, action}` — фронту не показываем, только в логи.
    """
    event = (payload.get("event") or "").lower().strip()
    data = payload.get("data") or {}
    yg_task_id = data.get("id") or payload.get("id")

    if not yg_task_id:
        logger.warning("yougile.webhook_no_id", extra={"event": event})
        return {"status": "skipped", "reason": "no-id"}

    task: Task | None = Task.query.filter_by(
        company_id=company.id,
        yougile_task_id=yg_task_id,
    ).first()

    # Карточка, которую мы не знаем (не связана с нашей задачей). Игнорим —
    # импорт «новых из YG» делается вручную пользователем, не автоматом.
    if task is None:
        if event.startswith("task-created"):
            return {"status": "skipped", "reason": "unlinked-create"}
        return {"status": "skipped", "reason": "not-linked"}

    if event.startswith("task-deleted"):
        return _apply_deleted(task)
    if event.startswith("task-restored"):
        return _apply_restored(task)
    if event.startswith("task-moved") or event.startswith("task-renamed") \
            or event.startswith("task-updated") or event.startswith("task-completed"):
        return _apply_updated(company, task, data)

    return {"status": "skipped", "reason": f"event:{event}"}


# ── обработчики ───────────────────────────────────────────────────────────

def _apply_updated(company: Company, task: Task, data: dict[str, Any]) -> dict:
    """Применить изменения title/description/deadline/completed/columnId."""
    incoming_title = (data.get("title") or "").strip()
    incoming_deadline = None
    dl = data.get("deadline") or {}
    if isinstance(dl, dict):
        incoming_deadline = _ts_to_dt(dl.get("deadline"))
    incoming_completed = bool(data.get("completed"))

    incoming_hash = _sync_hash(
        title=incoming_title or task.name,
        deadline_ms=_dt_to_ms(incoming_deadline) if incoming_deadline else None,
        completed=incoming_completed,
    )
    # Антицикл: это наш собственный echo.
    if incoming_hash and incoming_hash == task.yougile_sync_hash:
        return {"status": "skipped", "reason": "self-echo"}

    fields: dict[str, Any] = {}
    if incoming_title and incoming_title != task.name:
        fields["name"] = incoming_title

    if incoming_deadline is not None and incoming_deadline != task.deadline:
        fields["deadline"] = incoming_deadline

    new_col = data.get("columnId")
    if new_col and new_col != task.yougile_column_id:
        fields["yougile_column_id"] = new_col

    new_id_short = data.get("idTaskProject") or data.get("idTaskCommon")
    if new_id_short and new_id_short != task.yougile_id_short:
        fields["yougile_id_short"] = new_id_short

    # Архивация по «выполнено» — два сигнала: completed=True, либо move в
    # yg_completed_column_id. Любого хватит.
    should_archive = incoming_completed or (
        company.yg_completed_column_id
        and new_col == company.yg_completed_column_id
    )
    if should_archive and not task.is_archived:
        # Инвариант GW: нельзя архивировать задачу с активным юнитом. Если по
        # ней сейчас кто-то работает — не архивируем (иначе юнит «повиснет» на
        # архивной задаче). Пользователь закроет задачу сам, когда остановит юнит.
        if task_repo.has_active_unit(task.id):
            logger.info("yougile.webhook_archive_skipped_active_unit",
                        extra={"task_id": task.id})
        else:
            fields["is_archived"] = True
            fields["archived_at"] = datetime.now(timezone.utc)
    elif not incoming_completed and task.is_archived and not company.yg_completed_column_id:
        # Если завершённость снимали в YG (а у нас не задана completed-колонка
        # для авто-архива), отменяем archive.
        fields["is_archived"] = False
        fields["archived_at"] = None

    if not fields and incoming_hash == task.yougile_sync_hash:
        return {"status": "no-changes"}

    fields["yougile_synced_at"] = datetime.now(timezone.utc)
    fields["yougile_sync_hash"] = incoming_hash
    task_repo.update(task, **fields)
    db.session.commit()

    # Закрытие, пришедшее из YouGile, — тоже опорная точка ленты «Мой Groove».
    if fields.get("is_archived"):
        from app.services.feed_service import on_task_closed
        on_task_closed(task)

    payload = _broadcast_task_update(task)
    logger.info("yougile.webhook_applied",
                extra={"task_id": task.id, "fields": list(fields.keys())})
    return {"status": "applied", "fields": list(fields.keys()), "payload": payload}


def _apply_deleted(task: Task) -> dict:
    """task-deleted → не удаляем у нас, а разрываем связь и оставляем
    системный комментарий. По договорённости с пользователем (см. ответ 3)."""
    if not task.yougile_task_id:
        return {"status": "skipped", "reason": "already-unlinked"}

    yg_url = task.link_yougile
    task_repo.update(
        task,
        link_yougile=None,
        yougile_task_id=None,
        yougile_id_short=None,
        yougile_project_id=None,
        yougile_board_id=None,
        yougile_column_id=None,
        yougile_synced_at=None,
        yougile_sync_hash=None,
    )
    db.session.commit()

    # Системный комментарий пишем от лица автора задачи (он наверняка
    # существует в company; запрос дёшев — id в кеше SA).
    author = user_repo.get_by_id(task.author_id) if task.author_id else None
    if author is not None:
        _post_system_comment(
            task, author,
            f"🔗 Карточка в YouGile удалена, связь разорвана"
            + (f" (была: {yg_url})" if yg_url else ""),
        )
    _broadcast_task_update(task)
    logger.info("yougile.webhook_deleted_unlinked", extra={"task_id": task.id})
    return {"status": "unlinked"}


def _apply_restored(task: Task) -> dict:
    """task-restored: если у нас архивная — снимаем архив. Связь не
    восстанавливаем (это отдельный пользовательский экшен)."""
    if not task.is_archived:
        return {"status": "no-changes"}
    task_repo.update(task, is_archived=False, archived_at=None)
    db.session.commit()
    _broadcast_task_update(task)
    return {"status": "restored"}


# ── broadcast ────────────────────────────────────────────────────────────

def _broadcast_task_update(task: Task) -> dict:
    """Шлём task:updated в комнату all. enrich_task требует current_user_id
    (для is_favorite/color); webhook'и идут без пользователя, поэтому
    передаём 0 — фронт получит is_favorite=false/color=null и подмёрджит
    локально, если у него были индивидуальные данные."""
    from app.api.tasks import _enrich_task  # noqa: WPS433
    payload = _enrich_task(task, current_user_id=0)
    socketio.emit("task:updated", payload, room="all")
    return payload

"""Исходящий push изменений GW-задачи в YouGile.

Вызывается из `api/tasks.py` post-action: после успешного update / archive /
restore. Внутри — best-effort: если интеграция не включена, юзер не
подключён или YG лежит — молча логируем и идём дальше; пользовательский
запрос на GW-задачу НЕ должен ломаться из-за внешнего сервиса.

Асинхронность. Сам push выполняется в фоновом greenlet'е
(`socketio.start_background_task`), а не в обработчике запроса: HTTP к YouGile
может занять до таймаута × ретраи, и заставлять пользователя ждать ответа YG
на каждую правку связанной задачи недопустимо. Запрос отвечает сразу, push
догоняет в фоне. ORM-объекты в greenlet не передаём (DetachedInstanceError) —
перечитываем задачу/пользователя по id внутри app-контекста.

Антицикл: пишем `yougile_sync_hash` от состояния, которое только что
отправили — webhook отбросит своё же эхо.
"""
from __future__ import annotations

from datetime import datetime, timezone

from app.extensions import db, socketio
from app.integrations.yougile.account_service import build_client_for_user
from app.integrations.yougile.client import YougileError
from app.integrations.yougile.task_service import _dt_to_ms, _sync_hash
from app.models.task import Task
from app.models.user import User
from app.repositories import task_repo, user_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)

_PUSHABLE_FIELDS = frozenset({"name", "deadline"})


# ── фоновый диспетчер ──────────────────────────────────────────────────────

def _dispatch(fn, *args) -> None:
    """Запустить fn(*args) в фоновом greenlet'е с app-контекстом.

    Если app-контекст недоступен (например, в юнит-тестах вне запроса) —
    выполняем синхронно, чтобы не терять поведение.
    """
    try:
        from flask import current_app
        app = current_app._get_current_object()
    except RuntimeError:
        app = None

    if app is None:
        fn(*args)
        return

    def _job():
        with app.app_context():
            try:
                fn(*args)
            except Exception as e:  # noqa: BLE001
                logger.warning("yougile.push_job_failed",
                               extra={"fn": getattr(fn, "__name__", "?"), "err": str(e)})

    socketio.start_background_task(_job)


def _client_for(actor_id: int):
    user = user_repo.get_by_id(actor_id)
    if user is None:
        return None
    try:
        return build_client_for_user(user)
    except Exception as e:  # noqa: BLE001
        logger.warning("yougile.push_client_unavailable",
                       extra={"user_id": actor_id, "err": str(e)})
        return None


# ── update (name/deadline) ─────────────────────────────────────────────────

def push_after_update(actor: User, task: Task, changed_fields: set[str]) -> None:
    """Отправить в YG только то, что реально поменялось у нас и интересует YG.

    Маппинг GW→YG: name → title, deadline → deadline.deadline (ms). Остальное
    (department/responsible/stage) — внутреннее, в YG не шлём. Реальная работа
    — в фоне (см. модульный docstring).
    """
    if not task.yougile_task_id:
        return
    relevant = _PUSHABLE_FIELDS & set(changed_fields)
    if not relevant:
        return
    _dispatch(_run_after_update, actor.id, task.id, frozenset(relevant))


def _run_after_update(actor_id: int, task_id: int, changed_fields: frozenset) -> None:
    task = task_repo.get_by_id(task_id)
    if task is None or not task.yougile_task_id:
        return
    client = _client_for(actor_id)
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
    if not body:
        return

    new_hash = _sync_hash(
        title=task.name,
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


# ── archive / restore ──────────────────────────────────────────────────────

def push_after_archive(actor: User, task: Task, *, archived: bool) -> None:
    """Архивация/восстановление в GW → completed/columnId в YG (в фоне)."""
    if not task.yougile_task_id:
        return
    _dispatch(_run_after_archive, actor.id, task.id, archived)


def _run_after_archive(actor_id: int, task_id: int, archived: bool) -> None:
    task = task_repo.get_by_id(task_id)
    if task is None or not task.yougile_task_id:
        return
    client = _client_for(actor_id)
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
            title=task.name,
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

"""Служебные хуки generic-моста для YouGile-пуша.

Домен задач живёт в tasksvc; исходящая синхра в YouGile (личные ключи
пользователей, антицикл sync_hash) — пока во Flask. tasksvc после
обновления/архивирования задачи публикует в gw2:tasks:events служебные
события `_yg_task_updated` / `_yg_task_archived` (мост наружу их не эмитит,
а диспатчит сюда). Best-effort, как прежние вызовы push_after_* из
api/tasks.py: любая ошибка — только в лог. Уйдёт вместе с интеграцией
в фазе 4 (push переедет внутрь tasksvc).
"""
from app.repositories import task_repo, user_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)


def handle_task_updated(app, socketio, payload: dict) -> None:
    try:
        with app.app_context():
            task = task_repo.get_by_id(payload.get("task_id") or 0)
            actor = user_repo.get_by_id(payload.get("actor_user_id") or 0)
            if task is None or actor is None:
                return
            from app.integrations.yougile.task_push import push_after_update
            push_after_update(actor, task, set(payload.get("changed") or []))
    except Exception:  # noqa: BLE001
        logger.exception("yougile.bridge_hook.update_failed")


def handle_task_archived(app, socketio, payload: dict) -> None:
    try:
        with app.app_context():
            task = task_repo.get_by_id(payload.get("task_id") or 0)
            actor = user_repo.get_by_id(payload.get("actor_user_id") or 0)
            if task is None or actor is None:
                return
            from app.integrations.yougile.task_push import push_after_archive
            push_after_archive(actor, task, archived=bool(payload.get("archived")))
    except Exception:  # noqa: BLE001
        logger.exception("yougile.bridge_hook.archive_failed")

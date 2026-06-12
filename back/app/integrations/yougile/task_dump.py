"""Дамп задачи для ответов /api/yougile и сокет-событий task:*.

Домен задач живёт в tasksvc (back-go/tasks); во Flask остался только
YouGile (до фазы 4), которому нужен полный dump задачи в том же виде, что
отдаёт /api/tasks (is_favorite, has_units, active_users, личный цвет).
Это перенос _enrich_task/_yougile_enabled из прежнего api/tasks.py.
"""
from app.repositories import task_repo, company_repo
from app.schemas.task import TaskSchema

_task_schema = TaskSchema()


def yougile_enabled(company_id) -> bool:
    if company_id is None:
        return True
    company = company_repo.get_by_id(company_id)
    if company is None or not company.settings:
        return True
    return bool(company.settings.get("uses_yougile", True))


def enrich_task(task, current_user_id: int) -> dict:
    data = _task_schema.dump(task)
    data["is_favorite"] = task_repo.is_favorite(task.id, current_user_id)
    data["has_units"] = task_repo.has_any_units(task.id)
    data["active_users"] = task_repo.get_active_users(task.id)
    # Цвет — индивидуальный для каждого пользователя (user_task_colors).
    data["color"] = task_repo.get_user_color(task.id, current_user_id)
    if not yougile_enabled(task.company_id):
        data["link_yougile"] = None
    return data

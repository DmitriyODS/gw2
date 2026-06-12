from datetime import datetime
from app.extensions import db
from app.repositories import task_repo, department_repo, user_repo, stage_repo
from app.services.ai_client import schedule_reindex
from app.utils.logger import get_logger

logger = get_logger(__name__)

# Домен задач живёт в tasksvc (back-go/tasks); во Flask осталось только
# создание задачи для YouGile-импорта (integrations/yougile/task_service.py)
# с прежними бизнес-проверками. Уйдёт вместе с интеграцией в фазе 4.


class TaskServiceError(Exception):
    def __init__(self, message: str, code: str = "TASK_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def _validate_responsible(user_id, company_id):
    if user_id is None:
        return
    user = user_repo.get_by_id(user_id)
    if user is None:
        raise TaskServiceError("Сотрудник не найден", "USER_NOT_FOUND", 404)
    if user.is_hidden:
        raise TaskServiceError("Сотрудник не найден", "USER_NOT_FOUND", 404)
    if user.company_id is not None and user.company_id != company_id:
        raise TaskServiceError("Сотрудник из другой компании", "USER_FOREIGN", 422)


def _validate_stage(stage_id, company_id):
    if stage_id is None:
        return
    stage = stage_repo.get_by_id(stage_id)
    if stage is None:
        raise TaskServiceError("Этап не найден", "STAGE_NOT_FOUND", 404)
    if stage.company_id != company_id:
        raise TaskServiceError("Этап принадлежит другой компании", "STAGE_FOREIGN", 422)


def create_task(
    name: str,
    author_id: int,
    department_id: int,
    company_id: int,
    received_at: datetime = None,
    link_yougile: str = None,
    deadline: datetime = None,
    responsible_user_id: int = None,
    stage_id: int = None,
) -> object:
    dept = department_repo.get_by_id(department_id)
    if dept is None:
        raise TaskServiceError("Отдел не найден", "DEPT_NOT_FOUND", 404)
    if dept.company_id != company_id:
        raise TaskServiceError("Отдел принадлежит другой компании", "DEPT_FOREIGN", 422)

    # По умолчанию ответственный = автор задачи.
    if responsible_user_id is None:
        responsible_user_id = author_id
    _validate_responsible(responsible_user_id, company_id)
    _validate_stage(stage_id, company_id)

    task = task_repo.create(
        name=name,
        author_id=author_id,
        department_id=department_id,
        company_id=company_id,
        received_at=received_at,
        link_yougile=link_yougile,
        deadline=deadline,
        responsible_user_id=responsible_user_id,
        stage_id=stage_id,
    )
    db.session.commit()
    logger.info("task.create", extra={"extra": {"task_id": task.id, "author_id": author_id, "event": "task.create"}})
    schedule_reindex(task.id)
    return task

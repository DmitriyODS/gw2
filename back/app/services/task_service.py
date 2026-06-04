from datetime import datetime, timezone
from app.extensions import db
from app.repositories import task_repo, department_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)


class TaskServiceError(Exception):
    def __init__(self, message: str, code: str = "TASK_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def create_task(
    name: str,
    author_id: int,
    department_id: int,
    company_id: int,
    received_at: datetime = None,
    link_yougile: str = None,
    deadline: datetime = None,
) -> object:
    dept = department_repo.get_by_id(department_id)
    if dept is None:
        raise TaskServiceError("Отдел не найден", "DEPT_NOT_FOUND", 404)
    if dept.company_id != company_id:
        raise TaskServiceError("Отдел принадлежит другой компании", "DEPT_FOREIGN", 422)

    task = task_repo.create(
        name=name,
        author_id=author_id,
        department_id=department_id,
        company_id=company_id,
        received_at=received_at,
        link_yougile=link_yougile,
        deadline=deadline,
    )
    db.session.commit()
    logger.info("task.create", extra={"extra": {"task_id": task.id, "author_id": author_id, "event": "task.create"}})
    return task


def update_task(task_id: int, **kwargs) -> object:
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    if "department_id" in kwargs:
        dept = department_repo.get_by_id(kwargs["department_id"])
        if dept is None:
            raise TaskServiceError("Отдел не найден", "DEPT_NOT_FOUND", 404)
        if dept.company_id != task.company_id:
            raise TaskServiceError("Отдел принадлежит другой компании", "DEPT_FOREIGN", 422)

    task_repo.update(task, **kwargs)
    db.session.commit()
    return task


def delete_task(task_id: int) -> None:
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    task_repo.delete(task)
    db.session.commit()
    logger.info("task.delete", extra={"extra": {"task_id": task_id, "event": "task.delete"}})


def archive_task(task_id: int) -> object:
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    if task.is_archived:
        raise TaskServiceError("Задача уже архивирована", "ALREADY_ARCHIVED", 422)

    if task_repo.has_active_unit(task_id):
        raise TaskServiceError(
            "Нельзя архивировать задачу с активным юнитом",
            "HAS_ACTIVE_UNIT", 422
        )

    now = datetime.now(timezone.utc)
    task_repo.update(task, is_archived=True, archived_at=now)
    db.session.commit()
    logger.info("task.archive", extra={"extra": {"task_id": task_id, "event": "task.archive"}})
    return task


def restore_task(task_id: int) -> object:
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    if not task.is_archived:
        raise TaskServiceError("Задача не архивирована", "NOT_ARCHIVED", 422)

    task_repo.update(task, is_archived=False, archived_at=None)
    db.session.commit()
    logger.info("task.restore", extra={"extra": {"task_id": task_id, "event": "task.restore"}})
    return task


def toggle_favorite(task_id: int, user_id: int) -> bool:
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    is_fav = task_repo.toggle_favorite(task_id, user_id)
    db.session.commit()
    return is_fav


def set_user_color(task_id: int, user_id: int, color):
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    task_repo.set_user_color(task_id, user_id, color)
    db.session.commit()

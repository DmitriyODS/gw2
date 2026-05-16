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
    received_at: datetime = None,
    link_yougile: str = None,
    deadline: datetime = None,
) -> object:
    dept = department_repo.get_by_id(department_id)
    if dept is None:
        raise TaskServiceError("Отдел не найден", "DEPT_NOT_FOUND", 404)

    task = task_repo.create(
        name=name,
        author_id=author_id,
        department_id=department_id,
        received_at=received_at,
        link_yougile=link_yougile,
        deadline=deadline,
    )
    db.session.commit()
    logger.info("task.create", extra={"extra": {"task_id": task.id, "author_id": author_id, "event": "task.create"}})
    return task


def update_task(task_id: int, current_user_id: int, current_user_access: int, **kwargs) -> object:
    from app.utils.permissions import has_permission, Section, Bit
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    is_own = task.author_id == current_user_id
    if is_own:
        if not has_permission(current_user_access, Section.TASKS, Bit.OWN_EDIT):
            raise TaskServiceError("Недостаточно прав", "FORBIDDEN", 403)
    else:
        if not has_permission(current_user_access, Section.TASKS, Bit.OTHER_EDIT):
            raise TaskServiceError("Недостаточно прав", "FORBIDDEN", 403)

    if "department_id" in kwargs:
        dept = department_repo.get_by_id(kwargs["department_id"])
        if dept is None:
            raise TaskServiceError("Отдел не найден", "DEPT_NOT_FOUND", 404)

    task_repo.update(task, **kwargs)
    db.session.commit()
    return task


def delete_task(task_id: int, current_user_id: int, current_user_access: int) -> None:
    from app.utils.permissions import has_permission, Section, Bit
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    is_own = task.author_id == current_user_id
    if is_own:
        if not has_permission(current_user_access, Section.TASKS, Bit.OWN_DELETE):
            raise TaskServiceError("Недостаточно прав", "FORBIDDEN", 403)
    else:
        if not has_permission(current_user_access, Section.TASKS, Bit.OTHER_DELETE):
            raise TaskServiceError("Недостаточно прав", "FORBIDDEN", 403)

    task_repo.delete(task)
    db.session.commit()
    logger.info("task.delete", extra={"extra": {"task_id": task_id, "user_id": current_user_id, "event": "task.delete"}})


def archive_task(task_id: int, current_user_id: int, current_user_access: int) -> object:
    from app.utils.permissions import has_permission, Section, Bit
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    if task.is_archived:
        raise TaskServiceError("Задача уже архивирована", "ALREADY_ARCHIVED", 422)

    is_own = task.author_id == current_user_id
    perm = Bit.OWN_EDIT if is_own else Bit.OTHER_EDIT
    if not has_permission(current_user_access, Section.TASKS, perm):
        raise TaskServiceError("Недостаточно прав", "FORBIDDEN", 403)

    if task_repo.has_active_unit(task_id):
        raise TaskServiceError(
            "Нельзя архивировать задачу с активным юнитом",
            "HAS_ACTIVE_UNIT", 422
        )

    now = datetime.now(timezone.utc)
    task_repo.update(task, is_archived=True, archived_at=now)
    db.session.commit()
    logger.info("task.archive", extra={"extra": {"task_id": task_id, "user_id": current_user_id, "event": "task.archive"}})
    return task


def restore_task(task_id: int, current_user_id: int, current_user_access: int) -> object:
    from app.utils.permissions import has_permission, Section, Bit
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    if not task.is_archived:
        raise TaskServiceError("Задача не архивирована", "NOT_ARCHIVED", 422)

    is_own = task.author_id == current_user_id
    perm = Bit.OWN_EDIT if is_own else Bit.OTHER_EDIT
    if not has_permission(current_user_access, Section.TASKS, perm):
        raise TaskServiceError("Недостаточно прав", "FORBIDDEN", 403)

    task_repo.update(task, is_archived=False, archived_at=None)
    db.session.commit()
    logger.info("task.restore", extra={"extra": {"task_id": task_id, "user_id": current_user_id, "event": "task.restore"}})
    return task


def toggle_favorite(task_id: int, user_id: int) -> bool:
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)

    is_fav = task_repo.toggle_favorite(task_id, user_id)
    db.session.commit()
    return is_fav

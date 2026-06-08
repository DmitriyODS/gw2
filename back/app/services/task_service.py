from datetime import datetime, timezone
from app.extensions import db
from app.repositories import task_repo, department_repo, user_repo, stage_repo
from app.services.task_embedding_service import schedule_reindex
from app.utils.logger import get_logger

logger = get_logger(__name__)


# Какие поля задачи влияют на текст эмбеддинга (см. `_build_text_for_task`).
# При изменении любого из них в update_task — перегенерим эмбеддинг.
_REINDEX_FIELDS = frozenset({"name", "department_id", "responsible_user_id"})


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

    if "responsible_user_id" in kwargs:
        _validate_responsible(kwargs["responsible_user_id"], task.company_id)

    if "stage_id" in kwargs:
        _validate_stage(kwargs["stage_id"], task.company_id)

    task_repo.update(task, **kwargs)
    db.session.commit()
    if _REINDEX_FIELDS.intersection(kwargs.keys()):
        schedule_reindex(task.id)
    return task


def set_responsible(task_id: int, responsible_user_id):
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)
    _validate_responsible(responsible_user_id, task.company_id)
    task_repo.update(task, responsible_user_id=responsible_user_id)
    db.session.commit()
    schedule_reindex(task.id)
    return task


def set_stage(task_id: int, stage_id):
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise TaskServiceError("Задача не найдена", "NOT_FOUND", 404)
    _validate_stage(stage_id, task.company_id)
    task_repo.update(task, stage_id=stage_id)
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

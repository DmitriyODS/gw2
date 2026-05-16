from app.extensions import db
from app.repositories import unit_repo, task_repo, unit_type_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)


class UnitServiceError(Exception):
    def __init__(self, message: str, code: str = "UNIT_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def create_unit(task_id: int, name: str, unit_type_id: int, user_id: int) -> object:
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise UnitServiceError("Задача не найдена", "TASK_NOT_FOUND", 404)

    if task.is_archived:
        raise UnitServiceError("Нельзя создать юнит для архивной задачи", "TASK_ARCHIVED", 422)

    unit_type = unit_type_repo.get_by_id(unit_type_id)
    if unit_type is None:
        raise UnitServiceError("Тип юнита не найден", "TYPE_NOT_FOUND", 404)

    active = unit_repo.get_active_for_user(user_id)
    if active is not None:
        raise UnitServiceError("У вас уже есть активный юнит", "ACTIVE_UNIT_EXISTS", 409)

    unit = unit_repo.create(name=name, user_id=user_id, unit_type_id=unit_type_id, task_id=task_id)
    db.session.commit()

    logger.info("unit.start", extra={"extra": {"unit_id": unit.id, "task_id": task_id, "user_id": user_id, "event": "unit.start"}})
    return unit


def update_unit(unit_id: int, current_user_id: int, current_user_access: int, **kwargs) -> object:
    from app.utils.permissions import has_permission, Section, Bit
    unit = unit_repo.get_by_id(unit_id)
    if unit is None:
        raise UnitServiceError("Юнит не найден", "NOT_FOUND", 404)

    is_own = unit.user_id == current_user_id
    if is_own:
        if not has_permission(current_user_access, Section.UNITS, Bit.OWN_EDIT):
            raise UnitServiceError("Недостаточно прав", "FORBIDDEN", 403)
    else:
        if not has_permission(current_user_access, Section.UNITS, Bit.OTHER_EDIT):
            raise UnitServiceError("Недостаточно прав", "FORBIDDEN", 403)

    if "unit_type_id" in kwargs:
        unit_type = unit_type_repo.get_by_id(kwargs["unit_type_id"])
        if unit_type is None:
            raise UnitServiceError("Тип юнита не найден", "TYPE_NOT_FOUND", 404)

    unit_repo.update(unit, **kwargs)
    db.session.commit()
    return unit


def stop_unit(unit_id: int, current_user_id: int, current_user_access: int) -> object:
    from app.utils.permissions import has_permission, Section, Bit
    unit = unit_repo.get_by_id(unit_id)
    if unit is None:
        raise UnitServiceError("Юнит не найден", "NOT_FOUND", 404)

    if unit.datetime_end is not None:
        raise UnitServiceError("Юнит уже завершён", "ALREADY_STOPPED", 422)

    is_own = unit.user_id == current_user_id
    if is_own:
        if not has_permission(current_user_access, Section.UNITS, Bit.OWN_EDIT):
            raise UnitServiceError("Недостаточно прав", "FORBIDDEN", 403)
    else:
        if not has_permission(current_user_access, Section.UNITS, Bit.OTHER_EDIT):
            raise UnitServiceError("Недостаточно прав", "FORBIDDEN", 403)

    unit_repo.stop(unit)
    db.session.commit()

    logger.info("unit.stop", extra={"extra": {"unit_id": unit_id, "user_id": current_user_id, "event": "unit.stop"}})
    return unit


def delete_unit(unit_id: int, current_user_id: int, current_user_access: int) -> None:
    from app.utils.permissions import has_permission, Section, Bit
    unit = unit_repo.get_by_id(unit_id)
    if unit is None:
        raise UnitServiceError("Юнит не найден", "NOT_FOUND", 404)

    is_own = unit.user_id == current_user_id
    if is_own:
        if not has_permission(current_user_access, Section.UNITS, Bit.OWN_DELETE):
            raise UnitServiceError("Недостаточно прав", "FORBIDDEN", 403)
    else:
        if not has_permission(current_user_access, Section.UNITS, Bit.OTHER_DELETE):
            raise UnitServiceError("Недостаточно прав", "FORBIDDEN", 403)

    task_id = unit.task_id
    unit_repo.delete(unit)
    db.session.commit()
    logger.info("unit.delete", extra={"extra": {"unit_id": unit_id, "task_id": task_id, "user_id": current_user_id, "event": "unit.delete"}})

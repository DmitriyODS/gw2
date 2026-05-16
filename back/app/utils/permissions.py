from functools import wraps
from flask import abort
from flask_jwt_extended import verify_jwt_in_request, get_jwt_identity, get_jwt


BIGINT_MAX = 9223372036854775807


class Section:
    TASKS = 0
    UNITS = 1
    USERS = 2
    ROLES = 3
    STATS = 4
    BACKUP = 5
    DEPARTMENTS = 6
    UNIT_TYPES = 7


class Bit:
    # Задачи / Юниты
    VIEW = 0
    OWN_CREATE = 1
    OWN_EDIT = 2
    OWN_DELETE = 3
    OTHER_CREATE = 4
    OTHER_EDIT = 5
    OTHER_DELETE = 6
    # Пользователи / Роли / Отделы / Типы юнитов
    CREATE = 1
    EDIT = 2
    DELETE = 3
    # Роли
    ASSIGN = 4
    # Статистика
    VIEW_USERS = 1
    EXPORT_COMMON = 2
    EXPORT_USERS = 3
    # Копирование
    EXPORT = 1
    IMPORT = 2


def has_permission(access: int, section: int, bit: int) -> bool:
    """Проверить наличие разрешения."""
    byte_val = (access >> (section * 8)) & 0xFF
    return bool(byte_val & (1 << bit))


def require_permission(section: int, bit: int):
    """Декоратор для Flask route — проверяет JWT и права, возвращает 403 при отсутствии."""
    def decorator(fn):
        @wraps(fn)
        def wrapper(*args, **kwargs):
            verify_jwt_in_request()

            claims = get_jwt()
            if claims.get("force_change") and fn.__name__ != "change_default":
                abort(403, description="FORCE_PASSWORD_CHANGE")

            from app.repositories.user_repo import get_user_by_id
            user_id = get_jwt_identity()
            user = get_user_by_id(user_id)
            if user is None or user.is_hidden:
                abort(401, description="Пользователь не найден")
            if not has_permission(user.role.access, section, bit):
                abort(403, description="Недостаточно прав")
            return fn(*args, **kwargs)
        return wrapper
    return decorator


def require_auth(fn):
    """Декоратор — только проверка JWT и force_change. Без проверки прав."""
    @wraps(fn)
    def wrapper(*args, **kwargs):
        verify_jwt_in_request()

        claims = get_jwt()
        if claims.get("force_change"):
            abort(403, description="FORCE_PASSWORD_CHANGE")

        from app.repositories.user_repo import get_user_by_id
        user_id = get_jwt_identity()
        user = get_user_by_id(user_id)
        if user is None or user.is_hidden:
            abort(401, description="Пользователь не найден")
        return fn(*args, **kwargs)
    return wrapper

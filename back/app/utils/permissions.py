from functools import wraps
from flask import abort
from flask_jwt_extended import verify_jwt_in_request, get_jwt_identity, get_jwt


EMPLOYEE = 1
MANAGER = 2
ADMIN = 3
SUPERADMIN = 4


def get_user_level(user) -> int:
    return user.role.level if user and user.role else 0


def require_role(min_level: int):
    """Декоратор для Flask route — проверяет JWT и уровень роли."""
    def decorator(fn):
        @wraps(fn)
        def wrapper(*args, **kwargs):
            verify_jwt_in_request()

            claims = get_jwt()
            if claims.get("force_change") and fn.__name__ != "change_default":
                abort(403, description="FORCE_PASSWORD_CHANGE")

            from app.repositories.user_repo import get_user_by_id
            user_id = int(get_jwt_identity())
            user = get_user_by_id(user_id)
            if user is None or user.is_hidden:
                abort(401, description="Пользователь не найден")
            if get_user_level(user) < min_level:
                abort(403, description="Недостаточно прав")
            return fn(*args, **kwargs)
        return wrapper
    return decorator


def require_auth(fn):
    """Декоратор — только проверка JWT и force_change. Без проверки уровня роли."""
    @wraps(fn)
    def wrapper(*args, **kwargs):
        verify_jwt_in_request()

        claims = get_jwt()
        if claims.get("force_change"):
            abort(403, description="FORCE_PASSWORD_CHANGE")

        from app.repositories.user_repo import get_user_by_id
        user_id = int(get_jwt_identity())
        user = get_user_by_id(user_id)
        if user is None or user.is_hidden:
            abort(401, description="Пользователь не найден")
        return fn(*args, **kwargs)
    return wrapper

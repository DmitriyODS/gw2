from functools import wraps
from flask import abort, g, request
from flask_jwt_extended import verify_jwt_in_request, get_jwt_identity, get_jwt


EMPLOYEE = 1
MANAGER = 2
DIRECTOR = 3
ADMIN = 4


def get_user_level(user) -> int:
    return user.role.level if user and user.role else 0


def _resolve_current_user_and_check_force_change(fn_name: str):
    """Общая преамбула декораторов: верифицировать JWT, проверить
    force_change, загрузить пользователя из БД, сложить в `g.current_user`."""
    verify_jwt_in_request()
    claims = get_jwt()
    if claims.get("force_change") and fn_name != "change_default":
        abort(403, description="FORCE_PASSWORD_CHANGE")

    from app.repositories.user_repo import get_user_by_id
    user_id = int(get_jwt_identity())
    user = get_user_by_id(user_id)
    if user is None or user.is_hidden:
        abort(401, description="Пользователь не найден")

    # Доступ к отключённой компании запрещаем уже на уровне декоратора —
    # чтобы старые access-токены (где company_id ещё активен) не давали
    # пройти после отключения.
    if user.company_id is not None:
        company = user.company
        if company is None or not company.is_active:
            abort(403, description="COMPANY_DISABLED")

    g.current_user = user
    return user


def require_role(min_level: int):
    """Декоратор для Flask route — проверяет JWT и уровень роли."""
    def decorator(fn):
        @wraps(fn)
        def wrapper(*args, **kwargs):
            user = _resolve_current_user_and_check_force_change(fn.__name__)
            if get_user_level(user) < min_level:
                abort(403, description="Недостаточно прав")
            return fn(*args, **kwargs)
        return wrapper
    return decorator


def require_auth(fn):
    """Декоратор — только проверка JWT и force_change. Без проверки уровня роли."""
    @wraps(fn)
    def wrapper(*args, **kwargs):
        _resolve_current_user_and_check_force_change(fn.__name__)
        return fn(*args, **kwargs)
    return wrapper


def resolve_company_scope(user) -> int | None:
    """Определяет, в рамках какой компании выполняется запрос.
    - Обычные роли всегда работают со своей `user.company_id`.
    - Администратор системы (`is_root_admin` или просто без company_id) может
      явно указать `?company_id=<id>` в query — это его «текущий контекст».
      Если не указал — возвращается None, и обработчик сам решит, что делать
      (например, отдать данные по всем компаниям).
    """
    if user.company_id is not None:
        return user.company_id
    raw = request.args.get("company_id")
    if raw is None or raw == "":
        return None
    try:
        return int(raw)
    except (TypeError, ValueError):
        abort(400, description="Неверный company_id")


def require_company_scope(fn):
    """Гарантирует, что у обработчика есть `g.company_id`. Если пользователь
    привязан к компании — это его компания; если Администратор системы — берём
    из query-param. Если и того и другого нет — 400."""
    @wraps(fn)
    def wrapper(*args, **kwargs):
        user = getattr(g, "current_user", None)
        if user is None:
            user = _resolve_current_user_and_check_force_change(fn.__name__)
        cid = resolve_company_scope(user)
        if cid is None:
            abort(400, description="Требуется указать company_id")
        g.company_id = cid
        return fn(*args, **kwargs)
    return wrapper

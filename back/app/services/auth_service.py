from flask import current_app
from flask_jwt_extended import create_access_token, create_refresh_token, decode_token
from app.extensions import db
from app.repositories import user_repo
from app.services import login_throttle
from app.utils.logger import get_logger

logger = get_logger(__name__)


class AuthError(Exception):
    def __init__(self, message: str, code: str = "AUTH_ERROR", http_status: int = 400, extra: dict = None):
        self.message = message
        self.code = code
        self.http_status = http_status
        self.extra = extra or {}
        super().__init__(message)


def _raise_locked(seconds: int):
    raise AuthError(
        f"Слишком много неудачных попыток. Подождите {seconds} с.",
        "TOO_MANY_ATTEMPTS",
        429,
        {"retry_after_sec": seconds},
    )


def _build_claims(user) -> dict:
    """Полезные клеймы для фронта/декораторов. company_* у Администратора
    системы (без компании) — None; фронт это интерпретирует как «работа от
    лица системы», и в шапке появляется селектор компаний.

    company_settings прокидываем в JWT, чтобы фронт мог скрывать поля
    (YouGile/Этапы) без отдельного запроса /companies/me."""
    company = user.company
    return {
        "force_change": user.is_default_pass,
        "company_id": user.company_id,
        "company_name": company.name if company else None,
        "company_settings": company.settings if company else None,
        "role_level": user.role.level if user.role else 0,
        "is_root_admin": bool(user.is_root_admin),
    }


def _ensure_company_active(user):
    """Если у пользователя есть привязка к компании — она должна быть активна.
    Администраторы системы (без company_id) не блокируются."""
    if user.company_id is None:
        return
    company = user.company
    if company is None or not company.is_active:
        raise AuthError(
            "Ваша компания отключена. Обратитесь к администратору.",
            "COMPANY_DISABLED",
            403,
            {"company_name": company.name if company else None},
        )


def login(login: str, password: str) -> dict:
    # Активная блокировка — даже не проверяем пароль.
    locked_for = login_throttle.get_lock_remaining(login)
    if locked_for > 0:
        _raise_locked(locked_for)

    user = user_repo.get_by_login(login)
    if user is None or user.is_hidden:
        delay = login_throttle.register_failure(login)
        if delay > 0:
            _raise_locked(delay)
        raise AuthError("Неверный логин или пароль", "INVALID_CREDENTIALS", 401)

    ok = user_repo.verify_password_sql(password, user.hash_password)
    if not ok:
        delay = login_throttle.register_failure(login)
        if delay > 0:
            _raise_locked(delay)
        raise AuthError("Неверный логин или пароль", "INVALID_CREDENTIALS", 401)

    login_throttle.register_success(login)
    # Пароль верный — проверяем доступ компании. Делаем ПОСЛЕ верификации
    # пароля, чтобы по ответу нельзя было узнать, к какой компании
    # принадлежит чужой логин.
    _ensure_company_active(user)

    additional_claims = _build_claims(user)
    access_token = create_access_token(identity=str(user.id), additional_claims=additional_claims)
    refresh_token = create_refresh_token(identity=str(user.id))

    logger.info("auth.login", extra={"extra": {"user_id": user.id, "event": "auth.login"}})

    return {
        "access_token": access_token,
        "refresh_token": refresh_token,
        "user_id": user.id,
        "force_change": user.is_default_pass,
    }


def refresh(user_id: int) -> str:
    user = user_repo.get_by_id(user_id)
    if user is None or user.is_hidden:
        raise AuthError("Пользователь не найден", "NOT_FOUND", 401)
    _ensure_company_active(user)

    additional_claims = _build_claims(user)
    return create_access_token(identity=str(user.id), additional_claims=additional_claims)


def change_default_credentials(user_id: int, new_login: str, new_password: str, confirm_password: str) -> dict:
    if new_password != confirm_password:
        raise AuthError("Пароли не совпадают", "PASSWORDS_MISMATCH", 400)

    user = user_repo.get_by_id(user_id)
    if user is None:
        raise AuthError("Пользователь не найден", "NOT_FOUND", 404)

    if not user.is_default_pass:
        raise AuthError("Пароль уже был изменён", "ALREADY_CHANGED", 422)

    existing = user_repo.get_by_login(new_login)
    if existing and existing.id != user_id:
        raise AuthError("Логин уже занят", "LOGIN_TAKEN", 409)

    hashed = user_repo.hash_password_sql(new_password)
    user_repo.update(user, login=new_login, hash_password=hashed, is_default_pass=False)
    db.session.commit()

    # Перечитываем пользователя через repo, чтобы получить актуальные значения
    # для клеймов (post-change role/company не изменились, но используем общий
    # _build_claims, чтобы клеймы оставались согласованы с login/refresh).
    user = user_repo.get_by_id(user_id)
    additional_claims = _build_claims(user)
    access_token = create_access_token(identity=str(user.id), additional_claims=additional_claims)
    refresh_token = create_refresh_token(identity=str(user.id))

    logger.info("auth.change_default", extra={"extra": {"user_id": user.id, "event": "auth.change_default"}})

    return {"access_token": access_token, "refresh_token": refresh_token}

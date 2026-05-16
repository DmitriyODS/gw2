from flask import current_app
from flask_jwt_extended import create_access_token, create_refresh_token, decode_token
from app.extensions import db
from app.repositories import user_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)


class AuthError(Exception):
    def __init__(self, message: str, code: str = "AUTH_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def login(login: str, password: str) -> dict:
    user = user_repo.get_by_login(login)
    if user is None or user.is_hidden:
        raise AuthError("Неверный логин или пароль", "INVALID_CREDENTIALS", 401)

    ok = user_repo.verify_password_sql(password, user.hash_password)
    if not ok:
        raise AuthError("Неверный логин или пароль", "INVALID_CREDENTIALS", 401)

    additional_claims = {"force_change": user.is_default_pass}
    access_token = create_access_token(identity=user.id, additional_claims=additional_claims)
    refresh_token = create_refresh_token(identity=user.id)

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

    additional_claims = {"force_change": user.is_default_pass}
    return create_access_token(identity=user.id, additional_claims=additional_claims)


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

    additional_claims = {"force_change": False}
    access_token = create_access_token(identity=user.id, additional_claims=additional_claims)
    refresh_token = create_refresh_token(identity=user.id)

    logger.info("auth.change_default", extra={"extra": {"user_id": user.id, "event": "auth.change_default"}})

    return {"access_token": access_token, "refresh_token": refresh_token}

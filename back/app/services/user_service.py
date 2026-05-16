from flask import current_app
from app.extensions import db
from app.repositories import user_repo, role_repo
from app.utils.avatar import save_avatar, delete_avatar
from app.utils.logger import get_logger
from app.utils.permissions import BIGINT_MAX

logger = get_logger(__name__)


class UserServiceError(Exception):
    def __init__(self, message: str, code: str = "USER_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def create_user(fio: str, login: str, role_id: int, post: str = None) -> object:
    role = role_repo.get_by_id(role_id)
    if role is None:
        raise UserServiceError("Роль не найдена", "ROLE_NOT_FOUND", 404)

    existing = user_repo.get_by_login(login)
    if existing:
        raise UserServiceError("Логин уже занят", "LOGIN_TAKEN", 409)

    hashed = user_repo.hash_password_sql("admin")
    user = user_repo.create(fio=fio, login=login, hashed_password=hashed, role_id=role_id, post=post)
    db.session.commit()

    logger.info("user.create", extra={"extra": {"user_id": user.id, "event": "user.create"}})
    return user


def update_user(user_id: int, current_user_id: int, **kwargs) -> object:
    user = user_repo.get_by_id(user_id)
    if user is None or user.is_hidden:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    if "login" in kwargs:
        existing = user_repo.get_by_login(kwargs["login"])
        if existing and existing.id != user_id:
            raise UserServiceError("Логин уже занят", "LOGIN_TAKEN", 409)

    user_repo.update(user, **kwargs)
    db.session.commit()
    return user


def hide_user(user_id: int, current_user_id: int) -> None:
    if user_id == current_user_id:
        raise UserServiceError("Нельзя скрыть самого себя", "SELF_HIDE", 422)

    user = user_repo.get_by_id(user_id)
    if user is None or user.is_hidden:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    role = role_repo.get_by_id(user.role_id)
    if role and role.access == BIGINT_MAX:
        almighty_count = user_repo.count_almighty_holders(role.id)
        if almighty_count <= 1:
            raise UserServiceError(
                "Нельзя скрыть единственного носителя всесильной роли",
                "LAST_ALMIGHTY_USER", 422
            )

    user_repo.update(user, is_hidden=True)
    db.session.commit()
    logger.info("user.hide", extra={"extra": {"user_id": user_id, "event": "user.hide"}})


def update_me(user_id: int, fio: str = None, login: str = None, post: str = None,
              current_password: str = None, new_password: str = None, confirm_password: str = None) -> object:
    user = user_repo.get_by_id(user_id)
    if user is None:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    updates = {}

    if fio is not None:
        updates["fio"] = fio

    if post is not None:
        updates["post"] = post

    if login is not None:
        existing = user_repo.get_by_login(login)
        if existing and existing.id != user_id:
            raise UserServiceError("Логин уже занят", "LOGIN_TAKEN", 409)
        updates["login"] = login

    if new_password:
        if not current_password:
            raise UserServiceError("Введите текущий пароль", "CURRENT_PASSWORD_REQUIRED", 400)
        if not user_repo.verify_password_sql(current_password, user.hash_password):
            raise UserServiceError("Неверный текущий пароль", "WRONG_PASSWORD", 400)
        if new_password != confirm_password:
            raise UserServiceError("Пароли не совпадают", "PASSWORDS_MISMATCH", 400)
        updates["hash_password"] = user_repo.hash_password_sql(new_password)

    if updates:
        user_repo.update(user, **updates)
        db.session.commit()

    return user


def upload_avatar(user_id: int, file_bytes: bytes) -> object:
    user = user_repo.get_by_id(user_id)
    if user is None:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    upload_folder = current_app.config["UPLOAD_FOLDER"]

    if user.avatar_path:
        delete_avatar(user.avatar_path, upload_folder)

    avatar_path = save_avatar(file_bytes, upload_folder)
    user_repo.update(user, avatar_path=avatar_path)
    db.session.commit()
    return user


def delete_user_avatar(user_id: int) -> object:
    user = user_repo.get_by_id(user_id)
    if user is None:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    if user.avatar_path:
        upload_folder = current_app.config["UPLOAD_FOLDER"]
        delete_avatar(user.avatar_path, upload_folder)
        user_repo.update(user, avatar_path=None)
        db.session.commit()

    return user


def assign_role(user_id: int, role_id: int, current_user_id: int) -> object:
    if user_id == current_user_id:
        raise UserServiceError("Нельзя изменить свою роль", "SELF_ROLE_CHANGE", 422)

    user = user_repo.get_by_id(user_id)
    if user is None or user.is_hidden:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    old_role = role_repo.get_by_id(user.role_id)
    if old_role and old_role.access == BIGINT_MAX:
        almighty_count = user_repo.count_almighty_holders(old_role.id)
        if almighty_count <= 1:
            raise UserServiceError(
                "Нельзя изменить роль единственного носителя всесильной роли",
                "LAST_ALMIGHTY_USER", 422
            )

    new_role = role_repo.get_by_id(role_id)
    if new_role is None:
        raise UserServiceError("Роль не найдена", "ROLE_NOT_FOUND", 404)

    user_repo.update(user, role_id=role_id)
    db.session.commit()
    return user

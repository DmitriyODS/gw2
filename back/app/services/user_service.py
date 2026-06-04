from flask import current_app
from app.extensions import db
from app.repositories import user_repo, role_repo
from app.utils.avatar import save_avatar, delete_avatar
from app.utils.logger import get_logger
from app.utils.permissions import MANAGER, ADMIN

logger = get_logger(__name__)


class UserServiceError(Exception):
    def __init__(self, message: str, code: str = "USER_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def create_user(fio: str, login: str, role_id: int, current_user_level: int,
                company_id: int = None,
                post: str = None, password: str = None,
                phone: str = None, email: str = None) -> object:
    role = role_repo.get_by_id(role_id)
    if role is None:
        raise UserServiceError("Роль не найдена", "ROLE_NOT_FOUND", 404)

    # Нельзя назначить роль равную или выше своей. Раньше было "выше менеджера
    # без супер-админа"; теперь общее правило: своя роль — потолок.
    if role.level >= current_user_level:
        raise UserServiceError("Нельзя назначить роль равную или выше собственной",
                               "ROLE_LEVEL_FORBIDDEN", 403)

    if login and user_repo.get_by_login(login):
        raise UserServiceError("Логин уже занят", "LOGIN_TAKEN", 409)
    if email and user_repo.get_by_email(email):
        raise UserServiceError("Email уже используется", "EMAIL_TAKEN", 409)

    is_default = password is None
    hashed = user_repo.hash_password_sql(password if password else "admin")
    user = user_repo.create(
        fio=fio, login=login, hashed_password=hashed, role_id=role_id,
        company_id=company_id, post=post, phone=phone, email=email,
        is_default_pass=is_default,
    )
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


def hide_user(user_id: int, current_user_id: int, current_user_level: int) -> None:
    if user_id == current_user_id:
        raise UserServiceError("Нельзя скрыть самого себя", "SELF_HIDE", 422)

    user = user_repo.get_by_id(user_id)
    if user is None or user.is_hidden:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    # Защита: нельзя скрыть пользователя с более высоким уровнем
    if user.role.level >= current_user_level:
        raise UserServiceError("Нельзя удалить пользователя с такой же или более высокой ролью", "ROLE_LEVEL_FORBIDDEN", 403)

    # Защита корневого Администратора системы (is_root_admin) — его нельзя
    # скрыть никому. Запасная защита: если в системе всего один Администратор,
    # его тоже нельзя скрыть.
    if user.is_root_admin:
        raise UserServiceError(
            "Корневого Администратора системы нельзя удалить",
            "ROOT_ADMIN", 422,
        )
    if user.role.level >= ADMIN:
        if user_repo.count_by_level(ADMIN) <= 1:
            raise UserServiceError(
                "Нельзя скрыть единственного Администратора системы",
                "LAST_ADMIN", 422
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


def assign_role(user_id: int, role_id: int, current_user_id: int, current_user_level: int) -> object:
    if user_id == current_user_id:
        raise UserServiceError("Нельзя изменить свою роль", "SELF_ROLE_CHANGE", 422)

    user = user_repo.get_by_id(user_id)
    if user is None or user.is_hidden:
        raise UserServiceError("Пользователь не найден", "NOT_FOUND", 404)

    new_role = role_repo.get_by_id(role_id)
    if new_role is None:
        raise UserServiceError("Роль не найдена", "ROLE_NOT_FOUND", 404)

    # Проверка уровня: нельзя назначить роль выше своего уровня
    if new_role.level >= current_user_level:
        raise UserServiceError("Нельзя назначить роль равную или выше собственной", "ROLE_LEVEL_FORBIDDEN", 403)

    # Корневого Администратора системы (is_root_admin) разжаловать нельзя
    # никому. И защита: последний Администратор системы остаётся в роли.
    if user.is_root_admin:
        raise UserServiceError(
            "Корневому Администратору системы нельзя сменить роль",
            "ROOT_ADMIN", 422,
        )
    if user.role.level >= ADMIN:
        if user_repo.count_by_level(ADMIN) <= 1:
            raise UserServiceError(
                "Нельзя изменить роль единственного Администратора системы",
                "LAST_ADMIN", 422
            )

    user_repo.update(user, role_id=role_id)
    db.session.commit()
    return user

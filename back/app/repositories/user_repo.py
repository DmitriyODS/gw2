from typing import Optional
from sqlalchemy import text
from app.extensions import db
from app.models import User, Role


def get_all(include_hidden: bool = False) -> list[User]:
    q = db.select(User).join(User.role)
    if not include_hidden:
        q = q.where(User.is_hidden.is_(False))
    return db.session.execute(q.order_by(User.id)).scalars().all()


def get_by_id(user_id: int) -> Optional[User]:
    return db.session.execute(
        db.select(User).join(User.role).where(User.id == user_id)
    ).scalar_one_or_none()


def get_by_login(login: str) -> Optional[User]:
    return db.session.execute(
        db.select(User).join(User.role).where(User.login == login)
    ).scalar_one_or_none()


def create(fio: str, login: str, hashed_password: str, role_id: int, post: Optional[str] = None,
           is_default_pass: bool = True) -> User:
    user = User(
        fio=fio,
        login=login,
        hash_password=hashed_password,
        role_id=role_id,
        post=post,
        is_default_pass=is_default_pass,
    )
    db.session.add(user)
    db.session.flush()
    return user


def hash_password_sql(password: str) -> str:
    result = db.session.execute(
        text("SELECT crypt(:password, gen_salt('bf')) AS hash"),
        {"password": password}
    )
    return result.scalar_one()


def verify_password_sql(password: str, stored_hash: str) -> bool:
    result = db.session.execute(
        text("SELECT crypt(:password, :hash) = :hash AS ok"),
        {"password": password, "hash": stored_hash}
    )
    return bool(result.scalar_one())


def update(user: User, **kwargs) -> User:
    for key, value in kwargs.items():
        setattr(user, key, value)
    db.session.flush()
    return user


def count_by_level(level: int) -> int:
    """Количество видимых пользователей с указанным уровнем роли."""
    return db.session.execute(
        db.select(db.func.count(User.id))
        .join(User.role)
        .where(Role.level == level, User.is_hidden.is_(False))
    ).scalar_one()


get_user_by_id = get_by_id

from typing import Optional
from app.extensions import db
from app.models import Role
from app.utils.permissions import BIGINT_MAX


def get_all() -> list[Role]:
    return db.session.execute(db.select(Role).order_by(Role.id)).scalars().all()


def get_by_id(role_id: int) -> Optional[Role]:
    return db.session.get(Role, role_id)


def create(name: str, access: int) -> Role:
    role = Role(name=name, access=access)
    db.session.add(role)
    db.session.flush()
    return role


def update(role: Role, **kwargs) -> Role:
    for key, value in kwargs.items():
        setattr(role, key, value)
    db.session.flush()
    return role


def delete(role: Role) -> None:
    db.session.delete(role)
    db.session.flush()


def count_almighty() -> int:
    """Количество ролей с полным доступом (access == BIGINT_MAX)."""
    return db.session.execute(
        db.select(db.func.count(Role.id)).where(Role.access == BIGINT_MAX)
    ).scalar_one()

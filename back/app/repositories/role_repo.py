from typing import Optional
from app.extensions import db
from app.models import Role


def get_all() -> list[Role]:
    return db.session.execute(db.select(Role).order_by(Role.level)).scalars().all()


def get_by_id(role_id: int) -> Optional[Role]:
    return db.session.get(Role, role_id)

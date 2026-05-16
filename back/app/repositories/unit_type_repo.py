from typing import Optional
from app.extensions import db
from app.models import UnitType


def get_all() -> list[UnitType]:
    return db.session.execute(db.select(UnitType).order_by(UnitType.name)).scalars().all()


def get_by_id(type_id: int) -> Optional[UnitType]:
    return db.session.get(UnitType, type_id)


def get_by_name(name: str) -> Optional[UnitType]:
    return db.session.execute(
        db.select(UnitType).where(UnitType.name == name)
    ).scalar_one_or_none()


def create(name: str) -> UnitType:
    ut = UnitType(name=name)
    db.session.add(ut)
    db.session.flush()
    return ut


def update(ut: UnitType, name: str) -> UnitType:
    ut.name = name
    db.session.flush()
    return ut


def delete(ut: UnitType) -> None:
    db.session.delete(ut)
    db.session.flush()

from typing import Optional
from app.extensions import db
from app.models import UnitType


def get_all(company_id: int) -> list[UnitType]:
    return db.session.execute(
        db.select(UnitType)
        .where(UnitType.company_id == company_id)
        .order_by(UnitType.name)
    ).scalars().all()


def get_by_id(type_id: int) -> Optional[UnitType]:
    return db.session.get(UnitType, type_id)


def get_by_name(name: str, company_id: int) -> Optional[UnitType]:
    return db.session.execute(
        db.select(UnitType).where(
            UnitType.name == name,
            UnitType.company_id == company_id,
        )
    ).scalar_one_or_none()


def create(name: str, company_id: int) -> UnitType:
    ut = UnitType(name=name, company_id=company_id)
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

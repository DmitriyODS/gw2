from datetime import datetime, timezone
from typing import Optional
from sqlalchemy import desc
from sqlalchemy.orm import selectinload
from app.extensions import db
from app.models import Unit


def get_by_task(task_id: int) -> list[Unit]:
    # user/unit_type сериализуются UnitSchema — грузим батчем, не лениво.
    return db.session.execute(
        db.select(Unit)
        .options(selectinload(Unit.user), selectinload(Unit.unit_type))
        .where(Unit.task_id == task_id)
        .order_by(desc(Unit.datetime_start))
    ).scalars().all()


def get_by_id(unit_id: int) -> Optional[Unit]:
    return db.session.execute(
        db.select(Unit).where(Unit.id == unit_id)
    ).scalar_one_or_none()


def get_active_for_user(user_id: int) -> Optional[Unit]:
    return db.session.execute(
        db.select(Unit).where(Unit.user_id == user_id, Unit.datetime_end.is_(None))
    ).scalar_one_or_none()


def create(name: str, user_id: int, unit_type_id: int, task_id: int, company_id: int) -> Unit:
    unit = Unit(
        name=name,
        user_id=user_id,
        unit_type_id=unit_type_id,
        task_id=task_id,
        company_id=company_id,
    )
    db.session.add(unit)
    db.session.flush()
    return unit


def update(unit: Unit, **kwargs) -> Unit:
    for key, value in kwargs.items():
        setattr(unit, key, value)
    unit.is_edited = True
    db.session.flush()
    return unit


def stop(unit: Unit) -> Unit:
    unit.datetime_end = datetime.now(timezone.utc)
    db.session.flush()
    return unit


def delete(unit: Unit) -> None:
    db.session.delete(unit)
    db.session.flush()

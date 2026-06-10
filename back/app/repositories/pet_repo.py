"""Питомцы-Грувики, поглаживания и рейды. Только I/O."""
from datetime import date, datetime

from sqlalchemy import func

from app.extensions import db
from app.models.groove import Pet, PetStroke, GrooveRaid
from app.models.task import Task
from app.models.unit import Unit
from app.models.user import User


# ───────────────────────────── питомцы ─────────────────────────────

def get_pet(user_id: int) -> Pet | None:
    return db.session.get(Pet, user_id)


def get_or_create(user_id: int, company_id: int) -> Pet:
    pet = db.session.get(Pet, user_id)
    if pet is None:
        pet = Pet(user_id=user_id, company_id=company_id)
        db.session.add(pet)
        db.session.flush()
    return pet


def list_company_pets(company_id: int) -> list[Pet]:
    """Зоопарк компании: питомцы видимых сотрудников, старшие стадии первыми."""
    return db.session.execute(
        db.select(Pet)
        .join(User, User.id == Pet.user_id)
        .where(Pet.company_id == company_id, User.is_hidden.is_(False))
        .order_by(Pet.stage.desc(), Pet.xp.desc())
    ).scalars().all()


def finished_units_for_user(user_id: int, since: datetime,
                            limit: int = 100) -> list[Unit]:
    """Завершённые юниты пользователя для определения «характера» питомца."""
    return db.session.execute(
        db.select(Unit)
        .where(Unit.user_id == user_id,
               Unit.datetime_end.isnot(None),
               Unit.datetime_start >= since)
        .order_by(Unit.datetime_start.desc())
        .limit(limit)
    ).scalars().all()


# ──────────────────────────── поглаживания ─────────────────────────

def add_stroke(pet_user_id: int, user_id: int, day: date) -> bool:
    """True — погладил; False — уже гладил сегодня."""
    existing = db.session.execute(
        db.select(PetStroke.id).where(
            PetStroke.pet_user_id == pet_user_id,
            PetStroke.user_id == user_id,
            PetStroke.day == day,
        )
    ).scalar_one_or_none()
    if existing is not None:
        return False
    db.session.add(PetStroke(pet_user_id=pet_user_id, user_id=user_id, day=day))
    return True


def strokes_today(pet_user_ids: list[int], day: date) -> dict[int, int]:
    if not pet_user_ids:
        return {}
    rows = db.session.execute(
        db.select(PetStroke.pet_user_id, func.count(PetStroke.id))
        .where(PetStroke.pet_user_id.in_(pet_user_ids), PetStroke.day == day)
        .group_by(PetStroke.pet_user_id)
    ).all()
    return {pet_user_id: count for pet_user_id, count in rows}


def my_strokes_today(user_id: int, day: date) -> set[int]:
    """Чьих питомцев я уже погладил сегодня (set из pet_user_id)."""
    rows = db.session.execute(
        db.select(PetStroke.pet_user_id)
        .where(PetStroke.user_id == user_id, PetStroke.day == day)
    ).scalars().all()
    return set(rows)


# ────────────────────────────── рейды ──────────────────────────────

def get_raid(company_id: int, week_start: date) -> GrooveRaid | None:
    return db.session.execute(
        db.select(GrooveRaid).where(
            GrooveRaid.company_id == company_id,
            GrooveRaid.week_start == week_start,
        )
    ).scalar_one_or_none()


def create_raid(company_id: int, week_start: date, boss: str, target: int,
                reward: str) -> GrooveRaid:
    raid = GrooveRaid(company_id=company_id, week_start=week_start, boss=boss,
                      target=target, reward=reward)
    db.session.add(raid)
    db.session.flush()
    return raid


def count_closed_between(company_id: int, start: datetime, end: datetime) -> int:
    return db.session.execute(
        db.select(func.count(Task.id)).where(
            Task.company_id == company_id,
            Task.is_archived.is_(True),
            Task.archived_at >= start,
            Task.archived_at < end,
        )
    ).scalar_one()

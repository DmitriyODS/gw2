from typing import Optional
from sqlalchemy import func
from sqlalchemy.orm import selectinload
from app.extensions import db
from app.models import Company, User, Task


def get_all() -> list[Company]:
    return db.session.execute(
        db.select(Company)
        .options(selectinload(Company.director))
        .order_by(Company.created_at.desc())
    ).scalars().all()


def get_by_id(company_id: int) -> Optional[Company]:
    return db.session.execute(
        db.select(Company)
        .options(selectinload(Company.director))
        .where(Company.id == company_id)
    ).scalar_one_or_none()


def get_by_name(name: str) -> Optional[Company]:
    return db.session.execute(
        db.select(Company).where(Company.name == name)
    ).scalar_one_or_none()


def create(name: str, description: Optional[str] = None,
           director_id: Optional[int] = None, settings: Optional[dict] = None) -> Company:
    company = Company(
        name=name,
        description=description,
        director_id=director_id,
        settings=settings or {},
    )
    db.session.add(company)
    db.session.flush()
    return company


def update(company: Company, **kwargs) -> Company:
    for key, value in kwargs.items():
        setattr(company, key, value)
    db.session.flush()
    return company


def delete(company: Company) -> None:
    db.session.delete(company)
    db.session.flush()


def stats_by_company_id(company_id: int) -> dict:
    """Кол-ва сотрудников и задач — для строки таблицы Компаний."""
    stats = stats_by_company_ids([company_id])
    return stats.get(company_id, {"employees": 0, "tasks": 0})


def stats_by_company_ids(company_ids: list[int]) -> dict[int, dict]:
    """Батч-статистика для списка компаний без N+1."""
    if not company_ids:
        return {}

    employees_rows = db.session.execute(
        db.select(
            User.company_id.label("company_id"),
            func.count(User.id).label("employees"),
        ).where(
            User.company_id.in_(company_ids),
            User.is_hidden.is_(False),
        ).group_by(User.company_id)
    ).all()

    tasks_rows = db.session.execute(
        db.select(
            Task.company_id.label("company_id"),
            func.count(Task.id).label("tasks"),
        ).where(
            Task.company_id.in_(company_ids),
        ).group_by(Task.company_id)
    ).all()

    result: dict[int, dict] = {cid: {"employees": 0, "tasks": 0} for cid in company_ids}
    for row in employees_rows:
        result[row.company_id]["employees"] = int(row.employees)
    for row in tasks_rows:
        result[row.company_id]["tasks"] = int(row.tasks)
    return result

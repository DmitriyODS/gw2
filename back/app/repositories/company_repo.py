from typing import Optional
from sqlalchemy import func
from app.extensions import db
from app.models import Company, User, Task


def get_all() -> list[Company]:
    return db.session.execute(
        db.select(Company).order_by(Company.created_at.desc())
    ).scalars().all()


def get_by_id(company_id: int) -> Optional[Company]:
    return db.session.get(Company, company_id)


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
    employees = db.session.execute(
        db.select(func.count(User.id)).where(
            User.company_id == company_id, User.is_hidden.is_(False),
        )
    ).scalar_one()
    tasks = db.session.execute(
        db.select(func.count(Task.id)).where(Task.company_id == company_id)
    ).scalar_one()
    return {"employees": employees, "tasks": tasks}

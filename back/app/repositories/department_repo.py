from typing import Optional
from app.extensions import db
from app.models import Department


def get_all(company_id: int) -> list[Department]:
    return db.session.execute(
        db.select(Department)
        .where(Department.company_id == company_id)
        .order_by(Department.name)
    ).scalars().all()


def get_by_id(dept_id: int) -> Optional[Department]:
    return db.session.get(Department, dept_id)


def get_by_name(name: str, company_id: int) -> Optional[Department]:
    return db.session.execute(
        db.select(Department).where(
            Department.name == name,
            Department.company_id == company_id,
        )
    ).scalar_one_or_none()


def create(name: str, company_id: int) -> Department:
    dept = Department(name=name, company_id=company_id)
    db.session.add(dept)
    db.session.flush()
    return dept


def update(dept: Department, name: str) -> Department:
    dept.name = name
    db.session.flush()
    return dept


def delete(dept: Department) -> None:
    db.session.delete(dept)
    db.session.flush()

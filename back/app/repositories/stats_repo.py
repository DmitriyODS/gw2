from datetime import datetime
from typing import Optional
from sqlalchemy import func, distinct, and_
from app.extensions import db
from app.models import Task, Unit, UnitType, Department, User


def _apply_company(q, company_id: Optional[int], col):
    return q.where(col == company_id) if company_id is not None else q


def get_common_metrics(period_start: datetime, period_end: datetime,
                       company_id: Optional[int] = None) -> dict:
    debt_q = db.select(func.count(Task.id)).where(
        Task.is_archived.is_(False),
        Task.received_at < period_start,
    )
    received_q = db.select(func.count(Task.id)).where(
        Task.received_at >= period_start,
        Task.received_at <= period_end,
    )
    closed_q = db.select(func.count(Task.id)).where(
        Task.is_archived.is_(True),
        Task.archived_at >= period_start,
        Task.archived_at <= period_end,
    )
    remaining_q = db.select(func.count(Task.id)).where(Task.is_archived.is_(False))

    debt = db.session.execute(_apply_company(debt_q, company_id, Task.company_id)).scalar_one()
    received = db.session.execute(_apply_company(received_q, company_id, Task.company_id)).scalar_one()
    closed = db.session.execute(_apply_company(closed_q, company_id, Task.company_id)).scalar_one()
    remaining = db.session.execute(_apply_company(remaining_q, company_id, Task.company_id)).scalar_one()

    return {"debt": debt, "received": received, "closed": closed, "remaining": remaining}


def get_tasks_by_hours(period_start: datetime, period_end: datetime,
                       company_id: Optional[int] = None) -> list[dict]:
    q = (
        db.select(
            Task.id,
            Task.name,
            func.coalesce(
                func.sum(
                    func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
                ), 0
            ).label("total_hours")
        )
        .join(Unit, Unit.task_id == Task.id, isouter=True)
        .where(
            and_(
                Unit.datetime_start >= period_start,
                Unit.datetime_start <= period_end,
            )
        )
        .group_by(Task.id, Task.name)
        .order_by(func.sum(
            func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start)
        ).desc())
    )
    rows = db.session.execute(_apply_company(q, company_id, Task.company_id)).all()
    return [{"task_id": r.id, "name": r.name, "total_hours": round(r.total_hours, 2)} for r in rows]


def get_tasks_by_employees(period_start: datetime, period_end: datetime,
                           company_id: Optional[int] = None) -> list[dict]:
    q = (
        db.select(
            User.id,
            User.fio,
            func.count(distinct(Unit.task_id)).label("tasks_count"),
            func.coalesce(
                func.sum(
                    func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
                ), 0
            ).label("total_hours")
        )
        .join(Unit, Unit.user_id == User.id)
        .where(
            Unit.datetime_start >= period_start,
            Unit.datetime_start <= period_end,
        )
        .group_by(User.id, User.fio)
        .order_by(func.sum(
            func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start)
        ).desc())
    )
    rows = db.session.execute(_apply_company(q, company_id, User.company_id)).all()
    return [{"user_id": r.id, "fio": r.fio, "tasks_count": r.tasks_count, "total_hours": round(r.total_hours, 2)} for r in rows]


def get_by_unit_types(period_start: datetime, period_end: datetime,
                      company_id: Optional[int] = None) -> list[dict]:
    q = (
        db.select(
            UnitType.id,
            UnitType.name,
            func.coalesce(
                func.sum(
                    func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
                ), 0
            ).label("total_hours"),
            func.count(distinct(Unit.task_id)).label("tasks_count")
        )
        .join(Unit, Unit.unit_type_id == UnitType.id)
        .where(
            Unit.datetime_start >= period_start,
            Unit.datetime_start <= period_end,
        )
        .group_by(UnitType.id, UnitType.name)
        .order_by(func.sum(
            func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start)
        ).desc())
    )
    rows = db.session.execute(_apply_company(q, company_id, UnitType.company_id)).all()
    return [{"type_id": r.id, "name": r.name, "total_hours": round(r.total_hours, 2), "tasks_count": r.tasks_count} for r in rows]


def get_by_departments(period_start: datetime, period_end: datetime,
                       company_id: Optional[int] = None) -> list[dict]:
    q = (
        db.select(
            Department.id,
            Department.name,
            func.count(distinct(Task.id)).label("tasks_count")
        )
        .join(Task, Task.department_id == Department.id)
        .where(
            Task.received_at >= period_start,
            Task.received_at <= period_end,
        )
        .group_by(Department.id, Department.name)
        .order_by(func.count(distinct(Task.id)).desc())
    )
    rows = db.session.execute(_apply_company(q, company_id, Department.company_id)).all()
    return [{"dept_id": r.id, "name": r.name, "tasks_count": r.tasks_count} for r in rows]


def get_by_unit_types_per_user(period_start: datetime, period_end: datetime,
                               company_id: Optional[int] = None) -> list[dict]:
    q = (
        db.select(
            User.id,
            User.fio,
            UnitType.id.label("type_id"),
            UnitType.name.label("type_name"),
            func.coalesce(
                func.sum(
                    func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
                ), 0
            ).label("hours"),
            func.count(distinct(Unit.task_id)).label("tasks_count")
        )
        .join(Unit, Unit.user_id == User.id)
        .join(UnitType, UnitType.id == Unit.unit_type_id)
        .where(
            Unit.datetime_start >= period_start,
            Unit.datetime_start <= period_end,
        )
        .group_by(User.id, User.fio, UnitType.id, UnitType.name)
        .order_by(User.fio, UnitType.name)
    )
    rows = db.session.execute(_apply_company(q, company_id, User.company_id)).all()

    result = {}
    for r in rows:
        if r.id not in result:
            result[r.id] = {"user_id": r.id, "fio": r.fio, "unit_types": []}
        result[r.id]["unit_types"].append({
            "type_id": r.type_id,
            "name": r.type_name,
            "hours": round(r.hours, 2),
            "tasks_count": r.tasks_count,
        })
    return list(result.values())


def get_calendar(period_start: datetime, period_end: datetime,
                 company_id: Optional[int] = None) -> list[dict]:
    received_q = (
        db.select(
            func.date(Task.received_at).label("date"),
            func.count(Task.id).label("received")
        )
        .where(Task.received_at >= period_start, Task.received_at <= period_end)
        .group_by(func.date(Task.received_at))
    )
    closed_q = (
        db.select(
            func.date(Task.archived_at).label("date"),
            func.count(Task.id).label("closed")
        )
        .where(
            Task.is_archived.is_(True),
            Task.archived_at >= period_start,
            Task.archived_at <= period_end,
        )
        .group_by(func.date(Task.archived_at))
    )
    hours_q = (
        db.select(
            func.date(Unit.datetime_start).label("date"),
            func.sum(
                func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
            ).label("total_hours")
        )
        .where(Unit.datetime_start >= period_start, Unit.datetime_start <= period_end)
        .group_by(func.date(Unit.datetime_start))
    )

    if company_id is not None:
        received_q = received_q.where(Task.company_id == company_id)
        closed_q = closed_q.where(Task.company_id == company_id)
        hours_q = hours_q.join(Task, Task.id == Unit.task_id).where(Task.company_id == company_id)

    received_rows = db.session.execute(received_q).all()
    closed_rows = db.session.execute(closed_q).all()
    hours_rows = db.session.execute(hours_q).all()

    calendar: dict = {}
    for r in received_rows:
        d = str(r.date)
        calendar.setdefault(d, {"date": d, "received": 0, "closed": 0, "total_hours": 0.0})
        calendar[d]["received"] = r.received
    for r in closed_rows:
        d = str(r.date)
        calendar.setdefault(d, {"date": d, "received": 0, "closed": 0, "total_hours": 0.0})
        calendar[d]["closed"] = r.closed
    for r in hours_rows:
        d = str(r.date)
        calendar.setdefault(d, {"date": d, "received": 0, "closed": 0, "total_hours": 0.0})
        calendar[d]["total_hours"] = round(r.total_hours or 0, 2)

    return sorted(calendar.values(), key=lambda x: x["date"])


def get_user_tasks_detail(user_id: int, period_start: datetime, period_end: datetime) -> dict:
    rows = db.session.execute(
        db.select(
            Task.id,
            Task.name,
            func.coalesce(
                func.sum(
                    func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
                ), 0
            ).label("total_hours")
        )
        .join(Unit, Unit.task_id == Task.id)
        .where(
            Unit.user_id == user_id,
            Unit.datetime_start >= period_start,
            Unit.datetime_start <= period_end,
        )
        .group_by(Task.id, Task.name)
        .order_by(func.sum(
            func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start)
        ).desc())
    ).all()
    return {
        "tasks": [{"task_id": r.id, "task_name": r.name, "total_hours": round(float(r.total_hours), 2)} for r in rows],
        "tasks_count": len(rows),
    }


def get_profile_stats(user_id: int, period_start: datetime, period_end: datetime) -> dict:
    total_hours = db.session.execute(
        db.select(
            func.coalesce(
                func.sum(
                    func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
                ), 0
            )
        )
        .where(
            Unit.user_id == user_id,
            Unit.datetime_start >= period_start,
            Unit.datetime_start <= period_end,
        )
    ).scalar_one()

    tasks_count = db.session.execute(
        db.select(func.count(distinct(Unit.task_id)))
        .where(
            Unit.user_id == user_id,
            Unit.datetime_start >= period_start,
            Unit.datetime_start <= period_end,
        )
    ).scalar_one()

    by_types_rows = db.session.execute(
        db.select(
            UnitType.id,
            UnitType.name,
            func.coalesce(
                func.sum(
                    func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
                ), 0
            ).label("hours"),
            func.count(distinct(Unit.task_id)).label("tasks_count")
        )
        .join(Unit, Unit.unit_type_id == UnitType.id)
        .where(
            Unit.user_id == user_id,
            Unit.datetime_start >= period_start,
            Unit.datetime_start <= period_end,
        )
        .group_by(UnitType.id, UnitType.name)
    ).all()

    return {
        "total_hours": round(float(total_hours), 2),
        "tasks_count": tasks_count,
        "by_unit_types": [
            {"type_id": r.id, "name": r.name, "hours": round(r.hours, 2), "tasks_count": r.tasks_count}
            for r in by_types_rows
        ],
    }


def get_responsibles(company_id: Optional[int] = None) -> list[dict]:
    """Сотрудники с количеством открытых/закрытых задач, где они responsible.
    Сортировка: больше открытых → выше."""
    q = (
        db.select(
            User.id,
            User.fio,
            User.avatar_path,
            User.post,
            func.sum(db.case((Task.is_archived.is_(False), 1), else_=0)).label("open_count"),
            func.sum(db.case((Task.is_archived.is_(True), 1), else_=0)).label("closed_count"),
        )
        .join(Task, Task.responsible_user_id == User.id)
        .where(User.is_hidden.is_(False))
        .group_by(User.id, User.fio, User.avatar_path, User.post)
        .order_by(
            func.sum(db.case((Task.is_archived.is_(False), 1), else_=0)).desc(),
            func.sum(db.case((Task.is_archived.is_(True), 1), else_=0)).desc(),
        )
    )
    rows = db.session.execute(_apply_company(q, company_id, Task.company_id)).all()
    return [
        {
            "user_id": r.id,
            "fio": r.fio,
            "avatar_path": r.avatar_path,
            "post": r.post,
            "open_count": int(r.open_count or 0),
            "closed_count": int(r.closed_count or 0),
        }
        for r in rows
    ]

from datetime import datetime
from typing import Optional
from sqlalchemy import text, func, distinct, and_
from app.extensions import db
from app.models import Task, Unit, UnitType, Department, User


def get_common_metrics(period_start: datetime, period_end: datetime) -> dict:
    debt = db.session.execute(
        db.select(func.count(Task.id)).where(
            Task.is_archived.is_(False),
            Task.received_at < period_start,
        )
    ).scalar_one()

    received = db.session.execute(
        db.select(func.count(Task.id)).where(
            Task.received_at >= period_start,
            Task.received_at <= period_end,
        )
    ).scalar_one()

    closed = db.session.execute(
        db.select(func.count(Task.id)).where(
            Task.is_archived.is_(True),
            Task.archived_at >= period_start,
            Task.archived_at <= period_end,
        )
    ).scalar_one()

    remaining = db.session.execute(
        db.select(func.count(Task.id)).where(
            Task.is_archived.is_(False),
            Task.received_at <= period_end,
        )
    ).scalar_one()

    return {"debt": debt, "received": received, "closed": closed, "remaining": remaining}


def get_tasks_by_hours(period_start: datetime, period_end: datetime) -> list[dict]:
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
    ).all()
    return [{"task_id": r.id, "name": r.name, "total_hours": round(r.total_hours, 2)} for r in rows]


def get_tasks_by_employees(period_start: datetime, period_end: datetime) -> list[dict]:
    rows = db.session.execute(
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
    ).all()
    return [{"user_id": r.id, "fio": r.fio, "tasks_count": r.tasks_count, "total_hours": round(r.total_hours, 2)} for r in rows]


def get_by_unit_types(period_start: datetime, period_end: datetime) -> list[dict]:
    rows = db.session.execute(
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
    ).all()
    return [{"type_id": r.id, "name": r.name, "total_hours": round(r.total_hours, 2), "tasks_count": r.tasks_count} for r in rows]


def get_by_departments(period_start: datetime, period_end: datetime) -> list[dict]:
    rows = db.session.execute(
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
    ).all()
    return [{"dept_id": r.id, "name": r.name, "tasks_count": r.tasks_count} for r in rows]


def get_by_unit_types_per_user(period_start: datetime, period_end: datetime) -> list[dict]:
    rows = db.session.execute(
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
    ).all()

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


def get_calendar(period_start: datetime, period_end: datetime) -> list[dict]:
    received_rows = db.session.execute(
        db.select(
            func.date(Task.received_at).label("date"),
            func.count(Task.id).label("received")
        )
        .where(Task.received_at >= period_start, Task.received_at <= period_end)
        .group_by(func.date(Task.received_at))
    ).all()

    closed_rows = db.session.execute(
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
    ).all()

    hours_rows = db.session.execute(
        db.select(
            func.date(Unit.datetime_start).label("date"),
            func.sum(
                func.extract("epoch", func.coalesce(Unit.datetime_end, func.now()) - Unit.datetime_start) / 3600
            ).label("total_hours")
        )
        .where(Unit.datetime_start >= period_start, Unit.datetime_start <= period_end)
        .group_by(func.date(Unit.datetime_start))
    ).all()

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

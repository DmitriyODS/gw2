from datetime import datetime
from typing import Optional
from sqlalchemy import exists, and_
from sqlalchemy.orm import selectinload
from app.extensions import db
from app.models import Task, Unit, Favorite, UserTaskColor

# Домен задач живёт в tasksvc (back-go/tasks); здесь — только то, что нужно
# YouGile-интеграции (до фазы 4): создание/обновление при импорте и вебхуках
# + срез для enrich-дампа сокет-событий.

_TASK_LOAD_OPTIONS = (
    selectinload(Task.author),
    selectinload(Task.responsible),
    selectinload(Task.department),
    selectinload(Task.stage),
)


def get_by_id(task_id: int) -> Optional[Task]:
    return db.session.execute(
        db.select(Task).options(*_TASK_LOAD_OPTIONS).where(Task.id == task_id)
    ).scalar_one_or_none()


def create(
    name: str,
    author_id: int,
    department_id: int,
    company_id: int,
    received_at: Optional[datetime] = None,
    link_yougile: Optional[str] = None,
    deadline: Optional[datetime] = None,
    responsible_user_id: Optional[int] = None,
    stage_id: Optional[int] = None,
) -> Task:
    task = Task(
        name=name,
        author_id=author_id,
        department_id=department_id,
        company_id=company_id,
        link_yougile=link_yougile,
        deadline=deadline,
        responsible_user_id=responsible_user_id,
        stage_id=stage_id,
    )
    if received_at:
        task.received_at = received_at
    db.session.add(task)
    db.session.flush()
    return task


def update(task: Task, **kwargs) -> Task:
    for key, value in kwargs.items():
        setattr(task, key, value)
    db.session.flush()
    return task


def has_active_unit(task_id: int) -> bool:
    return db.session.execute(
        db.select(exists().where(and_(Unit.task_id == task_id, Unit.datetime_end.is_(None))))
    ).scalar_one()


def is_favorite(task_id: int, user_id: int) -> bool:
    return db.session.execute(
        db.select(exists().where(and_(Favorite.task_id == task_id, Favorite.user_id == user_id)))
    ).scalar_one()


def has_any_units(task_id: int) -> bool:
    return db.session.execute(
        db.select(exists().where(Unit.task_id == task_id))
    ).scalar_one()


def get_active_users(task_id: int) -> list[dict]:
    from app.models import User as UserModel
    rows = db.session.execute(
        db.select(UserModel.id, UserModel.fio, UserModel.avatar_path)
        .join(Unit, Unit.user_id == UserModel.id)
        .where(Unit.task_id == task_id, Unit.datetime_end.is_(None))
    ).all()
    return [{"id": r.id, "fio": r.fio, "avatar_path": r.avatar_path} for r in rows]


def get_user_color(task_id: int, user_id: int) -> Optional[str]:
    return db.session.execute(
        db.select(UserTaskColor.color).where(
            UserTaskColor.task_id == task_id, UserTaskColor.user_id == user_id
        )
    ).scalar_one_or_none()

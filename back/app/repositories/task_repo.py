from datetime import datetime
from typing import Optional
from sqlalchemy import desc, asc, func, exists, and_
from app.extensions import db
from app.models import Task, Unit, Favorite, UserTaskColor


def get_by_id(task_id: int) -> Optional[Task]:
    return db.session.execute(
        db.select(Task).where(Task.id == task_id)
    ).scalar_one_or_none()


def get_list(
    current_user_id: int,
    company_id: Optional[int],
    tab: str = "active",
    search: Optional[str] = None,
    sort: str = "last_activity",
    dept_id: Optional[int] = None,
    stage_id: Optional[int] = None,
    responsible_user_id: Optional[int] = None,
    received_from: Optional[datetime] = None,
    received_to: Optional[datetime] = None,
    has_units: Optional[str] = None,
    author_id: Optional[int] = None,
    page: int = 1,
    per_page: int = 30,
) -> dict:
    q = db.select(Task)
    # Multi-tenancy: основной фильтр. None означает «во всех компаниях»
    # (доступно только Администратору системы, без явно выбранной компании
    # в селекторе) — на практике обработчик должен либо требовать
    # company_id, либо явно решать.
    if company_id is not None:
        q = q.where(Task.company_id == company_id)

    # Вкладка
    if tab == "active":
        q = q.where(Task.is_archived.is_(False))
    elif tab == "favorites":
        q = q.where(
            Task.is_archived.is_(False),
            exists().where(
                and_(Favorite.task_id == Task.id, Favorite.user_id == current_user_id)
            )
        )
    elif tab == "archive":
        q = q.where(Task.is_archived.is_(True))

    # Поиск
    if search:
        q = q.where(Task.name.ilike(f"%{search}%"))

    # Фильтр по отделу
    if dept_id:
        q = q.where(Task.department_id == dept_id)

    # Фильтр по этапу
    if stage_id is not None:
        q = q.where(Task.stage_id == stage_id)

    # Фильтр по ответственному
    if responsible_user_id is not None:
        q = q.where(Task.responsible_user_id == responsible_user_id)

    # Период поступления
    if received_from:
        q = q.where(Task.received_at >= received_from)
    if received_to:
        q = q.where(Task.received_at <= received_to)

    # Фильтр по автору
    if author_id is not None:
        q = q.where(Task.author_id == author_id)

    # Фильтр по юнитам
    if has_units == "none":
        q = q.where(~exists().where(Unit.task_id == Task.id))
    elif has_units == "mine":
        q = q.where(
            exists().where(and_(Unit.task_id == Task.id, Unit.user_id == current_user_id))
        )

    # Сортировка
    if sort == "last_activity":
        last_unit_subq = (
            db.select(func.max(Unit.datetime_start))
            .where(Unit.task_id == Task.id)
            .correlate(Task)
            .scalar_subquery()
        )
        q = q.order_by(desc(last_unit_subq).nulls_last(), desc(Task.created_at))
    elif sort == "created_at":
        q = q.order_by(desc(Task.created_at))
    elif sort == "received_at":
        q = q.order_by(desc(Task.received_at))
    elif sort == "deadline":
        q = q.order_by(asc(Task.deadline).nulls_last())

    total = db.session.execute(db.select(func.count()).select_from(q.subquery())).scalar_one()

    offset = (page - 1) * per_page
    tasks = db.session.execute(q.offset(offset).limit(per_page)).scalars().all()

    return {"items": tasks, "total": total, "page": page, "per_page": per_page}


def get_stale(threshold: datetime, company_id: Optional[int] = None,
              limit: int = 100) -> list[Task]:
    """Активные (не в архиве) задачи, поступившие раньше threshold — те, что
    «висят» дольше порога. Сначала самые старые, чтобы напоминание подсвечивало
    залежавшиеся в первую очередь."""
    q = (
        db.select(Task)
        .where(Task.is_archived.is_(False), Task.received_at < threshold)
        .order_by(asc(Task.received_at))
        .limit(limit)
    )
    if company_id is not None:
        q = q.where(Task.company_id == company_id)
    return db.session.execute(q).scalars().all()


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


def get_contributors(task_id: int) -> list[dict]:
    """Сотрудники, у которых хоть когда-либо был юнит по задаче (distinct)."""
    from app.models import User as UserModel
    rows = db.session.execute(
        db.select(UserModel.id, UserModel.fio, UserModel.avatar_path)
        .join(Unit, Unit.user_id == UserModel.id)
        .where(Unit.task_id == task_id)
        .distinct()
        .order_by(UserModel.fio.asc())
    ).all()
    return [{"id": r.id, "fio": r.fio, "avatar_path": r.avatar_path} for r in rows]


def count_by_company(company_id: int) -> int:
    """Кол-во задач (включая архивные) — для статистики таблицы компаний."""
    return db.session.execute(
        db.select(db.func.count(Task.id)).where(Task.company_id == company_id)
    ).scalar_one()


def update(task: Task, **kwargs) -> Task:
    for key, value in kwargs.items():
        setattr(task, key, value)
    db.session.flush()
    return task


def delete(task: Task) -> None:
    db.session.delete(task)
    db.session.flush()


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


def get_active_users_by_task_ids(task_ids: list) -> dict:
    if not task_ids:
        return {}
    from app.models import User as UserModel
    rows = db.session.execute(
        db.select(Unit.task_id, UserModel.id, UserModel.fio, UserModel.avatar_path)
        .join(UserModel, Unit.user_id == UserModel.id)
        .where(Unit.task_id.in_(task_ids), Unit.datetime_end.is_(None))
    ).all()
    result: dict = {}
    for r in rows:
        result.setdefault(r.task_id, []).append({"id": r.id, "fio": r.fio, "avatar_path": r.avatar_path})
    return result


def toggle_favorite(task_id: int, user_id: int) -> bool:
    """Добавить или убрать из избранного. Возвращает новое состояние (True = добавлено)."""
    fav = db.session.execute(
        db.select(Favorite).where(
            Favorite.task_id == task_id, Favorite.user_id == user_id
        )
    ).scalar_one_or_none()
    if fav:
        db.session.delete(fav)
        db.session.flush()
        return False
    else:
        db.session.add(Favorite(task_id=task_id, user_id=user_id))
        db.session.flush()
        return True


def get_user_color(task_id: int, user_id: int) -> Optional[str]:
    return db.session.execute(
        db.select(UserTaskColor.color).where(
            UserTaskColor.task_id == task_id, UserTaskColor.user_id == user_id
        )
    ).scalar_one_or_none()


def get_user_colors_by_task_ids(task_ids: list, user_id: int) -> dict:
    if not task_ids:
        return {}
    rows = db.session.execute(
        db.select(UserTaskColor.task_id, UserTaskColor.color).where(
            UserTaskColor.user_id == user_id,
            UserTaskColor.task_id.in_(task_ids),
        )
    ).all()
    return {r.task_id: r.color for r in rows}


def set_user_color(task_id: int, user_id: int, color: Optional[str]) -> None:
    rec = db.session.execute(
        db.select(UserTaskColor).where(
            UserTaskColor.task_id == task_id, UserTaskColor.user_id == user_id
        )
    ).scalar_one_or_none()
    if color is None:
        if rec is not None:
            db.session.delete(rec)
            db.session.flush()
        return
    if rec is None:
        db.session.add(UserTaskColor(task_id=task_id, user_id=user_id, color=color))
    else:
        rec.color = color
    db.session.flush()

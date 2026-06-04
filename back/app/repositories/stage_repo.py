from typing import Optional
from sqlalchemy import func
from app.extensions import db
from app.models import Stage


def get_all(company_id: int) -> list[Stage]:
    return db.session.execute(
        db.select(Stage)
        .where(Stage.company_id == company_id)
        .order_by(Stage.order, Stage.id)
    ).scalars().all()


def get_by_id(stage_id: int) -> Optional[Stage]:
    return db.session.get(Stage, stage_id)


def get_by_name(name: str, company_id: int) -> Optional[Stage]:
    return db.session.execute(
        db.select(Stage).where(
            Stage.name == name,
            Stage.company_id == company_id,
        )
    ).scalar_one_or_none()


def next_order(company_id: int) -> int:
    max_order = db.session.execute(
        db.select(func.max(Stage.order)).where(Stage.company_id == company_id)
    ).scalar_one()
    return (max_order or 0) + 1


def create(name: str, color: str, company_id: int, order: int) -> Stage:
    stage = Stage(name=name, color=color, company_id=company_id, order=order)
    db.session.add(stage)
    db.session.flush()
    return stage


def update(stage: Stage, **kwargs) -> Stage:
    for k, v in kwargs.items():
        setattr(stage, k, v)
    db.session.flush()
    return stage


def delete(stage: Stage) -> None:
    db.session.delete(stage)
    db.session.flush()


def reorder(company_id: int, ordered_ids: list[int]) -> list[Stage]:
    """Применяет новый порядок этапов одной транзакцией. Идентификаторы,
    не принадлежащие компании, игнорируются."""
    stages = {s.id: s for s in get_all(company_id)}
    for idx, sid in enumerate(ordered_ids):
        stage = stages.get(sid)
        if stage is not None:
            stage.order = idx + 1
    db.session.flush()
    return get_all(company_id)

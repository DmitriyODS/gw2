from datetime import datetime, timezone
from typing import Optional
from sqlalchemy.orm import joinedload
from app.extensions import db
from app.models import Comment


def get_by_id(comment_id: int) -> Optional[Comment]:
    return db.session.execute(
        db.select(Comment).where(Comment.id == comment_id)
    ).scalar_one_or_none()


def list_by_task(task_id: int) -> list[Comment]:
    return db.session.execute(
        db.select(Comment)
        .options(joinedload(Comment.author))
        .where(Comment.task_id == task_id, Comment.deleted_at.is_(None))
        .order_by(Comment.created_at.asc())
    ).scalars().all()


def create(task_id: int, author_id: int, text: str) -> Comment:
    c = Comment(task_id=task_id, author_id=author_id, text=text)
    db.session.add(c)
    db.session.flush()
    return c


def update_text(comment: Comment, text: str) -> Comment:
    comment.text = text
    comment.updated_at = datetime.now(timezone.utc)
    db.session.flush()
    return comment


def soft_delete(comment: Comment) -> Comment:
    comment.deleted_at = datetime.now(timezone.utc)
    db.session.flush()
    return comment

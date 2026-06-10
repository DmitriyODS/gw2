"""Лента «Мой Groove»: события, реакции, комментарии. Только I/O."""
from sqlalchemy import func
from sqlalchemy.orm import selectinload

from app.extensions import db
from app.models.groove import FeedEvent, FeedReaction, FeedComment
from app.models.unit import Unit
from app.models.user import User


# ───────────────────────────── события ─────────────────────────────

def create_event(company_id: int, user_id, kind: str, payload: dict) -> FeedEvent:
    event = FeedEvent(company_id=company_id, user_id=user_id, kind=kind,
                      payload=payload or {})
    db.session.add(event)
    db.session.flush()
    return event


def get_event(event_id: int) -> FeedEvent | None:
    return db.session.get(FeedEvent, event_id)


def list_events(company_id: int, before_id: int | None = None,
                limit: int = 30) -> list[FeedEvent]:
    q = (db.select(FeedEvent)
         .options(selectinload(FeedEvent.user))
         .where(FeedEvent.company_id == company_id)
         .order_by(FeedEvent.id.desc())
         .limit(limit))
    if before_id is not None:
        q = q.where(FeedEvent.id < before_id)
    return db.session.execute(q).scalars().all()


# ───────────────────────────── реакции ─────────────────────────────

def toggle_reaction(event_id: int, user_id: int, emoji: str) -> bool:
    """Возвращает True, если реакция добавлена, False — если снята."""
    existing = db.session.execute(
        db.select(FeedReaction).where(
            FeedReaction.event_id == event_id,
            FeedReaction.user_id == user_id,
            FeedReaction.emoji == emoji,
        )
    ).scalar_one_or_none()
    if existing is not None:
        db.session.delete(existing)
        return False
    db.session.add(FeedReaction(event_id=event_id, user_id=user_id, emoji=emoji))
    return True


def reaction_counts(event_ids: list[int]) -> dict[int, dict[str, int]]:
    if not event_ids:
        return {}
    rows = db.session.execute(
        db.select(FeedReaction.event_id, FeedReaction.emoji,
                  func.count(FeedReaction.id))
        .where(FeedReaction.event_id.in_(event_ids))
        .group_by(FeedReaction.event_id, FeedReaction.emoji)
    ).all()
    result: dict[int, dict[str, int]] = {}
    for event_id, emoji, count in rows:
        result.setdefault(event_id, {})[emoji] = count
    return result


def my_reactions(event_ids: list[int], user_id: int) -> dict[int, list[str]]:
    if not event_ids:
        return {}
    rows = db.session.execute(
        db.select(FeedReaction.event_id, FeedReaction.emoji)
        .where(FeedReaction.event_id.in_(event_ids),
               FeedReaction.user_id == user_id)
    ).all()
    result: dict[int, list[str]] = {}
    for event_id, emoji in rows:
        result.setdefault(event_id, []).append(emoji)
    return result


def reaction_count_for(event_id: int, emoji: str) -> int:
    return db.session.execute(
        db.select(func.count(FeedReaction.id)).where(
            FeedReaction.event_id == event_id,
            FeedReaction.emoji == emoji,
        )
    ).scalar_one()


# ─────────────────────────── комментарии ───────────────────────────

def comment_counts(event_ids: list[int]) -> dict[int, int]:
    if not event_ids:
        return {}
    rows = db.session.execute(
        db.select(FeedComment.event_id, func.count(FeedComment.id))
        .where(FeedComment.event_id.in_(event_ids))
        .group_by(FeedComment.event_id)
    ).all()
    return {event_id: count for event_id, count in rows}


def list_comments(event_id: int) -> list[FeedComment]:
    return db.session.execute(
        db.select(FeedComment)
        .options(selectinload(FeedComment.author))
        .where(FeedComment.event_id == event_id)
        .order_by(FeedComment.created_at.asc())
    ).scalars().all()


def create_comment(event_id: int, author_id, text: str,
                   reply_to_id: int | None = None,
                   is_bot: bool = False) -> FeedComment:
    comment = FeedComment(event_id=event_id, author_id=author_id, text=text,
                          reply_to_id=reply_to_id, is_bot=is_bot)
    db.session.add(comment)
    db.session.flush()
    return comment


def get_comment(comment_id: int) -> FeedComment | None:
    return db.session.get(FeedComment, comment_id)


def delete_comment(comment: FeedComment) -> None:
    db.session.delete(comment)


# ─────────────────────── wrapped «Моя неделя» ──────────────────────

def count_user_events(company_id: int, user_id: int, kind: str, since) -> int:
    return db.session.execute(
        db.select(func.count(FeedEvent.id)).where(
            FeedEvent.company_id == company_id,
            FeedEvent.user_id == user_id,
            FeedEvent.kind == kind,
            FeedEvent.created_at >= since,
        )
    ).scalar_one()


def reactions_received(user_id: int, since) -> int:
    """Реакции коллег на мои события за период (свои не считаем)."""
    return db.session.execute(
        db.select(func.count(FeedReaction.id))
        .join(FeedEvent, FeedEvent.id == FeedReaction.event_id)
        .where(FeedEvent.user_id == user_id,
               FeedReaction.user_id != user_id,
               FeedReaction.created_at >= since)
    ).scalar_one()


def kudos_received(company_id: int, user_id: int, since) -> int:
    return db.session.execute(
        db.select(func.count(FeedEvent.id)).where(
            FeedEvent.company_id == company_id,
            FeedEvent.kind == "kudos",
            FeedEvent.created_at >= since,
            FeedEvent.payload["to_user_id"].as_integer() == user_id,
        )
    ).scalar_one()


# ─────────────────────────── live-блок ─────────────────────────────

def list_active_units(company_id: int) -> list[Unit]:
    """Активные юниты компании с видимыми владельцами — «Сейчас в эфире»."""
    return db.session.execute(
        db.select(Unit)
        .options(selectinload(Unit.user), selectinload(Unit.task))
        .join(User, User.id == Unit.user_id)
        .where(Unit.company_id == company_id,
               Unit.datetime_end.is_(None),
               User.is_hidden.is_(False))
        .order_by(Unit.datetime_start.asc())
    ).scalars().all()

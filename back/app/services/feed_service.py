"""Лента «Мой Groove»: запись событий, реакции, комментарии, кудосы, заряды.

`record_event` — единственная точка создания событий ленты. Хуки
`on_unit_*` / `on_task_closed` дёргаются из unit_service/task_service ПОСЛЕ
их коммита и никогда не роняют основной запрос: геймификация не должна
ломать работу.
"""
from __future__ import annotations

from flask import current_app
from redis import Redis

from app.extensions import db, socketio
from app.repositories import feed_repo, user_repo, unit_repo
from app.schemas.groove import (FeedEventSchema, FeedCommentSchema,
                                FEED_REACTIONS)
from app.utils.permissions import DIRECTOR
from app.utils.logger import get_logger

logger = get_logger(__name__)

_event_schema = FeedEventSchema()
_comment_schema = FeedCommentSchema()

FEED_PAGE_LIMIT = 30

ZAP_SENT_DAILY_MAX = 10

_redis_client: Redis | None = None


class FeedServiceError(Exception):
    def __init__(self, message: str, code: str = "FEED_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def _redis() -> Redis:
    global _redis_client
    if _redis_client is None:
        _redis_client = Redis.from_url(current_app.config["REDIS_URL"],
                                       decode_responses=True)
    return _redis_client


def _broadcast(event_name: str, data: dict) -> None:
    try:
        socketio.emit(event_name, data, room="all")
    except Exception as e:
        logger.warning("groove.broadcast_failed",
                       extra={"extra": {"event": event_name, "err": str(e)}})


# ─────────────────────────── запись событий ────────────────────────

def record_event(company_id: int, user_id, kind: str, payload: dict,
                 *, bot_comment: bool = False):
    """Создаёт событие, коммитит и вещает feed:new. Возвращает событие."""
    event = feed_repo.create_event(company_id, user_id, kind, payload)
    db.session.commit()
    data = _event_schema.dump(event)
    data["reactions"] = {}
    data["comments_count"] = 0
    _broadcast("feed:new", data)
    if bot_comment:
        from app.services.groove_ai_service import schedule_bot_comment
        schedule_bot_comment(event.id)
    return event


# ─────────────── хуки из unit_service / task_service ───────────────

def _safe(fn):
    try:
        fn()
    except Exception as e:
        try:
            db.session.rollback()
        except Exception:
            pass
        logger.warning("groove.hook_failed", extra={"extra": {"err": str(e)}})


def on_unit_started(unit) -> None:
    def _job():
        record_event(unit.company_id, unit.user_id, "unit_started", {
            "unit_id": unit.id,
            "unit_name": unit.name,
            "task_id": unit.task_id,
            "task_name": unit.task.name if unit.task else None,
        })
    _safe(_job)


def on_unit_stopped(unit) -> None:
    def _job():
        minutes = 0
        if unit.datetime_end and unit.datetime_start:
            minutes = max(0, int((unit.datetime_end - unit.datetime_start)
                                 .total_seconds() // 60))
        record_event(unit.company_id, unit.user_id, "unit_stopped", {
            "unit_id": unit.id,
            "unit_name": unit.name,
            "task_id": unit.task_id,
            "task_name": unit.task.name if unit.task else None,
            "minutes": minutes,
        })
        from app.services import pet_service
        # 1 грув за факт + по груву за каждые полчаса, не больше 5 за юнит.
        pet_service.award_beans(unit.user_id, unit.company_id, "unit",
                                min(5, 1 + minutes // 30))
    _safe(_job)


def on_task_closed(task, actor_id=None) -> None:
    def _job():
        hero_id = actor_id or task.responsible_user_id or task.author_id
        record_event(task.company_id, hero_id, "task_closed", {
            "task_id": task.id,
            "task_name": task.name,
        }, bot_comment=True)
        from app.services import pet_service
        if hero_id:
            pet_service.award_beans(hero_id, task.company_id, "task_closed", 5)
        pet_service.on_task_closed_raid(task.company_id)
    _safe(_job)


# ───────────────────────────── лента ───────────────────────────────

def get_feed_page(company_id: int, user_id: int, before_id=None,
                  limit: int = FEED_PAGE_LIMIT) -> dict:
    limit = max(1, min(int(limit or FEED_PAGE_LIMIT), 100))
    events = feed_repo.list_events(company_id, before_id, limit)
    ids = [e.id for e in events]
    counts = feed_repo.reaction_counts(ids)
    mine = feed_repo.my_reactions(ids, user_id)
    comments = feed_repo.comment_counts(ids)
    items = []
    for e in events:
        data = _event_schema.dump(e)
        data["reactions"] = counts.get(e.id, {})
        data["my_reactions"] = mine.get(e.id, [])
        data["comments_count"] = comments.get(e.id, 0)
        items.append(data)
    return {"items": items, "has_more": len(events) == limit}


# ─────────────────────────── реакции ───────────────────────────────

def toggle_reaction(event_id: int, user_id: int, company_id: int, emoji: str) -> dict:
    if emoji not in FEED_REACTIONS:
        raise FeedServiceError("Недопустимая реакция", "BAD_EMOJI", 422)
    event = feed_repo.get_event(event_id)
    if event is None or event.company_id != company_id:
        raise FeedServiceError("Событие не найдено", "NOT_FOUND", 404)
    added = feed_repo.toggle_reaction(event_id, user_id, emoji)
    db.session.commit()
    count = feed_repo.reaction_count_for(event_id, emoji)
    if added and event.user_id and event.user_id != user_id:
        from app.services import pet_service
        pet_service.award_beans(event.user_id, company_id, "reaction", 1)
    _broadcast("feed:reaction", {
        "event_id": event_id, "emoji": emoji, "count": count,
        "user_id": user_id, "added": added, "company_id": company_id,
    })
    return {"added": added, "count": count}


# ─────────────────────────── комментарии ───────────────────────────

def list_comments(event_id: int, company_id: int) -> list[dict]:
    event = feed_repo.get_event(event_id)
    if event is None or event.company_id != company_id:
        raise FeedServiceError("Событие не найдено", "NOT_FOUND", 404)
    return _comment_schema.dump(feed_repo.list_comments(event_id), many=True)


def add_comment(event_id: int, author_id: int, company_id: int, text: str,
                reply_to_id=None) -> dict:
    event = feed_repo.get_event(event_id)
    if event is None or event.company_id != company_id:
        raise FeedServiceError("Событие не найдено", "NOT_FOUND", 404)
    if reply_to_id is not None:
        parent = feed_repo.get_comment(reply_to_id)
        if parent is None or parent.event_id != event_id:
            raise FeedServiceError("Комментарий не найден", "REPLY_NOT_FOUND", 404)
    comment = feed_repo.create_comment(event_id, author_id, text.strip(),
                                       reply_to_id=reply_to_id)
    db.session.commit()
    data = _comment_schema.dump(comment)
    _broadcast("feed:comment", {"event_id": event_id, "comment": data,
                                "company_id": company_id})
    return data


def delete_comment(comment_id: int, user_id: int, user_level: int) -> None:
    comment = feed_repo.get_comment(comment_id)
    if comment is None:
        raise FeedServiceError("Комментарий не найден", "NOT_FOUND", 404)
    if comment.author_id != user_id and user_level < DIRECTOR:
        raise FeedServiceError("Недостаточно прав", "FORBIDDEN", 403)
    event = comment.event
    feed_repo.delete_comment(comment)
    db.session.commit()
    _broadcast("feed:comment_deleted", {
        "event_id": event.id, "comment_id": comment_id,
        "company_id": event.company_id,
    })


# ───────────────────────────── кудосы ──────────────────────────────

def send_kudos(company_id: int, from_user_id: int, to_user_id: int, text: str):
    if from_user_id == to_user_id:
        raise FeedServiceError("Нельзя благодарить самого себя", "SELF_KUDOS", 422)
    target = user_repo.get_by_id(to_user_id)
    if target is None or target.is_hidden or target.company_id != company_id:
        raise FeedServiceError("Сотрудник не найден", "USER_NOT_FOUND", 404)
    event = record_event(company_id, from_user_id, "kudos", {
        "to_user_id": target.id,
        "to_fio": target.fio,
        "to_avatar_path": target.avatar_path,
        "text": text.strip(),
    }, bot_comment=True)
    from app.services import pet_service
    pet_service.award_beans(to_user_id, company_id, "kudos", 2)
    return event


# ─────────────────────── live и заряды энергии ─────────────────────

def get_live(company_id: int) -> list[dict]:
    units = feed_repo.list_active_units(company_id)
    zaps = {}
    if units:
        try:
            values = _redis().mget([f"gw2:groove:zaps:{u.id}" for u in units])
            zaps = {u.id: int(v or 0) for u, v in zip(units, values)}
        except Exception:
            zaps = {}
    return [{
        "unit_id": u.id,
        "unit_name": u.name,
        "task_id": u.task_id,
        "task_name": u.task.name if u.task else None,
        "started_at": u.datetime_start.isoformat() if u.datetime_start else None,
        "user": {"id": u.user.id, "fio": u.user.fio,
                 "avatar_path": u.user.avatar_path} if u.user else None,
        "zaps": zaps.get(u.id, 0),
    } for u in units]


def send_zap(company_id: int, from_user_id: int, to_user_id: int) -> dict:
    if from_user_id == to_user_id:
        raise FeedServiceError("Себя зарядить нельзя", "SELF_ZAP", 422)
    unit = unit_repo.get_active_for_user(to_user_id)
    if unit is None or unit.company_id != company_id:
        raise FeedServiceError("Коллега сейчас не в эфире", "NOT_LIVE", 422)

    from app.services import pet_service
    if not pet_service.take_daily_budget(from_user_id, "zap_sent", 1,
                                         ZAP_SENT_DAILY_MAX):
        raise FeedServiceError("Заряды на сегодня закончились", "ZAP_LIMIT", 429)

    zaps = 1
    try:
        r = _redis()
        key = f"gw2:groove:zaps:{unit.id}"
        zaps = r.incr(key)
        r.expire(key, 24 * 3600)
    except Exception:
        pass

    pet_service.award_beans(to_user_id, company_id, "zap", 1)
    sender = user_repo.get_by_id(from_user_id)
    try:
        socketio.emit("groove:zap", {
            "from_user_id": from_user_id,
            "from_fio": sender.fio if sender else "Коллега",
            "to_user_id": to_user_id,
            "unit_id": unit.id,
            "zaps": zaps,
        }, room=f"user_{to_user_id}")
        socketio.emit("groove:zap-count", {
            "unit_id": unit.id, "zaps": zaps, "company_id": company_id,
        }, room="all")
    except Exception:
        pass
    return {"zaps": zaps}

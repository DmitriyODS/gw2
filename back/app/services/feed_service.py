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
        # Работа лечит больного Грувика (совсем короткие юниты не считаются).
        if minutes >= pet_service.RECOVERY_MIN_UNIT_MINUTES:
            pet_service.add_recovery(unit.user_id, unit.company_id, 1)
        # Дневной квест: завершённые юниты и минуты в фокусе.
        pet_service.bump_quest(unit.user_id, "units_finished", 1)
        if minutes > 0:
            pet_service.bump_quest(unit.user_id, "unit_minutes", minutes)
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
            pet_service.add_recovery(hero_id, task.company_id, 1)
            pet_service.bump_quest(hero_id, "tasks_closed", 1)
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


# ───────────────────── wrapped «Моя неделя» ────────────────────────

_WEEKDAYS_RU = ["понедельник", "вторник", "среда", "четверг",
                "пятница", "суббота", "воскресенье"]


def get_wrapped(company_id: int, user_id: int) -> dict:
    """Личный итог последних 7 дней — карточки-истории «Моя неделя»."""
    from datetime import datetime, timedelta, timezone as tz
    from app.repositories import pet_repo
    from app.services import pet_service

    since = datetime.now(tz.utc) - timedelta(days=7)
    units = pet_repo.finished_units_for_user(user_id, since, limit=300)

    total_minutes = 0
    longest = None
    by_day: dict[int, int] = {}
    start_hours: list[int] = []
    for u in units:
        minutes = max(0, int((u.datetime_end - u.datetime_start).total_seconds() // 60))
        total_minutes += minutes
        if longest is None or minutes > longest[1]:
            longest = (u, minutes)
        local_start = u.datetime_start.astimezone(pet_service.MSK)
        by_day[local_start.weekday()] = by_day.get(local_start.weekday(), 0) + minutes
        start_hours.append(local_start.hour)

    best_day = None
    if by_day:
        day_idx, day_minutes = max(by_day.items(), key=lambda kv: kv[1])
        best_day = {"label": _WEEKDAYS_RU[day_idx], "minutes": day_minutes}

    peak_hour = None
    if start_hours:
        start_hours.sort()
        peak_hour = start_hours[len(start_hours) // 2]

    closed = feed_repo.count_user_events(company_id, user_id, "task_closed", since)
    reactions = feed_repo.reactions_received(user_id, since)
    kudos = feed_repo.kudos_received(company_id, user_id, since)

    soulmate = None
    mate = pet_repo.soulmate_for_user(user_id, since)
    if mate is not None:
        mate_user, mate_units = mate
        soulmate = {
            "user": {"id": mate_user.id, "fio": mate_user.fio,
                     "avatar_path": mate_user.avatar_path},
            "units": mate_units,
        }

    pet = pet_repo.get_or_create(user_id, company_id)
    db.session.commit()

    stats = {
        "units": len(units),
        "minutes": total_minutes,
        "closed": closed,
        "longest": ({"name": longest[0].name, "minutes": longest[1]}
                    if longest else None),
        "best_day": best_day,
        "peak_hour": peak_hour,
        "reactions": reactions,
        "kudos": kudos,
        "soulmate": soulmate,
        "pet": {"name": pet.name, "stage": pet.stage, "species": pet.species,
                "feed_streak": pet.feed_streak, "sick": pet.sick_since is not None},
    }
    from app.services.groove_ai_service import get_wrapped_phrase
    stats["ai_phrase"] = get_wrapped_phrase(company_id, user_id, stats)
    return stats


def share_wrapped(company_id: int, user_id: int) -> None:
    """Опубликовать итог недели в ленту (не чаще раза в день)."""
    key = f"gw2:groove:wrapped_share:{user_id}"
    try:
        r = _redis()
        if r.exists(key):
            raise FeedServiceError("Итог недели уже опубликован сегодня",
                                   "ALREADY_SHARED", 429)
    except FeedServiceError:
        raise
    except Exception:
        r = None
    stats = get_wrapped(company_id, user_id)
    record_event(company_id, user_id, "wrapped", {
        "units": stats["units"],
        "minutes": stats["minutes"],
        "closed": stats["closed"],
        "best_day": stats["best_day"]["label"] if stats["best_day"] else None,
        "reactions": stats["reactions"],
        "kudos": stats["kudos"],
    }, bot_comment=True)
    if r is not None:
        try:
            r.setex(key, 24 * 3600, "1")
        except Exception:
            pass


# ─────────────────────── live и заряды энергии ─────────────────────

def get_live(company_id: int, viewer_id: int) -> dict:
    units = feed_repo.list_active_units(company_id)
    zaps = {}
    if units:
        try:
            values = _redis().mget([f"gw2:groove:zaps:{u.id}" for u in units])
            zaps = {u.id: int(v or 0) for u, v in zip(units, values)}
        except Exception:
            zaps = {}
    from app.services import pet_service
    zaps_left = pet_service.daily_left(viewer_id, "zap_sent", ZAP_SENT_DAILY_MAX)
    return {
        "items": [{
            "unit_id": u.id,
            "unit_name": u.name,
            "task_id": u.task_id,
            "task_name": u.task.name if u.task else None,
            "started_at": u.datetime_start.isoformat() if u.datetime_start else None,
            "user": {"id": u.user.id, "fio": u.user.fio,
                     "avatar_path": u.user.avatar_path} if u.user else None,
            "zaps": zaps.get(u.id, 0),
        } for u in units],
        # Личный дневной запас зарядов зрителя (обнуляется в полночь МСК).
        "zaps_left": zaps_left,
        "zaps_max": ZAP_SENT_DAILY_MAX,
    }


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
    zaps_left = pet_service.daily_left(from_user_id, "zap_sent", ZAP_SENT_DAILY_MAX)
    return {"zaps": zaps, "zaps_left": zaps_left, "zaps_max": ZAP_SENT_DAILY_MAX}

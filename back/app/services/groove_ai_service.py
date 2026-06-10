"""ИИ-слой раздела «Мой Groove» — Грувик как живой талисман компании.

Три механики:
1. «Грувик комментирует» — под событиями-вехами ленты (эволюция, стрик,
   победа над боссом, кудос, иногда закрытая задача) асинхронно появляется
   короткий комментарий от бота. Вероятностный гейт, чтобы не спамить.
2. Утренний дайджест — раз в день (после DIGEST_HOUR_MSK) системное событие
   `ai_digest` со сводкой вчерашнего дня. Дедупликация через Redis.
3. Фразы при кормлении — пул реплик генерится раз в сутки per-company и
   лежит в Redis; кормление берёт случайную мгновенно (без похода в LLM).

Всё fail-safe: AI выключен/упал — бот молчит, кормление отвечает статикой.
"""
from __future__ import annotations

import json
import random
import time
from datetime import datetime, timedelta, timezone

from flask import current_app
from redis import Redis

from app.extensions import db, socketio
from app.utils.logger import get_logger

logger = get_logger(__name__)

MSK = timezone(timedelta(hours=3))

DIGEST_HOUR_MSK = 9
TICK_INTERVAL_SEC = 15 * 60

_PHRASES_KEY = "gw2:groove:phrases:{cid}"
_PHRASES_TTL = 48 * 3600
_DIGEST_KEY = "gw2:groove:digest:{cid}:{day}"
_DIGEST_TTL = 48 * 3600

# Вероятность комментария Грувика по виду события.
BOT_COMMENT_PROB = {
    "pet_evolved": 1.0,
    "streak": 1.0,
    "raid_won": 1.0,
    "kudos": 0.8,
    "raid_started": 0.8,
    "task_closed": 0.25,
}

# Фолбэк-реплики кормления, если AI выключен или пул пуст.
STATIC_PHRASES = [
    "Ням! Сегодня грувы особенно хрустящие.",
    "Спасибо! Чувствую, как расту.",
    "Ещё парочка таких — и я эволюционирую!",
    "Вкуснотища. Кто молодец? Ты молодец.",
    "Грув-грув! Продолжаем в том же духе.",
    "М-м-м, со вкусом закрытой задачи.",
    "Я бы и от поглаживания не отказался…",
    "Заряжен и готов к подвигам!",
]

_SYSTEM_PROMPT = (
    "Ты — Грувик, питомец-талисман корпоративной платформы Groove Work. "
    "Характер: добрый, ироничный, поддерживающий, без пафоса и канцелярита. "
    "Отвечай на русском, без кавычек и преамбул."
)

_redis_client: Redis | None = None


def _redis() -> Redis:
    global _redis_client
    if _redis_client is None:
        _redis_client = Redis.from_url(current_app.config["REDIS_URL"],
                                       decode_responses=True)
    return _redis_client


# ─────────────────────── фразы при кормлении ───────────────────────

def get_feed_phrase(company_id: int) -> str:
    try:
        raw = _redis().get(_PHRASES_KEY.format(cid=company_id))
        if raw:
            pool = json.loads(raw)
            if pool:
                return random.choice(pool)
    except Exception:
        pass
    return random.choice(STATIC_PHRASES)


def refresh_phrases(company_id: int) -> bool:
    from app.services.ai_client import get_ai_client
    client = get_ai_client(company_id)
    if client is None:
        return False
    try:
        text = client.chat(
            messages=[
                {"role": "system", "content": _SYSTEM_PROMPT},
                {"role": "user", "content": (
                    "Придумай 12 коротких реплик (до 90 символов каждая), "
                    "которые ты говоришь, когда тебя кормят грувами — внутренней "
                    "валютой за выполненную работу. Разные настроения: радость, "
                    "юмор, лёгкая ирония, благодарность. Можно изредка эмодзи. "
                    "Ответ — строго по одной реплике на строку, без нумерации."
                )},
            ],
            max_tokens=500, temperature=1.0, timeout=25.0,
        )
        pool = [ln.strip().strip('-•"«»').strip() for ln in text.splitlines()]
        pool = [p for p in pool if 3 <= len(p) <= 120][:12]
        if not pool:
            return False
        _redis().setex(_PHRASES_KEY.format(cid=company_id), _PHRASES_TTL,
                       json.dumps(pool, ensure_ascii=False))
        return True
    except Exception as e:
        logger.warning("groove.ai.phrases_failed",
                       extra={"extra": {"company_id": company_id, "err": str(e)}})
        return False


# ───────────────────── комментарии Грувика-бота ────────────────────

def _bot_prompt_for_event(event) -> str | None:
    p = event.payload or {}
    fio = event.user.fio if event.user else None
    first_name = fio.split()[1] if fio and len(fio.split()) > 1 else (fio or "коллега")
    if event.kind == "pet_evolved":
        return (f"Питомец сотрудника {first_name} по имени «{p.get('pet_name', 'Грувик')}» "
                f"эволюционировал до стадии {p.get('stage')}. Поздравь хозяина "
                "коротко и забавно (1-2 предложения, до 160 символов).")
    if event.kind == "streak":
        return (f"{first_name} кормит питомца {p.get('days')} дней подряд. "
                "Отметь постоянство, добавь лёгкую шутку (до 160 символов).")
    if event.kind == "raid_won":
        return (f"Команда победила недельного босса «{p.get('boss')}» — закрыла "
                f"{p.get('target')} задач. Триумфальный командный комментарий "
                "(до 180 символов).")
    if event.kind == "raid_started":
        return (f"Начался недельный рейд: босс «{p.get('boss')}», нужно закрыть "
                f"{p.get('target')} задач командой. Подзадорь команду "
                "(до 160 символов).")
    if event.kind == "kudos":
        return (f"{first_name} публично поблагодарил(а) коллегу "
                f"{p.get('to_fio')}: «{(p.get('text') or '')[:200]}». "
                "Поддержи тёплую атмосферу одной фразой (до 140 символов).")
    if event.kind == "task_closed":
        return (f"{first_name} закрыл(а) задачу «{(p.get('task_name') or '')[:120]}». "
                "Коротко похвали, можно с юмором (до 120 символов).")
    return None


def schedule_bot_comment(event_id: int) -> None:
    """Асинхронно (eventlet-greenlet) добавить комментарий Грувика к событию."""
    app = current_app._get_current_object()

    def _job():
        with app.app_context():
            try:
                _make_bot_comment(event_id)
            except Exception as e:
                db.session.rollback()
                logger.warning("groove.ai.bot_comment_failed",
                               extra={"extra": {"event_id": event_id, "err": str(e)}})

    try:
        socketio.start_background_task(_job)
    except Exception as e:
        logger.warning("groove.ai.spawn_failed",
                       extra={"extra": {"event_id": event_id, "err": str(e)}})


def _make_bot_comment(event_id: int) -> None:
    from app.repositories import feed_repo
    from app.services.ai_client import get_ai_client

    event = feed_repo.get_event(event_id)
    if event is None:
        return
    prob = BOT_COMMENT_PROB.get(event.kind, 0.0)
    if random.random() > prob:
        return
    client = get_ai_client(event.company_id)
    if client is None:
        return
    prompt = _bot_prompt_for_event(event)
    if prompt is None:
        return
    text = client.chat(
        messages=[
            {"role": "system", "content": _SYSTEM_PROMPT},
            {"role": "user", "content": prompt},
        ],
        max_tokens=140, temperature=0.95, timeout=25.0,
    ).strip().strip('"«»').strip()
    if not text:
        return
    comment = feed_repo.create_comment(event.id, None, text, is_bot=True)
    db.session.commit()
    from app.services.feed_service import _broadcast, _comment_schema
    _broadcast("feed:comment", {
        "event_id": event.id,
        "comment": _comment_schema.dump(comment),
        "company_id": event.company_id,
    })


# ───────────────────────── утренний дайджест ───────────────────────

def _digest_context(company_id: int) -> dict:
    from app.repositories import stats_repo
    now = datetime.now(MSK)
    end = now.replace(hour=0, minute=0, second=0, microsecond=0)
    start = end - timedelta(days=1)
    try:
        common = stats_repo.get_common_metrics(start, end, company_id)
        employees = stats_repo.get_tasks_by_employees(start, end, company_id) or []
    except Exception:
        return {}
    leader = employees[0] if employees else None
    total_hours = sum((e.get("total_hours") or 0) for e in employees)
    return {
        "closed": common.get("closed", 0),
        "received": common.get("received", 0),
        "hours": round(total_hours, 1),
        "leader_fio": leader["fio"] if leader else None,
    }


def generate_digest(company_id: int) -> bool:
    from app.services.ai_client import get_ai_client
    client = get_ai_client(company_id)
    if client is None:
        return False
    ctx = _digest_context(company_id)
    lines = ["Напиши утренний пост-дайджест для ленты команды: поприветствуй, "
             "подведи итог вчерашнего дня, пожелай хорошего дня. 2-3 предложения, "
             "до 350 символов, живо и с юмором."]
    if ctx.get("closed"):
        lines.append(f"Вчера закрыто задач: {ctx['closed']}.")
    if ctx.get("received"):
        lines.append(f"Поступило новых: {ctx['received']}.")
    if ctx.get("hours"):
        lines.append(f"Команда отработала часов: {ctx['hours']}.")
    if ctx.get("leader_fio"):
        lines.append(f"Герой вчерашнего дня — {ctx['leader_fio']}.")
    if not any((ctx.get("closed"), ctx.get("received"), ctx.get("hours"))):
        lines.append("Вчера было тихо — обыграй это мягко, без упрёков.")
    try:
        text = client.chat(
            messages=[
                {"role": "system", "content": _SYSTEM_PROMPT},
                {"role": "user", "content": " ".join(lines)},
            ],
            max_tokens=260, temperature=0.9, timeout=25.0,
        ).strip().strip('"«»').strip()
    except Exception as e:
        logger.warning("groove.ai.digest_failed",
                       extra={"extra": {"company_id": company_id, "err": str(e)}})
        return False
    if not text:
        return False
    from app.services.feed_service import record_event
    record_event(company_id, None, "ai_digest",
                 {"text": text, "date": datetime.now(MSK).date().isoformat()})
    return True


# ─────────────────────────── фоновый цикл ──────────────────────────

def _tick(app) -> None:
    from app.models.company import Company
    with app.app_context():
        company_ids = [c.id for c in Company.query.filter_by(ai_enabled=True).all()]
    for cid in company_ids:
        with app.app_context():
            try:
                r = _redis()
                # Пул фраз кормления: держим свежим всегда.
                if not r.exists(_PHRASES_KEY.format(cid=cid)):
                    refresh_phrases(cid)
                # Дайджест: один раз в день после DIGEST_HOUR_MSK.
                now = datetime.now(MSK)
                if now.hour >= DIGEST_HOUR_MSK:
                    key = _DIGEST_KEY.format(cid=cid, day=now.date().isoformat())
                    if not r.exists(key) and generate_digest(cid):
                        r.setex(key, _DIGEST_TTL, "1")
            except Exception as e:
                logger.warning("groove.ai.tick_failed",
                               extra={"extra": {"company_id": cid, "err": str(e)}})


def run_groove_ai_loop(app) -> None:
    logger.info("groove.ai.loop_start",
                extra={"extra": {"interval_sec": TICK_INTERVAL_SEC}})
    try:
        _tick(app)
    except Exception as e:
        logger.warning("groove.ai.initial_tick_failed", extra={"extra": {"err": str(e)}})
    while True:
        try:
            time.sleep(TICK_INTERVAL_SEC)
        except Exception:
            return
        try:
            _tick(app)
        except Exception as e:
            logger.warning("groove.ai.loop_tick_failed", extra={"extra": {"err": str(e)}})

"""TV-факт дня для брендового слайда.

Генерируем раз в час по каждой компании с включённым AI. Храним в Redis:
`gw2:ai:tv_fact:{company_id}` → JSON.

Жанры чередуются 50/50:
- "general"  — общий познавательный факт о работе / продуктивности.
- "context"  — наблюдение по статистике компании за последние 7 дней.

Никаких ретраев на месте: если генерация упала — просто пропускаем тик, на
следующем перегенерим. Лучше показать прошлый факт, чем рисковать стабильностью
цикла.
"""
from __future__ import annotations

import json
import random
import time
from datetime import datetime, timedelta, timezone

from flask import current_app
from redis import Redis

from app.repositories import stats_repo
from app.services.ai_client import get_ai_client
from app.utils.logger import get_logger


logger = get_logger(__name__)

# МСК = UTC+3, без DST. Используем для красивых дат в контексте.
MSK = timezone(timedelta(hours=3))

# Периодичность генерации. Раз в час хорошо балансирует свежесть и затраты
# (1 chat-completion / час / компания ≈ копейки).
TICK_INTERVAL_SEC = 60 * 60

# TTL вдвое больше тика — если по какой-то причине следующий тик пропустится,
# факт всё ещё будет на табло, а не превратится в фолбэк.
_FACT_TTL_SEC = 2 * TICK_INTERVAL_SEC

_REDIS_KEY_FMT = "gw2:ai:tv_fact:{cid}"

_redis_client: Redis | None = None


def _redis() -> Redis:
    global _redis_client
    if _redis_client is None:
        _redis_client = Redis.from_url(current_app.config["REDIS_URL"],
                                       decode_responses=True)
    return _redis_client


# ──────────────────────────── контекст компании ────────────────────────────

def _now_msk() -> datetime:
    return datetime.now(MSK)


def _week_window_msk() -> tuple[datetime, datetime]:
    """Окно за последние 7 дней (включая сегодня) в МСК. Берём неделю, а
    не сегодняшний день: на свежей базе или после выходного сегодняшние
    цифры часто нулевые — тогда «контекст» вырождался в «закрыто 0,
    поступило 0», и модель скатывалась к воде."""
    now = _now_msk()
    end = now.replace(hour=23, minute=59, second=59, microsecond=0)
    start = (end - timedelta(days=6)).replace(hour=0, minute=0, second=0, microsecond=0)
    return start, end


def _context_for_company(company_id: int) -> dict:
    """Метрики недели для контекстного факта. Пустой dict — на ошибке."""
    try:
        start, end = _week_window_msk()
        common = stats_repo.get_common_metrics(start, end, company_id)
        employees = stats_repo.get_tasks_by_employees(start, end, company_id) or []
        depts = stats_repo.get_by_departments(start, end, company_id) or []
    except Exception as e:
        logger.warning("ai.tv_facts.context_failed",
                       extra={"company_id": company_id, "err": str(e)})
        return {}
    leader = employees[0] if employees else None
    top_dept = depts[0] if depts else None
    total_hours = sum((e.get("total_hours") or 0) for e in employees)
    return {
        "closed_week": common.get("closed", 0),
        "received_week": common.get("received", 0),
        "team_hours_week": round(total_hours, 1),
        "leader_fio": leader["fio"] if leader else None,
        "leader_hours": leader["total_hours"] if leader else None,
        "top_dept": top_dept["name"] if top_dept else None,
    }


# ──────────────────────────── промпты ────────────────────────────

_SYSTEM_PROMPT = (
    "Ты — короткий и остроумный спикер на корпоративном табло. "
    "Никаких эмодзи, кавычек и преамбул. Только сам факт, 1–2 предложения, "
    "до 220 символов, на русском."
)

_GENERAL_PROMPT = (
    "Сформулируй один интересный или забавный факт про работу, "
    "продуктивность, тайм-менеджмент или командное взаимодействие. "
    "Без банальностей, без воды."
)


def _context_prompt(ctx: dict) -> str:
    lines = [
        "Сделай короткое, живое наблюдение или вывод по статистике команды "
        "за последние 7 дней. Цифры можно округлять для красоты."
    ]
    if ctx.get("closed_week") is not None:
        lines.append(f"Закрыто задач за неделю: {ctx['closed_week']}.")
    if ctx.get("received_week") is not None:
        lines.append(f"Поступило задач за неделю: {ctx['received_week']}.")
    if ctx.get("team_hours_week") is not None:
        lines.append(f"Часы команды за неделю: {ctx['team_hours_week']}.")
    if ctx.get("leader_fio"):
        lines.append(f"Лидер недели — {ctx['leader_fio']} ({ctx['leader_hours']} ч).")
    if ctx.get("top_dept"):
        lines.append(f"Самый активный отдел — {ctx['top_dept']}.")
    return " ".join(lines)


def _has_meaningful_context(ctx: dict) -> bool:
    """Есть ли в контексте что-то, кроме нулей. Если нет — фолбэк на general."""
    if not ctx:
        return False
    return bool(
        (ctx.get("closed_week") or 0) > 0 or
        (ctx.get("received_week") or 0) > 0 or
        (ctx.get("team_hours_week") or 0) > 0
    )


def _pick_kind() -> str:
    return random.choice(("general", "context"))


# ──────────────────────────── генерация ────────────────────────────

def generate_fact(company_id: int) -> dict | None:
    """Генерирует факт и кладёт в Redis. Всегда честно дёргает модель —
    кэш разруливает уровень выше (TTL в Redis + интервал фонового цикла).

    Если для компании AI выключен — возвращает None и затирает кэш, чтобы
    табло сразу упало на фолбэк-слайд.
    """
    client = get_ai_client(company_id)
    if client is None:
        # выключили AI у компании — старый факт убираем, не показываем
        try:
            _redis().delete(_REDIS_KEY_FMT.format(cid=company_id))
        except Exception:
            pass
        return None

    kind = _pick_kind()
    user_prompt = _GENERAL_PROMPT
    if kind == "context":
        ctx = _context_for_company(company_id)
        if _has_meaningful_context(ctx):
            user_prompt = _context_prompt(ctx)
        else:
            # нулевая статистика — стыдно показывать «закрыто 0 задач».
            kind = "general"

    try:
        text = client.chat(
            messages=[
                {"role": "system", "content": _SYSTEM_PROMPT},
                {"role": "user", "content": user_prompt},
            ],
            max_tokens=180,
            temperature=0.9,
            timeout=20.0,
        )
    except Exception as e:
        logger.warning("ai.tv_facts.gen_failed",
                       extra={"company_id": company_id, "err": str(e)})
        return None

    text = text.strip().strip('"«»').strip()
    if not text:
        return None

    payload = {
        "text": text,
        "kind": kind,
        "generated_at": datetime.now(timezone.utc).isoformat(),
    }
    try:
        _redis().setex(_REDIS_KEY_FMT.format(cid=company_id), _FACT_TTL_SEC,
                       json.dumps(payload, ensure_ascii=False))
    except Exception as e:
        logger.warning("ai.tv_facts.redis_set_failed",
                       extra={"company_id": company_id, "err": str(e)})
    return payload


def get_fact(company_id: int) -> dict | None:
    try:
        raw = _redis().get(_REDIS_KEY_FMT.format(cid=company_id))
    except Exception:
        return None
    if not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return None


# ──────────────────────────── фоновый цикл ────────────────────────────

def regenerate_for_all_companies(app) -> None:
    """Один проход: догенерить факт для каждой компании с включённым AI."""
    from app.models.company import Company  # локально, чтобы избежать циклов
    with app.app_context():
        company_ids = [c.id for c in Company.query.filter_by(ai_enabled=True).all()]
    for cid in company_ids:
        with app.app_context():
            try:
                generate_fact(cid)
            except Exception as e:
                logger.warning("ai.tv_facts.iter_failed",
                               extra={"company_id": cid, "err": str(e)})


def run_tv_facts_loop(app) -> None:
    """Бесконечный цикл, поднимается через socketio.start_background_task.

    При старте — один проход для всех компаний (на случай, если в Redis
    ничего нет). Дальше — спать TICK_INTERVAL_SEC и повторять.
    """
    logger.info("ai.tv_facts.loop_start",
                extra={"interval_sec": TICK_INTERVAL_SEC})
    try:
        regenerate_for_all_companies(app)
    except Exception as e:
        logger.warning("ai.tv_facts.initial_pass_failed", extra={"err": str(e)})

    while True:
        try:
            time.sleep(TICK_INTERVAL_SEC)
        except Exception:
            return
        try:
            regenerate_for_all_companies(app)
        except Exception as e:
            logger.warning("ai.tv_facts.tick_failed", extra={"err": str(e)})

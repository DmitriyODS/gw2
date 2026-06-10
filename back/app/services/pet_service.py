"""Грувики: экономика грувов, кормление, эволюция, зоопарк, рейды.

Принципы здоровой геймификации:
- питомец никогда не деградирует и не умирает (отпуск — не наказание);
- у всех источников грувов дневные капы (грайндить ночью бессмысленно);
- награды привязаны к уже сделанной работе, а не стимулируют её имитацию.

Дневные счётчики — в Redis (`gw2:groove:daily:{user_id}:{дата МСК}`); при
недоступном Redis — fail-open (начисляем без капа, лучше так, чем ронять).
"""
from __future__ import annotations

import math
from datetime import date, datetime, timedelta, timezone

from flask import current_app
from redis import Redis

from app.extensions import db, socketio
from app.repositories import pet_repo, user_repo
from app.schemas.groove import PetSchema
from app.utils.logger import get_logger

logger = get_logger(__name__)

_pet_schema = PetSchema()

MSK = timezone(timedelta(hours=3))

# Накопительные пороги XP для стадий: яйцо → малыш → непоседа → подросток
# → взрослый → герой → легенда.
STAGE_XP = [0, 40, 120, 280, 550, 950, 1500]
MAX_STAGE = len(STAGE_XP) - 1

FEED_COST = 3
FEED_XP = 12
FEED_DAILY_MAX = 6

# Дневные капы грувов по источникам.
DAILY_CAPS = {
    "unit": 15,          # завершённые юниты
    "task_closed": 25,   # закрытые задачи
    "reaction": 10,      # полученные реакции
    "kudos": 10,         # полученные благодарности
    "zap": 10,           # полученные заряды
    "stroke_in": 5,      # моего питомца погладили
    "stroke_out": 5,     # я погладил чужого
}

# Магазин аксессуаров (эмодзи-маппинг — на фронте, utils/groove.js).
SHOP_PRICES = {
    "party": 30, "cap": 40, "bow": 40, "scarf": 50,
    "glasses": 60, "headphones": 60, "tophat": 80, "crown": 200,
}
RAID_REWARD_ITEM = "helmet"   # только за победу в рейде, не продаётся
RAID_WIN_BEANS = 15

STREAK_MILESTONES = {3, 5, 7, 10, 14, 21, 30, 50, 100}

BOSSES = ["Дедлайнозавр", "Багоблин", "Прокрастинатор",
          "Совещаниус", "Хаос-гоблин", "Технодолг"]

_redis_client: Redis | None = None


class PetServiceError(Exception):
    def __init__(self, message: str, code: str = "PET_ERROR", http_status: int = 400):
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


def _today_msk() -> date:
    return datetime.now(MSK).date()


# ─────────────────────── дневные бюджеты (Redis) ───────────────────

def _daily_key(user_id: int) -> str:
    return f"gw2:groove:daily:{user_id}:{_today_msk().isoformat()}"


def take_daily_budget(user_id: int, source: str, want: int, cap: int) -> int:
    """Сколько из `want` ещё помещается в дневной кап. Атомарно резервирует."""
    if want <= 0:
        return 0
    try:
        r = _redis()
        key = _daily_key(user_id)
        used = int(r.hget(key, source) or 0)
        granted = max(0, min(want, cap - used))
        if granted > 0:
            r.hincrby(key, source, granted)
            r.expire(key, 48 * 3600)
        return granted
    except Exception:
        return want   # Redis лёг — не наказываем пользователя


def _peek_daily(user_id: int, source: str) -> int:
    try:
        return int(_redis().hget(_daily_key(user_id), source) or 0)
    except Exception:
        return 0


# ─────────────────────────── сериализация ──────────────────────────

def dump_pet(pet) -> dict:
    data = _pet_schema.dump(pet)
    data["next_stage_xp"] = STAGE_XP[pet.stage + 1] if pet.stage < MAX_STAGE else None
    return data


def _emit_pet_update(pet) -> None:
    """Синхронизация питомца между вкладками владельца."""
    try:
        socketio.emit("pet:update", dump_pet(pet), room=f"user_{pet.user_id}")
    except Exception:
        pass


# ───────────────────────── начисление грувов ───────────────────────

def award_beans(user_id: int, company_id: int, source: str, amount: int) -> int:
    """Начислить грувы с учётом дневного капа источника. Никогда не бросает."""
    try:
        cap = DAILY_CAPS.get(source, 10)
        granted = take_daily_budget(user_id, source, amount, cap)
        if granted <= 0:
            return 0
        pet = pet_repo.get_or_create(user_id, company_id)
        pet.beans += granted
        db.session.commit()
        _emit_pet_update(pet)
        return granted
    except Exception as e:
        db.session.rollback()
        logger.warning("groove.award_failed",
                       extra={"extra": {"user_id": user_id, "source": source,
                                        "err": str(e)}})
        return 0


# ────────────────────────── питомец владельца ──────────────────────

def get_my_pet(user_id: int, company_id: int) -> dict:
    pet = pet_repo.get_or_create(user_id, company_id)
    db.session.commit()
    data = dump_pet(pet)
    data["feeds_left"] = max(0, FEED_DAILY_MAX - _peek_daily(user_id, "feeds"))
    return data


def _detect_species(user_id: int) -> str:
    """Характер по паттерну работы за 60 дней: сова/жаворонок/спринтер/марафонец."""
    since = datetime.now(timezone.utc) - timedelta(days=60)
    units = pet_repo.finished_units_for_user(user_id, since)
    if not units:
        return "fox"
    durations = []
    start_hours = []
    for u in units:
        durations.append((u.datetime_end - u.datetime_start).total_seconds() / 60)
        start_hours.append(u.datetime_start.astimezone(MSK).hour)
    avg = sum(durations) / len(durations)
    start_hours.sort()
    median_hour = start_hours[len(start_hours) // 2]
    if avg >= 100:
        return "marathoner"
    if avg <= 35 and len(units) >= 10:
        return "sprinter"
    if median_hour < 11:
        return "lark"
    if median_hour >= 17:
        return "owl"
    return "fox"


def feed_pet(user_id: int, company_id: int) -> dict:
    pet = pet_repo.get_or_create(user_id, company_id)
    if pet.beans < FEED_COST:
        raise PetServiceError("Не хватает грувов", "NO_BEANS", 422)
    if take_daily_budget(user_id, "feeds", 1, FEED_DAILY_MAX) <= 0:
        raise PetServiceError("Грувик сыт — приходите завтра", "FED_ENOUGH", 429)

    pet.beans -= FEED_COST
    pet.xp += FEED_XP

    today = _today_msk()
    streak_event = None
    if pet.last_fed_date != today:
        if pet.last_fed_date == today - timedelta(days=1):
            pet.feed_streak += 1
        else:
            pet.feed_streak = 1
        pet.last_fed_date = today
        if pet.feed_streak in STREAK_MILESTONES:
            streak_event = pet.feed_streak

    evolved_to = None
    while pet.stage < MAX_STAGE and pet.xp >= STAGE_XP[pet.stage + 1]:
        pet.stage += 1
        evolved_to = pet.stage
    if evolved_to is not None:
        pet.species = _detect_species(user_id)

    db.session.commit()

    from app.services.feed_service import record_event
    if streak_event is not None:
        record_event(company_id, user_id, "streak",
                     {"days": streak_event, "pet_name": pet.name},
                     bot_comment=True)
    if evolved_to is not None:
        record_event(company_id, user_id, "pet_evolved",
                     {"stage": evolved_to, "species": pet.species,
                      "pet_name": pet.name},
                     bot_comment=True)

    _emit_pet_update(pet)
    from app.services.groove_ai_service import get_feed_phrase
    data = dump_pet(pet)
    data["feeds_left"] = max(0, FEED_DAILY_MAX - _peek_daily(user_id, "feeds"))
    data["phrase"] = get_feed_phrase(company_id)
    data["evolved"] = evolved_to is not None
    return data


def rename_pet(user_id: int, company_id: int, name: str) -> dict:
    pet = pet_repo.get_or_create(user_id, company_id)
    pet.name = name.strip()
    db.session.commit()
    _emit_pet_update(pet)
    return dump_pet(pet)


def buy_item(user_id: int, company_id: int, item: str) -> dict:
    price = SHOP_PRICES.get(item)
    if price is None:
        raise PetServiceError("Такого товара нет", "NO_ITEM", 404)
    pet = pet_repo.get_or_create(user_id, company_id)
    owned = list(pet.accessories or [])
    if item in owned:
        raise PetServiceError("Уже куплено", "ALREADY_OWNED", 422)
    if pet.beans < price:
        raise PetServiceError("Не хватает грувов", "NO_BEANS", 422)
    pet.beans -= price
    pet.accessories = owned + [item]
    pet.hat = item
    db.session.commit()
    _emit_pet_update(pet)
    return dump_pet(pet)


def equip_item(user_id: int, company_id: int, item) -> dict:
    pet = pet_repo.get_or_create(user_id, company_id)
    if item is not None and item not in (pet.accessories or []):
        raise PetServiceError("Аксессуар не куплен", "NOT_OWNED", 422)
    pet.hat = item
    db.session.commit()
    _emit_pet_update(pet)
    return dump_pet(pet)


# ─────────────────────────── зоопарк ───────────────────────────────

def get_zoo(company_id: int, viewer_id: int) -> list[dict]:
    pets = pet_repo.list_company_pets(company_id)
    today = _today_msk()
    strokes = pet_repo.strokes_today([p.user_id for p in pets], today)
    my = pet_repo.my_strokes_today(viewer_id, today)
    result = []
    for p in pets:
        data = dump_pet(p)
        data["strokes_today"] = strokes.get(p.user_id, 0)
        data["stroked_by_me"] = p.user_id in my
        result.append(data)
    return result


def stroke_pet(viewer_id: int, target_user_id: int, company_id: int) -> dict:
    if viewer_id == target_user_id:
        raise PetServiceError("Своего Грувика гладьте сколько угодно — грувы за это не положены",
                              "SELF_STROKE", 422)
    target = user_repo.get_by_id(target_user_id)
    if target is None or target.is_hidden or target.company_id != company_id:
        raise PetServiceError("Сотрудник не найден", "USER_NOT_FOUND", 404)
    pet = pet_repo.get_or_create(target_user_id, company_id)
    if not pet_repo.add_stroke(target_user_id, viewer_id, _today_msk()):
        raise PetServiceError("Сегодня вы уже погладили этого Грувика", "ALREADY_STROKED", 422)
    db.session.commit()
    award_beans(target_user_id, company_id, "stroke_in", 1)
    award_beans(viewer_id, company_id, "stroke_out", 1)
    viewer = user_repo.get_by_id(viewer_id)
    try:
        socketio.emit("groove:stroke", {
            "from_fio": viewer.fio if viewer else "Коллега",
            "pet_name": pet.name,
        }, room=f"user_{target_user_id}")
    except Exception:
        pass
    today = _today_msk()
    return {"strokes_today": pet_repo.strokes_today([target_user_id], today)
                                     .get(target_user_id, 0)}


# ────────────────────────────── рейды ──────────────────────────────

def _week_start_msk(d: date | None = None) -> date:
    d = d or _today_msk()
    return d - timedelta(days=d.weekday())


def _msk_midnight(d: date) -> datetime:
    return datetime(d.year, d.month, d.day, tzinfo=MSK)


def _ensure_raid(company_id: int):
    week_start = _week_start_msk()
    raid = pet_repo.get_raid(company_id, week_start)
    if raid is not None:
        return raid
    prev_start = week_start - timedelta(days=7)
    prev_closed = pet_repo.count_closed_between(
        company_id, _msk_midnight(prev_start), _msk_midnight(week_start))
    target = max(10, int(math.ceil(prev_closed * 1.2 / 5.0)) * 5)
    boss = BOSSES[week_start.isocalendar()[1] % len(BOSSES)]
    raid = pet_repo.create_raid(company_id, week_start, boss, target,
                                RAID_REWARD_ITEM)
    db.session.commit()
    from app.services.feed_service import record_event
    record_event(company_id, None, "raid_started",
                 {"boss": boss, "target": target,
                  "week_start": week_start.isoformat()})
    return raid


def _raid_progress(company_id: int, raid) -> int:
    return pet_repo.count_closed_between(
        company_id, _msk_midnight(raid.week_start),
        datetime.now(timezone.utc) + timedelta(seconds=1))


def get_raid_state(company_id: int) -> dict:
    raid = _ensure_raid(company_id)
    progress = _raid_progress(company_id, raid)
    week_end = raid.week_start + timedelta(days=7)
    return {
        "id": raid.id,
        "boss": raid.boss,
        "target": raid.target,
        "progress": min(progress, raid.target) if raid.defeated_at else progress,
        "reward": raid.reward,
        "defeated": raid.defeated_at is not None,
        "week_start": raid.week_start.isoformat(),
        "days_left": max(0, (week_end - _today_msk()).days),
    }


def on_task_closed_raid(company_id: int) -> None:
    """Прогресс рейда после закрытия задачи. Никогда не бросает."""
    try:
        raid = _ensure_raid(company_id)
        progress = _raid_progress(company_id, raid)
        defeated_now = False
        if raid.defeated_at is None and progress >= raid.target:
            raid.defeated_at = datetime.now(timezone.utc)
            for pet in pet_repo.list_company_pets(company_id):
                pet.beans += RAID_WIN_BEANS
                owned = list(pet.accessories or [])
                if raid.reward not in owned:
                    pet.accessories = owned + [raid.reward]
            db.session.commit()
            defeated_now = True
            from app.services.feed_service import record_event
            record_event(company_id, None, "raid_won", {
                "boss": raid.boss, "target": raid.target,
                "reward": raid.reward, "beans": RAID_WIN_BEANS,
            }, bot_comment=True)
        try:
            socketio.emit("raid:update", {
                "company_id": company_id,
                "progress": progress,
                "target": raid.target,
                "boss": raid.boss,
                "defeated": raid.defeated_at is not None,
                "defeated_now": defeated_now,
            }, room="all")
        except Exception:
            pass
    except Exception as e:
        db.session.rollback()
        logger.warning("groove.raid_failed",
                       extra={"extra": {"company_id": company_id, "err": str(e)}})

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

# ── Ежедневный квест ───────────────────────────────────────────────
# Грувик каждый день предлагает простую дневную цель — мягкий якорь,
# чтобы было приятно начать день и закрыть один-два конкретных пункта.
# Награда за выполнение — бонус-грувы (поверх обычных капов).
QUEST_REWARD_BEANS = 20
QUEST_TEMPLATES = [
    {"kind": "tasks_closed",   "target": 2,
     "title": "Закрыть 2 задачи", "unit": "задач",
     "hint": "Грувик ждёт пару записей в архив — наш командный счётчик подскочит."},
    {"kind": "tasks_closed",   "target": 3,
     "title": "Закрыть 3 задачи", "unit": "задач",
     "hint": "Тройка закрытий — Грувик мяукает от восторга. Поехали!"},
    {"kind": "units_finished", "target": 3,
     "title": "Завершить 3 юнита", "unit": "юнитов",
     "hint": "Три полноценных подхода. Можно по 25–50 минут — удобно!"},
    {"kind": "unit_minutes",   "target": 60,
     "title": "60 минут в фокусе", "unit": "мин",
     "hint": "Один час спокойной работы. Грувик обещает вести себя тихо."},
    {"kind": "unit_minutes",   "target": 90,
     "title": "Полтора часа фокуса", "unit": "мин",
     "hint": "1ч30мин чистого времени. Один большой юнит или несколько — как удобнее."},
    {"kind": "feed_pet",       "target": 1,
     "title": "Покормить Грувика", "unit": "раз",
     "hint": "Не забудьте про талисмана — он заскучал."},
]

# ── Болезнь ────────────────────────────────────────────────────────
# Грувик заболевает, если хозяин SICK_AFTER_DAYS дней не завершал юниты.
# Лечение — recovery-очки: работа (юнит ≥15 мин, закрытая задача),
# «куриный бульон» (лечебное кормление) и забота коллег (поглаживания).
# XP и уровень при болезни не теряются — рост просто замораживается.
SICK_AFTER_DAYS = 5
RECOVERY_TARGET = 3
SICK_FEED_COST = 1
SICK_FEED_DAILY_MAX = 2
RECOVERY_MIN_UNIT_MINUTES = 15

# ── Характер ───────────────────────────────────────────────────────
# Пересчитывается по юнитам за 21 день (см. _detect_personality).
PERSONALITIES = {
    "lazy":      {"title": "Ленивец-мечтатель",
                  "hint": "работает редко, любит подремать и пофилософствовать"},
    "night":     {"title": "Ночной активист",
                  "hint": "оживает после заката, ночь — его стихия"},
    "early":     {"title": "Ранняя пташка",
                  "hint": "лучшие дела делает до обеда, бодрится с утра"},
    "energizer": {"title": "Бодрячок-энерджайзер",
                  "hint": "куча коротких подходов, энергия бьёт ключом"},
    "zen":       {"title": "Дзен-марафонец",
                  "hint": "длинные сосредоточенные сессии, спокоен как удав"},
    "steady":    {"title": "Уравновешенный трудяга",
                  "hint": "ровный стабильный ритм, надёжен и рассудителен"},
}

# Русские названия для AI-промптов (стадии ≡ фронтовым PET_STAGES).
PET_STAGES_TITLES = ["Яйцо", "Малыш", "Непоседа", "Подросток",
                     "Взрослый", "Герой", "Легенда"]
PET_SPECIES_TITLES = {
    "egg": "ещё не вылупившийся", "owl": "сова", "lark": "жаворонок",
    "sprinter": "спринтер", "marathoner": "марафонец", "fox": "лис-универсал",
    "cat": "котёнок", "dog": "щенок", "tiger": "тигрёнок", "bear": "медвежонок",
    "rabbit": "крольчонок", "frog": "лягушонок", "panda": "панда",
    "penguin": "пингвинёнок", "monkey": "обезьянка", "chick": "цыплёнок",
    "unicorn": "единорог", "dragon": "дракон",
}

# Магазин «видов» Грувика. Естественный вид (определённый эволюцией)
# всегда бесплатен и автоматически разблокирован; покупные виды дают
# возможность переключаться между обликами без потери стадии/XP.
SPECIES_SHOP = {
    "cat": 80, "dog": 80, "rabbit": 80, "frog": 80,
    "chick": 100, "monkey": 100, "panda": 120,
    "tiger": 140, "bear": 140, "penguin": 140,
    "unicorn": 250, "dragon": 250,
}
# Виды, которые проявляются «естественно» через _detect_species. Их нельзя
# купить — они приходят с эволюцией. Используются как стартовые в unlocked.
NATURAL_SPECIES = {"owl", "lark", "sprinter", "marathoner", "fox"}

# ── Сезонные товары ────────────────────────────────────────────────
# Аксессуар сезона продаётся только в свой сезон — повод заглянуть в магазин.
SEASONAL_ITEMS = {"flower": 45, "icecream": 45, "pumpkin": 45, "santa": 45}
_SEASON_BY_MONTH = {
    12: ("Зима", "santa"), 1: ("Зима", "santa"), 2: ("Зима", "santa"),
    3: ("Весна", "flower"), 4: ("Весна", "flower"), 5: ("Весна", "flower"),
    6: ("Лето", "icecream"), 7: ("Лето", "icecream"), 8: ("Лето", "icecream"),
    9: ("Осень", "pumpkin"), 10: ("Осень", "pumpkin"), 11: ("Осень", "pumpkin"),
}

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


def daily_left(user_id: int, source: str, cap: int) -> int:
    """Сколько осталось из дневного лимита источника (сбрасывается в полночь МСК)."""
    return max(0, cap - _peek_daily(user_id, source))


# ─────────────────────────── сериализация ──────────────────────────

def dump_pet(pet) -> dict:
    data = _pet_schema.dump(pet)
    data["next_stage_xp"] = STAGE_XP[pet.stage + 1] if pet.stage < MAX_STAGE else None
    data["sick"] = pet.sick_since is not None
    data["recovery"] = pet.recovery
    data["recovery_target"] = RECOVERY_TARGET
    data["personality"] = pet.personality
    data["personality_title"] = PERSONALITIES.get(pet.personality, {}).get("title")
    # Доступные облики: всё разблокированное + текущий вид (на случай,
    # если он ещё не был добавлен — старые питомцы до миграции).
    unlocked = list(pet.unlocked_species or [])
    if pet.species and pet.species not in unlocked and pet.species != "egg":
        unlocked.append(pet.species)
    data["unlocked_species"] = unlocked
    data["quest"] = _quest_snapshot(pet)
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
    if pet.personality is None:
        pet.personality = _detect_personality(user_id)
    _ensure_today_quest(pet)
    db.session.commit()
    data = dump_pet(pet)
    if pet.sick_since is not None:
        data["feeds_left"] = daily_left(user_id, "sick_feeds", SICK_FEED_DAILY_MAX)
        data["feeds_max"] = SICK_FEED_DAILY_MAX
    else:
        data["feeds_left"] = daily_left(user_id, "feeds", FEED_DAILY_MAX)
        data["feeds_max"] = FEED_DAILY_MAX
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


def _detect_personality(user_id: int) -> str:
    """Характер по юнитам за 21 день: ритм, время суток, длина сессий."""
    since = datetime.now(timezone.utc) - timedelta(days=21)
    units = pet_repo.finished_units_for_user(user_id, since, limit=200)
    if len(units) <= 2:
        return "lazy"
    per_week = len(units) / 3.0
    durations = []
    start_hours = []
    for u in units:
        durations.append((u.datetime_end - u.datetime_start).total_seconds() / 60)
        start_hours.append(u.datetime_start.astimezone(MSK).hour)
    avg = sum(durations) / len(durations)
    start_hours.sort()
    median_hour = start_hours[len(start_hours) // 2]
    if per_week <= 3:
        return "lazy"
    if median_hour >= 19:
        return "night"
    if median_hour < 10:
        return "early"
    if per_week >= 12 and avg <= 60:
        return "energizer"
    if avg >= 110:
        return "zen"
    return "steady"


# ───────────────────────────── болезнь ─────────────────────────────

def _apply_recovery(pet, amount: int = 1) -> bool:
    """Прибавить recovery-очки больному питомцу. True — выздоровел.
    Без commit'а — коммитит вызывающий."""
    if pet.sick_since is None:
        return False
    pet.recovery = min(RECOVERY_TARGET, pet.recovery + amount)
    if pet.recovery >= RECOVERY_TARGET:
        pet.sick_since = None
        pet.recovery = 0
        return True
    return False


def add_recovery(user_id: int, company_id: int, amount: int = 1) -> None:
    """Лечение работой/заботой. Никогда не бросает (зовётся из хуков)."""
    try:
        pet = pet_repo.get_pet(user_id)
        if pet is None or pet.sick_since is None:
            return
        recovered = _apply_recovery(pet, amount)
        db.session.commit()
        if recovered:
            from app.services.feed_service import record_event
            record_event(company_id, user_id, "pet_recovered",
                         {"pet_name": pet.name}, bot_comment=True)
        _emit_pet_update(pet)
    except Exception as e:
        db.session.rollback()
        logger.warning("groove.recovery_failed",
                       extra={"extra": {"user_id": user_id, "err": str(e)}})


def check_sickness_for_company(company_id: int) -> int:
    """Помечает больными питомцев тех, кто давно не работал. Возвращает
    число новых заболевших. Вызывается фоновым циклом заботы."""
    pets = pet_repo.list_company_pets(company_id)
    candidates = [p for p in pets if p.stage >= 1 and p.sick_since is None]
    if not candidates:
        return 0
    last_ends = pet_repo.last_unit_end_by_users([p.user_id for p in candidates])
    threshold = datetime.now(timezone.utc) - timedelta(days=SICK_AFTER_DAYS)
    sick_count = 0
    for pet in candidates:
        last = last_ends.get(pet.user_id)
        # Ни одного юнита в принципе — не наказываем (свежий пользователь).
        if last is None or last >= threshold:
            continue
        pet.sick_since = datetime.now(timezone.utc)
        pet.recovery = 0
        db.session.commit()
        sick_count += 1
        from app.services.feed_service import record_event
        record_event(company_id, pet.user_id, "pet_sick",
                     {"pet_name": pet.name}, bot_comment=True)
        _emit_pet_update(pet)
    return sick_count


def refresh_personalities_for_company(company_id: int) -> None:
    """Дневной пересчёт характеров всех питомцев компании."""
    for pet in pet_repo.list_company_pets(company_id):
        new = _detect_personality(pet.user_id)
        if new != pet.personality:
            pet.personality = new
    db.session.commit()


SICK_PHRASES = [
    "Апчхи… Спасибо за бульон. Кажется, мне уже чуточку лучше.",
    "Тёплый бульончик… Ещё бы пару закрытых задач — и я на ногах!",
    "Болею… Поработай немного — твоя энергия меня лечит.",
    "Кх-кх… Говорят, лучшее лекарство — завершённый юнит хозяина.",
]


def feed_pet(user_id: int, company_id: int) -> dict:
    pet = pet_repo.get_or_create(user_id, company_id)

    # Больного кормим лечебным бульоном: дёшево, без XP, +1 к выздоровлению.
    if pet.sick_since is not None:
        if pet.beans < SICK_FEED_COST:
            raise PetServiceError("Не хватает грувов даже на бульон", "NO_BEANS", 422)
        if take_daily_budget(user_id, "sick_feeds", 1, SICK_FEED_DAILY_MAX) <= 0:
            raise PetServiceError("Бульон — не больше двух мисок в день",
                                  "FED_ENOUGH", 429)
        pet.beans -= SICK_FEED_COST
        recovered = _apply_recovery(pet, 1)
        db.session.commit()
        if recovered:
            from app.services.feed_service import record_event
            record_event(company_id, user_id, "pet_recovered",
                         {"pet_name": pet.name}, bot_comment=True)
        _emit_pet_update(pet)
        import random as _random
        data = dump_pet(pet)
        # Выздоровел — счётчики сразу по «здоровой» шкале кормлений.
        if recovered:
            data["feeds_left"] = daily_left(user_id, "feeds", FEED_DAILY_MAX)
            data["feeds_max"] = FEED_DAILY_MAX
        else:
            data["feeds_left"] = daily_left(user_id, "sick_feeds", SICK_FEED_DAILY_MAX)
            data["feeds_max"] = SICK_FEED_DAILY_MAX
        data["phrase"] = ("Ура, я снова здоров! Спасибо, что выходил меня!"
                          if recovered else _random.choice(SICK_PHRASES))
        data["evolved"] = False
        data["recovered"] = recovered
        return data

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
        pet.personality = _detect_personality(user_id)
        unlocked = list(pet.unlocked_species or [])
        if pet.species not in unlocked:
            pet.unlocked_species = unlocked + [pet.species]

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
    # Кормление двигает дневной квест feed_pet, если такой выпал.
    bump_quest(user_id, "feed_pet", 1)
    from app.services.groove_ai_service import get_feed_phrase
    data = dump_pet(pet)
    data["feeds_left"] = daily_left(user_id, "feeds", FEED_DAILY_MAX)
    data["feeds_max"] = FEED_DAILY_MAX
    data["phrase"] = get_feed_phrase(company_id)
    data["evolved"] = evolved_to is not None
    return data


def rename_pet(user_id: int, company_id: int, name: str) -> dict:
    pet = pet_repo.get_or_create(user_id, company_id)
    pet.name = name.strip()
    db.session.commit()
    _emit_pet_update(pet)
    return dump_pet(pet)


def current_season() -> tuple[str, str]:
    """(заголовок сезона, key сезонного аксессуара) по текущему месяцу МСК."""
    return _SEASON_BY_MONTH[_today_msk().month]


def get_shop_state() -> dict:
    season_title, seasonal_item = current_season()
    return {
        "prices": {**SHOP_PRICES, seasonal_item: SEASONAL_ITEMS[seasonal_item]},
        "seasonal_item": seasonal_item,
        "season_title": season_title,
        "species_prices": dict(SPECIES_SHOP),
    }


def buy_item(user_id: int, company_id: int, item: str) -> dict:
    price = SHOP_PRICES.get(item)
    if price is None and item in SEASONAL_ITEMS:
        _, seasonal_item = current_season()
        if item != seasonal_item:
            raise PetServiceError("Этот аксессуар вернётся в свой сезон",
                                  "OUT_OF_SEASON", 422)
        price = SEASONAL_ITEMS[item]
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


def buy_species(user_id: int, company_id: int, species: str) -> dict:
    """Разблокировать новый облик Грувика и сразу его надеть."""
    price = SPECIES_SHOP.get(species)
    if price is None:
        raise PetServiceError("Такого вида в магазине нет", "NO_ITEM", 404)
    pet = pet_repo.get_or_create(user_id, company_id)
    unlocked = list(pet.unlocked_species or [])
    if species in unlocked:
        raise PetServiceError("Этот вид уже разблокирован", "ALREADY_OWNED", 422)
    if pet.beans < price:
        raise PetServiceError("Не хватает грувов", "NO_BEANS", 422)
    pet.beans -= price
    pet.unlocked_species = unlocked + [species]
    pet.species = species
    db.session.commit()
    _emit_pet_update(pet)
    return dump_pet(pet)


def switch_species(user_id: int, company_id: int, species: str) -> dict:
    """Сменить облик на ранее разблокированный (без оплаты)."""
    pet = pet_repo.get_or_create(user_id, company_id)
    unlocked = list(pet.unlocked_species or [])
    # Природный (определённый эволюцией) вид доступен всегда — он
    # автоматически считается «своим» даже если не лежит в unlocked.
    natural_ok = species in NATURAL_SPECIES and pet.stage >= 2
    if species not in unlocked and not natural_ok:
        raise PetServiceError("Этот вид ещё не разблокирован", "NOT_OWNED", 422)
    pet.species = species
    if species not in unlocked:
        pet.unlocked_species = unlocked + [species]
    db.session.commit()
    _emit_pet_update(pet)
    return dump_pet(pet)


# ─────────────────────────── квест дня ─────────────────────────────

def _pick_quest_template(user_id: int, day: date) -> dict:
    """Детерминированный выбор шаблона по (user_id, day): один и тот же
    день — тот же квест (стабильность при перезапросе)."""
    idx = (user_id * 1009 + day.toordinal()) % len(QUEST_TEMPLATES)
    return QUEST_TEMPLATES[idx]


def _ensure_today_quest(pet) -> None:
    """Назначает свежий квест, если предыдущий устарел. Без commit'а."""
    today = _today_msk()
    if pet.quest_date == today and pet.quest_kind:
        return
    tpl = _pick_quest_template(pet.user_id, today)
    pet.quest_date = today
    pet.quest_kind = tpl["kind"]
    pet.quest_target = int(tpl["target"])
    pet.quest_progress = 0
    pet.quest_claimed = False


def _quest_template_for(kind: str) -> dict | None:
    for t in QUEST_TEMPLATES:
        if t["kind"] == kind:
            return t
    return None


def _quest_snapshot(pet) -> dict | None:
    if not pet.quest_kind or not pet.quest_target:
        return None
    tpl = _quest_template_for(pet.quest_kind) or {}
    target = int(pet.quest_target)
    progress = min(int(pet.quest_progress or 0), target)
    return {
        "kind": pet.quest_kind,
        "title": tpl.get("title", "Дневной квест"),
        "hint": tpl.get("hint", ""),
        "unit": tpl.get("unit", ""),
        "target": target,
        "progress": progress,
        "done": progress >= target,
        "claimed": bool(pet.quest_claimed),
        "reward": QUEST_REWARD_BEANS,
    }


def bump_quest(user_id: int, kind: str, amount: int = 1) -> None:
    """Прибавить прогресс к дневному квесту, если совпадает по типу.
    Никогда не бросает (зовётся из хуков юнитов/задач)."""
    if amount <= 0:
        return
    try:
        pet = pet_repo.get_pet(user_id)
        if pet is None:
            return
        _ensure_today_quest(pet)
        if pet.quest_kind != kind or pet.quest_claimed:
            db.session.commit()
            return
        target = int(pet.quest_target or 0)
        was_done = (pet.quest_progress or 0) >= target
        pet.quest_progress = min(target, int(pet.quest_progress or 0) + amount)
        db.session.commit()
        now_done = (pet.quest_progress or 0) >= target
        if not was_done and now_done:
            _emit_pet_update(pet)   # сообщить владельцу — можно забрать награду
        else:
            _emit_pet_update(pet)
    except Exception as e:
        db.session.rollback()
        logger.warning("groove.quest_bump_failed",
                       extra={"extra": {"user_id": user_id, "kind": kind,
                                        "err": str(e)}})


def claim_quest(user_id: int, company_id: int) -> dict:
    pet = pet_repo.get_or_create(user_id, company_id)
    _ensure_today_quest(pet)
    target = int(pet.quest_target or 0)
    if pet.quest_claimed:
        raise PetServiceError("Награда уже забрана сегодня", "ALREADY_CLAIMED", 422)
    if (pet.quest_progress or 0) < target:
        raise PetServiceError("Квест ещё не выполнен", "NOT_DONE", 422)
    pet.quest_claimed = True
    pet.beans += QUEST_REWARD_BEANS
    db.session.commit()
    from app.services.feed_service import record_event
    tpl = _quest_template_for(pet.quest_kind) or {}
    record_event(company_id, user_id, "quest_done",
                 {"pet_name": pet.name, "title": tpl.get("title", "Квест дня"),
                  "reward": QUEST_REWARD_BEANS}, bot_comment=True)
    _emit_pet_update(pet)
    data = dump_pet(pet)
    return data


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
    # Забота лечит: поглаживание больного Грувика даёт ему очко выздоровления.
    add_recovery(target_user_id, company_id, 1)
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


# ──────────────────────── фоновый цикл заботы ──────────────────────

CARE_TICK_INTERVAL_SEC = 60 * 60


def run_groove_care_loop(app) -> None:
    """Раз в час: проверка болезней + дневной пересчёт характеров.
    Работает для ВСЕХ компаний (в отличие от AI-цикла — болезнь не
    требует включённого ИИ)."""
    import time as _time
    logger.info("groove.care.loop_start",
                extra={"extra": {"interval_sec": CARE_TICK_INTERVAL_SEC}})
    while True:
        try:
            _care_tick(app)
        except Exception as e:
            logger.warning("groove.care.tick_failed", extra={"extra": {"err": str(e)}})
        try:
            _time.sleep(CARE_TICK_INTERVAL_SEC)
        except Exception:
            return


def _care_tick(app) -> None:
    from app.models.company import Company
    with app.app_context():
        company_ids = [c.id for c in Company.query.filter_by(is_active=True).all()]
    for cid in company_ids:
        with app.app_context():
            try:
                check_sickness_for_company(cid)
                # Характеры пересчитываем раз в день (метка в Redis).
                key = f"gw2:groove:personality:{cid}:{_today_msk().isoformat()}"
                try:
                    r = _redis()
                    if not r.exists(key):
                        refresh_personalities_for_company(cid)
                        r.setex(key, 48 * 3600, "1")
                except Exception:
                    refresh_personalities_for_company(cid)
            except Exception as e:
                db.session.rollback()
                logger.warning("groove.care.company_failed",
                               extra={"extra": {"company_id": cid, "err": str(e)}})


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

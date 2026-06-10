"""Инструменты Грувика для запроса статистики через function-calling.

Подключаются к `AIClient.chat_with_tools` из `_make_pet_reply` в чате с
Грувиком. Все запросы жёстко скоупятся по `company_id` владельца — Грувик
видит только данные своей компании, чужие компании невидимы в принципе.

Дизайн:
- `TOOL_SCHEMAS` — список OpenAI-совместимых tool-definition'ов.
- `dispatch(name, args, company_id, user_id)` — ядро: запускает нужную
  функцию, возвращает компактный dict (LLM пережёвывает 1–2 КБ ответа без
  труда — большие списки режутся `limit`).
- Все периоды нормализуются в `_resolve_period(period)` → один формат
  (`start_utc`, `end_utc`, `label`). Поддерживаются дружелюбные алиасы:
  today / yesterday / this_week / last_week / this_month / last_month / 7d / 30d.

Любая внутренняя ошибка возвращается как `{"error": "..."}` — LLM сам
адекватно реагирует на сбой, а Грувик не молчит.
"""
from __future__ import annotations

from datetime import datetime, timedelta, timezone
from typing import Any

from app.repositories import stats_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)

MSK = timezone(timedelta(hours=3))

# Максимум строк, которые отдаём LLM в любом списочном ответе.
_LIST_LIMIT_DEFAULT = 10
_LIST_LIMIT_MAX = 30


# ───────────────────────── периоды ─────────────────────────

def _today_msk() -> datetime:
    now = datetime.now(MSK)
    return now.replace(hour=0, minute=0, second=0, microsecond=0)


def _to_utc(dt: datetime) -> datetime:
    return dt.astimezone(timezone.utc)


def _resolve_period(period: str | None) -> dict:
    """Превратить дружелюбный код периода в окно `start/end` (UTC) + ярлык."""
    code = (period or "this_week").strip().lower()
    today = _today_msk()
    # начало недели — понедельник 00:00 МСК
    week_start = today - timedelta(days=today.weekday())
    month_start = today.replace(day=1)

    if code == "today":
        start, end, label = today, today + timedelta(days=1), "сегодня"
    elif code == "yesterday":
        start, end, label = today - timedelta(days=1), today, "вчера"
    elif code in ("this_week", "week"):
        start, end, label = week_start, week_start + timedelta(days=7), "эта неделя"
    elif code == "last_week":
        start = week_start - timedelta(days=7)
        end, label = week_start, "прошлая неделя"
    elif code in ("this_month", "month"):
        if month_start.month == 12:
            next_month = month_start.replace(year=month_start.year + 1, month=1)
        else:
            next_month = month_start.replace(month=month_start.month + 1)
        start, end, label = month_start, next_month, "этот месяц"
    elif code == "last_month":
        end = month_start
        if month_start.month == 1:
            start = month_start.replace(year=month_start.year - 1, month=12)
        else:
            start = month_start.replace(month=month_start.month - 1)
        label = "прошлый месяц"
    elif code in ("7d", "7days", "last_7_days"):
        start, end, label = today - timedelta(days=6), today + timedelta(days=1), "последние 7 дней"
    elif code in ("30d", "30days", "last_30_days"):
        start, end, label = today - timedelta(days=29), today + timedelta(days=1), "последние 30 дней"
    else:
        start, end, label = week_start, week_start + timedelta(days=7), "эта неделя"

    return {
        "start_utc": _to_utc(start),
        "end_utc": _to_utc(end),
        "label": label,
        "code": code,
    }


_PERIOD_ENUM = [
    "today", "yesterday", "this_week", "last_week",
    "this_month", "last_month", "7d", "30d",
]


# ───────────────────────── tool schemas ────────────────────

TOOL_SCHEMAS: list[dict] = [
    {
        "type": "function",
        "function": {
            "name": "get_stats_summary",
            "description": (
                "Общие метрики компании за период: поступило задач, закрыто, "
                "ещё в работе (на конец периода), долг до периода, суммарные часы команды. "
                "Использовать для ответа на вопросы типа «сколько задач поступило/закрыто», "
                "«сколько часов отработали», «как у нас дела за неделю»."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "period": {
                        "type": "string",
                        "enum": _PERIOD_ENUM,
                        "description": "Период статистики. По умолчанию this_week.",
                    },
                },
                "additionalProperties": False,
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "list_departments",
            "description": (
                "Список всех отделов компании с количеством поступивших задач "
                "за указанный период. Полезно когда пользователь упоминает отдел "
                "по названию или спрашивает «какие отделы у нас есть»."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "period": {
                        "type": "string",
                        "enum": _PERIOD_ENUM,
                        "description": "Период для счётчиков. По умолчанию this_week.",
                    },
                },
                "additionalProperties": False,
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_top_employees",
            "description": (
                "Топ сотрудников компании по отработанному времени за период. "
                "Возвращает ФИО, кол-во разных задач и суммарные часы. "
                "Использовать для вопросов «кто больше всего работал», «лидеры недели»."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "period": {"type": "string", "enum": _PERIOD_ENUM},
                    "limit": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": _LIST_LIMIT_MAX,
                        "description": f"Сколько строк вернуть (1..{_LIST_LIMIT_MAX}). По умолчанию {_LIST_LIMIT_DEFAULT}.",
                    },
                },
                "additionalProperties": False,
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_stats_by_unit_types",
            "description": (
                "Распределение работы по типам юнитов (звонок, разработка, "
                "встреча и т. п.) за период: часы и количество задач."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "period": {"type": "string", "enum": _PERIOD_ENUM},
                },
                "additionalProperties": False,
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_stats_calendar",
            "description": (
                "Динамика по дням за период: на каждый день — поступило, закрыто "
                "и часы. Используй для вопросов «как менялось», «в какой день "
                "было больше всего работы»."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "period": {"type": "string", "enum": _PERIOD_ENUM},
                },
                "additionalProperties": False,
            },
        },
    },
]


# ───────────────────────── реализация ──────────────────────

def _round_hours(value: Any) -> float:
    try:
        return round(float(value or 0), 1)
    except (TypeError, ValueError):
        return 0.0


def _summary(args: dict, company_id: int) -> dict:
    period = _resolve_period(args.get("period"))
    common = stats_repo.get_common_metrics(period["start_utc"], period["end_utc"], company_id)
    by_emp = stats_repo.get_tasks_by_employees(period["start_utc"], period["end_utc"], company_id) or []
    total_hours = sum(_round_hours(e.get("total_hours")) for e in by_emp)
    return {
        "period": period["label"],
        "received": common.get("received", 0),
        "closed": common.get("closed", 0),
        "in_progress_now": common.get("remaining", 0),
        "debt_before_period": common.get("debt", 0),
        "team_hours": round(total_hours, 1),
    }


def _list_departments(args: dict, company_id: int) -> dict:
    period = _resolve_period(args.get("period"))
    rows = stats_repo.get_by_departments(period["start_utc"], period["end_utc"], company_id) or []
    return {
        "period": period["label"],
        "departments": [
            {"id": r["dept_id"], "name": r["name"], "received_count": r["tasks_count"]}
            for r in rows
        ],
    }


def _top_employees(args: dict, company_id: int) -> dict:
    period = _resolve_period(args.get("period"))
    raw_limit = args.get("limit") or _LIST_LIMIT_DEFAULT
    try:
        limit = max(1, min(_LIST_LIMIT_MAX, int(raw_limit)))
    except (TypeError, ValueError):
        limit = _LIST_LIMIT_DEFAULT
    rows = stats_repo.get_tasks_by_employees(period["start_utc"], period["end_utc"], company_id) or []
    return {
        "period": period["label"],
        "employees": [
            {
                "fio": r["fio"],
                "tasks_count": r["tasks_count"],
                "hours": _round_hours(r["total_hours"]),
            }
            for r in rows[:limit]
        ],
    }


def _by_unit_types(args: dict, company_id: int) -> dict:
    period = _resolve_period(args.get("period"))
    rows = stats_repo.get_by_unit_types(period["start_utc"], period["end_utc"], company_id) or []
    return {
        "period": period["label"],
        "unit_types": [
            {
                "name": r["name"],
                "hours": _round_hours(r["total_hours"]),
                "tasks_count": r["tasks_count"],
            }
            for r in rows[:_LIST_LIMIT_MAX]
        ],
    }


def _calendar(args: dict, company_id: int) -> dict:
    period = _resolve_period(args.get("period"))
    rows = stats_repo.get_calendar(period["start_utc"], period["end_utc"], company_id) or []
    return {
        "period": period["label"],
        "days": [
            {
                "date": r["date"],
                "received": r["received"],
                "closed": r["closed"],
                "hours": _round_hours(r["total_hours"]),
            }
            for r in rows
        ],
    }


_HANDLERS = {
    "get_stats_summary": _summary,
    "list_departments": _list_departments,
    "get_top_employees": _top_employees,
    "get_stats_by_unit_types": _by_unit_types,
    "get_stats_calendar": _calendar,
}


def dispatch(name: str, args: dict | None, *, company_id: int, user_id: int | None = None) -> dict:
    """Запустить инструмент по имени и вернуть его результат как dict.

    Любой сбой репозитория или неизвестное имя → `{"error": "..."}`.
    """
    handler = _HANDLERS.get(name)
    if handler is None:
        return {"error": f"unknown_tool:{name}"}
    try:
        return handler(args or {}, company_id)
    except Exception as e:
        logger.warning(
            "groove.ai.tool_failed",
            extra={"extra": {"tool": name, "company_id": company_id,
                             "user_id": user_id, "err": str(e)}},
        )
        return {"error": "tool_execution_failed"}

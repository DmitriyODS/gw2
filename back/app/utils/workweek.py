"""Выходные дни компании (Company.settings.weekend_days, 0=Пн … 6=Вс)."""
from __future__ import annotations

from datetime import date, timedelta

from app.extensions import db

DEFAULT_WEEKEND = (5, 6)   # суббота и воскресенье


def weekend_days(company_id: int | None) -> set[int]:
    """Множество выходных дней недели компании. На любой мусор в настройках
    или отсутствие компании отвечает дефолтом Сб+Вс."""
    from app.models.company import Company, get_setting
    if company_id is None:
        return set(DEFAULT_WEEKEND)
    company = db.session.get(Company, company_id)
    raw = get_setting(company, "weekend_days", list(DEFAULT_WEEKEND))
    try:
        return {int(d) for d in (raw or []) if 0 <= int(d) <= 6}
    except (TypeError, ValueError):
        return set(DEFAULT_WEEKEND)


def is_weekend(d: date, weekend: set[int]) -> bool:
    return d.weekday() in weekend


def working_days_between(start: date, end: date, weekend: set[int]) -> int:
    """Число рабочих дней в интервале (start, end]."""
    if end <= start or len(weekend) >= 7:
        return 0
    n = 0
    d = start
    while d < end:
        d += timedelta(days=1)
        if d.weekday() not in weekend:
            n += 1
    return n

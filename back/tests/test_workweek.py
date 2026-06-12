"""Тесты утилиты рабочих дней (app/utils/workweek.py)."""
from datetime import date

from app.utils.workweek import is_weekend, working_days_between


def test_working_days_between_skips_weekend():
    # Пт 2026-06-05 → Пт 2026-06-12: Сб и Вс выпадают, остаётся 5 рабочих.
    assert working_days_between(date(2026, 6, 5), date(2026, 6, 12), {5, 6}) == 5


def test_working_days_between_without_weekend_counts_all():
    assert working_days_between(date(2026, 6, 1), date(2026, 6, 8), set()) == 7


def test_working_days_between_all_days_off():
    assert working_days_between(date(2026, 6, 1), date(2026, 6, 30), set(range(7))) == 0


def test_working_days_between_empty_interval():
    assert working_days_between(date(2026, 6, 5), date(2026, 6, 5), {5, 6}) == 0


def test_is_weekend():
    assert is_weekend(date(2026, 6, 13), {5, 6})       # суббота
    assert not is_weekend(date(2026, 6, 12), {5, 6})   # пятница

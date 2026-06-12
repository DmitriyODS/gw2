"""Тест утреннего брифинга Грувика (GET /api/groove/morning).

Создаём в dev-БД давно «висящую» задачу на реальном сотруднике и проверяем,
что брифинг её подсвечивает: персональная сводка + настроение питомца +
реплика. AI в тестах обычно выключен — проверяем статичный фолбэк.
"""
from datetime import datetime, timedelta, timezone

import pytest


def _auth_headers(app, user_id):
    from conftest import make_token
    with app.app_context():
        from app.repositories.user_repo import get_by_id
        u = get_by_id(user_id)
        claims = {
            "company_id": u.company_id,
            "company_name": u.company.name if u.company else None,
            "role_level": u.role.level if u.role else 0,
            "is_root_admin": bool(u.is_root_admin),
        }
    token = make_token(app, user_id, claims)
    return {"Authorization": f"Bearer {token}"}


def _employee_with_department(app):
    """Сотрудник компании + отдел этой компании (для создания задачи)."""
    from app.extensions import db
    from app.models import User, Department
    with app.app_context():
        row = db.session.execute(
            db.select(User.id, User.company_id).where(
                User.is_hidden.is_(False),
                User.company_id.isnot(None),
            ).order_by(User.id)
        ).first()
        if row is None:
            return None
        user_id, company_id = row
        dept_id = db.session.execute(
            db.select(Department.id).where(Department.company_id == company_id).limit(1)
        ).scalar_one_or_none()
        if dept_id is None:
            return None
        return user_id, company_id, dept_id


def _set_weekend_days(app, company_id, days):
    """Задаёт выходные компании, возвращает прежний settings для отката."""
    from app.extensions import db
    from app.models.company import Company
    with app.app_context():
        company = db.session.get(Company, company_id)
        old_settings = dict(company.settings or {})
        company.settings = {**old_settings, "weekend_days": days}
        db.session.commit()
        return old_settings


def _restore_settings(app, company_id, settings):
    from app.extensions import db
    from app.models.company import Company
    with app.app_context():
        company = db.session.get(Company, company_id)
        company.settings = settings
        db.session.commit()


def test_morning_briefing_highlights_stale_task(app):
    ctx = _employee_with_department(app)
    if ctx is None:
        pytest.skip("Нет сотрудника с отделом для теста брифинга")
    user_id, company_id, dept_id = ctx

    from app.extensions import db
    from app.repositories import task_repo

    # Без выходных — рабочий режим брифинга детерминирован в любой день недели.
    old_settings = _set_weekend_days(app, company_id, [])

    with app.app_context():
        task = task_repo.create(
            name="ТЕСТ: древняя задача брифинга",
            author_id=user_id,
            department_id=dept_id,
            company_id=company_id,
            responsible_user_id=user_id,
            received_at=datetime.now(timezone.utc) - timedelta(days=3650),
        )
        task_id = task.id
        db.session.commit()

    try:
        client = app.test_client()
        resp = client.get("/api/groove/morning?part=morning",
                          headers=_auth_headers(app, user_id))
        assert resp.status_code == 200
        data = resp.get_json()

        assert data["show"] is True
        assert data["greeting"] == "Доброе утро"
        assert data["open_count"] >= 1
        assert data["stale_count"] >= 1
        # Самая старая задача (наша 10-летняя) — первой в списке.
        assert data["stale"][0]["id"] == task_id
        assert data["stale"][0]["days_pending"] >= 3000
        assert isinstance(data["message"], str) and data["message"].strip()
        assert data["mood"] in ("sick", "buried", "reminder", "fresh")
        assert data["pet"]["name"]
    finally:
        with app.app_context():
            from app.models import Task
            db.session.execute(db.delete(Task).where(Task.id == task_id))
            db.session.commit()
        _restore_settings(app, company_id, old_settings)


def test_morning_briefing_weekend_suggests_rest(app):
    """В выходной компании Грувик не пилит за задачи: mood=weekend, список
    засидевшихся пуст, реплика зовёт отдыхать."""
    ctx = _employee_with_department(app)
    if ctx is None:
        pytest.skip("Нет сотрудника с отделом для теста брифинга")
    user_id, company_id, _ = ctx

    from app.services.pet_service import MSK
    today_wd = datetime.now(MSK).date().weekday()
    old_settings = _set_weekend_days(app, company_id, [today_wd])

    try:
        client = app.test_client()
        resp = client.get("/api/groove/morning?part=morning",
                          headers=_auth_headers(app, user_id))
        assert resp.status_code == 200
        data = resp.get_json()

        assert data["show"] is True
        assert data["mood"] == "weekend"
        assert data["stale"] == []
        assert data["stale_count"] == 0
        assert isinstance(data["message"], str) and data["message"].strip()
    finally:
        _restore_settings(app, company_id, old_settings)


def test_morning_briefing_part_maps_greeting(app):
    ctx = _employee_with_department(app)
    if ctx is None:
        pytest.skip("Нет сотрудника с отделом для теста брифинга")
    user_id = ctx[0]
    client = app.test_client()
    resp = client.get("/api/groove/morning?part=evening",
                      headers=_auth_headers(app, user_id))
    assert resp.status_code == 200
    data = resp.get_json()
    # show может быть False (нет задач), но при show=True приветствие — вечернее.
    if data.get("show"):
        assert data["greeting"] == "Добрый вечер"

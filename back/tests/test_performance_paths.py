"""Точечные проверки на батчинг и лишние запросы в горячих путях."""
from contextlib import contextmanager
from datetime import datetime, timezone

from sqlalchemy import event

from tests.conftest import make_token


@contextmanager
def _count_queries(app):
    from app.extensions import db

    statements = []

    def _before_cursor_execute(conn, cursor, statement, parameters, context, executemany):
        statements.append(statement)

    with app.app_context():
        event.listen(db.engine, "before_cursor_execute", _before_cursor_execute)
        try:
            yield statements
        finally:
            event.remove(db.engine, "before_cursor_execute", _before_cursor_execute)


def _root_admin_id(app):
    from app.extensions import db
    from app.models import User
    with app.app_context():
        return db.session.execute(
            db.select(User.id).where(User.is_root_admin.is_(True)).limit(1)
        ).scalar_one()


def _company_user_id(app):
    from app.extensions import db
    from app.models import User
    with app.app_context():
        return db.session.execute(
            db.select(User.id).where(
                User.is_hidden.is_(False),
                User.company_id.isnot(None),
            ).limit(1)
        ).scalar_one()


def test_common_stats_is_single_query(app):
    from app.extensions import db
    from app.models import Company
    from app.repositories import stats_repo

    with app.app_context():
        company_id = db.session.execute(
            db.select(Company.id).limit(1)
        ).scalar_one()
        year = datetime.now(timezone.utc).year
        period_start = datetime(year, 1, 1, tzinfo=timezone.utc)
        period_end = datetime(year, 12, 31, 23, 59, 59, tzinfo=timezone.utc)

        with _count_queries(app) as statements:
            data = stats_repo.get_common_metrics(period_start, period_end, company_id)

    assert len(statements) == 1
    assert set(data.keys()) == {"debt", "received", "closed", "remaining"}


def test_company_list_uses_batched_stats(app, monkeypatch):
    from app.repositories import company_repo

    admin_id = _root_admin_id(app)
    calls = {"count": 0}
    original = company_repo.stats_by_company_ids

    def wrapped(company_ids):
        calls["count"] += 1
        return original(company_ids)

    monkeypatch.setattr(company_repo, "stats_by_company_ids", wrapped)

    client = app.test_client()
    resp = client.get("/api/companies", headers={"Authorization": f"Bearer {make_token(app, admin_id)}"})

    assert resp.status_code == 200
    assert calls["count"] == 1


def test_tasks_list_reads_yougile_flag_once(app, monkeypatch):
    import app.api.tasks as tasks_api

    user_id = _company_user_id(app)
    calls = {"count": 0}
    original = tasks_api._yougile_enabled

    def wrapped(company_id):
        calls["count"] += 1
        return original(company_id)

    monkeypatch.setattr(tasks_api, "_yougile_enabled", wrapped)

    client = app.test_client()
    resp = client.get("/api/tasks?per_page=1", headers={"Authorization": f"Bearer {make_token(app, user_id)}"})

    assert resp.status_code == 200
    assert calls["count"] == 1

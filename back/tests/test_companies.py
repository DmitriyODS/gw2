"""Smoke-тест Companies CRUD + блокировки отключённой компании."""
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


def _root_admin_id(app):
    from app.extensions import db
    from app.models import User
    with app.app_context():
        return db.session.execute(
            db.select(User.id).where(User.is_root_admin.is_(True)).limit(1)
        ).scalar_one()


def test_companies_list_requires_admin(app):
    # Любая не-admin роль (берём первого сотрудника) → 403.
    from app.extensions import db
    from app.models import User
    with app.app_context():
        non_admin = db.session.execute(
            db.select(User.id).where(
                User.is_hidden.is_(False),
                User.company_id.isnot(None),
            ).limit(1)
        ).scalar_one()
    client = app.test_client()
    resp = client.get("/api/companies", headers=_auth_headers(app, non_admin))
    assert resp.status_code == 403


def test_companies_list_for_root_admin(app):
    admin_id = _root_admin_id(app)
    client = app.test_client()
    resp = client.get("/api/companies", headers=_auth_headers(app, admin_id))
    assert resp.status_code == 200
    data = resp.get_json()
    assert "items" in data and len(data["items"]) >= 1
    # Проверяем enriched-поля
    first = data["items"][0]
    assert "employees_count" in first
    assert "tasks_count" in first


def test_company_disabled_blocks_company_user(app):
    """Если выключить компанию — её сотрудник получает 403 COMPANY_DISABLED."""
    from app.extensions import db
    from app.models import Company, User
    from app.services import company_service
    with app.app_context():
        company = db.session.execute(
            db.select(Company).where(Company.is_active.is_(True)).limit(1)
        ).scalar_one()
        # Берём её сотрудника (любого, не админ системы).
        member_id = db.session.execute(
            db.select(User.id).where(
                User.company_id == company.id, User.is_hidden.is_(False),
            ).limit(1)
        ).scalar_one()
        company_id = company.id
        company_service.set_active(company_id, False)
    try:
        client = app.test_client()
        resp = client.get("/api/tasks", headers=_auth_headers(app, member_id))
        assert resp.status_code == 403
    finally:
        with app.app_context():
            company_service.set_active(company_id, True)

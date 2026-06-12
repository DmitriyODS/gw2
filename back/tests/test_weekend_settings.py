"""Тесты выходных дней компании (GET/PUT /api/companies/<id>/weekend-settings).

Руководитель меняет выходные своей компании; сотрудник получает 403 по роли.
"""
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


def _user_with_level(app, min_level=None, max_level=None):
    """Видимый пользователь компании с ролью в заданных границах."""
    from app.extensions import db
    from app.models import User, Role
    with app.app_context():
        q = db.select(User.id, User.company_id).join(Role, User.role_id == Role.id).where(
            User.is_hidden.is_(False),
            User.company_id.isnot(None),
        )
        if min_level is not None:
            q = q.where(Role.level >= min_level)
        if max_level is not None:
            q = q.where(Role.level <= max_level)
        row = db.session.execute(q.order_by(User.id)).first()
        return (row[0], row[1]) if row else None


def test_weekend_settings_roundtrip_as_director(app):
    found = _user_with_level(app, min_level=3)
    if found is None:
        pytest.skip("Нет Руководителя с компанией для теста")
    user_id, company_id = found
    headers = _auth_headers(app, user_id)
    client = app.test_client()

    old = client.get(f"/api/companies/{company_id}/weekend-settings", headers=headers)
    assert old.status_code == 200
    old_days = old.get_json()["weekend_days"]

    try:
        resp = client.put(f"/api/companies/{company_id}/weekend-settings",
                          headers=headers, json={"weekend_days": [4, 5, 6]})
        assert resp.status_code == 200
        assert resp.get_json()["weekend_days"] == [4, 5, 6]

        again = client.get(f"/api/companies/{company_id}/weekend-settings", headers=headers)
        assert again.get_json()["weekend_days"] == [4, 5, 6]

        bad = client.put(f"/api/companies/{company_id}/weekend-settings",
                         headers=headers, json={"weekend_days": [7]})
        assert bad.status_code == 400
    finally:
        client.put(f"/api/companies/{company_id}/weekend-settings",
                   headers=headers, json={"weekend_days": old_days})


def test_weekend_settings_forbidden_for_employee(app):
    found = _user_with_level(app, max_level=2)
    if found is None:
        pytest.skip("Нет сотрудника/менеджера для теста")
    user_id, company_id = found
    resp = app.test_client().get(
        f"/api/companies/{company_id}/weekend-settings",
        headers=_auth_headers(app, user_id))
    assert resp.status_code == 403

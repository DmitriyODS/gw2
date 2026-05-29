"""Общие фикстуры для интеграционных тестов.

Поднимаем настоящее Flask-приложение (фабрика create_app) поверх dev-БД и
Redis. Если они недоступны — интеграционные тесты, которым нужен `app`,
автоматически пропускаются (skip), а чистые юнит-тесты call_state работают и
без них.
"""
import os

import pytest

# Переменные окружения dev-стенда (pytest не читает .flaskenv сам).
os.environ.setdefault("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
os.environ.setdefault("REDIS_URL", "redis://localhost:6379/0")
os.environ.setdefault("JWT_SECRET_KEY", "dev-jwt-secret-key-min-32-chars-local-xxxx")
os.environ.setdefault("SECRET_KEY", "dev-flask-secret-key-min-32-chars-local-xxxx")
os.environ.setdefault("UPLOAD_FOLDER", "./uploads")
# Без grace-задержки на отдельный поток в тестах ждать не нужно — но оставим
# дефолт; тест rejoin не зависит от истечения окна.


@pytest.fixture(scope="session")
def app():
    try:
        from app import create_app
        application = create_app("production")
    except Exception as e:  # noqa: BLE001
        pytest.skip(f"Не удалось создать приложение (нет БД/Redis?): {e}")
    # Проверим доступность БД.
    from app.extensions import db
    try:
        with application.app_context():
            db.session.execute(db.text("SELECT 1"))
    except Exception as e:  # noqa: BLE001
        pytest.skip(f"БД недоступна: {e}")
    return application


@pytest.fixture
def two_users(app):
    """Берём двух реально существующих не скрытых пользователей из dev-БД."""
    from app.extensions import db
    from app.models import User
    with app.app_context():
        rows = db.session.execute(
            db.select(User.id).where(User.is_hidden.is_(False)).order_by(User.id).limit(2)
        ).scalars().all()
    if len(rows) < 2:
        pytest.skip("В БД меньше двух пользователей для теста звонка")
    return rows[0], rows[1]


def make_token(app, user_id: int) -> str:
    from flask_jwt_extended import create_access_token
    with app.app_context():
        return create_access_token(identity=str(user_id))

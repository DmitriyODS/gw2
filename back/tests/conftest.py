"""Общие фикстуры для интеграционных тестов.

Поднимаем настоящее Flask-приложение (фабрика create_app) поверх dev-БД и
Redis. Если они недоступны — тесты, которым нужен `app`, автоматически
пропускаются (skip), а чистые юнит-тесты (yougile и т. п.) работают и без них.
Go-микросервис звонков не нужен: шлюз проверяется против in-process
fake gRPC-сервера (см. test_call_flow.py).
"""
import os

import pytest

# Переменные окружения dev-стенда (pytest не читает .flaskenv сам).
os.environ.setdefault("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
os.environ.setdefault("REDIS_URL", "redis://localhost:6379/0")
os.environ.setdefault("SECRET_KEY", "dev-flask-secret-key-min-32-chars-local-xxxx")
os.environ.setdefault("UPLOAD_FOLDER", "./uploads")

# Dev-ключи PASETO (та же пара, что в dev.sh/.flaskenv): токены выпускает
# authsvc, но тестам он не нужен — подписываем сами приватным dev-ключом.
DEV_PASETO_SEED_HEX = "68eb779b2f672beb8fcd58d72a81ce1565a1417aed3788d1362bf4faaa3f62ac"
DEV_PASETO_PUBLIC_HEX = "15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1"
os.environ.setdefault("PASETO_PUBLIC_KEY", DEV_PASETO_PUBLIC_HEX)


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
    """Берём двух реально существующих не скрытых пользователей из ОДНОЙ компании
    (multi-tenancy v3.0: звонки запрещены между разными компаниями)."""
    from app.extensions import db
    from app.models import User
    with app.app_context():
        rows = db.session.execute(
            db.select(User.id).where(
                User.is_hidden.is_(False),
                User.company_id.isnot(None),
            ).order_by(User.company_id, User.id).limit(2)
        ).scalars().all()
    if len(rows) < 2:
        pytest.skip("В БД меньше двух сотрудников компании для теста звонка")
    return rows[0], rows[1]


def make_token(app, user_id: int, claims: dict | None = None) -> str:
    """PASETO v4.public access-токен, как его выпускает authsvc."""
    import json
    from datetime import datetime, timedelta, timezone

    import pyseto
    from pyseto import Key

    now = datetime.now(timezone.utc)
    payload = {
        "sub": str(user_id),
        "type": "access",
        "iat": now.isoformat(),
        "exp": (now + timedelta(minutes=15)).isoformat(),
        "force_change": False,
        **(claims or {}),
    }
    key = Key.from_asymmetric_key_params(4, d=bytes.fromhex(DEV_PASETO_SEED_HEX))
    return pyseto.encode(key, json.dumps(payload).encode()).decode()


def cleanup_call_artifacts(app, call_id=None, conversation_id=None):
    """Подчистить созданные тестом звонок/плашки/диалог в общей dev-БД.
    conversation_id передаём только если диалог создал сам тест."""
    from app.extensions import db
    from app.models import Call, CallParticipant, Conversation, Message
    with app.app_context():
        try:
            if call_id is not None:
                db.session.execute(db.delete(Message).where(Message.call_id == call_id))
                db.session.execute(db.delete(CallParticipant).where(CallParticipant.call_id == call_id))
                db.session.execute(db.delete(Call).where(Call.id == call_id))
            if conversation_id is not None:
                db.session.execute(db.delete(Message).where(Message.conversation_id == conversation_id))
                db.session.execute(db.delete(Conversation).where(Conversation.id == conversation_id))
            db.session.commit()
        except Exception:  # noqa: BLE001
            db.session.rollback()

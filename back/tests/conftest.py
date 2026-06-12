"""Общие фикстуры для интеграционных тестов.

Поднимаем настоящее Flask-приложение (фабрика create_app) поверх dev-БД и
Redis. Если они недоступны — тесты, которым нужен `app`, автоматически
пропускаются (skip), а чистые юнит-тесты (yougile и т. п.) работают и без них.
Go-микросервисы (callsvc, msgsvc) не нужны: шлюзы проверяются против
in-process fake gRPC-серверов (fake_messenger здесь, fake_calls —
в test_call_flow.py).
"""
import os
import socket
from concurrent import futures

import grpc
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


# ─────────────── gRPC-хелперы для фейковых микросервисов ───────────────

def grpc_direct_execute(method, request):
    """В pytest eventlet не monkey-patch'ится: tpool-путь gRPC-клиентов отдал
    бы управление hub'у, и фоновые гринлеты create_app повисли бы на не-зелёном
    time.sleep. Транспорт для шлюза прозрачен — зовём gRPC напрямую."""
    return method(request, timeout=5)


def free_port() -> int:
    """Порт, на котором заведомо никто не слушает (для *_down фикстур)."""
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return port


def reset_messenger_stub():
    # messenger_client кэширует channel/stub в module-globals — сбрасываем,
    # чтобы клиент пересоздал их на подменённый MESSENGER_GRPC_ADDR.
    from app.services import messenger_client
    if messenger_client._channel is not None:
        messenger_client._channel.close()
    messenger_client._channel = None
    messenger_client._stub = None


class FakeMessengerService:
    """Canned-ответы MessengerService + запись входящих запросов для ассертов.
    Дефолтные ответы пустые: EnsureDialog → conversation_id=0,
    GetCallMessage → пустой message_json (плашки нет)."""

    def __init__(self):
        self.requests = []
        self.responses = {}  # имя RPC -> ответ или callable(request)

    def _respond(self, name, request, default):
        self.requests.append((name, request))
        resp = self.responses.get(name, default)
        return resp(request) if callable(resp) else resp

    def EnsureDialog(self, request, context):
        from app.grpc import messenger_pb2
        return self._respond("EnsureDialog", request,
                             messenger_pb2.EnsureDialogResponse())

    def CreateCallMessage(self, request, context):
        from app.grpc import messenger_pb2
        return self._respond("CreateCallMessage", request,
                             messenger_pb2.CreateCallMessageResponse())

    def GetCallMessage(self, request, context):
        from app.grpc import messenger_pb2
        return self._respond("GetCallMessage", request,
                             messenger_pb2.GetCallMessageResponse())

    def PostBotMessage(self, request, context):
        from app.grpc import messenger_pb2
        return self._respond("PostBotMessage", request,
                             messenger_pb2.PostBotMessageResponse())

    def ListRecentMessages(self, request, context):
        from app.grpc import messenger_pb2
        return self._respond("ListRecentMessages", request,
                             messenger_pb2.ListRecentMessagesResponse())


@pytest.fixture
def fake_messenger(monkeypatch):
    """In-process fake msgsvc: gRPC-сервер с canned-ответами вместо
    настоящего back-go/messenger."""
    from app.grpc import messenger_pb2_grpc
    from app.services import messenger_client

    servicer = FakeMessengerService()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    messenger_pb2_grpc.add_MessengerServiceServicer_to_server(servicer, server)
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()

    monkeypatch.setenv("MESSENGER_GRPC_ADDR", f"127.0.0.1:{port}")
    monkeypatch.setattr(messenger_client, "_execute", grpc_direct_execute)
    reset_messenger_stub()
    yield servicer
    server.stop(None)
    reset_messenger_stub()


@pytest.fixture
def messenger_down(monkeypatch):
    """MESSENGER_GRPC_ADDR указывает на порт, где никто не слушает."""
    from app.services import messenger_client

    monkeypatch.setenv("MESSENGER_GRPC_ADDR", f"127.0.0.1:{free_port()}")
    monkeypatch.setattr(messenger_client, "_execute", grpc_direct_execute)
    reset_messenger_stub()
    yield
    reset_messenger_stub()

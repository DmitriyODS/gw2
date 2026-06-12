import os
from flask import Flask, jsonify
from flasgger import Swagger

from app.config import config
from app.extensions import db, socketio, limiter, migrate


def create_app(config_name: str = None) -> Flask:
    if config_name is None:
        config_name = os.getenv("FLASK_ENV", "production")

    app = Flask(__name__)
    app.config.from_object(config[config_name])

    _init_extensions(app)
    _register_blueprints(app)
    _register_socket_events(app)
    _register_error_handlers(app)
    _register_security_headers(app)
    _init_swagger(app)
    _start_call_bridge(app)
    _start_service_bridge(app)
    _start_presence_sweeper(app)

    return app


def _start_presence_sweeper(app: Flask) -> None:
    """Фоновая задача presence — раз в SWEEP_INTERVAL чистит соединения, от
    которых давно не было heartbeat'а (мобильные в фоне, мёртвые сокеты)."""
    from app.sockets.presence import start_sweeper
    start_sweeper(app)


def _init_extensions(app: Flask) -> None:
    db.init_app(app)
    migrate.init_app(app, db)
    limiter.init_app(app)
    # message_queue нужен ТОЛЬКО при нескольких процессах socketio (горизонтальное
    # масштабирование через Redis pubsub). У нас один процесс и в dev, и в prod —
    # с message_queue эмит уходит в Redis-канал, а локальный subscriber-loop в
    # eventlet поднимается не всегда корректно, из-за чего сокеты в том же
    # процессе теряют события. Включать имеет смысл только при WEB_CONCURRENCY > 1.
    use_mq = os.getenv("SOCKETIO_MESSAGE_QUEUE", "false").lower() in ("1", "true", "yes")
    socketio.init_app(
        app,
        message_queue=app.config["REDIS_URL"] if use_mq else None,
        cors_allowed_origins="*",
        async_mode="eventlet",
    )

    # Импорт моделей чтобы Alembic их видел
    with app.app_context():
        from app.models import (  # noqa
            Company,
            Role, User, Department, Task, Favorite, UnitType, Unit, UserTaskColor,
            Conversation, Message, MessageAttachment,
            Call, CallParticipant,
            FeedEvent, FeedReaction, FeedComment, Pet, PetStroke, GrooveRaid,
        )


def _register_blueprints(app: Flask) -> None:
    from app.api import register_blueprints
    register_blueprints(app)

    # Liveness для docker-compose healthcheck (без БД и авторизации).
    @app.route("/healthz")
    def healthz():
        return jsonify({"ok": True})

    @app.route("/uploads/<path:filename>")
    def serve_upload(filename):
        from flask import send_from_directory
        import os
        folder = app.config["UPLOAD_FOLDER"]
        if not os.path.isabs(folder):
            # Разрешаем относительный путь от back/ (родитель пакета app)
            folder = os.path.join(app.root_path, '..', folder)
        return send_from_directory(os.path.abspath(folder), filename)


def _register_socket_events(app: Flask) -> None:
    from app.sockets import register_events
    register_events(socketio)


def _register_error_handlers(app: Flask) -> None:
    @app.errorhandler(400)
    def bad_request(e):
        return jsonify({"error": "BAD_REQUEST", "message": str(e.description)}), 400

    @app.errorhandler(401)
    def unauthorized(e):
        return jsonify({"error": "UNAUTHORIZED", "message": str(e.description)}), 401

    @app.errorhandler(403)
    def forbidden(e):
        return jsonify({"error": "FORBIDDEN", "message": str(e.description)}), 403

    @app.errorhandler(404)
    def not_found(e):
        return jsonify({"error": "NOT_FOUND", "message": str(e.description)}), 404

    @app.errorhandler(409)
    def conflict(e):
        return jsonify({"error": "CONFLICT", "message": str(e.description)}), 409

    @app.errorhandler(422)
    def unprocessable(e):
        return jsonify({"error": "UNPROCESSABLE", "message": str(e.description)}), 422

    @app.errorhandler(429)
    def rate_limit(e):
        return jsonify({"error": "RATE_LIMIT", "message": "Превышен лимит запросов"}), 429

    @app.errorhandler(500)
    def internal_error(e):
        from app.utils.logger import get_logger
        get_logger(__name__).error("internal_error", extra={"extra": {"error": str(e)}})
        return jsonify({"error": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервера"}), 500


def _register_security_headers(app: Flask) -> None:
    @app.after_request
    def set_security_headers(response):
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["X-Frame-Options"] = "DENY"
        response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
        return response


def _init_swagger(app: Flask) -> None:
    swagger_config = {
        "headers": [],
        "specs": [
            {
                "endpoint": "apispec",
                "route": "/apispec.json",
                "rule_filter": lambda rule: True,
                "model_filter": lambda tag: True,
            }
        ],
        "static_url_path": "/flasgger_static",
        "swagger_ui": True,
        "specs_route": "/apidocs/",
    }
    template = {
        "swagger": "2.0",
        "info": {
            "title": "Groove Work API",
            "description": "REST API платформы учёта задач, времени и общения Groove Work v3.0",
            "version": "3.6.0",
        },
        "securityDefinitions": {
            "BearerAuth": {
                "type": "apiKey",
                "in": "header",
                "name": "Authorization",
                "description": "Формат: Bearer <access_token>",
            }
        },
        "consumes": ["application/json"],
        "produces": ["application/json"],
    }
    Swagger(app, config=swagger_config, template=template)


def _start_call_bridge(app: Flask) -> None:
    """Слушатель Redis-канала событий Go-микросервиса звонков: транслирует
    call:ended и обновления плашки звонка в Socket.IO-комнаты пользователей.
    Реконсиляция зависших звонков при старте теперь — забота самого callsvc."""
    from app.sockets.call_bridge import start_call_bridge
    start_call_bridge(app, socketio)


def _start_service_bridge(app: Flask) -> None:
    """Обобщённый слушатель Redis-каналов gw2:<svc>:events Go-микросервисов
    (пока — мессенджер): обычные события ретранслируются в Socket.IO-комнаты,
    служебные (на «_») диспатчатся в python-обработчики."""
    from app.sockets.service_bridge import start_service_bridge
    start_service_bridge(app, socketio)


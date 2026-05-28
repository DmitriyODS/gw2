import os
from flask import Flask, jsonify
from flasgger import Swagger

from app.config import config
from app.extensions import db, jwt, socketio, limiter, migrate


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
    _finalize_stuck_calls(app)

    return app


def _init_extensions(app: Flask) -> None:
    db.init_app(app)
    migrate.init_app(app, db)
    jwt.init_app(app)
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
            Role, User, Department, Task, Favorite, UnitType, Unit, UserTaskColor,
            Conversation, Message, MessageAttachment,
            Call, CallParticipant,
        )


def _register_blueprints(app: Flask) -> None:
    from app.api import register_blueprints
    register_blueprints(app)

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
            "description": "REST API платформы учёта задач и времени Groove Work v2.0",
            "version": "2.6.0",
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


def _finalize_stuck_calls(app: Flask) -> None:
    """При старте процесса финализируем все звонки, оставшиеся в БД со
    статусом 'ringing' или 'active'. Они физически не могут быть «активны»
    — `call_state` живёт в памяти процесса и теряется при рестарте; без
    очистки системные плашки в чате (kind='call') вечно показывают
    «Идёт вызов / Идёт сейчас», и кнопка «Присоединиться» зовёт в
    несуществующий звонок. 'ringing' → 'missed' (никто не успел поднять),
    'active' → 'ended' (длительность как была на момент крэша)."""
    from datetime import datetime, timezone
    from app.models import Call, CallParticipant
    with app.app_context():
        stuck = db.session.execute(
            db.select(Call).where(Call.status.in_(["ringing", "active"]))
        ).scalars().all()
        if not stuck:
            return
        now = datetime.now(timezone.utc)
        for c in stuck:
            c.status = "missed" if c.status == "ringing" else "ended"
            c.ended_at = c.started_at if c.status == "missed" else now
            for p in db.session.execute(
                db.select(CallParticipant).where(
                    CallParticipant.call_id == c.id,
                    CallParticipant.left_at.is_(None),
                )
            ).scalars().all():
                p.left_at = c.ended_at
        try:
            db.session.commit()
        except Exception:
            db.session.rollback()
        from app.utils.logger import get_logger
        get_logger(__name__).info("calls.startup_cleanup",
                                   extra={"extra": {"count": len(stuck)}})


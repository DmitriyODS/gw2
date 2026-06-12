import os


class Config:
    SECRET_KEY = os.environ.get("SECRET_KEY", "change-me-in-production")
    SQLALCHEMY_DATABASE_URI = os.environ.get("DATABASE_URL", "postgresql://grovework:grovework@localhost:5432/grovework")
    SQLALCHEMY_TRACK_MODIFICATIONS = False
    SQLALCHEMY_ENGINE_OPTIONS = {
        "pool_pre_ping": True,
        "pool_size": 10,
        "max_overflow": 20,
    }

    # Публичный ключ PASETO v4.public (hex, 32 байта): access-токены выпускает
    # микросервис авторизации (back-go/auth), Flask только проверяет подпись.
    PASETO_PUBLIC_KEY = os.environ.get("PASETO_PUBLIC_KEY", "")

    REDIS_URL = os.environ.get("REDIS_URL", "redis://localhost:6379/0")

    UPLOAD_FOLDER = os.environ.get("UPLOAD_FOLDER", "/app/uploads")
    # 50 MB — общий лимит на запрос; аватарка отдельно проверяет 2 MB в коде,
    # вложения мессенджера лимитируются в messenger_service.
    MAX_CONTENT_LENGTH = 50 * 1024 * 1024
    MESSENGER_ATTACHMENT_MAX = 25 * 1024 * 1024  # 25 MB на одно вложение

    RATELIMIT_DEFAULT = "200 per minute"
    RATELIMIT_STORAGE_URI = os.environ.get("REDIS_URL", "redis://localhost:6379/1")

    SWAGGER = {
        "title": "Groove Work API",
        "version": "2.0.0",
        "description": "REST API платформы учёта задач и времени Groove Work",
        "uiversion": 3,
        "specs_route": "/apidocs/",
    }


class DevelopmentConfig(Config):
    DEBUG = True
    SQLALCHEMY_ECHO = True


class ProductionConfig(Config):
    DEBUG = False


config = {
    "development": DevelopmentConfig,
    "production": ProductionConfig,
    "default": DevelopmentConfig,
}

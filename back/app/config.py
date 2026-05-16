import os
from datetime import timedelta


class Config:
    SECRET_KEY = os.environ.get("SECRET_KEY", "change-me-in-production")
    SQLALCHEMY_DATABASE_URI = os.environ.get("DATABASE_URL", "postgresql://grovework:grovework@localhost:5432/grovework")
    SQLALCHEMY_TRACK_MODIFICATIONS = False
    SQLALCHEMY_ENGINE_OPTIONS = {
        "pool_pre_ping": True,
        "pool_size": 10,
        "max_overflow": 20,
    }

    JWT_SECRET_KEY = os.environ.get("JWT_SECRET_KEY", "change-jwt-secret-in-production")
    JWT_ACCESS_TOKEN_EXPIRES = timedelta(minutes=15)
    JWT_REFRESH_TOKEN_EXPIRES = timedelta(days=30)
    JWT_TOKEN_LOCATION = ["headers"]

    REDIS_URL = os.environ.get("REDIS_URL", "redis://localhost:6379/0")

    UPLOAD_FOLDER = os.environ.get("UPLOAD_FOLDER", "/app/uploads")
    MAX_CONTENT_LENGTH = 2 * 1024 * 1024  # 2 MB

    RATELIMIT_DEFAULT = "200 per minute"
    RATELIMIT_STORAGE_URI = os.environ.get("REDIS_URL", "redis://localhost:6379/1")

    SWAGGER = {
        "title": "Grove Work API",
        "version": "2.0.0",
        "description": "REST API платформы учёта задач и времени Grove Work",
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

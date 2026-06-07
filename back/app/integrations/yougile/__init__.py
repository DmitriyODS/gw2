"""Интеграция с YouGile REST API v2.

См. план интеграции в CLAUDE.md. На этапе 1 здесь только примитивы
(HTTP-клиент, парсер URL, шифрование ключа); бизнес-логика синхронизации,
эндпоинты и сервис подключения подъезжают на этапах 2–4.
"""
from .client import YougileClient, YougileError, YougileAuthError, YougileRateLimited
from .parser import parse_task_url
from .crypto import encrypt_key, decrypt_key, make_fingerprint, YougileSecretMisconfigured

__all__ = [
    "YougileClient",
    "YougileError",
    "YougileAuthError",
    "YougileRateLimited",
    "parse_task_url",
    "encrypt_key",
    "decrypt_key",
    "make_fingerprint",
    "YougileSecretMisconfigured",
]

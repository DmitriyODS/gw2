"""Шифрование/расшифровка AI API-ключей компаний.

Ключ шифрования — `AI_KEY_ENCRYPTION_KEY` в env (Fernet-формат, base64-32 байта).
Хранить рядом с БД нельзя: бэкап БД без `.env` не должен раскрывать ключи.

Если переменная не задана — функции шифрования бросают `AiSecretMisconfigured`.
Это сознательный hard-fail: молча хранить ключи в открытом виде или терять их
при перешифровке — недопустимо.
"""
from __future__ import annotations

import base64
import os
from functools import lru_cache

from cryptography.fernet import Fernet, InvalidToken


class AiSecretMisconfigured(RuntimeError):
    """AI_KEY_ENCRYPTION_KEY не задан или некорректен."""


@lru_cache(maxsize=1)
def _fernet() -> Fernet:
    raw = os.environ.get("AI_KEY_ENCRYPTION_KEY", "").strip()
    if not raw:
        raise AiSecretMisconfigured(
            "AI_KEY_ENCRYPTION_KEY не задан. Сгенерировать: "
            "python -c \"from cryptography.fernet import Fernet; "
            "print(Fernet.generate_key().decode())\""
        )
    try:
        return Fernet(raw.encode())
    except (ValueError, base64.binascii.Error) as e:
        raise AiSecretMisconfigured(f"AI_KEY_ENCRYPTION_KEY некорректен: {e}") from e


def encrypt_api_key(plain: str) -> bytes:
    if not plain:
        raise ValueError("empty api key")
    return _fernet().encrypt(plain.encode())


def decrypt_api_key(enc: bytes | None) -> str | None:
    if not enc:
        return None
    try:
        return _fernet().decrypt(bytes(enc)).decode()
    except InvalidToken:
        # Ключ шифрования сменили, а ключи компаний не перешифровали — лучше
        # вернуть None (фичи AI выключатся), чем уронить весь запрос.
        return None


def make_hint(plain: str) -> str:
    """Короткая маска ключа для UI: первые 3 + … + последние 4 символа."""
    if not plain:
        return ""
    if len(plain) <= 8:
        return "…" + plain[-2:]
    return f"{plain[:3]}…{plain[-4:]}"

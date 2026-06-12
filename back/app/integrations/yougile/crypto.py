"""Шифрование/расшифровка API-ключей YouGile.

Ключ Fernet'а лежит в `YOUGILE_ENC_KEY` env-переменной. Принципиально
отдельная переменная от AI_KEY_ENCRYPTION_KEY (живёт в env aisvc):
компрометация одного секрета не должна снимать защиту со второго; разные
жизненные циклы (AI-ключи — один на компанию, YouGile-ключи — по одному
на каждого юзера).

При отсутствии или некорректном значении переменной — hard-fail при первом
обращении, потому что сохранить ключ YG в открытом виде или потерять его
при сбое ключа шифрования недопустимо.
"""
from __future__ import annotations

import base64
import os
from functools import lru_cache

from cryptography.fernet import Fernet, InvalidToken


class YougileSecretMisconfigured(RuntimeError):
    """YOUGILE_ENC_KEY не задан или некорректен."""


@lru_cache(maxsize=1)
def _fernet() -> Fernet:
    raw = os.environ.get("YOUGILE_ENC_KEY", "").strip()
    if not raw:
        raise YougileSecretMisconfigured(
            "YOUGILE_ENC_KEY не задан. Сгенерировать: "
            "python -c \"from cryptography.fernet import Fernet; "
            "print(Fernet.generate_key().decode())\""
        )
    try:
        return Fernet(raw.encode())
    except (ValueError, base64.binascii.Error) as e:
        raise YougileSecretMisconfigured(f"YOUGILE_ENC_KEY некорректен: {e}") from e


def encrypt_key(plain: str) -> bytes:
    if not plain:
        raise ValueError("empty yougile key")
    return _fernet().encrypt(plain.encode())


def decrypt_key(enc: bytes | None) -> str | None:
    if not enc:
        return None
    try:
        return _fernet().decrypt(bytes(enc)).decode()
    except InvalidToken:
        # YOUGILE_ENC_KEY сменили без миграции — обработчик увидит None и
        # покажет «переподключите аккаунт», вместо 500.
        return None


def make_fingerprint(plain: str) -> str:
    """Последние 4 символа ключа для UI («…X9aQ»). Хранится открыто."""
    if not plain:
        return ""
    return plain[-4:] if len(plain) >= 4 else plain

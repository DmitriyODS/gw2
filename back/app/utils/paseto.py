"""Проверка PASETO access-токенов (v4.public, Ed25519).

Токены выпускает Go-микросервис авторизации (back-go/auth, authsvc) и
подписывает приватным ключом; Flask хранит только публичный ключ
(PASETO_PUBLIC_KEY) — проверяет подпись, но выпустить токен не может.

Клеймы совпадают с прежними JWT additional_claims: sub (id строкой),
type=access, exp/iat (RFC3339), force_change, company_id, company_name,
company_settings, role_level, is_root_admin.
"""
import json
from datetime import datetime, timezone

import pyseto
from pyseto import Key
from flask import abort, current_app, g, request

_key_cache: dict[str, object] = {}


def _public_key():
    """Ключ кэшируем по hex-значению: конфиг не меняется в рантайме,
    а Key.from_asymmetric_key_params на каждый запрос — лишняя работа."""
    hex_key = current_app.config.get("PASETO_PUBLIC_KEY") or ""
    if not hex_key:
        raise ValueError("PASETO_PUBLIC_KEY не задан")
    key = _key_cache.get(hex_key)
    if key is None:
        key = Key.from_asymmetric_key_params(4, x=bytes.fromhex(hex_key))
        _key_cache[hex_key] = key
    return key


def verify_access_token(token: str) -> dict:
    """Проверить подпись/срок/тип access-токена. Возвращает клеймы,
    бросает ValueError на любом дефекте."""
    if not token:
        raise ValueError("empty token")
    decoded = pyseto.decode(_public_key(), token)
    claims = json.loads(decoded.payload)

    if claims.get("type") != "access":
        raise ValueError("not an access token")

    exp_raw = claims.get("exp")
    if not exp_raw:
        raise ValueError("no exp claim")
    exp = datetime.fromisoformat(exp_raw)
    if exp <= datetime.now(timezone.utc):
        raise ValueError("token expired")

    if not str(claims.get("sub") or "").isdigit():
        raise ValueError("bad subject")
    return claims


def verify_request_token() -> dict:
    """Достать Bearer-токен из запроса, проверить и сложить клеймы в g.
    401 при отсутствии/невалидности — как verify_jwt_in_request раньше."""
    header = request.headers.get("Authorization", "")
    if not header.startswith("Bearer "):
        abort(401, description="Требуется авторизация")
    try:
        claims = verify_access_token(header[len("Bearer "):])
    except Exception:
        abort(401, description="Требуется авторизация")
    g.paseto_claims = claims
    return claims


def current_claims() -> dict:
    """Клеймы текущего запроса (после декоратора require_auth/require_role)."""
    claims = getattr(g, "paseto_claims", None)
    if claims is None:
        claims = verify_request_token()
    return claims


def request_user_id() -> str:
    """id пользователя текущего запроса строкой (формат прежнего
    get_jwt_identity; вызывающие делают int())."""
    return current_claims()["sub"]

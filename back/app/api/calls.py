"""REST API звонков.

Сам процесс звонка идёт через сокеты (call:start/accept/decline/leave/end +
webrtc:signal). REST нужен для:
  - GET /api/calls/history  — история звонков пользователя
  - GET /api/calls/ice-servers — конфигурация STUN/TURN для клиента
  - GET /api/calls/active   — есть ли у меня сейчас активный звонок
                              (для восстановления состояния при F5)
"""
import os
import hmac
import hashlib
import base64
import time

from flask import Blueprint, jsonify
from flask_jwt_extended import get_jwt_identity

from app.schemas import CallSchema
from app.services import call_service
from app.sockets import call_state
from app.utils.permissions import require_auth

bp = Blueprint("calls", __name__, url_prefix="/api/calls")

_calls_schema = CallSchema(many=True)
_call_schema = CallSchema()


@bp.get("/history")
@require_auth
def call_history():
    """История звонков текущего пользователя.
    ---
    tags: [calls]
    security: [BearerAuth: []]
    responses:
      200:
        description: Звонки в хронологическом порядке (новые сверху)
    """
    user_id = int(get_jwt_identity())
    calls = call_service.list_history_for_user(user_id, limit=100)
    return jsonify(_calls_schema.dump(calls)), 200


@bp.get("/ice-servers")
@require_auth
def ice_servers():
    """Конфигурация STUN/TURN для WebRTC.

    Если в окружении заданы TURN_HOST + TURN_SECRET — выдаём временные
    coturn-креды по схеме REST API (rfc-style ephemeral credentials). Если
    задан TURN_HOST без secret — отдаём публичные креды TURN_USER/TURN_PASS.
    В dev без TURN — только STUN.
    ---
    tags: [calls]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список iceServers для RTCPeerConnection
    """
    servers = [{"urls": ["stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"]}]

    turn_host = os.getenv("TURN_HOST")
    turn_secret = os.getenv("TURN_SECRET")
    turn_user_static = os.getenv("TURN_USER")
    turn_pass_static = os.getenv("TURN_PASS")
    turn_port = os.getenv("TURN_PORT", "3478")
    turn_realm = os.getenv("TURN_REALM", "grovework")

    if turn_host and turn_secret:
        # Эфемерные креды: username = expiry_ts:user_id, password = HMAC-SHA1(secret, username)
        ttl = int(os.getenv("TURN_TTL", "3600"))
        user_id = int(get_jwt_identity())
        expiry = int(time.time()) + ttl
        username = f"{expiry}:gw{user_id}"
        sig = hmac.new(turn_secret.encode(), username.encode(), hashlib.sha1).digest()
        password = base64.b64encode(sig).decode()
        servers.append({
            "urls": [
                f"turn:{turn_host}:{turn_port}?transport=udp",
                f"turn:{turn_host}:{turn_port}?transport=tcp",
            ],
            "username": username,
            "credential": password,
        })
    elif turn_host and turn_user_static and turn_pass_static:
        servers.append({
            "urls": [
                f"turn:{turn_host}:{turn_port}?transport=udp",
                f"turn:{turn_host}:{turn_port}?transport=tcp",
            ],
            "username": turn_user_static,
            "credential": turn_pass_static,
        })

    return jsonify({"iceServers": servers}), 200


@bp.get("/active")
@require_auth
def active_call():
    """Есть ли у меня активный звонок (для UI при перезагрузке вкладки).
    ---
    tags: [calls]
    security: [BearerAuth: []]
    responses:
      200:
        description: Активный звонок или null
    """
    user_id = int(get_jwt_identity())
    call_id = call_state.get_user_active_call(user_id)
    if call_id is None:
        return jsonify({"call": None}), 200

    from app.extensions import db
    from app.models import Call
    call = db.session.get(Call, call_id)
    if not call:
        return jsonify({"call": None}), 200
    return jsonify({"call": _call_schema.dump(call)}), 200

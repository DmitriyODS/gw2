"""Остаток messenger-блюпринта после выноса мессенджера в Go-сервис msgsvc.

REST /api/messenger/* обслуживает msgsvc (nginx/vite роутят мимо Flask);
здесь остался только presence — он in-memory в процессе Flask вместе
с Socket.IO (см. sockets/presence.py).
"""
from flask import Blueprint, jsonify

from app.utils.permissions import require_auth

bp = Blueprint("messenger", __name__, url_prefix="/api/messenger")


@bp.get("/presence")
@require_auth
def presence_list():
    """
    Список id пользователей, которые сейчас онлайн.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    responses:
      200:
        description: Онлайн-пользователи
    """
    from app.sockets import presence
    return jsonify({"online": presence.online_user_ids()}), 200

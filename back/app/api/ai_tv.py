"""Публичные AI-эндпоинты для ТВ-режима.

Доступ — любой авторизованный пользователь. Компания берётся из его
профиля (для root_admin'а — из query-param ?company_id=, как и везде).
"""
from flask import Blueprint, g, jsonify

from app.services.tv_facts_service import get_fact
from app.utils.permissions import require_company_scope, require_auth


bp = Blueprint("ai_tv", __name__, url_prefix="/api/ai")


@bp.get("/tv-fact")
@require_auth
@require_company_scope
def tv_fact():
    """
    Текущий факт дня для ТВ-табло. Если AI выключен / факт не сгенерён —
    возвращаем `null` с 200 OK, чтобы фронт молча упал на фолбэк-слайд.
    """
    fact = get_fact(g.company_id)
    return jsonify(fact), 200

"""Управление AI-настройками компании.

Доступ:
- Администратор системы (`is_root_admin`) — для любой компании.
- Руководитель компании (`DIRECTOR`, level 3) — только для СВОЕЙ компании.

Сырой ключ наружу не отдаём НИКОГДА — даже суперадмину. После сохранения он
есть только в зашифрованном виде в БД; для UI остаётся короткая маска.
"""
from flask import Blueprint, g, jsonify, request
from marshmallow import ValidationError

from app.extensions import db, socketio
from app.models.company import Company
from app.models.task import Task
from app.schemas.ai_settings import AiSettingsSchema, AiSettingsUpdateSchema
from app.services.ai_client import get_ai_client, invalidate_ai_client
from app.services.task_embedding_service import (
    count_embeddings, find_unindexed_task_ids, run_backfill,
)
from app.utils.ai_secret import (
    AiSecretMisconfigured, encrypt_api_key, make_hint,
)
from app.utils.logger import get_logger
from app.utils.permissions import DIRECTOR, require_role


bp = Blueprint("ai_settings", __name__, url_prefix="/api/companies")
logger = get_logger(__name__)

_out = AiSettingsSchema()
_upd = AiSettingsUpdateSchema()


def _resolve_company(company_id: int) -> Company | None:
    """Возвращает компанию или None (404). Проверка прав — отдельно."""
    return db.session.get(Company, company_id)


def _check_access(company: Company) -> tuple[dict, int] | None:
    user = g.current_user
    if getattr(user, "is_root_admin", False):
        return None
    if user.company_id == company.id:
        return None
    return {"error": "FORBIDDEN", "message": "Нет доступа к настройкам этой компании"}, 403


@bp.get("/<int:company_id>/ai-settings")
@require_role(DIRECTOR)
def get_ai_settings(company_id: int):
    company = _resolve_company(company_id)
    if company is None:
        return jsonify({"error": "NOT_FOUND"}), 404
    err = _check_access(company)
    if err:
        return jsonify(err[0]), err[1]
    return jsonify(_out.dump(company)), 200


@bp.put("/<int:company_id>/ai-settings")
@require_role(DIRECTOR)
def update_ai_settings(company_id: int):
    company = _resolve_company(company_id)
    if company is None:
        return jsonify({"error": "NOT_FOUND"}), 404
    err = _check_access(company)
    if err:
        return jsonify(err[0]), err[1]
    try:
        data = _upd.load(request.get_json() or {})
    except ValidationError as ve:
        return jsonify({"error": "VALIDATION", "details": ve.messages}), 400

    if "enabled" in data:
        company.ai_enabled = bool(data["enabled"])
    if "model_chat" in data:
        company.ai_model_chat = data["model_chat"].strip()
    if "model_embedding" in data:
        company.ai_model_embedding = data["model_embedding"].strip()

    # api_key: None / "" → не менять; clear_key=true → стереть; иначе зашифровать.
    if data.get("clear_key"):
        company.ai_api_key_enc = None
        company.ai_key_hint = None
    else:
        new_key = (data.get("api_key") or "").strip()
        if new_key:
            try:
                company.ai_api_key_enc = encrypt_api_key(new_key)
            except AiSecretMisconfigured as e:
                logger.error("ai.encrypt_failed", extra={"err": str(e)})
                return jsonify({
                    "error": "AI_KEY_NOT_CONFIGURED",
                    "message": "На сервере не задан AI_KEY_ENCRYPTION_KEY",
                }), 500
            company.ai_key_hint = make_hint(new_key)

    db.session.commit()
    invalidate_ai_client(company.id)
    return jsonify(_out.dump(company)), 200


@bp.post("/<int:company_id>/ai-settings/test")
@require_role(DIRECTOR)
def test_ai_settings(company_id: int):
    """Проверяет реальную связь с моделью: один tiny-chat + один embedding.

    Не сохраняет ничего. Возвращает {chat, embedding, latency_ms, error}.
    Если AI выключен / ключа нет — отдаёт `409 AI_DISABLED`, чтобы UI показал
    «сначала введите ключ и включите AI».
    """
    company = _resolve_company(company_id)
    if company is None:
        return jsonify({"error": "NOT_FOUND"}), 404
    err = _check_access(company)
    if err:
        return jsonify(err[0]), err[1]
    client = get_ai_client(company.id)
    if client is None:
        return jsonify({
            "error": "AI_DISABLED",
            "message": "AI выключен или ключ не задан",
        }), 409
    return jsonify(client.test()), 200


@bp.get("/<int:company_id>/ai-settings/indexing")
@require_role(DIRECTOR)
def indexing_status(company_id: int):
    """Сколько задач компании уже проиндексировано и сколько ещё надо.

    Нужно для UI «Переиндексировать N задач» — после включения AI на
    существующей базе семантический поиск молчит, пока бэкфилл не
    пройдёт по всем задачам.
    """
    company = _resolve_company(company_id)
    if company is None:
        return jsonify({"error": "NOT_FOUND"}), 404
    err = _check_access(company)
    if err:
        return jsonify(err[0]), err[1]
    total = db.session.query(Task).filter_by(company_id=company.id).count()
    indexed = count_embeddings(company.id, model=company.ai_model_embedding)
    pending = max(0, len(find_unindexed_task_ids(company.id)))
    return jsonify({
        "total_tasks": total,
        "indexed": indexed,
        "pending": pending,
        "model": company.ai_model_embedding,
        "ai_enabled": company.ai_enabled and company.ai_api_key_enc is not None,
    }), 200


@bp.post("/<int:company_id>/ai-settings/reindex-tasks")
@require_role(DIRECTOR)
def reindex_tasks(company_id: int):
    """Запустить бэкфилл эмбеддингов для компании в фоне.

    Возвращаем 202 Accepted с ожидаемым количеством — реальный прогресс
    смотри через GET .../indexing.
    """
    company = _resolve_company(company_id)
    if company is None:
        return jsonify({"error": "NOT_FOUND"}), 404
    err = _check_access(company)
    if err:
        return jsonify(err[0]), err[1]
    if get_ai_client(company.id) is None:
        return jsonify({
            "error": "AI_DISABLED",
            "message": "AI выключен или ключ не задан",
        }), 409

    cid = company.id
    from flask import current_app
    app = current_app._get_current_object()

    def _job():
        with app.app_context():
            try:
                run_backfill(cid)
            except Exception as e:
                logger.warning("ai.reindex.batch_failed",
                               extra={"company_id": cid, "err": str(e)})

    pending = len(find_unindexed_task_ids(cid))
    socketio.start_background_task(_job)
    return jsonify({"queued": True, "pending": pending}), 202

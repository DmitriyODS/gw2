from flask import Blueprint, request, jsonify
from marshmallow import ValidationError

from app.schemas.company import (
    CompanySchema, CompanyCreateSchema, CompanyUpdateSchema, CompanyToggleActiveSchema,
)
from app.repositories import company_repo
from app.services import company_service
from app.services.company_service import CompanyServiceError
from app.utils.permissions import require_role, ADMIN

bp = Blueprint("companies", __name__, url_prefix="/api/companies")

_schema = CompanySchema()
_many_schema = CompanySchema(many=True)
_create_schema = CompanyCreateSchema()
_update_schema = CompanyUpdateSchema()
_toggle_schema = CompanyToggleActiveSchema()


def _enrich(company) -> dict:
    data = _schema.dump(company)
    stats = company_repo.stats_by_company_id(company.id)
    data["employees_count"] = stats["employees"]
    data["tasks_count"] = stats["tasks"]
    return data


@bp.get("")
@require_role(ADMIN)
def list_companies():
    """
    Список всех компаний (для Администратора системы).
    ---
    tags: [companies]
    security: [BearerAuth: []]
    responses:
      200: {description: Список компаний}
    """
    companies = company_repo.get_all()
    stats_map = company_repo.stats_by_company_ids([c.id for c in companies])
    items = []
    for c in companies:
        data = _schema.dump(c)
        stats = stats_map.get(c.id, {"employees": 0, "tasks": 0})
        data["employees_count"] = stats["employees"]
        data["tasks_count"] = stats["tasks"]
        items.append(data)
    return jsonify({"items": items, "total": len(items)}), 200


@bp.get("/<int:company_id>")
@require_role(ADMIN)
def get_company(company_id: int):
    company = company_repo.get_by_id(company_id)
    if company is None:
        return jsonify({"error": "NOT_FOUND", "message": "Компания не найдена"}), 404
    return jsonify(_enrich(company)), 200


@bp.post("")
@require_role(ADMIN)
def create_company():
    """
    Создать компанию.
    ---
    tags: [companies]
    security: [BearerAuth: []]
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        company = company_service.create_company(**data)
    except CompanyServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_enrich(company)), 201


@bp.patch("/<int:company_id>")
@require_role(ADMIN)
def update_company(company_id: int):
    """
    Изменить компанию.
    ---
    tags: [companies]
    security: [BearerAuth: []]
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        company = company_service.update_company(company_id, **data)
    except CompanyServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_enrich(company)), 200


@bp.patch("/<int:company_id>/toggle-active")
@require_role(ADMIN)
def toggle_active(company_id: int):
    """
    Включить/отключить компанию. При отключённой компании её сотрудники не
    могут войти в систему (получают экран блокировки).
    ---
    tags: [companies]
    security: [BearerAuth: []]
    """
    try:
        data = _toggle_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        company = company_service.set_active(company_id, data["is_active"])
    except CompanyServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_enrich(company)), 200


@bp.delete("/<int:company_id>")
@require_role(ADMIN)
def delete_company(company_id: int):
    """
    Удалить компанию (каскадно удаляет задачи, юниты, чаты и звонки!).
    ---
    tags: [companies]
    security: [BearerAuth: []]
    """
    try:
        company_service.delete_company(company_id)
    except CompanyServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify({"message": "Компания удалена"}), 200

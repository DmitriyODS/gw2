from flask import Blueprint, request, jsonify, g
from marshmallow import ValidationError

from app.schemas.stage import (
    StageSchema, StageCreateSchema, StageUpdateSchema, StageReorderSchema,
)
from app.services import stage_service
from app.services.stage_service import StageServiceError
from app.utils.permissions import require_role, require_company_scope, EMPLOYEE, MANAGER

bp = Blueprint("stages", __name__, url_prefix="/api/stages")

_schema = StageSchema()
_many_schema = StageSchema(many=True)
_create_schema = StageCreateSchema()
_update_schema = StageUpdateSchema()
_reorder_schema = StageReorderSchema()


@bp.get("")
@require_role(EMPLOYEE)
@require_company_scope
def list_stages():
    """
    Список этапов компании (в порядке `order`).
    ---
    tags: [stages]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: company_id, schema: {type: integer}, description: Для Администратора системы}
    """
    return jsonify(_many_schema.dump(stage_service.list_stages(g.company_id))), 200


@bp.post("")
@require_role(MANAGER)
@require_company_scope
def create_stage():
    """
    Создать этап.
    ---
    tags: [stages]
    security: [BearerAuth: []]
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        stage = stage_service.create_stage(g.company_id, **data)
    except StageServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_schema.dump(stage)), 201


@bp.patch("/<int:stage_id>")
@require_role(MANAGER)
@require_company_scope
def update_stage(stage_id: int):
    """
    Изменить этап.
    ---
    tags: [stages]
    security: [BearerAuth: []]
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        stage = stage_service.update_stage(g.company_id, stage_id, **data)
    except StageServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_schema.dump(stage)), 200


@bp.delete("/<int:stage_id>")
@require_role(MANAGER)
@require_company_scope
def delete_stage(stage_id: int):
    """
    Удалить этап.
    ---
    tags: [stages]
    security: [BearerAuth: []]
    """
    try:
        stage_service.delete_stage(g.company_id, stage_id)
    except StageServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify({"message": "Этап удалён"}), 200


@bp.patch("/reorder")
@require_role(MANAGER)
@require_company_scope
def reorder_stages():
    """
    Применить новый порядок этапов компании.
    ---
    tags: [stages]
    security: [BearerAuth: []]
    """
    try:
        data = _reorder_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    stages = stage_service.reorder_stages(g.company_id, data["ids"])
    return jsonify(_many_schema.dump(stages)), 200

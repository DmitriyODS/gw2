from flask import Blueprint, request, jsonify, g
from marshmallow import ValidationError

from app.schemas.unit_type import UnitTypeSchema, UnitTypeCreateSchema, UnitTypeUpdateSchema
from app.repositories import unit_type_repo
from app.extensions import db
from app.utils.permissions import require_role, require_company_scope, EMPLOYEE, MANAGER

bp = Blueprint("unit_types", __name__, url_prefix="/api/unit-types")

_schema = UnitTypeSchema()
_many_schema = UnitTypeSchema(many=True)
_create_schema = UnitTypeCreateSchema()
_update_schema = UnitTypeUpdateSchema()


@bp.get("")
@require_role(EMPLOYEE)
@require_company_scope
def list_unit_types():
    """
    Список типов юнитов компании.
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: company_id, schema: {type: integer}, description: Для Администратора системы}
    responses:
      200:
        description: Список типов юнитов
    """
    return jsonify(_many_schema.dump(unit_type_repo.get_all(g.company_id))), 200


@bp.post("")
@require_role(MANAGER)
@require_company_scope
def create_unit_type():
    """
    Создать тип юнита.
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    if unit_type_repo.get_by_name(data["name"], g.company_id):
        return jsonify({"error": "DUPLICATE", "message": "Тип юнита с таким именем уже существует"}), 409

    ut = unit_type_repo.create(data["name"], g.company_id)
    db.session.commit()
    return jsonify(_schema.dump(ut)), 201


@bp.patch("/<int:type_id>")
@require_role(MANAGER)
@require_company_scope
def update_unit_type(type_id: int):
    """
    Изменить тип юнита.
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    ut = unit_type_repo.get_by_id(type_id)
    if ut is None or ut.company_id != g.company_id:
        return jsonify({"error": "NOT_FOUND", "message": "Тип юнита не найден"}), 404

    existing = unit_type_repo.get_by_name(data["name"], g.company_id)
    if existing and existing.id != type_id:
        return jsonify({"error": "DUPLICATE", "message": "Тип юнита с таким именем уже существует"}), 409

    ut = unit_type_repo.update(ut, data["name"])
    db.session.commit()
    return jsonify(_schema.dump(ut)), 200


@bp.delete("/<int:type_id>")
@require_role(MANAGER)
@require_company_scope
def delete_unit_type(type_id: int):
    """
    Удалить тип юнита (каскадно удаляет все юниты с этим типом!).
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    """
    ut = unit_type_repo.get_by_id(type_id)
    if ut is None or ut.company_id != g.company_id:
        return jsonify({"error": "NOT_FOUND", "message": "Тип юнита не найден"}), 404

    unit_type_repo.delete(ut)
    db.session.commit()
    return jsonify({"message": "Тип юнита удалён"}), 200

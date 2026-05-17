from flask import Blueprint, request, jsonify
from marshmallow import ValidationError

from app.schemas.unit_type import UnitTypeSchema, UnitTypeCreateSchema, UnitTypeUpdateSchema
from app.repositories import unit_type_repo
from app.extensions import db
from app.utils.permissions import require_role, EMPLOYEE, MANAGER

bp = Blueprint("unit_types", __name__, url_prefix="/api/unit-types")

_schema = UnitTypeSchema()
_many_schema = UnitTypeSchema(many=True)
_create_schema = UnitTypeCreateSchema()
_update_schema = UnitTypeUpdateSchema()


@bp.get("")
@require_role(EMPLOYEE)
def list_unit_types():
    """
    Список типов юнитов.
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список типов юнитов
    """
    return jsonify(_many_schema.dump(unit_type_repo.get_all())), 200


@bp.post("")
@require_role(MANAGER)
def create_unit_type():
    """
    Создать тип юнита.
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [name]
            properties:
              name: {type: string}
    responses:
      201:
        description: Тип юнита создан
      409:
        description: Тип с таким именем уже существует
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    if unit_type_repo.get_by_name(data["name"]):
        return jsonify({"error": "DUPLICATE", "message": "Тип юнита с таким именем уже существует"}), 409

    ut = unit_type_repo.create(data["name"])
    db.session.commit()
    return jsonify(_schema.dump(ut)), 201


@bp.patch("/<int:type_id>")
@require_role(MANAGER)
def update_unit_type(type_id: int):
    """
    Изменить тип юнита.
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: type_id
        schema: {type: integer}
        required: true
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [name]
            properties:
              name: {type: string}
    responses:
      200:
        description: Тип юнита обновлён
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    ut = unit_type_repo.get_by_id(type_id)
    if ut is None:
        return jsonify({"error": "NOT_FOUND", "message": "Тип юнита не найден"}), 404

    existing = unit_type_repo.get_by_name(data["name"])
    if existing and existing.id != type_id:
        return jsonify({"error": "DUPLICATE", "message": "Тип юнита с таким именем уже существует"}), 409

    ut = unit_type_repo.update(ut, data["name"])
    db.session.commit()
    return jsonify(_schema.dump(ut)), 200


@bp.delete("/<int:type_id>")
@require_role(MANAGER)
def delete_unit_type(type_id: int):
    """
    Удалить тип юнита (каскадно удаляет все юниты с этим типом!).
    ---
    tags: [unit-types]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: type_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Тип юнита удалён
    """
    ut = unit_type_repo.get_by_id(type_id)
    if ut is None:
        return jsonify({"error": "NOT_FOUND", "message": "Тип юнита не найден"}), 404

    unit_type_repo.delete(ut)
    db.session.commit()
    return jsonify({"message": "Тип юнита удалён"}), 200

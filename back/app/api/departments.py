from flask import Blueprint, request, jsonify
from marshmallow import ValidationError

from app.schemas.department import DepartmentSchema, DepartmentCreateSchema, DepartmentUpdateSchema
from app.repositories import department_repo
from app.extensions import db
from app.utils.permissions import require_permission, Section, Bit

bp = Blueprint("departments", __name__, url_prefix="/api/departments")

_schema = DepartmentSchema()
_many_schema = DepartmentSchema(many=True)
_create_schema = DepartmentCreateSchema()
_update_schema = DepartmentUpdateSchema()


@bp.get("")
@require_permission(Section.DEPARTMENTS, Bit.VIEW)
def list_departments():
    """
    Список отделов.
    ---
    tags: [departments]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список отделов
    """
    return jsonify(_many_schema.dump(department_repo.get_all())), 200


@bp.post("")
@require_permission(Section.DEPARTMENTS, Bit.CREATE)
def create_department():
    """
    Создать отдел.
    ---
    tags: [departments]
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
        description: Отдел создан
      409:
        description: Отдел с таким именем уже существует
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    if department_repo.get_by_name(data["name"]):
        return jsonify({"error": "DUPLICATE", "message": "Отдел с таким именем уже существует"}), 409

    dept = department_repo.create(data["name"])
    db.session.commit()
    return jsonify(_schema.dump(dept)), 201


@bp.patch("/<int:dept_id>")
@require_permission(Section.DEPARTMENTS, Bit.EDIT)
def update_department(dept_id: int):
    """
    Изменить отдел.
    ---
    tags: [departments]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: dept_id
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
        description: Отдел обновлён
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    dept = department_repo.get_by_id(dept_id)
    if dept is None:
        return jsonify({"error": "NOT_FOUND", "message": "Отдел не найден"}), 404

    existing = department_repo.get_by_name(data["name"])
    if existing and existing.id != dept_id:
        return jsonify({"error": "DUPLICATE", "message": "Отдел с таким именем уже существует"}), 409

    dept = department_repo.update(dept, data["name"])
    db.session.commit()
    return jsonify(_schema.dump(dept)), 200


@bp.delete("/<int:dept_id>")
@require_permission(Section.DEPARTMENTS, Bit.DELETE)
def delete_department(dept_id: int):
    """
    Удалить отдел.
    ---
    tags: [departments]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: dept_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Отдел удалён
    """
    dept = department_repo.get_by_id(dept_id)
    if dept is None:
        return jsonify({"error": "NOT_FOUND", "message": "Отдел не найден"}), 404

    department_repo.delete(dept)
    db.session.commit()
    return jsonify({"message": "Отдел удалён"}), 200

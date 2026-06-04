from flask import Blueprint, request, jsonify, g
from marshmallow import ValidationError

from app.schemas.department import DepartmentSchema, DepartmentCreateSchema, DepartmentUpdateSchema
from app.repositories import department_repo
from app.extensions import db
from app.utils.permissions import require_role, require_company_scope, EMPLOYEE, MANAGER

bp = Blueprint("departments", __name__, url_prefix="/api/departments")

_schema = DepartmentSchema()
_many_schema = DepartmentSchema(many=True)
_create_schema = DepartmentCreateSchema()
_update_schema = DepartmentUpdateSchema()


def _check_company(dept, target_company_id: int) -> bool:
    return dept.company_id == target_company_id


@bp.get("")
@require_role(EMPLOYEE)
@require_company_scope
def list_departments():
    """
    Список отделов компании.
    ---
    tags: [departments]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: company_id, schema: {type: integer}, description: Для Администратора системы}
    responses:
      200:
        description: Список отделов
    """
    return jsonify(_many_schema.dump(department_repo.get_all(g.company_id))), 200


@bp.post("")
@require_role(MANAGER)
@require_company_scope
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

    if department_repo.get_by_name(data["name"], g.company_id):
        return jsonify({"error": "DUPLICATE", "message": "Отдел с таким именем уже существует"}), 409

    dept = department_repo.create(data["name"], g.company_id)
    db.session.commit()
    return jsonify(_schema.dump(dept)), 201


@bp.patch("/<int:dept_id>")
@require_role(MANAGER)
@require_company_scope
def update_department(dept_id: int):
    """
    Изменить отдел.
    ---
    tags: [departments]
    security: [BearerAuth: []]
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    dept = department_repo.get_by_id(dept_id)
    if dept is None or not _check_company(dept, g.company_id):
        return jsonify({"error": "NOT_FOUND", "message": "Отдел не найден"}), 404

    existing = department_repo.get_by_name(data["name"], g.company_id)
    if existing and existing.id != dept_id:
        return jsonify({"error": "DUPLICATE", "message": "Отдел с таким именем уже существует"}), 409

    dept = department_repo.update(dept, data["name"])
    db.session.commit()
    return jsonify(_schema.dump(dept)), 200


@bp.delete("/<int:dept_id>")
@require_role(MANAGER)
@require_company_scope
def delete_department(dept_id: int):
    """
    Удалить отдел.
    ---
    tags: [departments]
    security: [BearerAuth: []]
    """
    dept = department_repo.get_by_id(dept_id)
    if dept is None or not _check_company(dept, g.company_id):
        return jsonify({"error": "NOT_FOUND", "message": "Отдел не найден"}), 404

    department_repo.delete(dept)
    db.session.commit()
    return jsonify({"message": "Отдел удалён"}), 200

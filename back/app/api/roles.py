from flask import Blueprint, request, jsonify
from marshmallow import ValidationError

from app.schemas import RoleSchema, RoleCreateSchema, RoleUpdateSchema
from app.repositories import role_repo
from app.extensions import db
from app.utils.permissions import require_permission, Section, Bit, BIGINT_MAX
from app.utils.logger import get_logger

bp = Blueprint("roles", __name__, url_prefix="/api/roles")
logger = get_logger(__name__)

_role_schema = RoleSchema()
_roles_schema = RoleSchema(many=True)
_create_schema = RoleCreateSchema()
_update_schema = RoleUpdateSchema()


@bp.get("")
@require_permission(Section.ROLES, Bit.VIEW)
def list_roles():
    """
    Список ролей.
    ---
    tags: [roles]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список ролей
    """
    roles = role_repo.get_all()
    return jsonify(_roles_schema.dump(roles)), 200


@bp.post("")
@require_permission(Section.ROLES, Bit.CREATE)
def create_role():
    """
    Создать роль.
    ---
    tags: [roles]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [name, access]
            properties:
              name: {type: string}
              access: {type: integer}
    responses:
      201:
        description: Роль создана
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    role = role_repo.create(data["name"], data["access"])
    db.session.commit()
    logger.info("role.create", extra={"extra": {"role_id": role.id, "event": "role.create"}})
    return jsonify(_role_schema.dump(role)), 201


@bp.patch("/<int:role_id>")
@require_permission(Section.ROLES, Bit.EDIT)
def update_role(role_id: int):
    """
    Изменить роль.
    ---
    tags: [roles]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: role_id
        schema: {type: integer}
        required: true
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              name: {type: string}
              access: {type: integer}
    responses:
      200:
        description: Роль обновлена
      422:
        description: Нельзя изменить единственную всесильную роль
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    role = role_repo.get_by_id(role_id)
    if role is None:
        return jsonify({"error": "NOT_FOUND", "message": "Роль не найдена"}), 404

    if role.access == BIGINT_MAX and role_repo.count_almighty() <= 1:
        new_access = data.get("access", role.access)
        if new_access != BIGINT_MAX:
            return jsonify({
                "error": "LAST_ALMIGHTY_ROLE",
                "message": "Нельзя изменить единственную роль с полным доступом"
            }), 422

    role_repo.update(role, **data)
    db.session.commit()
    return jsonify(_role_schema.dump(role)), 200


@bp.delete("/<int:role_id>")
@require_permission(Section.ROLES, Bit.DELETE)
def delete_role(role_id: int):
    """
    Удалить роль.
    ---
    tags: [roles]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: role_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Роль удалена
      422:
        description: Нельзя удалить единственную всесильную роль
    """
    role = role_repo.get_by_id(role_id)
    if role is None:
        return jsonify({"error": "NOT_FOUND", "message": "Роль не найдена"}), 404

    if role.access == BIGINT_MAX and role_repo.count_almighty() <= 1:
        return jsonify({
            "error": "LAST_ALMIGHTY_ROLE",
            "message": "Нельзя удалить единственную роль с полным доступом"
        }), 422

    role_repo.delete(role)
    db.session.commit()
    logger.info("role.delete", extra={"extra": {"role_id": role_id, "event": "role.delete"}})
    return jsonify({"message": "Роль удалена"}), 200

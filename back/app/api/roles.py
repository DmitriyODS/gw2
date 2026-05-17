from flask import Blueprint, jsonify

from app.schemas.role import RoleSchema
from app.repositories import role_repo
from app.utils.permissions import require_auth

bp = Blueprint("roles", __name__, url_prefix="/api/roles")

_roles_schema = RoleSchema(many=True)


@bp.get("")
@require_auth
def list_roles():
    """
    Список фиксированных ролей.
    ---
    tags: [roles]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список ролей с уровнями
    """
    roles = role_repo.get_all()
    return jsonify(_roles_schema.dump(roles)), 200

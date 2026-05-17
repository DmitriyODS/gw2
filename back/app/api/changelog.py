import json
import os
from flask import Blueprint, jsonify
from flask_jwt_extended import jwt_required

bp = Blueprint("changelog", __name__, url_prefix="/api/changelog")


def _load():
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "data", "changelog.json")
    with open(path, encoding="utf-8") as f:
        return json.load(f)


@bp.get("")
@jwt_required()
def get_changelog():
    """
    Получить лог изменений приложения.
    ---
    tags: [changelog]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список версий
        content:
          application/json:
            schema:
              type: object
              properties:
                versions:
                  type: array
                  items:
                    type: object
    """
    return jsonify(_load())

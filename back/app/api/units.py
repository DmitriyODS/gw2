from flask import Blueprint, request, jsonify
from flask_jwt_extended import get_jwt_identity
from marshmallow import ValidationError

from app.schemas.unit import UnitSchema, UnitUpdateSchema
from app.services import unit_service
from app.services.unit_service import UnitServiceError
from app.repositories import unit_repo, user_repo
from app.utils.permissions import require_role, require_auth, EMPLOYEE, get_user_level

bp = Blueprint("units", __name__, url_prefix="/api/units")

_unit_schema = UnitSchema()
_update_schema = UnitUpdateSchema()


@bp.get("/active")
@require_auth
def get_active_unit():
    """
    Получить активный юнит текущего пользователя.
    ---
    tags: [units]
    security: [BearerAuth: []]
    responses:
      200:
        description: Активный юнит или null
    """
    current_user_id = int(get_jwt_identity())
    unit = unit_repo.get_active_for_user(current_user_id)
    if unit is None:
        return jsonify(None), 200
    return jsonify(_unit_schema.dump(unit)), 200


@bp.patch("/<int:unit_id>")
@require_role(EMPLOYEE)
def update_unit(unit_id: int):
    """
    Редактировать юнит.
    ---
    tags: [units]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: unit_id
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
              unit_type_id: {type: integer}
              datetime_start: {type: string, format: date-time}
              datetime_end: {type: string, format: date-time}
    responses:
      200:
        description: Юнит обновлён
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    current_user = user_repo.get_by_id(current_user_id)

    try:
        unit = unit_service.update_unit(unit_id, current_user_id, get_user_level(current_user), **data)
    except UnitServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    unit_data = _unit_schema.dump(unit)
    socketio.emit("unit:updated", {"unit_id": unit.id, "task_id": unit.task_id, **unit_data}, room="all")

    return jsonify(unit_data), 200


@bp.delete("/<int:unit_id>")
@require_role(EMPLOYEE)
def delete_unit(unit_id: int):
    """
    Удалить юнит.
    ---
    tags: [units]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: unit_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Юнит удалён
    """
    current_user_id = int(get_jwt_identity())
    current_user = user_repo.get_by_id(current_user_id)

    unit = unit_repo.get_by_id(unit_id)
    task_id = unit.task_id if unit else None
    owner_id = unit.user_id if unit else None

    try:
        unit_service.delete_unit(unit_id, current_user_id, get_user_level(current_user))
    except UnitServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    socketio.emit("unit:deleted", {"unit_id": unit_id, "task_id": task_id, "user_id": owner_id}, room="all")

    return jsonify({"message": "Юнит удалён"}), 200


@bp.post("/<int:unit_id>/stop")
@require_role(EMPLOYEE)
def stop_unit(unit_id: int):
    """
    Завершить юнит.
    ---
    tags: [units]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: unit_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Юнит завершён
    """
    current_user_id = int(get_jwt_identity())
    current_user = user_repo.get_by_id(current_user_id)

    unit_before = unit_repo.get_by_id(unit_id)
    owner_id = unit_before.user_id if unit_before else None

    try:
        unit = unit_service.stop_unit(unit_id, current_user_id, get_user_level(current_user))
    except UnitServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    socketio.emit("unit:stopped", {
        "unit_id": unit.id,
        "task_id": unit.task_id,
        "user_id": unit.user_id,
        "datetime_end": unit.datetime_end.isoformat(),
    }, room="all")

    if owner_id and owner_id != current_user_id:
        stopper = user_repo.get_by_id(current_user_id)
        socketio.emit("unit:force_stopped", {
            "unit_id": unit.id,
            "stopped_by_fio": stopper.fio if stopper else "Неизвестный",
        }, room=f"user_{owner_id}")

    return jsonify(_unit_schema.dump(unit)), 200

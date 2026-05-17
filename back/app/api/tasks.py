from datetime import datetime
from flask import Blueprint, request, jsonify
from flask_jwt_extended import get_jwt_identity
from marshmallow import ValidationError

from app.schemas import TaskSchema, TaskCreateSchema, TaskUpdateSchema
from app.schemas.unit import UnitSchema
from app.services import task_service
from app.services.task_service import TaskServiceError
from app.repositories import task_repo, unit_repo
from app.utils.permissions import require_role, require_auth, EMPLOYEE

bp = Blueprint("tasks", __name__, url_prefix="/api/tasks")

_task_schema = TaskSchema()
_unit_schema = UnitSchema(many=True)
_create_schema = TaskCreateSchema()
_update_schema = TaskUpdateSchema()


def _enrich_task(task, current_user_id: int) -> dict:
    data = _task_schema.dump(task)
    data["is_favorite"] = task_repo.is_favorite(task.id, current_user_id)
    data["has_units"] = task_repo.has_any_units(task.id)
    return data


@bp.get("")
@require_role(EMPLOYEE)
def list_tasks():
    """
    Список задач с фильтрами и пагинацией.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: tab, schema: {type: string, enum: [active, favorites, archive]}, description: Вкладка}
      - {in: query, name: search, schema: {type: string}, description: Поиск по названию}
      - {in: query, name: sort, schema: {type: string, enum: [last_activity, created_at, deadline]}}
      - {in: query, name: dept_id, schema: {type: integer}}
      - {in: query, name: received_from, schema: {type: string, format: date}}
      - {in: query, name: received_to, schema: {type: string, format: date}}
      - {in: query, name: has_units, schema: {type: string, enum: [none, mine]}}
      - {in: query, name: page, schema: {type: integer, default: 1}}
      - {in: query, name: per_page, schema: {type: integer, default: 30}}
    responses:
      200:
        description: Список задач
    """
    args = request.args
    current_user_id = int(get_jwt_identity())

    received_from = None
    received_to = None
    try:
        if args.get("received_from"):
            received_from = datetime.fromisoformat(args["received_from"])
        if args.get("received_to"):
            received_to = datetime.fromisoformat(args["received_to"])
    except ValueError:
        return jsonify({"error": "VALIDATION_ERROR", "message": "Неверный формат даты"}), 400

    result = task_repo.get_list(
        current_user_id=current_user_id,
        tab=args.get("tab", "active"),
        search=args.get("search"),
        sort=args.get("sort", "last_activity"),
        dept_id=int(args["dept_id"]) if args.get("dept_id") else None,
        received_from=received_from,
        received_to=received_to,
        has_units=args.get("has_units"),
        page=int(args.get("page", 1)),
        per_page=int(args.get("per_page", 30)),
    )

    items = [_enrich_task(t, current_user_id) for t in result["items"]]
    return jsonify({
        "items": items,
        "total": result["total"],
        "page": result["page"],
        "per_page": result["per_page"],
    }), 200


@bp.post("")
@require_role(EMPLOYEE)
def create_task():
    """
    Создать задачу.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [name, department_id]
            properties:
              name: {type: string}
              link_yougile: {type: string}
              department_id: {type: integer}
              received_at: {type: string, format: date}
              deadline: {type: string, format: date}
    responses:
      201:
        description: Задача создана
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    try:
        task = task_service.create_task(author_id=current_user_id, **data)
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    task_data = _enrich_task(task, current_user_id)
    socketio.emit("task:created", task_data, room="all")

    return jsonify(task_data), 201


@bp.get("/<int:task_id>")
@require_role(EMPLOYEE)
def get_task(task_id: int):
    """
    Получить задачу по ID.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Задача
      404:
        description: Не найдена
    """
    task = task_repo.get_by_id(task_id)
    if task is None:
        return jsonify({"error": "NOT_FOUND", "message": "Задача не найдена"}), 404

    current_user_id = int(get_jwt_identity())
    return jsonify(_enrich_task(task, current_user_id)), 200


@bp.patch("/<int:task_id>")
@require_role(EMPLOYEE)
def update_task(task_id: int):
    """
    Редактировать задачу.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
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
              link_yougile: {type: string}
              department_id: {type: integer}
              received_at: {type: string, format: date}
              deadline: {type: string, format: date}
    responses:
      200:
        description: Задача обновлена
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    try:
        task = task_service.update_task(task_id, **data)
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    task_data = _enrich_task(task, current_user_id)
    socketio.emit("task:updated", task_data, room="all")

    return jsonify(task_data), 200


@bp.delete("/<int:task_id>")
@require_role(EMPLOYEE)
def delete_task(task_id: int):
    """
    Удалить задачу.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Задача удалена
    """
    current_user_id = int(get_jwt_identity())
    try:
        task_service.delete_task(task_id)
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    socketio.emit("task:deleted", {"task_id": task_id}, room="all")

    return jsonify({"message": "Задача удалена"}), 200


@bp.post("/<int:task_id>/archive")
@require_role(EMPLOYEE)
def archive_task(task_id: int):
    """
    Архивировать задачу.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Задача архивирована
      422:
        description: У задачи есть активный юнит
    """
    current_user_id = int(get_jwt_identity())
    try:
        task = task_service.archive_task(task_id)
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    socketio.emit("task:archived", {"task_id": task_id, "archived_at": task.archived_at.isoformat()}, room="all")

    return jsonify(_task_schema.dump(task)), 200


@bp.post("/<int:task_id>/restore")
@require_role(EMPLOYEE)
def restore_task(task_id: int):
    """
    Восстановить задачу из архива.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Задача восстановлена
    """
    current_user_id = int(get_jwt_identity())
    try:
        task = task_service.restore_task(task_id)
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    socketio.emit("task:restored", {"task_id": task_id}, room="all")

    return jsonify(_task_schema.dump(task)), 200


@bp.post("/<int:task_id>/favorite")
@require_auth
def toggle_favorite(task_id: int):
    """
    Добавить/убрать задачу из избранного.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Статус избранного изменён
    """
    current_user_id = int(get_jwt_identity())
    try:
        is_fav = task_service.toggle_favorite(task_id, current_user_id)
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify({"is_favorite": is_fav}), 200


@bp.get("/<int:task_id>/units")
@require_role(EMPLOYEE)
def get_task_units(task_id: int):
    """
    Список юнитов задачи.
    ---
    tags: [units]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Список юнитов
    """
    task = task_repo.get_by_id(task_id)
    if task is None:
        return jsonify({"error": "NOT_FOUND", "message": "Задача не найдена"}), 404

    units = unit_repo.get_by_task(task_id)
    return jsonify(UnitSchema(many=True).dump(units)), 200


@bp.post("/<int:task_id>/units")
@require_role(EMPLOYEE)
def create_unit(task_id: int):
    """
    Создать юнит для задачи.
    ---
    tags: [units]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: task_id
        schema: {type: integer}
        required: true
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [name, unit_type_id]
            properties:
              name: {type: string}
              unit_type_id: {type: integer}
    responses:
      201:
        description: Юнит создан
      409:
        description: Уже есть активный юнит
    """
    from app.schemas.unit import UnitCreateSchema
    from app.services import unit_service
    from app.services.unit_service import UnitServiceError

    try:
        data = UnitCreateSchema().load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    try:
        unit = unit_service.create_unit(
            task_id=task_id,
            name=data["name"],
            unit_type_id=data["unit_type_id"],
            user_id=current_user_id,
        )
    except UnitServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    unit_data = UnitSchema().dump(unit)
    socketio.emit("unit:started", unit_data, room="all")

    return jsonify(unit_data), 201

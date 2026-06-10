from datetime import datetime, timedelta, timezone
from flask import Blueprint, request, jsonify, g
from flask_jwt_extended import get_jwt_identity
from marshmallow import ValidationError

from app.schemas import (
    TaskSchema, TaskCreateSchema, TaskUpdateSchema, TaskColorSchema,
    TaskResponsibleSchema, TaskStageSchema,
    CommentSchema, CommentCreateSchema, CommentUpdateSchema,
)
from app.schemas.unit import UnitSchema
from app.services import task_service, comment_service
from app.services.task_service import TaskServiceError
from app.services.comment_service import CommentServiceError
from app.repositories import task_repo, unit_repo, comment_repo, company_repo
from app.utils.permissions import require_role, require_auth, require_company_scope, EMPLOYEE

bp = Blueprint("tasks", __name__, url_prefix="/api/tasks")

_task_schema = TaskSchema()
_unit_schema = UnitSchema(many=True)
_create_schema = TaskCreateSchema()
_update_schema = TaskUpdateSchema()
_color_schema = TaskColorSchema()
_responsible_schema = TaskResponsibleSchema()
_stage_schema = TaskStageSchema()
_comment_schema = CommentSchema()
_comments_schema = CommentSchema(many=True)
_comment_create_schema = CommentCreateSchema()
_comment_update_schema = CommentUpdateSchema()


def _yougile_enabled(company_id):
    if company_id is None:
        return True
    company = company_repo.get_by_id(company_id)
    if company is None or not company.settings:
        return True
    return bool(company.settings.get("uses_yougile", True))


def _enrich_task(task, current_user_id: int, active_users: list = None, user_color: str = None,
                 yougile_enabled: bool | None = None, is_favorite: bool | None = None,
                 has_units: bool | None = None) -> dict:
    data = _task_schema.dump(task)
    data["is_favorite"] = is_favorite if is_favorite is not None else task_repo.is_favorite(task.id, current_user_id)
    data["has_units"] = has_units if has_units is not None else task_repo.has_any_units(task.id)
    data["active_users"] = active_users if active_users is not None else task_repo.get_active_users(task.id)
    # Цвет — индивидуальный для каждого пользователя (см. user_task_colors).
    data["color"] = user_color if user_color is not None else task_repo.get_user_color(task.id, current_user_id)
    # YouGile-ссылку отдаём только если в компании включена эта интеграция.
    enabled = yougile_enabled if yougile_enabled is not None else _yougile_enabled(task.company_id)
    if not enabled:
        data["link_yougile"] = None
    return data


@bp.get("")
@require_role(EMPLOYEE)
@require_company_scope
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
      - {in: query, name: created_by_me, schema: {type: integer, enum: [0, 1]}, description: Только созданные текущим пользователем}
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

    # Поиск. Если у компании включён AI — целиком семантический по всем
    # проиндексированным задачам. Иначе — обычный LIKE по названию.
    # Никаких гибридов: пользователь либо ищет «по смыслу», либо нет.
    search_q = (args.get("search") or "").strip()
    ordered_ids = None
    if search_q and g.company_id is not None:
        from app.services.ai_client import get_ai_client
        if get_ai_client(g.company_id) is not None:
            from app.services.task_embedding_service import semantic_search
            try:
                hits = semantic_search(g.company_id, search_q)
            except Exception:
                hits = []
            # При включённом AI всегда отдаём семантическую выдачу — даже
            # пустую. Если задача не проиндексирована, LIKE её всё равно
            # бы нашёл «не туда»: запрос «исправить логику авторизации»
            # не должен матчить «логирование событий». Правильный путь —
            # бэкфилл индексов, кнопка для него есть в настройках AI.
            ordered_ids = [tid for tid, _ in hits]

    result = task_repo.get_list(
        current_user_id=current_user_id,
        company_id=g.company_id,
        tab=args.get("tab", "active"),
        search=search_q or None,
        sort=args.get("sort", "last_activity"),
        dept_id=int(args["dept_id"]) if args.get("dept_id") else None,
        stage_id=int(args["stage_id"]) if args.get("stage_id") else None,
        responsible_user_id=int(args["responsible_id"]) if args.get("responsible_id") else None,
        received_from=received_from,
        received_to=received_to,
        has_units=args.get("has_units"),
        author_id=current_user_id if args.get("created_by_me") == "1" else None,
        page=int(args.get("page", 1)),
        per_page=int(args.get("per_page", 30)),
        ordered_ids=ordered_ids,
    )

    task_ids = [t.id for t in result["items"]]
    active_users_map = task_repo.get_active_users_by_task_ids(task_ids)
    user_colors_map = task_repo.get_user_colors_by_task_ids(task_ids, current_user_id)
    favorite_ids = task_repo.get_favorite_task_ids(task_ids, current_user_id)
    with_units_ids = task_repo.get_task_ids_with_units(task_ids)
    yougile_enabled = _yougile_enabled(g.company_id)
    items = [
        _enrich_task(
            t, current_user_id,
            active_users_map.get(t.id, []),
            user_colors_map.get(t.id),
            yougile_enabled,
            is_favorite=(t.id in favorite_ids),
            has_units=(t.id in with_units_ids),
        )
        for t in result["items"]
    ]
    return jsonify({
        "items": items,
        "total": result["total"],
        "page": result["page"],
        "per_page": result["per_page"],
    }), 200


@bp.get("/stale")
@require_role(EMPLOYEE)
@require_company_scope
def stale_tasks():
    """
    Активные задачи, «висящие» дольше недели (по дате поступления).
    Используется для ежедневного напоминания «пора закрыть».
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: days, schema: {type: integer, default: 7}, description: Порог «давности» в днях}
    responses:
      200:
        description: Список давних задач
    """
    try:
        days = max(1, int(request.args.get("days", 7)))
    except (TypeError, ValueError):
        days = 7

    now = datetime.now(timezone.utc)
    threshold = now - timedelta(days=days)
    tasks = task_repo.get_stale(threshold, company_id=g.company_id)

    items = []
    for t in tasks:
        data = _task_schema.dump(t)
        received = t.received_at
        # received_at хранится timezone-aware; на всякий случай нормализуем.
        if received.tzinfo is None:
            received = received.replace(tzinfo=timezone.utc)
        data["days_pending"] = (now - received).days
        items.append(data)

    return jsonify({"items": items, "days": days, "total": len(items)}), 200


@bp.post("")
@require_role(EMPLOYEE)
@require_company_scope
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
        task = task_service.create_task(
            author_id=current_user_id,
            company_id=g.company_id,
            **data,
        )
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    task_data = _enrich_task(task, current_user_id)
    # Цвет — личный, не транслируем (получатели применят свой при следующем fetch).
    broadcast = {k: v for k, v in task_data.items() if k != "color"}
    socketio.emit("task:created", broadcast, room="all")

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

    # Двусторонняя синхра с YouGile: если задача связана и пользователь
    # подключён, пушим изменения в YG. Best-effort — внутри функции
    # все ошибки логируются и НЕ пробрасываются.
    try:
        from app.integrations.yougile.task_push import push_after_update
        push_after_update(g.current_user, task, set(data.keys()))
    except Exception:  # noqa: BLE001
        pass

    from app.extensions import socketio
    task_data = _enrich_task(task, current_user_id)
    # Цвет — индивидуальный, не транслируем чужим клиентам.
    broadcast = {k: v for k, v in task_data.items() if k != "color"}
    socketio.emit("task:updated", broadcast, room="all")

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
        task = task_service.archive_task(task_id, actor_id=current_user_id)
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    try:
        from app.integrations.yougile.task_push import push_after_archive
        push_after_archive(g.current_user, task, archived=True)
    except Exception:  # noqa: BLE001
        pass

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

    try:
        from app.integrations.yougile.task_push import push_after_archive
        push_after_archive(g.current_user, task, archived=False)
    except Exception:  # noqa: BLE001
        pass

    from app.extensions import socketio
    socketio.emit("task:restored", {"task_id": task_id}, room="all")

    return jsonify(_task_schema.dump(task)), 200


@bp.put("/<int:task_id>/color")
@require_role(EMPLOYEE)
def set_task_color(task_id: int):
    """
    Установить или снять цвет карточки задачи для текущего пользователя.
    Цвет индивидуален: одна и та же задача может выглядеть у разных
    пользователей в разный цвет.
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
              color:
                type: string
                nullable: true
                description: id цвета из набора или null чтобы убрать
    responses:
      200:
        description: Цвет применён
      404:
        description: Задача не найдена
    """
    try:
        data = _color_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    task = task_repo.get_by_id(task_id)
    if task is None:
        return jsonify({"error": "NOT_FOUND", "message": "Задача не найдена"}), 404

    current_user_id = int(get_jwt_identity())
    try:
        task_service.set_user_color(task_id, current_user_id, data.get("color"))
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify({"task_id": task_id, "color": data.get("color")}), 200


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


# === v3: ответственный, этап, контрибьюторы, комментарии ===========================


@bp.patch("/<int:task_id>/responsible")
@require_role(EMPLOYEE)
def set_task_responsible(task_id: int):
    """
    Назначить ответственного по задаче.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: path, name: task_id, schema: {type: integer}, required: true}
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              responsible_user_id: {type: integer, nullable: true}
    responses:
      200: {description: Ответственный обновлён}
    """
    try:
        data = _responsible_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    try:
        task = task_service.set_responsible(task_id, data["responsible_user_id"])
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    task_data = _enrich_task(task, current_user_id)
    broadcast = {k: v for k, v in task_data.items() if k != "color"}
    socketio.emit("task:updated", broadcast, room="all")
    return jsonify(task_data), 200


@bp.patch("/<int:task_id>/stage")
@require_role(EMPLOYEE)
def set_task_stage(task_id: int):
    """
    Установить этап задачи (канбан / drag-drop).
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: path, name: task_id, schema: {type: integer}, required: true}
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              stage_id: {type: integer, nullable: true}
    responses:
      200: {description: Этап обновлён}
    """
    try:
        data = _stage_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    try:
        task = task_service.set_stage(task_id, data["stage_id"])
    except TaskServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    task_data = _enrich_task(task, current_user_id)
    broadcast = {k: v for k, v in task_data.items() if k != "color"}
    socketio.emit("task:updated", broadcast, room="all")
    return jsonify(task_data), 200


@bp.get("/<int:task_id>/contributors")
@require_role(EMPLOYEE)
def get_task_contributors(task_id: int):
    """
    Сотрудники, работавшие над задачей (distinct по юнитам).
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: path, name: task_id, schema: {type: integer}, required: true}
    responses:
      200: {description: Список сотрудников}
    """
    task = task_repo.get_by_id(task_id)
    if task is None:
        return jsonify({"error": "NOT_FOUND", "message": "Задача не найдена"}), 404
    return jsonify({"items": task_repo.get_contributors(task_id)}), 200


@bp.get("/<int:task_id>/comments")
@require_role(EMPLOYEE)
def list_task_comments(task_id: int):
    """
    Список комментариев задачи.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: path, name: task_id, schema: {type: integer}, required: true}
    responses:
      200: {description: Список комментариев}
    """
    try:
        comments = comment_service.list_comments(task_id)
    except CommentServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify({"items": _comments_schema.dump(comments)}), 200


@bp.post("/<int:task_id>/comments")
@require_role(EMPLOYEE)
def create_task_comment(task_id: int):
    """
    Создать комментарий.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: path, name: task_id, schema: {type: integer}, required: true}
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [text]
            properties:
              text: {type: string}
    responses:
      201: {description: Комментарий создан}
    """
    try:
        data = _comment_create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    try:
        comment = comment_service.create_comment(task_id, current_user_id, data["text"])
    except CommentServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    payload = _comment_schema.dump(comment)
    socketio.emit("comment:new", payload, room="all")
    return jsonify(payload), 201


@bp.patch("/<int:task_id>/comments/<int:comment_id>")
@require_role(EMPLOYEE)
def update_task_comment(task_id: int, comment_id: int):
    """
    Редактировать комментарий.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: path, name: task_id, schema: {type: integer}, required: true}
      - {in: path, name: comment_id, schema: {type: integer}, required: true}
    requestBody:
      required: true
      content:
        application/json:
          schema: {type: object, required: [text], properties: {text: {type: string}}}
    responses:
      200: {description: Обновлён}
    """
    try:
        data = _comment_update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = int(get_jwt_identity())
    try:
        comment = comment_service.update_comment(comment_id, current_user_id, data["text"])
    except CommentServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    payload = _comment_schema.dump(comment)
    socketio.emit("comment:updated", payload, room="all")
    return jsonify(payload), 200


@bp.delete("/<int:task_id>/comments/<int:comment_id>")
@require_role(EMPLOYEE)
def delete_task_comment(task_id: int, comment_id: int):
    """
    Удалить (soft-delete) комментарий.
    ---
    tags: [tasks]
    security: [BearerAuth: []]
    parameters:
      - {in: path, name: task_id, schema: {type: integer}, required: true}
      - {in: path, name: comment_id, schema: {type: integer}, required: true}
    responses:
      200: {description: Удалён}
    """
    current_user_id = int(get_jwt_identity())
    try:
        comment_service.delete_comment(comment_id, current_user_id)
    except CommentServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    socketio.emit("comment:deleted", {"task_id": task_id, "comment_id": comment_id}, room="all")
    return jsonify({"message": "Удалён"}), 200

from flask import Blueprint, request, jsonify, send_file, current_app
from flask_jwt_extended import get_jwt_identity
from marshmallow import ValidationError
from io import BytesIO

from app.schemas import UserSchema, UserCreateSchema, UserUpdateSchema, UserMeUpdateSchema
from app.services import user_service
from app.services.user_service import UserServiceError
from app.repositories import user_repo
from app.utils.permissions import require_permission, require_auth, Section, Bit
from app.utils.avatar import generate_identicon

bp = Blueprint("users", __name__, url_prefix="/api/users")

_user_schema = UserSchema()
_users_schema = UserSchema(many=True)
_create_schema = UserCreateSchema()
_update_schema = UserUpdateSchema()
_me_schema = UserMeUpdateSchema()


@bp.get("")
@require_permission(Section.USERS, Bit.VIEW)
def list_users():
    """
    Список пользователей.
    ---
    tags: [users]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список пользователей
    """
    users = user_repo.get_all()
    return jsonify(_users_schema.dump(users)), 200


@bp.post("")
@require_permission(Section.USERS, Bit.CREATE)
def create_user():
    """
    Создать пользователя.
    ---
    tags: [users]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [fio, login, role_id]
            properties:
              fio: {type: string}
              login: {type: string, minLength: 3}
              post: {type: string}
              role_id: {type: integer}
    responses:
      201:
        description: Пользователь создан
    """
    try:
        data = _create_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    try:
        user = user_service.create_user(**data)
    except UserServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify(_user_schema.dump(user)), 201


@bp.get("/me")
@require_auth
def get_me():
    """
    Получить текущего пользователя.
    ---
    tags: [users]
    security: [BearerAuth: []]
    responses:
      200:
        description: Данные текущего пользователя
    """
    user_id = get_jwt_identity()
    user = user_repo.get_by_id(user_id)
    if user is None:
        return jsonify({"error": "NOT_FOUND", "message": "Пользователь не найден"}), 404
    return jsonify(_user_schema.dump(user)), 200


@bp.patch("/me")
@require_auth
def update_me():
    """
    Редактировать свой профиль.
    ---
    tags: [users]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              fio: {type: string}
              login: {type: string}
              post: {type: string}
              current_password: {type: string}
              new_password: {type: string, minLength: 8}
              confirm_password: {type: string}
    responses:
      200:
        description: Профиль обновлён
    """
    try:
        data = _me_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    user_id = get_jwt_identity()
    try:
        user = user_service.update_me(user_id=user_id, **data)
    except UserServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify(_user_schema.dump(user)), 200


@bp.post("/me/avatar")
@require_auth
def upload_avatar():
    """
    Загрузить аватарку (multipart/form-data, поле file).
    ---
    tags: [users]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        multipart/form-data:
          schema:
            type: object
            properties:
              file:
                type: string
                format: binary
    responses:
      200:
        description: Аватарка сохранена
      400:
        description: Недопустимый тип файла или размер
    """
    if "file" not in request.files:
        return jsonify({"error": "NO_FILE", "message": "Файл не передан"}), 400

    file = request.files["file"]
    file_bytes = file.read()

    if len(file_bytes) > 2 * 1024 * 1024:
        return jsonify({"error": "FILE_TOO_LARGE", "message": "Файл превышает 2 МБ"}), 400

    user_id = get_jwt_identity()
    try:
        user = user_service.upload_avatar(user_id, file_bytes)
    except (UserServiceError, ValueError) as e:
        msg = e.message if hasattr(e, "message") else str(e)
        code = e.code if hasattr(e, "code") else "UPLOAD_ERROR"
        status = e.http_status if hasattr(e, "http_status") else 400
        return jsonify({"error": code, "message": msg}), status

    return jsonify(_user_schema.dump(user)), 200


@bp.delete("/me/avatar")
@require_auth
def delete_avatar():
    """
    Удалить аватарку (вернуть identicon).
    ---
    tags: [users]
    security: [BearerAuth: []]
    responses:
      200:
        description: Аватарка удалена
    """
    user_id = get_jwt_identity()
    try:
        user = user_service.delete_user_avatar(user_id)
    except UserServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify(_user_schema.dump(user)), 200


@bp.get("/<int:user_id>")
@require_permission(Section.USERS, Bit.VIEW)
def get_user(user_id: int):
    """
    Получить пользователя по ID.
    ---
    tags: [users]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: user_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Данные пользователя
      404:
        description: Не найден
    """
    user = user_repo.get_by_id(user_id)
    if user is None or user.is_hidden:
        return jsonify({"error": "NOT_FOUND", "message": "Пользователь не найден"}), 404
    return jsonify(_user_schema.dump(user)), 200


@bp.patch("/<int:user_id>")
@require_permission(Section.USERS, Bit.EDIT)
def update_user(user_id: int):
    """
    Редактировать пользователя.
    ---
    tags: [users]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: user_id
        schema: {type: integer}
        required: true
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              fio: {type: string}
              login: {type: string}
              post: {type: string}
              role_id: {type: integer}
    responses:
      200:
        description: Пользователь обновлён
    """
    try:
        data = _update_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    current_user_id = get_jwt_identity()
    try:
        user = user_service.update_user(user_id, current_user_id, **data)
    except UserServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify(_user_schema.dump(user)), 200


@bp.delete("/<int:user_id>")
@require_permission(Section.USERS, Bit.DELETE)
def hide_user(user_id: int):
    """
    Скрыть пользователя (soft delete).
    ---
    tags: [users]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: user_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Пользователь скрыт
      422:
        description: Бизнес-правило нарушено
    """
    current_user_id = get_jwt_identity()
    try:
        user_service.hide_user(user_id, current_user_id)
    except UserServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify({"message": "Пользователь скрыт"}), 200


@bp.patch("/<int:user_id>/role")
@require_permission(Section.ROLES, Bit.ASSIGN)
def assign_role(user_id: int):
    """
    Назначить роль пользователю.
    ---
    tags: [users]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: user_id
        schema: {type: integer}
        required: true
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [role_id]
            properties:
              role_id: {type: integer}
    responses:
      200:
        description: Роль назначена
    """
    data = request.get_json(silent=True) or {}
    role_id = data.get("role_id")
    if not role_id:
        return jsonify({"error": "VALIDATION_ERROR", "message": "role_id обязателен"}), 400

    current_user_id = get_jwt_identity()
    try:
        user = user_service.assign_role(user_id, role_id, current_user_id)
    except UserServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify(_user_schema.dump(user)), 200


@bp.get("/<int:user_id>/identicon")
def get_identicon(user_id: int):
    """
    Получить identicon пользователя (PNG).
    ---
    tags: [users]
    parameters:
      - in: path
        name: user_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: PNG identicon
        content:
          image/png: {}
    """
    upload_folder = current_app.config["UPLOAD_FOLDER"]
    png_bytes = generate_identicon(user_id, upload_folder)
    return send_file(BytesIO(png_bytes), mimetype="image/png")

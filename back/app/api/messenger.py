from flask import Blueprint, request, jsonify
from flask_jwt_extended import get_jwt_identity
from marshmallow import ValidationError

from app.schemas import (
    MessageSchema, AttachmentSchema, ConversationListItemSchema,
    ConversationSchema, MessageCreateSchema, ConversationCreateSchema,
    UserDirectorySchema,
)
from app.services import messenger_service
from app.services.messenger_service import MessengerServiceError
from app.repositories import message_repo, user_repo
from app.utils.permissions import require_auth

bp = Blueprint("messenger", __name__, url_prefix="/api/messenger")

_msg = MessageSchema()
_msgs = MessageSchema(many=True)
_att = AttachmentSchema()
_conv_list = ConversationListItemSchema(many=True)
_conv = ConversationSchema()
_msg_create = MessageCreateSchema()
_conv_create = ConversationCreateSchema()
_dir = UserDirectorySchema()


@bp.get("/conversations")
@require_auth
def list_conversations():
    """
    Список диалогов текущего пользователя.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список диалогов с последним сообщением и счётчиком непрочитанных
    """
    user_id = int(get_jwt_identity())
    items = message_repo.list_user_conversations(user_id)
    return jsonify(_conv_list.dump(items)), 200


@bp.post("/conversations")
@require_auth
def open_or_create_conversation():
    """
    Найти или создать диалог с пользователем.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [user_id]
            properties:
              user_id: {type: integer}
    responses:
      200:
        description: Диалог
    """
    try:
        data = _conv_create.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    me = int(get_jwt_identity())
    try:
        conv = messenger_service.open_conversation(me, data["user_id"])
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    other = user_repo.get_by_id(conv.other_user_id(me))
    return jsonify({**_conv.dump(conv), "other_user": _dir.dump(other)}), 200


@bp.get("/conversations/<int:conversation_id>/messages")
@require_auth
def list_messages(conversation_id: int):
    """
    Сообщения диалога. Курсорная пагинация по before_id (старые → новые).
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: conversation_id
        schema: {type: integer}
        required: true
      - in: query
        name: before_id
        schema: {type: integer}
        required: false
      - in: query
        name: limit
        schema: {type: integer, default: 50}
        required: false
    responses:
      200:
        description: Сообщения
    """
    me = int(get_jwt_identity())
    try:
        conv = messenger_service.get_conversation_for_user(conversation_id, me)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    before_id = request.args.get("before_id", type=int)
    limit = min(int(request.args.get("limit", 50)), 200)
    msgs = message_repo.list_messages(conv.id, user_id=me, before_id=before_id, limit=limit)
    return jsonify(_msgs.dump(msgs)), 200


@bp.post("/conversations/<int:conversation_id>/messages")
@require_auth
def post_message(conversation_id: int):
    """
    Отправить сообщение.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: conversation_id
        schema: {type: integer}
        required: true
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              text: {type: string}
              attachment_ids:
                type: array
                items: {type: integer}
    responses:
      201:
        description: Сообщение отправлено
    """
    try:
        data = _msg_create.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    me = int(get_jwt_identity())
    try:
        conv, msg = messenger_service.send_message(
            conversation_id, me,
            text=data.get("text"),
            attachment_ids=data.get("attachment_ids") or [],
        )
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    payload = _msg.dump(msg)

    # WebSocket: уведомим обе стороны
    from app.extensions import socketio
    recipient_id = conv.other_user_id(me)
    socketio.emit("message:new", {
        "conversation_id": conv.id,
        "message": payload,
        "from_user_id": me,
    }, room=f"user_{recipient_id}")
    # Эхо отправителю — чтобы другие его вкладки/устройства тоже обновились
    socketio.emit("message:new", {
        "conversation_id": conv.id,
        "message": payload,
        "from_user_id": me,
    }, room=f"user_{me}")

    return jsonify(payload), 201


@bp.post("/conversations/<int:conversation_id>/read")
@require_auth
def mark_read(conversation_id: int):
    """
    Пометить все входящие сообщения диалога как прочитанные.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: conversation_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Помечено как прочитанное
    """
    me = int(get_jwt_identity())
    try:
        n = messenger_service.mark_conversation_read(conversation_id, me)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    if n > 0:
        from app.extensions import socketio
        conv = message_repo.get_conversation(conversation_id)
        other_id = conv.other_user_id(me)
        socketio.emit("message:read", {
            "conversation_id": conversation_id,
            "reader_id": me,
        }, room=f"user_{other_id}")

    return jsonify({"updated": n}), 200


@bp.post("/uploads")
@require_auth
def upload_attachment():
    """
    Загрузить файл для последующей отправки в сообщении.
    ---
    tags: [messenger]
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
      201:
        description: Файл сохранён, возвращён id для прикрепления к сообщению
    """
    if "file" not in request.files:
        return jsonify({"error": "NO_FILE", "message": "Файл не передан"}), 400

    me = int(get_jwt_identity())
    try:
        att = messenger_service.upload_attachment(me, request.files["file"])
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify(_att.dump(att)), 201


@bp.delete("/messages/<int:message_id>")
@require_auth
def delete_message_endpoint(message_id: int):
    """
    Удалить сообщение. scope=me (скрыть у себя) или all (удалить у всех —
    только своё). При scope=all — broadcast message:deleted обоим участникам.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: message_id
        schema: {type: integer}
        required: true
      - in: query
        name: scope
        schema: {type: string, enum: [me, all], default: me}
    responses:
      200:
        description: Сообщение удалено
    """
    scope = (request.args.get("scope") or "me").lower()
    me = int(get_jwt_identity())
    try:
        conv_id, for_all = messenger_service.delete_message(message_id, me, scope)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    if for_all:
        from app.extensions import socketio
        conv = message_repo.get_conversation(conv_id)
        if conv:
            other_id = conv.other_user_id(me)
            payload = {"conversation_id": conv_id, "message_id": message_id}
            socketio.emit("message:deleted", payload, room=f"user_{other_id}")
            socketio.emit("message:deleted", payload, room=f"user_{me}")

    return jsonify({"deleted": True, "scope": scope, "for_all": for_all}), 200


@bp.delete("/conversations/<int:conversation_id>")
@require_auth
def delete_conversation_endpoint(conversation_id: int):
    """
    Удалить диалог. scope=me (скрыть у себя — собеседник продолжит видеть
    переписку) или all (удалить у обоих — широковещательно).
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: conversation_id
        schema: {type: integer}
        required: true
      - in: query
        name: scope
        schema: {type: string, enum: [me, all], default: me}
    responses:
      200:
        description: Диалог удалён
    """
    scope = (request.args.get("scope") or "me").lower()
    me = int(get_jwt_identity())

    # Запомним собеседника ДО удаления, чтобы было кому слать broadcast.
    conv = message_repo.get_conversation(conversation_id)
    other_id = conv.other_user_id(me) if conv and me in (conv.user_a_id, conv.user_b_id) else None

    try:
        physical = messenger_service.delete_conversation(conversation_id, me, scope)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    if scope == 'all' and other_id is not None:
        from app.extensions import socketio
        payload = {"conversation_id": conversation_id}
        socketio.emit("conversation:deleted", payload, room=f"user_{other_id}")
        socketio.emit("conversation:deleted", payload, room=f"user_{me}")
    elif physical and other_id is not None:
        # Обе стороны независимо нажали «у себя» — физического чата больше нет,
        # уведомим самого пользователя на другие вкладки (собеседнику уже не нужно).
        from app.extensions import socketio
        socketio.emit("conversation:deleted",
                      {"conversation_id": conversation_id},
                      room=f"user_{me}")

    return jsonify({"deleted": True, "scope": scope, "physical": physical}), 200


@bp.post("/conversations/<int:conversation_id>/pin")
@require_auth
def toggle_pin(conversation_id: int):
    """
    Закрепить/открепить диалог (личное действие).
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: conversation_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Закрепление переключено
    """
    me = int(get_jwt_identity())
    try:
        pinned = messenger_service.toggle_pin(conversation_id, me)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    # Эхо в другие вкладки этого же пользователя
    from app.extensions import socketio
    socketio.emit("conversation:pin",
                  {"conversation_id": conversation_id, "is_pinned": pinned},
                  room=f"user_{me}")

    return jsonify({"is_pinned": pinned}), 200


@bp.get("/unread")
@require_auth
def unread_count():
    """
    Общее число непрочитанных сообщений у текущего пользователя.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    responses:
      200:
        description: Счётчик непрочитанных
    """
    me = int(get_jwt_identity())
    return jsonify({"total": message_repo.total_unread(me)}), 200

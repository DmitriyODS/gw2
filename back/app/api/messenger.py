from flask import Blueprint, request, jsonify
from flask_jwt_extended import get_jwt_identity
from marshmallow import ValidationError

from app.schemas import (
    MessageSchema, AttachmentSchema, ConversationListItemSchema,
    ConversationSchema, MessageCreateSchema, ConversationCreateSchema,
    UserDirectorySchema, ForwardSchema,
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
_forward = ForwardSchema()
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
    # Гарантируем, что личный чат техподдержки существует у сотрудника
    # компании — он должен быть всегда первым в списке, даже без переписки.
    # У Администратора системы своего dev-чата нет (он отвечает в чужие
    # через /support-inbox).
    me = user_repo.get_by_id(user_id)
    if me is not None and me.company_id is not None:
        try:
            messenger_service.open_dev_chat(user_id)
        except MessengerServiceError:
            pass  # не валим листинг, если создание чем-то заблокировано
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
    after_id = request.args.get("after_id", type=int)
    limit = min(int(request.args.get("limit", 50)), 200)
    msgs = message_repo.list_messages(
        conv.id, user_id=me, before_id=before_id, after_id=after_id, limit=limit,
    )
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
            reply_to_id=data.get("reply_to_id"),
            task_id=data.get("task_id"),
        )
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    payload = _msg.dump(msg)

    from app.extensions import socketio
    payload_event = {
        "conversation_id": conv.id,
        "message": payload,
        "from_user_id": me,
    }
    if conv.is_dev_chat:
        # Спец-чат компании: уведомляем всех сотрудников компании и всех
        # Администраторов системы (личные комнаты `user_{id}`).
        for uid in _dev_chat_user_ids(conv):
            socketio.emit("message:new", payload_event, room=f"user_{uid}")
    elif conv.is_pet_chat:
        # Чат с Грувиком видит только владелец: эхо в его вкладки. Ответ
        # питомца придёт отдельным message:new из groove_ai_service.
        socketio.emit("message:new", payload_event, room=f"user_{me}")
    else:
        recipient_id = conv.other_user_id(me)
        socketio.emit("message:new", payload_event, room=f"user_{recipient_id}")
        # Эхо отправителю — чтобы другие его вкладки/устройства тоже обновились
        socketio.emit("message:new", payload_event, room=f"user_{me}")

    return jsonify(payload), 201


def _dev_chat_user_ids(conv) -> list[int]:
    """Все, кому надо уведомить о новом сообщении в dev-чате:
    владелец чата + все Администраторы системы (`company_id IS NULL`)."""
    from app.models import User
    from app.extensions import db
    rows = db.session.execute(
        db.select(User.id).where(
            User.is_hidden.is_(False),
            db.or_(User.id == conv.user_a_id, User.company_id.is_(None)),
        )
    ).scalars().all()
    return list(rows)


@bp.post("/forward")
@require_auth
def forward_message_endpoint():
    """
    Переслать сообщение в один или несколько диалогов / пользователям.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [message_id]
            properties:
              message_id: {type: integer}
              conversation_ids:
                type: array
                items: {type: integer}
              user_ids:
                type: array
                items: {type: integer}
    responses:
      201:
        description: Сообщение переслано
    """
    try:
        data = _forward.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    me = int(get_jwt_identity())
    try:
        results = messenger_service.forward_message(
            data["message_id"], me,
            conversation_ids=data.get("conversation_ids") or [],
            user_ids=data.get("user_ids") or [],
        )
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    out = []
    for conv, msg in results:
        payload = _msg.dump(msg)
        out.append({"conversation_id": conv.id, "message": payload})
        recipient_id = conv.other_user_id(me)
        for room in (f"user_{recipient_id}", f"user_{me}"):
            socketio.emit("message:new", {
                "conversation_id": conv.id,
                "message": payload,
                "from_user_id": me,
            }, room=room)

    return jsonify({"forwarded": out}), 201


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
        payload = {"conversation_id": conversation_id, "reader_id": me}
        if conv.is_dev_chat:
            # В dev-чате о прочтении надо знать всем, кто видит эту переписку:
            # владельцу и всем Администраторам системы.
            for uid in _dev_chat_user_ids(conv):
                if uid != me:
                    socketio.emit("message:read", payload, room=f"user_{uid}")
        elif not conv.is_pet_chat:
            other_id = conv.other_user_id(me)
            socketio.emit("message:read", payload, room=f"user_{other_id}")

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


@bp.post("/messages/<int:message_id>/pin")
@require_auth
def toggle_message_pin_endpoint(message_id: int):
    """
    Закрепить/открепить сообщение в диалоге (видят оба участника).
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    parameters:
      - in: path
        name: message_id
        schema: {type: integer}
        required: true
    responses:
      200:
        description: Закрепление переключено
    """
    me = int(get_jwt_identity())
    try:
        conv, msg, pinned = messenger_service.toggle_message_pin(message_id, me)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    from app.extensions import socketio
    payload = {
        "conversation_id": conv.id,
        "message_id": message_id,
        "pinned": pinned,
        "message": _msg.dump(msg),
    }
    socketio.emit("message:pin", payload, room=f"user_{conv.user_a_id}")
    socketio.emit("message:pin", payload, room=f"user_{conv.user_b_id}")

    return jsonify({"pinned": pinned, "message": _msg.dump(msg)}), 200


@bp.get("/conversations/<int:conversation_id>/pinned")
@require_auth
def list_pinned_endpoint(conversation_id: int):
    """
    Закреплённые сообщения диалога.
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
        description: Список закреплённых сообщений
    """
    me = int(get_jwt_identity())
    try:
        msgs = messenger_service.list_pinned_messages(conversation_id, me)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_msgs.dump(msgs)), 200


@bp.get("/presence")
@require_auth
def presence_list():
    """
    Список id пользователей, которые сейчас онлайн.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    responses:
      200:
        description: Онлайн-пользователи
    """
    from app.sockets import presence
    return jsonify({"online": presence.online_user_ids()}), 200


@bp.get("/dev-chat")
@require_auth
def open_dev_chat():
    """
    Открыть/создать ЛИЧНЫЙ чат пользователя с техподдержкой. У каждого
    сотрудника свой чат. Администратор системы своего чата не имеет —
    он отвечает в чужие через /support-inbox.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    responses:
      200:
        description: Чат с техподдержкой
    """
    me = int(get_jwt_identity())
    try:
        conv = messenger_service.open_dev_chat(me)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_conv.dump(conv)), 200


@bp.get("/pet-chat")
@require_auth
def open_pet_chat():
    """
    Открыть/создать чат пользователя со своим Грувиком («Мой Groove»).
    Отвечает ИИ от лица питомца с его характером.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    responses:
      200:
        description: Чат с Грувиком
    """
    me = int(get_jwt_identity())
    try:
        conv = messenger_service.open_pet_chat(me)
    except MessengerServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(_conv.dump(conv)), 200


@bp.get("/support-inbox")
@require_auth
def support_inbox():
    """
    Список всех личных чатов пользователей с техподдержкой. Только для
    Администратора системы — он видит вкладку «Техподдержка» в мессенджере.
    ---
    tags: [messenger]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список dev-чатов всех пользователей
    """
    me = int(get_jwt_identity())
    me_user = user_repo.get_by_id(me)
    if me_user is None or me_user.company_id is not None:
        return jsonify({"error": "FORBIDDEN", "message": "Только Администратор системы"}), 403

    from sqlalchemy.orm import selectinload
    from app.extensions import db
    from app.models import Message, User

    convs = message_repo.list_dev_chats()
    conv_ids = [c.id for c in convs]
    owner_ids = [c.user_a_id for c in convs if c.user_a_id is not None]

    last_msg_by_conv: dict[int, Message] = {}
    unread_by_conv: dict[int, int] = {}
    if conv_ids:
        last_sub = (
            db.select(Message.conversation_id, db.func.max(Message.id).label("last_id"))
            .where(Message.conversation_id.in_(conv_ids))
            .group_by(Message.conversation_id)
            .subquery()
        )
        for msg in db.session.execute(
            db.select(Message)
            .options(
                selectinload(Message.attachments),
                selectinload(Message.sender),
                selectinload(Message.reply_to).selectinload(Message.sender),
                selectinload(Message.reply_to).selectinload(Message.attachments),
                selectinload(Message.forwarded_from),
                selectinload(Message.pinned_by),
                selectinload(Message.call),
                selectinload(Message.task),
                selectinload(Message.conversation),
            )
            .join(last_sub, Message.id == last_sub.c.last_id)
        ).scalars().all():
            last_msg_by_conv[msg.conversation_id] = msg

        unread_rows = db.session.execute(
            db.select(
                Message.conversation_id,
                db.func.count(Message.id).label("unread"),
            ).where(
                Message.conversation_id.in_(conv_ids),
                Message.sender_id.in_(owner_ids),
                Message.read_at.is_(None),
            ).group_by(Message.conversation_id)
        ).all()
        unread_by_conv = {row.conversation_id: int(row.unread) for row in unread_rows}

    owners = db.session.execute(
        db.select(User).where(User.id.in_(owner_ids))
    ).scalars().all()
    owner_by_id = {u.id: u for u in owners}

    items = []
    for c in convs:
        items.append({
            "conversation": c,
            "other_user": None,
            "owner_user": owner_by_id.get(c.user_a_id),
            "last_message": last_msg_by_conv.get(c.id),
            "unread_count": unread_by_conv.get(c.id, 0),
            "is_pinned": False,
            "pinned_at": None,
        })
    return jsonify(_conv_list.dump(items)), 200


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

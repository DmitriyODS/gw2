from marshmallow import Schema, fields, validate

from .user import UserDirectorySchema


class AttachmentSchema(Schema):
    id = fields.Int(dump_only=True)
    file_name = fields.Str(dump_only=True)
    mime_type = fields.Str(dump_only=True)
    size_bytes = fields.Int(dump_only=True)
    url = fields.Method("get_url", dump_only=True)

    def get_url(self, obj):
        return f"/uploads/{obj.file_path}"


class ReplyPreviewSchema(Schema):
    """Краткая выжимка сообщения, на которое отвечают (без рекурсии в reply_to)."""
    id = fields.Int(dump_only=True)
    sender_id = fields.Int(dump_only=True)
    sender_fio = fields.Method("get_sender_fio", dump_only=True)
    text = fields.Method("get_text", dump_only=True, allow_none=True)
    has_attachments = fields.Method("get_has_att", dump_only=True)

    def get_sender_fio(self, obj):
        return obj.sender.fio if obj.sender else None

    def get_text(self, obj):
        if not obj.text:
            return None
        return obj.text[:140]

    def get_has_att(self, obj):
        return bool(obj.attachments)


class CallInfoSchema(Schema):
    """Краткая информация о звонке для плашки в чате (kind='call')."""
    id = fields.Int(dump_only=True)
    kind = fields.Str(dump_only=True)        # p2p | group
    media = fields.Str(dump_only=True)       # audio | video
    status = fields.Str(dump_only=True)      # ringing | active | ended | missed
    started_at = fields.DateTime(dump_only=True)
    ended_at = fields.DateTime(dump_only=True, allow_none=True)
    initiator_id = fields.Int(dump_only=True)
    duration_sec = fields.Method("get_duration", dump_only=True, allow_none=True)

    def get_duration(self, obj):
        if obj.ended_at and obj.started_at:
            return int((obj.ended_at - obj.started_at).total_seconds())
        return None


class MessageSchema(Schema):
    id = fields.Int(dump_only=True)
    conversation_id = fields.Int(dump_only=True)
    sender_id = fields.Int(dump_only=True)
    text = fields.Str(dump_only=True, allow_none=True)
    created_at = fields.DateTime(dump_only=True)
    read_at = fields.DateTime(dump_only=True, allow_none=True)
    attachments = fields.List(fields.Nested(AttachmentSchema), dump_only=True)
    reply_to = fields.Nested(ReplyPreviewSchema, dump_only=True, allow_none=True)
    forwarded_from = fields.Method("get_forwarded_from", dump_only=True, allow_none=True)
    # 'text' — обычное; 'call' — системная плашка о звонке (фронт рендерит
    # отдельным компонентом, текст игнорируется, данные берутся из `call`).
    kind = fields.Str(dump_only=True)
    call = fields.Nested(CallInfoSchema, dump_only=True, allow_none=True)

    def get_forwarded_from(self, obj):
        if not obj.forwarded_from:
            return None
        return {"id": obj.forwarded_from.id, "fio": obj.forwarded_from.fio}


class ConversationListItemSchema(Schema):
    """Элемент списка диалогов. На вход дикт из repo.list_user_conversations."""
    id = fields.Method("get_id", dump_only=True)
    other_user = fields.Nested(UserDirectorySchema, dump_only=True)
    last_message = fields.Nested(MessageSchema, dump_only=True, allow_none=True)
    unread_count = fields.Int(dump_only=True)
    last_message_at = fields.Method("get_last_at", dump_only=True)
    is_pinned = fields.Bool(dump_only=True)
    pinned_at = fields.Method("get_pinned_at", dump_only=True)

    def get_id(self, obj):
        return obj["conversation"].id

    def get_last_at(self, obj):
        return obj["conversation"].last_message_at.isoformat() if obj["conversation"].last_message_at else None

    def get_pinned_at(self, obj):
        return obj["pinned_at"].isoformat() if obj.get("pinned_at") else None


class ConversationSchema(Schema):
    id = fields.Int(dump_only=True)
    user_a_id = fields.Int(dump_only=True)
    user_b_id = fields.Int(dump_only=True)
    created_at = fields.DateTime(dump_only=True)
    last_message_at = fields.DateTime(dump_only=True, allow_none=True)


class MessageCreateSchema(Schema):
    text = fields.Str(load_default=None, allow_none=True, validate=validate.Length(max=10000))
    attachment_ids = fields.List(fields.Int(), load_default=list)
    reply_to_id = fields.Int(load_default=None, allow_none=True)


class ConversationCreateSchema(Schema):
    user_id = fields.Int(required=True)


class ForwardSchema(Schema):
    message_id = fields.Int(required=True)
    conversation_ids = fields.List(fields.Int(), load_default=list)
    user_ids = fields.List(fields.Int(), load_default=list)

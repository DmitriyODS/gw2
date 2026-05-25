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


class MessageSchema(Schema):
    id = fields.Int(dump_only=True)
    conversation_id = fields.Int(dump_only=True)
    sender_id = fields.Int(dump_only=True)
    text = fields.Str(dump_only=True, allow_none=True)
    created_at = fields.DateTime(dump_only=True)
    read_at = fields.DateTime(dump_only=True, allow_none=True)
    attachments = fields.List(fields.Nested(AttachmentSchema), dump_only=True)


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


class ConversationCreateSchema(Schema):
    user_id = fields.Int(required=True)

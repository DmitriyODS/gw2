from marshmallow import Schema, fields, validate


class CallParticipantBriefSchema(Schema):
    """Кто участвует в звонке (для истории и push-входящего)."""
    user_id = fields.Int()
    fio = fields.Method("get_fio")
    avatar_path = fields.Method("get_avatar")
    role = fields.Str()
    joined_at = fields.DateTime()
    left_at = fields.DateTime()
    declined = fields.Bool()

    def get_fio(self, obj):
        return obj.user.fio if obj.user else None

    def get_avatar(self, obj):
        return obj.user.avatar_path if obj.user else None


class CallSchema(Schema):
    """Звонок в истории."""
    id = fields.Int()
    kind = fields.Str()
    status = fields.Str()
    media = fields.Str()
    started_at = fields.DateTime()
    ended_at = fields.DateTime()
    initiator_id = fields.Int()
    initiator_fio = fields.Method("get_initiator_fio")
    conversation_id = fields.Int(allow_none=True)
    duration_sec = fields.Method("get_duration")
    participants = fields.Nested(CallParticipantBriefSchema, many=True)

    def get_initiator_fio(self, obj):
        return obj.initiator.fio if obj.initiator else None

    def get_duration(self, obj):
        if obj.ended_at and obj.started_at:
            return int((obj.ended_at - obj.started_at).total_seconds())
        return None


class CallStartSchema(Schema):
    """Запрос на инициацию звонка."""
    user_ids = fields.List(fields.Int(), required=True, validate=validate.Length(min=1, max=8))
    media = fields.Str(validate=validate.OneOf(["audio", "video"]), load_default="video")
    conversation_id = fields.Int(load_default=None, allow_none=True)

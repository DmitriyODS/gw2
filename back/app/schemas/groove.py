from marshmallow import Schema, fields, validate

# Фиксированный набор реакций — продублирован на фронте в utils/groove.js.
FEED_REACTIONS = ("🔥", "💪", "👏", "🎉", "😮", "❤️")


class FeedUserRefSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)


class FeedEventSchema(Schema):
    id = fields.Int(dump_only=True)
    company_id = fields.Int(dump_only=True)
    kind = fields.Str(dump_only=True)
    payload = fields.Dict(dump_only=True)
    created_at = fields.DateTime(dump_only=True)
    user = fields.Nested(FeedUserRefSchema, dump_only=True, allow_none=True)


class FeedCommentSchema(Schema):
    id = fields.Int(dump_only=True)
    event_id = fields.Int(dump_only=True)
    text = fields.Str(dump_only=True)
    is_bot = fields.Bool(dump_only=True)
    reply_to_id = fields.Int(dump_only=True, allow_none=True)
    created_at = fields.DateTime(dump_only=True)
    author = fields.Nested(FeedUserRefSchema, dump_only=True, allow_none=True)


class FeedReactionToggleSchema(Schema):
    emoji = fields.Str(required=True, validate=validate.OneOf(FEED_REACTIONS))


class FeedCommentCreateSchema(Schema):
    text = fields.Str(required=True, validate=validate.Length(min=1, max=2000))
    reply_to_id = fields.Int(load_default=None, allow_none=True)


class KudosSchema(Schema):
    to_user_id = fields.Int(required=True)
    text = fields.Str(required=True, validate=validate.Length(min=1, max=500))


class ZapSchema(Schema):
    to_user_id = fields.Int(required=True)


class PetSchema(Schema):
    user_id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    species = fields.Str(dump_only=True)
    stage = fields.Int(dump_only=True)
    xp = fields.Int(dump_only=True)
    beans = fields.Int(dump_only=True)
    hat = fields.Str(dump_only=True, allow_none=True)
    accessories = fields.List(fields.Str(), dump_only=True)
    feed_streak = fields.Int(dump_only=True)
    last_fed_date = fields.Date(dump_only=True)
    user = fields.Nested(FeedUserRefSchema, dump_only=True)


class PetRenameSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=50))


class ShopBuySchema(Schema):
    item = fields.Str(required=True, validate=validate.Length(min=1, max=32))


class PetEquipSchema(Schema):
    item = fields.Str(required=False, load_default=None, allow_none=True,
                      validate=validate.Length(max=32))


class PetSpeciesSchema(Schema):
    species = fields.Str(required=True, validate=validate.Length(min=1, max=24))

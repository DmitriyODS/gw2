from marshmallow import Schema, fields, validate


STAGE_COLORS = ["red", "orange", "amber", "green", "teal", "blue", "violet", "pink"]


class StageSchema(Schema):
    id = fields.Int(dump_only=True)
    company_id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    color = fields.Str(dump_only=True)
    order = fields.Int(dump_only=True)


class StageCreateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    color = fields.Str(load_default="blue", validate=validate.OneOf(STAGE_COLORS))


class StageUpdateSchema(Schema):
    name = fields.Str(validate=validate.Length(min=1, max=255))
    color = fields.Str(validate=validate.OneOf(STAGE_COLORS))


class StageReorderSchema(Schema):
    ids = fields.List(fields.Int(), required=True, validate=validate.Length(min=1))

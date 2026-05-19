from marshmallow import Schema, fields, validate


class UnitTypeRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)


class UserRefSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)


class UnitSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    user_id = fields.Int(dump_only=True)
    user = fields.Nested(UserRefSchema, dump_only=True)
    unit_type_id = fields.Int(dump_only=True)
    unit_type = fields.Nested(UnitTypeRefSchema, dump_only=True)
    task_id = fields.Int(dump_only=True)
    is_edited = fields.Bool(dump_only=True)
    datetime_start = fields.DateTime(dump_only=True)
    datetime_end = fields.DateTime(dump_only=True, allow_none=True)
    created_at = fields.DateTime(dump_only=True)


class UnitCreateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=500))
    unit_type_id = fields.Int(required=True)


class UnitUpdateSchema(Schema):
    name = fields.Str(validate=validate.Length(min=1, max=500))
    unit_type_id = fields.Int()
    datetime_start = fields.DateTime()
    datetime_end = fields.DateTime(allow_none=True)

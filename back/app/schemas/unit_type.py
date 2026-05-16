from marshmallow import Schema, fields, validate


class UnitTypeSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)


class UnitTypeCreateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=255))


class UnitTypeUpdateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=255))

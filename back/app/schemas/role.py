from marshmallow import Schema, fields, validate


class RoleSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    access = fields.Int(dump_only=True)


class RoleCreateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=100))
    access = fields.Int(required=True, validate=validate.Range(min=0))


class RoleUpdateSchema(Schema):
    name = fields.Str(validate=validate.Length(min=1, max=100))
    access = fields.Int(validate=validate.Range(min=0))

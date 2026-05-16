from marshmallow import Schema, fields, validate


class DepartmentSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)


class DepartmentCreateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=255))


class DepartmentUpdateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=255))

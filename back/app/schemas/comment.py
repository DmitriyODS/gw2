from marshmallow import Schema, fields, validate


class CommentAuthorSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)


class CommentSchema(Schema):
    id = fields.Int(dump_only=True)
    task_id = fields.Int(dump_only=True)
    text = fields.Str(dump_only=True)
    created_at = fields.DateTime(dump_only=True)
    updated_at = fields.DateTime(dump_only=True, allow_none=True)
    deleted_at = fields.DateTime(dump_only=True, allow_none=True)
    author = fields.Nested(CommentAuthorSchema, dump_only=True)
    author_id = fields.Int(dump_only=True)


class CommentCreateSchema(Schema):
    text = fields.Str(required=True, validate=validate.Length(min=1, max=10000))


class CommentUpdateSchema(Schema):
    text = fields.Str(required=True, validate=validate.Length(min=1, max=10000))

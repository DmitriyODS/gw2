from marshmallow import Schema, fields, validate

# Фиксированный набор цветов-тегов задач (синхронизирован с
# front/src/utils/taskColors.js и токенами --tag-* в tokens.css).
TASK_COLORS = ["red", "orange", "amber", "green", "teal", "blue", "violet", "pink"]


class DeptRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)


class AuthorRefSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)


class TaskSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    created_at = fields.DateTime(dump_only=True)
    author = fields.Nested(AuthorRefSchema, dump_only=True)
    author_id = fields.Int(dump_only=True)
    link_yougile = fields.Str(dump_only=True, allow_none=True)
    received_at = fields.DateTime(dump_only=True)
    department = fields.Nested(DeptRefSchema, dump_only=True)
    department_id = fields.Int(dump_only=True)
    deadline = fields.DateTime(dump_only=True, allow_none=True)
    is_archived = fields.Bool(dump_only=True)
    archived_at = fields.DateTime(dump_only=True, allow_none=True)
    color = fields.Str(dump_only=True, allow_none=True)
    is_favorite = fields.Bool(dump_only=True)
    has_units = fields.Bool(dump_only=True)


class TaskCreateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=500))
    link_yougile = fields.Str(validate=validate.Length(max=2000), load_default=None, allow_none=True)
    department_id = fields.Int(required=True)
    received_at = fields.Date(load_default=None, allow_none=True)
    deadline = fields.Date(load_default=None, allow_none=True)


class TaskUpdateSchema(Schema):
    name = fields.Str(validate=validate.Length(min=1, max=500))
    link_yougile = fields.Str(validate=validate.Length(max=2000), allow_none=True)
    department_id = fields.Int()
    received_at = fields.Date(allow_none=True)
    deadline = fields.Date(allow_none=True)
    color = fields.Str(allow_none=True, validate=validate.OneOf(TASK_COLORS))

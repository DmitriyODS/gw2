from marshmallow import Schema, fields

# Домен задач живёт в tasksvc (back-go/tasks); схема осталась ради
# YouGile-интеграции (дампы задач в ответах /api/yougile и сокет-событиях,
# см. integrations/yougile/task_dump.py). Уйдёт вместе с ней в фазе 4.


class DeptRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)


class AuthorRefSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)


class StageRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    color = fields.Str(dump_only=True)
    order = fields.Int(dump_only=True)


class TaskSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    created_at = fields.DateTime(dump_only=True)
    author = fields.Nested(AuthorRefSchema, dump_only=True)
    author_id = fields.Int(dump_only=True)
    responsible = fields.Nested(AuthorRefSchema, dump_only=True, allow_none=True)
    responsible_user_id = fields.Int(dump_only=True, allow_none=True)
    link_yougile = fields.Str(dump_only=True, allow_none=True)
    received_at = fields.DateTime(dump_only=True)
    department = fields.Nested(DeptRefSchema, dump_only=True)
    department_id = fields.Int(dump_only=True)
    stage = fields.Nested(StageRefSchema, dump_only=True, allow_none=True)
    stage_id = fields.Int(dump_only=True, allow_none=True)
    deadline = fields.DateTime(dump_only=True, allow_none=True)
    is_archived = fields.Bool(dump_only=True)
    archived_at = fields.DateTime(dump_only=True, allow_none=True)
    color = fields.Str(dump_only=True, allow_none=True)
    is_favorite = fields.Bool(dump_only=True)
    has_units = fields.Bool(dump_only=True)
    company_id = fields.Int(dump_only=True)
    # YouGile-привязка: id карточки + структурные id (для бейджа на карточке
    # и подсветки колонки). Полный URL остаётся в link_yougile.
    yougile_task_id = fields.Str(dump_only=True, allow_none=True)
    yougile_id_short = fields.Str(dump_only=True, allow_none=True)
    yougile_project_id = fields.Str(dump_only=True, allow_none=True)
    yougile_board_id = fields.Str(dump_only=True, allow_none=True)
    yougile_column_id = fields.Str(dump_only=True, allow_none=True)

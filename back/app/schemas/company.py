from marshmallow import Schema, fields, validate


class CompanyDirectorRefSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    login = fields.Str(dump_only=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)


class CompanySettingsSchema(Schema):
    uses_yougile = fields.Bool(load_default=False)
    uses_stages = fields.Bool(load_default=False)
    uses_calls = fields.Bool(load_default=True)


class CompanySchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    description = fields.Str(dump_only=True, allow_none=True)
    is_active = fields.Bool(dump_only=True)
    settings = fields.Dict(dump_only=True)
    director = fields.Nested(CompanyDirectorRefSchema, dump_only=True, allow_none=True)
    director_id = fields.Int(dump_only=True, allow_none=True)
    created_at = fields.DateTime(dump_only=True)
    # Виртуальные поля для таблицы — заполняются обработчиком списка.
    employees_count = fields.Int(dump_only=True)
    tasks_count = fields.Int(dump_only=True)


class CompanyCreateSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    description = fields.Str(load_default=None, allow_none=True)
    director_id = fields.Int(load_default=None, allow_none=True)
    is_active = fields.Bool(load_default=True)
    settings = fields.Nested(CompanySettingsSchema, load_default=dict)


class CompanyUpdateSchema(Schema):
    name = fields.Str(validate=validate.Length(min=1, max=255))
    description = fields.Str(allow_none=True)
    director_id = fields.Int(allow_none=True)
    is_active = fields.Bool()
    settings = fields.Nested(CompanySettingsSchema, partial=True)


class CompanyToggleActiveSchema(Schema):
    is_active = fields.Bool(required=True)

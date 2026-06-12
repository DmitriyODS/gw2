from marshmallow import Schema, fields

# CRUD-схемы пользователей и схемы авторизации переехали в Go-микросервис
# back-go/auth (authsvc) вместе с /api/auth/* и /api/users/*. Здесь остался
# только публичный профиль — его вкладывают схемы мессенджера.


class RoleRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    level = fields.Int(dump_only=True)


class UserDirectorySchema(Schema):
    """Публичный профиль для каталога сотрудников и мессенджера —
    без is_default_pass и прочих внутренних полей."""
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    login = fields.Str(dump_only=True)
    post = fields.Str(dump_only=True, allow_none=True)
    role = fields.Nested(RoleRefSchema, dump_only=True)
    company_id = fields.Int(dump_only=True, allow_none=True)
    phone = fields.Str(dump_only=True, allow_none=True)
    email = fields.Str(dump_only=True, allow_none=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)
    last_seen_at = fields.DateTime(dump_only=True, allow_none=True)

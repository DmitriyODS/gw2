from marshmallow import Schema, fields, validate, validates, ValidationError


class RoleRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    level = fields.Int(dump_only=True)


class UserSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    login = fields.Str(dump_only=True)
    post = fields.Str(dump_only=True, allow_none=True)
    role = fields.Nested(RoleRefSchema, dump_only=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)
    is_default_pass = fields.Bool(dump_only=True)
    is_hidden = fields.Bool(dump_only=True)
    created_at = fields.DateTime(dump_only=True)


class UserDirectorySchema(Schema):
    """Публичный профиль для каталога сотрудников и мессенджера —
    без is_default_pass и прочих внутренних полей."""
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    login = fields.Str(dump_only=True)
    post = fields.Str(dump_only=True, allow_none=True)
    role = fields.Nested(RoleRefSchema, dump_only=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)
    last_seen_at = fields.DateTime(dump_only=True, allow_none=True)


class UserCreateSchema(Schema):
    fio = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    login = fields.Str(required=True, validate=validate.Length(min=3, max=100))
    post = fields.Str(validate=validate.Length(max=255), load_default=None)
    role_id = fields.Int(required=True)
    password = fields.Str(validate=validate.Length(min=8), load_default=None)


class UserUpdateSchema(Schema):
    fio = fields.Str(validate=validate.Length(min=1, max=255))
    login = fields.Str(validate=validate.Length(min=3, max=100))
    post = fields.Str(validate=validate.Length(max=255), allow_none=True)


class UserMeUpdateSchema(Schema):
    fio = fields.Str(validate=validate.Length(min=1, max=255))
    login = fields.Str(validate=validate.Length(min=3, max=100))
    post = fields.Str(validate=validate.Length(max=255), allow_none=True)
    current_password = fields.Str(load_default=None)
    new_password = fields.Str(validate=validate.Length(min=8), load_default=None)
    confirm_password = fields.Str(load_default=None)

    @validates("new_password")
    def validate_new_password(self, value):
        if value and len(value) < 8:
            raise ValidationError("Пароль должен содержать минимум 8 символов")


class ChangeDefaultSchema(Schema):
    new_login = fields.Str(required=True, validate=validate.Length(min=3, max=100))
    new_password = fields.Str(required=True, validate=validate.Length(min=8))
    confirm_password = fields.Str(required=True)


class LoginSchema(Schema):
    login = fields.Str(required=True)
    password = fields.Str(required=True)

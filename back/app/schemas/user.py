import re
from marshmallow import Schema, fields, validate, validates, validates_schema, ValidationError, pre_load


# Простой, но достаточный для бизнес-валидации regex для email (RFC5322-base).
_EMAIL_RE = re.compile(r"^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$")

# Нормализуем телефон в +7XXXXXXXXXX. Принимаем варианты с пробелами/скобками/
# дефисами; ведущая 8 заменяется на 7; 11 цифр итог.
def _normalize_phone(raw: str) -> str:
    digits = re.sub(r"\D", "", raw or "")
    if not digits:
        return ""
    if digits.startswith("8") and len(digits) == 11:
        digits = "7" + digits[1:]
    if len(digits) == 10:
        digits = "7" + digits
    if len(digits) != 11 or not digits.startswith("7"):
        raise ValidationError("Телефон должен быть российским мобильным (+7…)")
    return "+" + digits


class RoleRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)
    level = fields.Int(dump_only=True)


class CompanyRefSchema(Schema):
    id = fields.Int(dump_only=True)
    name = fields.Str(dump_only=True)


class UserSchema(Schema):
    id = fields.Int(dump_only=True)
    fio = fields.Str(dump_only=True)
    login = fields.Str(dump_only=True)
    post = fields.Str(dump_only=True, allow_none=True)
    role = fields.Nested(RoleRefSchema, dump_only=True)
    company_id = fields.Int(dump_only=True, allow_none=True)
    company = fields.Nested(CompanyRefSchema, dump_only=True, allow_none=True)
    phone = fields.Str(dump_only=True, allow_none=True)
    email = fields.Str(dump_only=True, allow_none=True)
    avatar_path = fields.Str(dump_only=True, allow_none=True)
    is_default_pass = fields.Bool(dump_only=True)
    is_hidden = fields.Bool(dump_only=True)
    is_root_admin = fields.Bool(dump_only=True)
    created_at = fields.DateTime(dump_only=True)


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


class _ContactsMixin:
    """Общие правила phone/email — переиспользуются в Create/Update/Me-схемах."""
    @staticmethod
    def _validate_phone(value):
        if value is None or value == "":
            return None
        return _normalize_phone(value)

    @staticmethod
    def _validate_email(value):
        if value is None or value == "":
            return None
        if not _EMAIL_RE.match(value):
            raise ValidationError("Неверный формат email")
        return value


class UserCreateSchema(Schema, _ContactsMixin):
    fio = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    login = fields.Str(required=True, validate=validate.Length(min=3, max=100))
    post = fields.Str(validate=validate.Length(max=255), load_default=None)
    role_id = fields.Int(required=True)
    company_id = fields.Int(load_default=None, allow_none=True)
    phone = fields.Str(load_default=None, allow_none=True)
    email = fields.Str(load_default=None, allow_none=True)
    password = fields.Str(validate=validate.Length(min=8), load_default=None)

    @pre_load
    def _normalize(self, data, **_):
        if not isinstance(data, dict):
            return data
        if "phone" in data:
            data["phone"] = self._validate_phone(data["phone"])
        if "email" in data:
            data["email"] = self._validate_email(data["email"])
        return data


class UserUpdateSchema(Schema, _ContactsMixin):
    fio = fields.Str(validate=validate.Length(min=1, max=255))
    login = fields.Str(validate=validate.Length(min=3, max=100))
    post = fields.Str(validate=validate.Length(max=255), allow_none=True)
    company_id = fields.Int(allow_none=True)
    phone = fields.Str(allow_none=True)
    email = fields.Str(allow_none=True)

    @pre_load
    def _normalize(self, data, **_):
        if not isinstance(data, dict):
            return data
        if "phone" in data:
            data["phone"] = self._validate_phone(data["phone"])
        if "email" in data:
            data["email"] = self._validate_email(data["email"])
        return data


class UserMeUpdateSchema(Schema, _ContactsMixin):
    fio = fields.Str(validate=validate.Length(min=1, max=255))
    login = fields.Str(validate=validate.Length(min=3, max=100))
    post = fields.Str(validate=validate.Length(max=255), allow_none=True)
    phone = fields.Str(allow_none=True)
    email = fields.Str(allow_none=True)
    current_password = fields.Str(load_default=None)
    new_password = fields.Str(validate=validate.Length(min=8), load_default=None)
    confirm_password = fields.Str(load_default=None)

    @pre_load
    def _normalize(self, data, **_):
        if not isinstance(data, dict):
            return data
        if "phone" in data:
            data["phone"] = self._validate_phone(data["phone"])
        if "email" in data:
            data["email"] = self._validate_email(data["email"])
        return data

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

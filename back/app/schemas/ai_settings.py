from marshmallow import Schema, fields, validate


class AiSettingsSchema(Schema):
    """Что отдаём наружу. Сырого ключа НЕТ — только маска key_hint."""

    enabled = fields.Boolean(attribute="ai_enabled")
    key_hint = fields.String(attribute="ai_key_hint", allow_none=True)
    has_key = fields.Method("_has_key")
    model_chat = fields.String(attribute="ai_model_chat")
    model_embedding = fields.String(attribute="ai_model_embedding")

    def _has_key(self, obj) -> bool:
        return bool(getattr(obj, "ai_api_key_enc", None))


class AiSettingsUpdateSchema(Schema):
    """Все поля опциональные. api_key — отдельно: пустая строка значит «не
    менять» (а не «удалить»). Чтобы удалить ключ, используется отдельный
    флаг clear_key=True."""

    enabled = fields.Boolean()
    api_key = fields.String(validate=validate.Length(max=512), load_default=None,
                            allow_none=True)
    clear_key = fields.Boolean(load_default=False)
    model_chat = fields.String(validate=validate.Length(min=1, max=64))
    model_embedding = fields.String(validate=validate.Length(min=1, max=64))

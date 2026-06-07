"""Схемы для эндпоинтов /api/yougile.

Главное правило: пароль и сам ключ YouGile в ответах НИКОГДА не показываем.
Наружу — только статус, last4 (fingerprint) и публичные id/названия выбранных
project/board/column.
"""
from marshmallow import Schema, fields, validate


# ── Подключение пользователя ───────────────────────────────────────────────

class YougileConnectStartSchema(Schema):
    """Получить список компаний по логину/паролю.

    Используется только на админском визарде — обычному пользователю list
    компаний не нужен, у него yg_company_id зафиксирован настройками.
    """
    login = fields.Email(required=True, validate=validate.Length(max=255))
    password = fields.String(required=True, load_only=True,
                             validate=validate.Length(min=1, max=512))


class YougileConnectFinishSchema(Schema):
    """Создать/перевыпустить ключ и сохранить привязку.

    Для обычного пользователя `yg_company_id` НЕ передаётся: бэк сам берёт
    его из настроек компании. Для админа в визарде — обязателен (он мог
    выбрать его в `/connect/start`).
    """
    login = fields.Email(required=True, validate=validate.Length(max=255))
    password = fields.String(required=True, load_only=True,
                             validate=validate.Length(min=1, max=512))
    yg_company_id = fields.String(load_default=None,
                                  validate=validate.Length(max=64))


class YougileRotateSchema(Schema):
    """Сброс ключа. Требуем повторно ввести пароль — это чувствительная
    операция, JWT-сессии недостаточно."""
    password = fields.String(required=True, load_only=True,
                             validate=validate.Length(min=1, max=512))


# ── Что показываем наружу ──────────────────────────────────────────────────

class YougileAccountStatusSchema(Schema):
    """Статус подключения текущего пользователя."""
    connected = fields.Boolean(required=True)
    yg_login = fields.String(allow_none=True)
    key_fingerprint = fields.String(allow_none=True)
    last_validated_at = fields.DateTime(allow_none=True)
    yg_company_id = fields.String(allow_none=True)
    # Признак «настроенной» интеграции компании — фронту удобно показать
    # фичу или баннер «попросите админа включить YouGile в компании».
    company_enabled = fields.Boolean(required=True)


class YougileCompanyListItemSchema(Schema):
    """Элемент списка `/auth/companies` — отдаём наружу только то, что нужно
    в UI визарда."""
    id = fields.String(required=True)
    name = fields.String(required=True)


class YougileProjectSchema(Schema):
    id = fields.String(required=True)
    title = fields.String(required=True)


class YougileBoardSchema(Schema):
    id = fields.String(required=True)
    title = fields.String(required=True)
    projectId = fields.String(allow_none=True)


class YougileColumnSchema(Schema):
    id = fields.String(required=True)
    title = fields.String(required=True)
    boardId = fields.String(allow_none=True)


# ── Настройки компании ────────────────────────────────────────────────────

class YougileCompanySettingsSchema(Schema):
    """Что отдаём админу: текущая конфигурация интеграции."""
    enabled = fields.Boolean(required=True)
    yg_company_id = fields.String(allow_none=True)
    yg_company_name = fields.String(allow_none=True)
    yg_project_id = fields.String(allow_none=True)
    yg_project_title = fields.String(allow_none=True)
    yg_board_id = fields.String(allow_none=True)
    yg_board_title = fields.String(allow_none=True)
    yg_first_column_id = fields.String(allow_none=True)
    yg_completed_column_id = fields.String(allow_none=True)
    webhook_registered = fields.Boolean(required=True)


class YougileImportTaskSchema(Schema):
    """Импорт карточки из YouGile в Groove Work.

    `url` — ссылка на карточку YG. Отдел всегда обязателен (как и при обычном
    создании задачи в GW). pull_deadline по умолчанию TRUE: подтянем дедлайн
    из YG-стикера, если он есть.
    """
    url = fields.String(required=True, validate=validate.Length(min=1, max=2000))
    department_id = fields.Integer(required=True)
    responsible_user_id = fields.Integer(load_default=None, allow_none=True)
    stage_id = fields.Integer(load_default=None, allow_none=True)
    pull_deadline = fields.Boolean(load_default=True)


class YougileExportTaskSchema(Schema):
    """Создать карточку в YouGile из существующей GW-задачи."""
    gw_task_id = fields.Integer(required=True)


class YougileCompanySettingsUpdateSchema(Schema):
    """Что админ может прислать. Все поля опциональны, чтобы можно было
    обновлять частями. enabled=False достаточно, чтобы выключить интеграцию,
    не теряя выбранные id."""
    enabled = fields.Boolean()
    yg_company_id = fields.String(validate=validate.Length(max=64), allow_none=True)
    yg_company_name = fields.String(validate=validate.Length(max=255), allow_none=True)
    yg_project_id = fields.String(validate=validate.Length(max=64), allow_none=True)
    yg_project_title = fields.String(validate=validate.Length(max=255), allow_none=True)
    yg_board_id = fields.String(validate=validate.Length(max=64), allow_none=True)
    yg_board_title = fields.String(validate=validate.Length(max=255), allow_none=True)
    yg_completed_column_id = fields.String(validate=validate.Length(max=64), allow_none=True)

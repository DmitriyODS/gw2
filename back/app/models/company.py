from datetime import datetime, timezone
from sqlalchemy.dialects.postgresql import JSONB
from app.extensions import db


DEFAULT_SETTINGS = {
    "uses_yougile": False,
    "uses_stages": False,
    "uses_calls": True,
}


class Company(db.Model):
    __tablename__ = "companies"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), nullable=False, unique=True)
    description = db.Column(db.Text)
    # Корневой Руководитель компании. SET NULL — компания без директора
    # допустима (только что создана, директор уволен и т.п.).
    director_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="SET NULL"),
                            nullable=True)
    is_active = db.Column(db.Boolean, nullable=False, default=True, server_default="true")
    # Настройки рабочего процесса. Хранятся в JSONB чтобы добавлять новые флаги
    # без миграций. Чтение в коде — через get_setting(company, key, default).
    settings = db.Column(JSONB, nullable=False, default=lambda: dict(DEFAULT_SETTINGS),
                         server_default='{}')
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    # AI / ProxyAPI. Ключ хранится зашифрованным (Fernet, ключ шифрования —
    # AI_KEY_ENCRYPTION_KEY в env). hint — открытая маска для UI.
    ai_enabled = db.Column(db.Boolean, nullable=False, default=False,
                           server_default='false')
    ai_api_key_enc = db.Column(db.LargeBinary, nullable=True)
    ai_key_hint = db.Column(db.String(16), nullable=True)
    ai_model_chat = db.Column(db.String(64), nullable=False,
                              default='gpt-4o-mini', server_default='gpt-4o-mini')
    ai_model_embedding = db.Column(db.String(64), nullable=False,
                                   default='text-embedding-3-small',
                                   server_default='text-embedding-3-small')

    # YouGile-интеграция. Включается флагом settings.uses_yougile +
    # необходимы yg_company_id/project/board. Колонка для новых задач —
    # yg_first_column_id (первая колонка выбранной доски); yg_completed_column_id
    # опционален и используется для синка статуса.
    yg_company_id = db.Column(db.String(64))
    yg_company_name = db.Column(db.String(255))
    yg_project_id = db.Column(db.String(64))
    yg_project_title = db.Column(db.String(255))
    yg_board_id = db.Column(db.String(64))
    yg_board_title = db.Column(db.String(255))
    yg_first_column_id = db.Column(db.String(64))
    yg_completed_column_id = db.Column(db.String(64))
    yg_webhook_id = db.Column(db.String(64))
    yg_webhook_secret = db.Column(db.String(64))

    director = db.relationship("User", foreign_keys=[director_id], post_update=True)

    __table_args__ = (
        db.Index("idx_companies_active", "is_active"),
    )


def get_setting(company: "Company", key: str, default=None):
    if company is None or company.settings is None:
        return DEFAULT_SETTINGS.get(key, default)
    return company.settings.get(key, DEFAULT_SETTINGS.get(key, default))

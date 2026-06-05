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

    director = db.relationship("User", foreign_keys=[director_id], post_update=True)

    __table_args__ = (
        db.Index("idx_companies_active", "is_active"),
    )


def get_setting(company: "Company", key: str, default=None):
    if company is None or company.settings is None:
        return DEFAULT_SETTINGS.get(key, default)
    return company.settings.get(key, DEFAULT_SETTINGS.get(key, default))

from datetime import datetime, timezone

from app.extensions import db


class UserYougileAccount(db.Model):
    """Привязка пользователя GW к личному API-ключу в YouGile.

    1:1 по `user_id` (UNIQUE). Ключ хранится зашифрованным Fernet'ом —
    YOUGILE_ENC_KEY в env, шифрование/расшифровка через
    `app.integrations.yougile.crypto`. `key_fingerprint` (last4) показываем
    в UI, чтобы пользователь понимал, какой именно ключ сейчас активен.

    yg_company_id хранится здесь же, хоть и совпадает с companies.yg_company_id
    на момент подключения — нужно, чтобы при смене настроек компании было
    видно, что персональный коннект «устарел» (не та компания) и его надо
    переподключить.
    """
    __tablename__ = "user_yougile_accounts"

    id = db.Column(db.Integer, primary_key=True)
    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                        nullable=False, unique=True)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)
    yg_company_id = db.Column(db.String(64), nullable=False)
    yg_user_id = db.Column(db.String(64))
    yg_login = db.Column(db.String(255), nullable=False)
    key_ciphertext = db.Column(db.LargeBinary, nullable=False)
    key_fingerprint = db.Column(db.String(8), nullable=False)
    last_validated_at = db.Column(db.DateTime(timezone=True))
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))
    updated_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc),
                           onupdate=lambda: datetime.now(timezone.utc))

    user = db.relationship("User", foreign_keys=[user_id])
    company = db.relationship("Company", foreign_keys=[company_id])

    __table_args__ = (
        db.Index("idx_user_yg_company", "company_id"),
    )

from datetime import datetime, timezone
from app.extensions import db


class User(db.Model):
    __tablename__ = "users"

    id = db.Column(db.Integer, primary_key=True)
    fio = db.Column(db.String(255), nullable=False)
    login = db.Column(db.String(100), nullable=False, unique=True)
    hash_password = db.Column(db.Text, nullable=False)
    post = db.Column(db.String(255))
    role_id = db.Column(db.Integer, db.ForeignKey("roles.id"), nullable=False)
    avatar_path = db.Column(db.String(500))
    is_default_pass = db.Column(db.Boolean, nullable=False, default=True)
    is_hidden = db.Column(db.Boolean, nullable=False, default=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))
    # Время последнего выхода из сети (обновляется при дисконнекте всех сокетов).
    # NULL — пользователь ещё ни разу не был онлайн в текущей версии.
    last_seen_at = db.Column(db.DateTime(timezone=True), nullable=True)

    role = db.relationship("Role", back_populates="users")
    tasks = db.relationship("Task", back_populates="author", lazy="dynamic")
    units = db.relationship("Unit", back_populates="user", lazy="dynamic")
    favorites = db.relationship("Favorite", back_populates="user", lazy="dynamic")

    __table_args__ = (
        db.Index("idx_users_login", "login"),
        db.Index("idx_users_role", "role_id"),
        db.Index("idx_users_visible", "is_hidden",
                 postgresql_where=db.text("is_hidden = FALSE")),
    )

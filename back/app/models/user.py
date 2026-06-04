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
    # NULL — пользователь не привязан ни к какой компании (Администратор системы).
    # SET NULL при удалении компании, чтобы её сотрудники не исчезали.
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="SET NULL"),
                           nullable=True)
    avatar_path = db.Column(db.String(500))
    phone = db.Column(db.String(20))
    email = db.Column(db.String(255))
    is_default_pass = db.Column(db.Boolean, nullable=False, default=True)
    is_hidden = db.Column(db.Boolean, nullable=False, default=False)
    # Корневой Администратор системы (первый супер-админ). Его никто не может
    # разжаловать, удалить или сменить ему роль. Должен быть один на систему.
    is_root_admin = db.Column(db.Boolean, nullable=False, default=False, server_default="false")
    created_at = db.Column(db.DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))
    last_seen_at = db.Column(db.DateTime(timezone=True), nullable=True)

    role = db.relationship("Role", back_populates="users")
    company = db.relationship("Company", foreign_keys=[company_id])
    tasks = db.relationship("Task", back_populates="author", lazy="dynamic",
                            foreign_keys="Task.author_id")
    units = db.relationship("Unit", back_populates="user", lazy="dynamic")
    favorites = db.relationship("Favorite", back_populates="user", lazy="dynamic")

    __table_args__ = (
        db.Index("idx_users_login", "login"),
        db.Index("idx_users_role", "role_id"),
        db.Index("idx_users_company", "company_id"),
        db.Index("idx_users_visible", "is_hidden",
                 postgresql_where=db.text("is_hidden = FALSE")),
        db.Index("uq_users_email_lower", db.func.lower(db.text("email")),
                 unique=True, postgresql_where=db.text("email IS NOT NULL")),
    )

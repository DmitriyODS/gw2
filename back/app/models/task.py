from datetime import datetime, timezone
from app.extensions import db


class Task(db.Model):
    __tablename__ = "tasks"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(500), nullable=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))
    author_id = db.Column(db.Integer, db.ForeignKey("users.id"), nullable=False)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)
    link_yougile = db.Column(db.Text)
    received_at = db.Column(db.DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))
    department_id = db.Column(db.Integer, db.ForeignKey("departments.id"), nullable=False)
    deadline = db.Column(db.DateTime(timezone=True))
    is_archived = db.Column(db.Boolean, nullable=False, default=False)
    archived_at = db.Column(db.DateTime(timezone=True))
    color = db.Column(db.String(20))
    responsible_user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="SET NULL"))
    stage_id = db.Column(db.Integer, db.ForeignKey("stages.id", ondelete="SET NULL"))

    # YouGile-привязка. link_yougile выше — это URL для UI; ниже структурное
    # представление, заполняется при привязке через /api/yougile.
    yougile_task_id = db.Column(db.String(64))
    yougile_project_id = db.Column(db.String(64))
    yougile_board_id = db.Column(db.String(64))
    yougile_column_id = db.Column(db.String(64))
    # Антицикл: хеш последнего state'а, который мы сами пушнули в YG. Если
    # webhook вернёт такой же — игнорируем (см. integrations/yougile/sync).
    yougile_synced_at = db.Column(db.DateTime(timezone=True))
    yougile_sync_hash = db.Column(db.String(64))

    author = db.relationship("User", back_populates="tasks", foreign_keys=[author_id])
    responsible = db.relationship("User", foreign_keys=[responsible_user_id])
    company = db.relationship("Company", foreign_keys=[company_id])
    department = db.relationship("Department", back_populates="tasks")
    stage = db.relationship("Stage", foreign_keys=[stage_id])
    units = db.relationship("Unit", back_populates="task", lazy="dynamic", cascade="all, delete-orphan")
    favorites = db.relationship("Favorite", back_populates="task", lazy="dynamic", cascade="all, delete-orphan")

    __table_args__ = (
        db.Index("idx_tasks_author", "author_id"),
        db.Index("idx_tasks_company", "company_id"),
        db.Index("idx_tasks_dept", "department_id"),
        db.Index("idx_tasks_archived", "is_archived"),
        db.Index("idx_tasks_received", "received_at"),
        db.Index("idx_tasks_responsible", "responsible_user_id"),
        db.Index("idx_tasks_stage", "stage_id"),
        db.Index("idx_tasks_archived_at", "archived_at",
                 postgresql_where=db.text("is_archived = TRUE")),
    )

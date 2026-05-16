from datetime import datetime, timezone
from app.extensions import db


class Unit(db.Model):
    __tablename__ = "units"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(500), nullable=False)
    user_id = db.Column(db.Integer, db.ForeignKey("users.id"), nullable=False)
    unit_type_id = db.Column(db.Integer, db.ForeignKey("unit_types.id", ondelete="CASCADE"), nullable=False)
    task_id = db.Column(db.Integer, db.ForeignKey("tasks.id", ondelete="CASCADE"), nullable=False)
    is_edited = db.Column(db.Boolean, nullable=False, default=False)
    datetime_start = db.Column(db.DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))
    datetime_end = db.Column(db.DateTime(timezone=True))
    created_at = db.Column(db.DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))

    user = db.relationship("User", back_populates="units")
    unit_type = db.relationship("UnitType", back_populates="units")
    task = db.relationship("Task", back_populates="units")

    __table_args__ = (
        db.Index("idx_units_user", "user_id"),
        db.Index("idx_units_task", "task_id"),
        db.Index("idx_units_active", "user_id",
                 postgresql_where=db.text("datetime_end IS NULL")),
    )

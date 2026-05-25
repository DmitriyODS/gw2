from datetime import datetime, timezone
from app.extensions import db


class Conversation(db.Model):
    """Личный диалог двух пользователей. Хранится в нормализованном виде:
    user_a_id < user_b_id, чтобы пара была уникальной независимо от того,
    кто инициировал переписку."""
    __tablename__ = "conversations"

    id = db.Column(db.Integer, primary_key=True)
    user_a_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    user_b_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))
    last_message_at = db.Column(db.DateTime(timezone=True), nullable=True)

    user_a = db.relationship("User", foreign_keys=[user_a_id])
    user_b = db.relationship("User", foreign_keys=[user_b_id])
    messages = db.relationship("Message", back_populates="conversation",
                               cascade="all, delete-orphan", lazy="dynamic")

    __table_args__ = (
        db.UniqueConstraint("user_a_id", "user_b_id", name="uq_conversation_pair"),
        db.Index("idx_conv_user_a", "user_a_id"),
        db.Index("idx_conv_user_b", "user_b_id"),
        db.Index("idx_conv_last_msg", "last_message_at"),
        db.CheckConstraint("user_a_id < user_b_id", name="ck_conversation_pair_order"),
    )

    def other_user_id(self, user_id: int) -> int:
        return self.user_b_id if self.user_a_id == user_id else self.user_a_id

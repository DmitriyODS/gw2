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
    # Компания, в рамках которой ведётся диалог. У обоих участников
    # company_id должен совпадать с этим значением (бизнес-инвариант).
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))
    last_message_at = db.Column(db.DateTime(timezone=True), nullable=True)
    # Soft-delete по сторонам. Когда обе стороны скрыли — диалог удаляется
    # физически вместе с сообщениями и файлами.
    hidden_for_a = db.Column(db.Boolean, nullable=False, default=False, server_default="false")
    hidden_for_b = db.Column(db.Boolean, nullable=False, default=False, server_default="false")
    # Личное закрепление: каждый пользователь может закрепить чат у себя.
    # Сортировка: pinned_at_<side> DESC, потом last_message_at DESC.
    pinned_at_a = db.Column(db.DateTime(timezone=True), nullable=True)
    pinned_at_b = db.Column(db.DateTime(timezone=True), nullable=True)

    user_a = db.relationship("User", foreign_keys=[user_a_id])
    user_b = db.relationship("User", foreign_keys=[user_b_id])
    messages = db.relationship("Message", back_populates="conversation",
                               cascade="all, delete-orphan", lazy="dynamic")

    __table_args__ = (
        db.UniqueConstraint("user_a_id", "user_b_id", name="uq_conversation_pair"),
        db.Index("idx_conv_user_a", "user_a_id"),
        db.Index("idx_conv_user_b", "user_b_id"),
        db.Index("idx_conv_company", "company_id"),
        db.Index("idx_conv_last_msg", "last_message_at"),
        db.CheckConstraint("user_a_id < user_b_id", name="ck_conversation_pair_order"),
    )

    def other_user_id(self, user_id: int) -> int:
        return self.user_b_id if self.user_a_id == user_id else self.user_a_id

    def side(self, user_id: int) -> str:
        """'a' если user_id == user_a_id, иначе 'b'."""
        return 'a' if self.user_a_id == user_id else 'b'

    def is_hidden_for(self, user_id: int) -> bool:
        return self.hidden_for_a if self.side(user_id) == 'a' else self.hidden_for_b

    def pinned_at_for(self, user_id: int):
        return self.pinned_at_a if self.side(user_id) == 'a' else self.pinned_at_b

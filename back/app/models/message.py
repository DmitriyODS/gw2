from datetime import datetime, timezone
from app.extensions import db


class Message(db.Model):
    __tablename__ = "messages"

    id = db.Column(db.Integer, primary_key=True)
    conversation_id = db.Column(db.Integer, db.ForeignKey("conversations.id", ondelete="CASCADE"),
                                nullable=False)
    sender_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    text = db.Column(db.Text, nullable=True)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))
    read_at = db.Column(db.DateTime(timezone=True), nullable=True)

    conversation = db.relationship("Conversation", back_populates="messages")
    sender = db.relationship("User")
    attachments = db.relationship("MessageAttachment", back_populates="message",
                                  cascade="all, delete-orphan", lazy="joined")

    __table_args__ = (
        db.Index("idx_msg_conv_created", "conversation_id", "created_at"),
        db.Index("idx_msg_unread_recipient", "conversation_id", "read_at"),
    )


class MessageAttachment(db.Model):
    __tablename__ = "message_attachments"

    id = db.Column(db.Integer, primary_key=True)
    message_id = db.Column(db.Integer, db.ForeignKey("messages.id", ondelete="CASCADE"),
                           nullable=True)
    uploader_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                            nullable=False)
    file_path = db.Column(db.String(500), nullable=False)
    file_name = db.Column(db.String(255), nullable=False)
    mime_type = db.Column(db.String(120), nullable=False)
    size_bytes = db.Column(db.Integer, nullable=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    message = db.relationship("Message", back_populates="attachments")

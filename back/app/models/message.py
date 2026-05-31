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
    # Скрыто стороной (для «удалить только у себя»). Стороны определяются
    # по conversation.user_a_id / user_b_id. Когда оба true — сообщение
    # физически удаляется фоновой проверкой (см. messenger_service).
    hidden_for_a = db.Column(db.Boolean, nullable=False, default=False, server_default="false")
    hidden_for_b = db.Column(db.Boolean, nullable=False, default=False, server_default="false")
    # Ответ на сообщение того же диалога. SET NULL при удалении исходного —
    # цитата просто пропадёт, само сообщение-ответ останется.
    reply_to_id = db.Column(db.Integer, db.ForeignKey("messages.id", ondelete="SET NULL"),
                            nullable=True)
    # Если сообщение переслано — сюда пишется автор оригинала (для метки
    # «Переслано от …»). При пересылке текст и файлы копируются.
    forwarded_from_user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="SET NULL"),
                                       nullable=True)
    # 'text' — обычное пользовательское сообщение; 'call' — системная плашка
    # о завершённом звонке (длительность, статус, тип), создаётся из
    # call_service после _finalize. text при kind='call' пустой; данные
    # фронт берёт из nested call (см. MessageSchema).
    kind = db.Column(db.String(16), nullable=False, default="text", server_default="text")
    # Для kind='call' — ссылка на запись звонка. SET NULL: если запись
    # звонка когда-то удалят, плашка останется в чате, но без деталей.
    call_id = db.Column(db.Integer, db.ForeignKey("calls.id", ondelete="SET NULL"),
                        nullable=True)
    # Закрепление сообщения. Общее для обоих участников диалога (как в
    # Telegram): закрепил один — закреплённое видят оба. pinned_at — момент
    # закрепления (для сортировки и метки), pinned_by_id — кто закрепил.
    pinned_at = db.Column(db.DateTime(timezone=True), nullable=True)
    pinned_by_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="SET NULL"),
                             nullable=True)

    conversation = db.relationship("Conversation", back_populates="messages")
    sender = db.relationship("User", foreign_keys=[sender_id])
    reply_to = db.relationship("Message", remote_side=[id], foreign_keys=[reply_to_id])
    forwarded_from = db.relationship("User", foreign_keys=[forwarded_from_user_id])
    pinned_by = db.relationship("User", foreign_keys=[pinned_by_id])
    call = db.relationship("Call", foreign_keys=[call_id])
    # selectin вместо joined: joined-load на коллекцию заставляет вызывать
    # .unique() на каждом Result, что ломает list_user_conversations.
    attachments = db.relationship("MessageAttachment", back_populates="message",
                                  cascade="all, delete-orphan", lazy="selectin")

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

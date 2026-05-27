from datetime import datetime, timezone
from app.extensions import db


class Call(db.Model):
    """Запись звонка (история). Сам сигналинг и активное состояние — in-memory
    (sockets/call_state.py); БД нужна, чтобы пользователь видел, кто и когда
    ему звонил."""
    __tablename__ = "calls"

    id = db.Column(db.Integer, primary_key=True)
    initiator_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                             nullable=False)
    # 'p2p' (1:1) или 'group' (3+ участника). p2p начинается с двух участников,
    # group — с initiator + список приглашённых.
    kind = db.Column(db.String(16), nullable=False, default="p2p", server_default="p2p")
    # 'ringing' (идёт звон, никто не принял), 'active' (хотя бы один принял),
    # 'ended' (все вышли или явное завершение). 'missed' — никто не ответил.
    status = db.Column(db.String(16), nullable=False, default="ringing", server_default="ringing")
    # 'audio' или 'video'. Кто-то из участников может включать/выключать своё
    # видео в процессе, поле — лишь стартовый режим (для иконки в истории).
    media = db.Column(db.String(8), nullable=False, default="video", server_default="video")
    started_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))
    ended_at = db.Column(db.DateTime(timezone=True), nullable=True)
    # Опционально: id парного диалога мессенджера — чтобы можно было сразу
    # перейти из истории звонков в чат. Для group-звонков пусто.
    conversation_id = db.Column(db.Integer, db.ForeignKey("conversations.id", ondelete="SET NULL"),
                                nullable=True)

    initiator = db.relationship("User", foreign_keys=[initiator_id])
    participants = db.relationship("CallParticipant", back_populates="call",
                                   cascade="all, delete-orphan", lazy="selectin")

    __table_args__ = (
        db.Index("idx_call_started", "started_at"),
        db.Index("idx_call_status", "status"),
    )


class CallParticipant(db.Model):
    """Участник звонка. Запись создаётся при invite (для каждого приглашённого);
    joined_at заполняется при accept, left_at — при leave/end."""
    __tablename__ = "call_participants"

    id = db.Column(db.Integer, primary_key=True)
    call_id = db.Column(db.Integer, db.ForeignKey("calls.id", ondelete="CASCADE"),
                        nullable=False)
    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                        nullable=False)
    # 'initiator' — кто начал звонок, 'invitee' — все приглашённые.
    role = db.Column(db.String(16), nullable=False, default="invitee", server_default="invitee")
    invited_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))
    joined_at = db.Column(db.DateTime(timezone=True), nullable=True)
    left_at = db.Column(db.DateTime(timezone=True), nullable=True)
    # Если участник нажал «отклонить» — это явный сигнал отказа, в отличие от
    # «не успел снять трубку» (left_at без joined_at и status='ended').
    declined = db.Column(db.Boolean, nullable=False, default=False, server_default="false")

    call = db.relationship("Call", back_populates="participants")
    user = db.relationship("User", foreign_keys=[user_id])

    __table_args__ = (
        db.Index("idx_callpart_user", "user_id"),
        db.Index("idx_callpart_call", "call_id"),
        db.UniqueConstraint("call_id", "user_id", name="uq_callpart_pair"),
    )

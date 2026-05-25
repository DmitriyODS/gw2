from datetime import datetime, timezone
from typing import Optional
from sqlalchemy import or_, and_, func

from app.extensions import db
from app.models import Conversation, Message, MessageAttachment, User


def _pair(a: int, b: int) -> tuple[int, int]:
    return (a, b) if a < b else (b, a)


def get_conversation_between(user_a: int, user_b: int) -> Optional[Conversation]:
    a, b = _pair(user_a, user_b)
    return db.session.execute(
        db.select(Conversation).where(Conversation.user_a_id == a, Conversation.user_b_id == b)
    ).scalar_one_or_none()


def get_or_create_conversation(user_a: int, user_b: int) -> Conversation:
    conv = get_conversation_between(user_a, user_b)
    if conv:
        return conv
    a, b = _pair(user_a, user_b)
    conv = Conversation(user_a_id=a, user_b_id=b)
    db.session.add(conv)
    db.session.flush()
    return conv


def get_conversation(conversation_id: int) -> Optional[Conversation]:
    return db.session.get(Conversation, conversation_id)


def list_user_conversations(user_id: int) -> list[dict]:
    """Список диалогов пользователя с информацией о собеседнике, последнем
    сообщении и количестве непрочитанных. Возвращает list[dict] для
    удобства сериализации."""
    convs = db.session.execute(
        db.select(Conversation).where(
            or_(Conversation.user_a_id == user_id, Conversation.user_b_id == user_id)
        ).order_by(Conversation.last_message_at.desc().nullslast(), Conversation.created_at.desc())
    ).scalars().all()

    if not convs:
        return []

    conv_ids = [c.id for c in convs]
    other_ids = [c.other_user_id(user_id) for c in convs]

    # Последние сообщения
    last_msg_subq = (
        db.select(
            Message.conversation_id,
            func.max(Message.id).label("last_id"),
        )
        .where(Message.conversation_id.in_(conv_ids))
        .group_by(Message.conversation_id)
        .subquery()
    )
    last_msgs = db.session.execute(
        db.select(Message).join(last_msg_subq, Message.id == last_msg_subq.c.last_id)
    ).scalars().all()
    last_msg_by_conv = {m.conversation_id: m for m in last_msgs}

    # Непрочитанные (от собеседника, read_at IS NULL)
    unread_rows = db.session.execute(
        db.select(Message.conversation_id, func.count(Message.id))
        .where(
            Message.conversation_id.in_(conv_ids),
            Message.sender_id != user_id,
            Message.read_at.is_(None),
        )
        .group_by(Message.conversation_id)
    ).all()
    unread_by_conv = {row[0]: row[1] for row in unread_rows}

    # Профили собеседников
    others = db.session.execute(
        db.select(User).join(User.role).where(User.id.in_(other_ids))
    ).scalars().all()
    other_by_id = {u.id: u for u in others}

    result = []
    for c in convs:
        other = other_by_id.get(c.other_user_id(user_id))
        result.append({
            "conversation": c,
            "other_user": other,
            "last_message": last_msg_by_conv.get(c.id),
            "unread_count": unread_by_conv.get(c.id, 0),
        })
    return result


def list_messages(conversation_id: int, before_id: Optional[int] = None,
                  limit: int = 50) -> list[Message]:
    q = db.select(Message).where(Message.conversation_id == conversation_id)
    if before_id is not None:
        q = q.where(Message.id < before_id)
    q = q.order_by(Message.id.desc()).limit(limit)
    rows = db.session.execute(q).scalars().all()
    # Возвращаем в хронологическом порядке (старые → новые)
    return list(reversed(rows))


def create_message(conversation_id: int, sender_id: int, text: Optional[str],
                   attachment_ids: list[int]) -> Message:
    msg = Message(
        conversation_id=conversation_id,
        sender_id=sender_id,
        text=text or None,
    )
    db.session.add(msg)
    db.session.flush()

    if attachment_ids:
        db.session.execute(
            db.update(MessageAttachment)
            .where(
                MessageAttachment.id.in_(attachment_ids),
                MessageAttachment.uploader_id == sender_id,
                MessageAttachment.message_id.is_(None),
            )
            .values(message_id=msg.id)
        )

    # Обновим last_message_at у диалога
    db.session.execute(
        db.update(Conversation)
        .where(Conversation.id == conversation_id)
        .values(last_message_at=msg.created_at)
    )
    db.session.flush()
    return msg


def mark_read(conversation_id: int, reader_id: int) -> int:
    """Помечает как прочитанные все сообщения от собеседника. Возвращает количество."""
    now = datetime.now(timezone.utc)
    result = db.session.execute(
        db.update(Message)
        .where(
            Message.conversation_id == conversation_id,
            Message.sender_id != reader_id,
            Message.read_at.is_(None),
        )
        .values(read_at=now)
    )
    db.session.flush()
    return result.rowcount or 0


def create_attachment(uploader_id: int, file_path: str, file_name: str,
                      mime_type: str, size_bytes: int) -> MessageAttachment:
    att = MessageAttachment(
        uploader_id=uploader_id,
        file_path=file_path,
        file_name=file_name,
        mime_type=mime_type,
        size_bytes=size_bytes,
    )
    db.session.add(att)
    db.session.flush()
    return att


def total_unread(user_id: int) -> int:
    """Общее число непрочитанных сообщений у пользователя по всем диалогам."""
    return db.session.execute(
        db.select(func.count(Message.id))
        .join(Conversation, Conversation.id == Message.conversation_id)
        .where(
            or_(Conversation.user_a_id == user_id, Conversation.user_b_id == user_id),
            Message.sender_id != user_id,
            Message.read_at.is_(None),
        )
    ).scalar_one() or 0


def get_attachment(attachment_id: int) -> Optional[MessageAttachment]:
    return db.session.get(MessageAttachment, attachment_id)

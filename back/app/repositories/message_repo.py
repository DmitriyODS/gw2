from datetime import datetime, timezone
from typing import Optional
from sqlalchemy import or_, and_, func
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import selectinload

from app.extensions import db
from app.models import Conversation, Message, MessageAttachment, User


def _pair(a: int, b: int) -> tuple[int, int]:
    return (a, b) if a < b else (b, a)


def _msg_load_options():
    """Всё, что сериализует MessageSchema, — одним батчем вместо ленивых
    подгрузок на каждое сообщение (sender, цитата, плашки звонка/задачи…)."""
    return (
        selectinload(Message.attachments),
        selectinload(Message.sender).selectinload(User.role),
        selectinload(Message.reply_to).selectinload(Message.sender),
        selectinload(Message.reply_to).selectinload(Message.attachments),
        selectinload(Message.forwarded_from),
        selectinload(Message.pinned_by),
        selectinload(Message.call),
        selectinload(Message.task),
        selectinload(Message.conversation),
    )


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
    # Multi-tenancy: компания диалога = компания собеседника, в чью «комнату»
    # этот чат попадает. Берём company_id любого из участников (они должны
    # быть из одной компании — это валидируется выше по стеку). Если оба
    # без company_id (Администратор системы пишет Администратору системы) —
    # берём первого попавшегося. На практике этого не должно происходить.
    ua = db.session.get(User, a)
    ub = db.session.get(User, b)
    company_id = ua.company_id if ua and ua.company_id else (ub.company_id if ub else None)
    conv = Conversation(user_a_id=a, user_b_id=b, company_id=company_id)
    db.session.add(conv)
    try:
        db.session.flush()
        return conv
    except IntegrityError:
        db.session.rollback()
        existing = get_conversation_between(user_a, user_b)
        if existing is None:
            raise
        return existing


def get_dev_chat_for_user(user_id: int) -> Optional[Conversation]:
    """Личный чат пользователя с техподдержкой (один на пользователя)."""
    return db.session.execute(
        db.select(Conversation).where(
            Conversation.is_dev_chat.is_(True),
            Conversation.user_a_id == user_id,
        )
    ).scalar_one_or_none()


def get_or_create_dev_chat_for_user(user_id: int, company_id: int) -> Conversation:
    conv = get_dev_chat_for_user(user_id)
    if conv:
        return conv
    conv = Conversation(user_a_id=user_id, user_b_id=None,
                        company_id=company_id, is_dev_chat=True)
    db.session.add(conv)
    try:
        db.session.flush()
        return conv
    except IntegrityError:
        db.session.rollback()
        existing = get_dev_chat_for_user(user_id)
        if existing is None:
            raise
        return existing


def get_pet_chat_for_user(user_id: int) -> Optional[Conversation]:
    """Чат пользователя со своим Грувиком (один на пользователя)."""
    return db.session.execute(
        db.select(Conversation).where(
            Conversation.is_pet_chat.is_(True),
            Conversation.user_a_id == user_id,
        )
    ).scalar_one_or_none()


def get_or_create_pet_chat_for_user(user_id: int, company_id: int) -> Conversation:
    conv = get_pet_chat_for_user(user_id)
    if conv:
        return conv
    conv = Conversation(user_a_id=user_id, user_b_id=None,
                        company_id=company_id, is_pet_chat=True)
    db.session.add(conv)
    try:
        db.session.flush()
        return conv
    except IntegrityError:
        db.session.rollback()
        existing = get_pet_chat_for_user(user_id)
        if existing is None:
            raise
        return existing


def has_human_message_since(conversation_id: int, since: datetime, before_id: int) -> bool:
    """Было ли в диалоге сообщение от человека (не бота) свежее `since`,
    не считая сообщения before_id и более новых. Для решения, нужен ли
    автоответ техподдержки на первое за сутки обращение."""
    row = db.session.execute(
        db.select(Message.id).where(
            Message.conversation_id == conversation_id,
            Message.id < before_id,
            Message.is_bot.is_(False),
            Message.created_at >= since,
        ).limit(1)
    ).scalar_one_or_none()
    return row is not None


def last_messages(conversation_id: int, limit: int = 12) -> list[Message]:
    """Последние сообщения диалога в хронологическом порядке (для контекста
    AI-ответа Грувика)."""
    rows = db.session.execute(
        db.select(Message)
        .where(Message.conversation_id == conversation_id)
        .order_by(Message.id.desc())
        .limit(limit)
    ).scalars().all()
    return list(reversed(rows))


def list_dev_chats() -> list[Conversation]:
    """Все личные чаты пользователей с техподдержкой (для Администратора системы).
    Сортировка: сначала с непрочитанными/свежим сообщением, потом пустые."""
    return list(db.session.execute(
        db.select(Conversation).options(selectinload(Conversation.company))
        .where(Conversation.is_dev_chat.is_(True))
        .order_by(Conversation.last_message_at.desc().nullslast(),
                  Conversation.created_at.desc())
    ).scalars().all())


def get_conversation(conversation_id: int) -> Optional[Conversation]:
    return db.session.get(Conversation, conversation_id)


def list_user_conversations(user_id: int) -> list[dict]:
    """Список диалогов пользователя с информацией о собеседнике, последнем
    сообщении и количестве непрочитанных. Скрытые «у себя» — не показываются.
    Для пользователя с company_id — dev_chat компании добавляется первым
    (если существует). Для Администратора системы — dev-чаты выгружаются
    отдельным эндпоинтом. Сортировка: закреплённые → остальные (по last_message_at)."""
    me = db.session.get(User, user_id)

    # Фильтр «не скрыто на моей стороне»
    not_hidden = or_(
        and_(Conversation.user_a_id == user_id, Conversation.hidden_for_a.is_(False)),
        and_(Conversation.user_b_id == user_id, Conversation.hidden_for_b.is_(False)),
    )
    convs = db.session.execute(
        db.select(Conversation).options(selectinload(Conversation.company)).where(
            Conversation.is_dev_chat.is_(False),
            Conversation.is_pet_chat.is_(False),
            or_(Conversation.user_a_id == user_id, Conversation.user_b_id == user_id),
            not_hidden,
        ).order_by(Conversation.last_message_at.desc().nullslast(), Conversation.created_at.desc())
    ).scalars().all()

    if not convs:
        convs = []

    # Доп. сортировка в Python: pinned-первыми по pinned_at_<side> DESC.
    convs.sort(key=lambda c: (
        c.pinned_at_for(user_id) is None,
        # для не-pinned ставим минимальный datetime, чтобы они шли после
        -((c.pinned_at_for(user_id) or _MIN_DT).timestamp()),
    ))

    conv_ids = [c.id for c in convs]
    other_ids = [c.other_user_id(user_id) for c in convs]

    # Последнее сообщение, не скрытое на моей стороне. Берём пары (conv_id, side)
    # и для каждой ищем максимальный id среди non-hidden_for_<side>.
    # Делаем двумя запросами по сторонам — проще, чем case-выражение.
    a_conv_ids = [c.id for c in convs if c.side(user_id) == 'a']
    b_conv_ids = [c.id for c in convs if c.side(user_id) == 'b']

    last_msg_by_conv: dict[int, Message] = {}
    for ids, hidden_col in ((a_conv_ids, Message.hidden_for_a), (b_conv_ids, Message.hidden_for_b)):
        if not ids:
            continue
        sub = (
            db.select(Message.conversation_id, func.max(Message.id).label("last_id"))
            .where(Message.conversation_id.in_(ids), hidden_col.is_(False))
            .group_by(Message.conversation_id)
            .subquery()
        )
        rows = db.session.execute(
            db.select(Message)
            .options(*_msg_load_options())
            .join(sub, Message.id == sub.c.last_id)
        ).scalars().all()
        for m in rows:
            last_msg_by_conv[m.conversation_id] = m

    # Непрочитанные (от собеседника, read_at IS NULL, не скрытые на моей стороне)
    unread_by_conv: dict[int, int] = {}
    for ids, hidden_col in ((a_conv_ids, Message.hidden_for_a), (b_conv_ids, Message.hidden_for_b)):
        if not ids:
            continue
        rows = db.session.execute(
            db.select(Message.conversation_id, func.count(Message.id))
            .where(
                Message.conversation_id.in_(ids),
                or_(Message.sender_id.is_(None), Message.sender_id != user_id),
                Message.read_at.is_(None),
                hidden_col.is_(False),
            )
            .group_by(Message.conversation_id)
        ).all()
        for cid, n in rows:
            unread_by_conv[cid] = n

    # Профили собеседников
    others = db.session.execute(
        db.select(User).options(selectinload(User.role)).where(User.id.in_(other_ids))
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
            "is_pinned": c.pinned_at_for(user_id) is not None,
            "pinned_at": c.pinned_at_for(user_id),
        })

    # Добавляем личный dev_chat пользователя первым (если он сотрудник компании
    # и чат уже существует — создание делает сервис до вызова list).
    # Администратор системы (company_id=None) видит все dev-чаты пользователей
    # через отдельный эндпоинт /dev-chats (support inbox).
    if me is not None and me.company_id is not None:
        dev = get_dev_chat_for_user(user_id)
        if dev is not None:
            # Последнее сообщение dev_chat (не скрытое нет понятия «стороны»)
            dev_last = db.session.execute(
                db.select(Message)
                .options(*_msg_load_options())
                .where(Message.conversation_id == dev.id)
                .order_by(Message.id.desc())
                .limit(1)
            ).scalar_one_or_none()
            # Автоответ техподдержки идёт с sender_id = NULL — обычное `!=`
            # его молча отбрасывает (трёхзначная логика SQL), как у Грувика.
            dev_unread = db.session.execute(
                db.select(func.count(Message.id))
                .where(
                    Message.conversation_id == dev.id,
                    or_(Message.sender_id.is_(None), Message.sender_id != user_id),
                    Message.read_at.is_(None),
                )
            ).scalar_one() or 0
            result.insert(0, {
                "conversation": dev,
                "other_user": None,
                "last_message": dev_last,
                "unread_count": dev_unread,
                "is_pinned": False,
                "pinned_at": None,
            })

        # Чат с Грувиком — самым первым (если уже создан кнопкой в «Моём Groove»).
        pet_conv = get_pet_chat_for_user(user_id)
        if pet_conv is not None:
            pet_last = db.session.execute(
                db.select(Message)
                .options(*_msg_load_options())
                .where(Message.conversation_id == pet_conv.id)
                .order_by(Message.id.desc())
                .limit(1)
            ).scalar_one_or_none()
            # Ответы Грувика идут с sender_id = NULL — обычное `!=` их молча
            # отбрасывает (трёхзначная логика SQL), поэтому явный OR IS NULL.
            pet_unread = db.session.execute(
                db.select(func.count(Message.id))
                .where(
                    Message.conversation_id == pet_conv.id,
                    or_(Message.sender_id.is_(None), Message.sender_id != user_id),
                    Message.read_at.is_(None),
                )
            ).scalar_one() or 0
            result.insert(0, {
                "conversation": pet_conv,
                "other_user": None,
                "last_message": pet_last,
                "unread_count": pet_unread,
                "is_pinned": False,
                "pinned_at": None,
            })

    return result


_MIN_DT = datetime(1970, 1, 1, tzinfo=timezone.utc)


def list_messages(conversation_id: int, user_id: int, before_id: Optional[int] = None,
                  after_id: Optional[int] = None, limit: int = 50) -> list[Message]:
    """Сообщения диалога без тех, что скрыты на стороне user_id.
    - before_id: пагинация назад в историю (для подгрузки старых сообщений).
    - after_id: только новые сообщения с момента last seen id (для silent poll)."""
    conv = get_conversation(conversation_id)
    if conv is None:
        return []
    hidden_col = Message.hidden_for_a if conv.side(user_id) == 'a' else Message.hidden_for_b
    q = db.select(Message).options(*_msg_load_options()).where(
        Message.conversation_id == conversation_id,
        hidden_col.is_(False),
    )
    if before_id is not None:
        q = q.where(Message.id < before_id)
    if after_id is not None:
        q = q.where(Message.id > after_id)
        # При after_id возвращаем в прямом порядке (старые → новые), без переворота
        rows = db.session.execute(q.order_by(Message.id.asc()).limit(limit)).scalars().all()
        return list(rows)
    q = q.order_by(Message.id.desc()).limit(limit)
    rows = db.session.execute(q).scalars().all()
    return list(reversed(rows))


def create_call_message(conversation_id: int, sender_id: int, call_id: int) -> Message:
    """Системная плашка о звонке в чате (kind='call').

    Текст пустой — фронт рендерит специальной плашкой по nested полю `call`.
    last_message_at и hidden_for_* обновляем как у обычного сообщения, чтобы
    диалог поднялся в списке и «вернулся» обеим сторонам, если был скрыт.
    """
    msg = Message(
        conversation_id=conversation_id,
        sender_id=sender_id,
        text=None,
        kind="call",
        call_id=call_id,
    )
    db.session.add(msg)
    db.session.flush()

    db.session.execute(
        db.update(Conversation)
        .where(Conversation.id == conversation_id)
        .values(last_message_at=msg.created_at, hidden_for_a=False, hidden_for_b=False)
    )
    db.session.flush()
    return msg


def create_message(conversation_id: int, sender_id: Optional[int], text: Optional[str],
                   attachment_ids: list[int], reply_to_id: Optional[int] = None,
                   forwarded_from_user_id: Optional[int] = None,
                   kind: str = "text", task_id: Optional[int] = None,
                   call_id: Optional[int] = None,
                   is_bot: bool = False) -> Message:
    msg = Message(
        conversation_id=conversation_id,
        sender_id=sender_id,
        text=text or None,
        reply_to_id=reply_to_id,
        forwarded_from_user_id=forwarded_from_user_id,
        kind=kind,
        task_id=task_id,
        call_id=call_id,
        is_bot=is_bot,
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

    # Обновим last_message_at у диалога. Заодно «возвращаем» диалог обеим
    # сторонам, если кто-то его раньше скрыл у себя: новое сообщение должно
    # снова показаться у получателя.
    db.session.execute(
        db.update(Conversation)
        .where(Conversation.id == conversation_id)
        .values(last_message_at=msg.created_at, hidden_for_a=False, hidden_for_b=False)
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
            or_(Message.sender_id.is_(None), Message.sender_id != reader_id),
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
    """Общее число непрочитанных сообщений у пользователя по всем не скрытым
    диалогам, без сообщений, скрытых на его стороне."""
    side_a = and_(
        Conversation.user_a_id == user_id,
        Conversation.hidden_for_a.is_(False),
        Message.hidden_for_a.is_(False),
    )
    side_b = and_(
        Conversation.user_b_id == user_id,
        Conversation.hidden_for_b.is_(False),
        Message.hidden_for_b.is_(False),
    )
    return db.session.execute(
        db.select(func.count(Message.id))
        .join(Conversation, Conversation.id == Message.conversation_id)
        .where(
            or_(side_a, side_b),
            or_(Message.sender_id.is_(None), Message.sender_id != user_id),
            Message.read_at.is_(None),
        )
    ).scalar_one() or 0


def get_attachment(attachment_id: int) -> Optional[MessageAttachment]:
    return db.session.get(MessageAttachment, attachment_id)


def get_message(message_id: int) -> Optional[Message]:
    return db.session.get(Message, message_id)


def delete_message(message: Message) -> None:
    """Удаляет сообщение. Вложения каскадно уходят через FK ondelete=CASCADE.
    Файлы на диске удаляются вызывающей стороной до commit."""
    db.session.delete(message)
    db.session.flush()


def recompute_last_message_at(conversation_id: int) -> None:
    """Пересчитать last_message_at у диалога после удаления сообщения."""
    last = db.session.execute(
        db.select(func.max(Message.created_at)).where(Message.conversation_id == conversation_id)
    ).scalar_one()
    db.session.execute(
        db.update(Conversation)
        .where(Conversation.id == conversation_id)
        .values(last_message_at=last)
    )
    db.session.flush()


def list_attachment_paths_of_conversation(conversation_id: int) -> list[str]:
    return list(db.session.execute(
        db.select(MessageAttachment.file_path)
        .join(Message, Message.id == MessageAttachment.message_id)
        .where(Message.conversation_id == conversation_id)
    ).scalars().all())


def delete_conversation(conversation: Conversation) -> None:
    db.session.delete(conversation)
    db.session.flush()


def hide_message_for(message: Message, side: str) -> bool:
    """Помечает сообщение скрытым на указанной стороне ('a' или 'b').
    Возвращает True, если после операции сообщение скрыто обеими сторонами
    (вызывающий должен физически удалить вместе с файлами)."""
    if side == 'a':
        message.hidden_for_a = True
    else:
        message.hidden_for_b = True
    db.session.flush()
    return bool(message.hidden_for_a and message.hidden_for_b)


def hide_conversation_for(conversation: Conversation, side: str) -> bool:
    """Помечает диалог скрытым на указанной стороне. Также скрывает все
    сообщения на этой стороне (чтобы при возврате/повторном открытии собеседник
    не видел старую переписку, которую другая сторона стёрла). Возвращает True,
    если оба пользователя теперь скрыли диалог — вызывающий удаляет физически."""
    if side == 'a':
        conversation.hidden_for_a = True
    else:
        conversation.hidden_for_b = True

    col = Message.hidden_for_a if side == 'a' else Message.hidden_for_b
    db.session.execute(
        db.update(Message).where(Message.conversation_id == conversation.id).values({col: True})
    )
    db.session.flush()
    return bool(conversation.hidden_for_a and conversation.hidden_for_b)


def set_pin(conversation: Conversation, side: str, pinned: bool) -> None:
    now = datetime.now(timezone.utc) if pinned else None
    if side == 'a':
        conversation.pinned_at_a = now
    else:
        conversation.pinned_at_b = now
    db.session.flush()


def set_message_pin(message: Message, pinned: bool, by_id: Optional[int]) -> None:
    """Закрепить/открепить сообщение. Закрепление общее для обоих участников."""
    if pinned:
        message.pinned_at = datetime.now(timezone.utc)
        message.pinned_by_id = by_id
    else:
        message.pinned_at = None
        message.pinned_by_id = None
    db.session.flush()


def list_pinned_messages(conversation_id: int, user_id: int) -> list[Message]:
    """Закреплённые сообщения диалога, не скрытые на стороне user_id.
    Самое свежее закрепление — первым."""
    conv = get_conversation(conversation_id)
    if conv is None:
        return []
    hidden_col = Message.hidden_for_a if conv.side(user_id) == 'a' else Message.hidden_for_b
    rows = db.session.execute(
        db.select(Message).options(*_msg_load_options()).where(
            Message.conversation_id == conversation_id,
            Message.pinned_at.isnot(None),
            hidden_col.is_(False),
        ).order_by(Message.pinned_at.desc())
    ).scalars().all()
    return list(rows)

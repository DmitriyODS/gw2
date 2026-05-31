"""Бизнес-логика звонков: запись в БД, валидация прав/занятости.

Сам сигналинг WebRTC обрабатывается в sockets/call_events.py, актуальное
состояние идущего звонка — в sockets/call_state.py. Здесь — только всё,
что переживает рестарт сервера (история) и валидация.
"""
from datetime import datetime, timezone
from typing import Optional

from app.extensions import db
from app.models import Call, CallParticipant, User, Conversation, Message
from app.repositories import message_repo
from app.sockets import call_state


class CallServiceError(Exception):
    def __init__(self, code: str, message: str, http_status: int = 400):
        self.code = code
        self.message = message
        self.http_status = http_status


def _now() -> datetime:
    return datetime.now(timezone.utc)


def start_call(initiator_id: int, invitee_ids: list[int],
               media: str = "video") -> Call:
    """Создать запись звонка и зарегистрировать его в in-memory state.

    Валидация:
    - инициатор не может звонить сам себе
    - все приглашённые должны существовать и не быть скрыты
    - инициатор и никто из приглашённых не должны быть уже заняты другим звонком
    """
    invitee_ids = list({uid for uid in invitee_ids if uid != initiator_id})
    if not invitee_ids:
        raise CallServiceError("EMPTY_INVITEES", "Не указаны участники звонка", 400)
    if len(invitee_ids) > 8:
        raise CallServiceError("TOO_MANY_INVITEES",
                               "Максимум 8 участников в одном звонке", 400)

    if call_state.is_user_busy(initiator_id):
        raise CallServiceError("BUSY", "Вы уже в звонке", 409)
    busy = [uid for uid in invitee_ids if call_state.is_user_busy(uid)]
    if busy:
        raise CallServiceError("INVITEE_BUSY",
                               "Один из участников уже разговаривает", 409)

    invitees = db.session.execute(
        db.select(User).where(User.id.in_(invitee_ids), User.is_hidden.is_(False))
    ).scalars().all()
    if len(invitees) != len(invitee_ids):
        raise CallServiceError("USER_NOT_FOUND",
                               "Один из участников не найден", 404)

    kind = "p2p" if len(invitee_ids) == 1 else "group"

    # Парная привязка к диалогу — только для p2p, чтобы в истории можно было
    # быстро открыть переписку с тем же собеседником. Для p2p создаём
    # диалог сразу, даже если его не было — иначе негде хранить системное
    # сообщение о звонке.
    conv_id = None
    if kind == "p2p":
        conv = message_repo.get_or_create_conversation(initiator_id, invitee_ids[0])
        conv_id = conv.id

    call = Call(
        initiator_id=initiator_id,
        kind=kind,
        status="ringing",
        media=media,
        started_at=_now(),
        conversation_id=conv_id,
    )
    db.session.add(call)
    db.session.flush()  # нужен id

    db.session.add(CallParticipant(
        call_id=call.id, user_id=initiator_id, role="initiator",
        invited_at=_now(), joined_at=_now(),
    ))
    for uid in invitee_ids:
        db.session.add(CallParticipant(
            call_id=call.id, user_id=uid, role="invitee", invited_at=_now(),
        ))

    # Системное сообщение в чате (только p2p). Создаётся в статусе ringing —
    # фронт рендерит плашку «Идёт звонок · Присоединиться» для приглашённого
    # и «Звоните…» для инициатора. После _finalize call обновится в БД
    # (status, ended_at), фронту прилетит message:updated.
    sys_msg_id = None
    if kind == "p2p" and conv_id:
        sys_msg = message_repo.create_call_message(conv_id, initiator_id, call.id)
        sys_msg_id = sys_msg.id

    db.session.commit()

    call_state.create_call(call.id, initiator_id, invitee_ids, kind, media)
    # Запомним id системного сообщения и conv_id — нужны при _finalize, чтобы
    # эмитить message:updated в комнаты обоих сторон.
    state = call_state.get_call(call.id)
    if state is not None:
        state["system_message_id"] = sys_msg_id
        state["conversation_id"] = conv_id
    return call


def invite_to_call(call_id: int, inviter_id: int, invitee_ids: list[int]) -> tuple[Call, list[int]]:
    """Пригласить новых участников в уже идущий звонок. Любой участник может
    позвать ещё людей. Возвращает (call, новые_приглашённые_ids).

    Валидация:
    - звонок должен существовать и инициатор приглашения — быть в нём;
    - новые приглашённые должны существовать, не быть скрыты и не быть заняты;
    - тех, кто уже в этом звонке, молча пропускаем.
    """
    state = call_state.get_call(call_id)
    if not state or inviter_id not in state["invited"]:
        raise CallServiceError("NOT_IN_CALL", "Вы не в этом звонке", 404)

    already = state["invited"]
    invitee_ids = list({uid for uid in invitee_ids
                        if uid != inviter_id and uid not in already})
    if not invitee_ids:
        return _get_call(call_id), []

    if len(already) + len(invitee_ids) > 9:  # инициатор + до 8 приглашённых
        raise CallServiceError("TOO_MANY_INVITEES",
                               "В звонке слишком много участников", 400)

    busy = [uid for uid in invitee_ids if call_state.is_user_busy(uid)]
    if busy:
        raise CallServiceError("INVITEE_BUSY",
                               "Один из приглашённых уже разговаривает", 409)

    users = db.session.execute(
        db.select(User).where(User.id.in_(invitee_ids), User.is_hidden.is_(False))
    ).scalars().all()
    if len(users) != len(invitee_ids):
        raise CallServiceError("USER_NOT_FOUND", "Один из участников не найден", 404)

    call = _get_call(call_id)
    for uid in invitee_ids:
        # CallParticipant мог остаться от прежнего выхода — переиспользуем.
        part = _get_participant(call_id, uid)
        if part is None:
            db.session.add(CallParticipant(
                call_id=call_id, user_id=uid, role="invitee", invited_at=_now(),
            ))
        else:
            part.invited_at = _now()
            part.left_at = None
            part.declined = False
        call_state.add_invitee(call_id, uid)

    # Звонок на двоих превратился в групповой.
    if call.kind == "p2p":
        call.kind = "group"
        call_state.set_kind(call_id, "group")

    db.session.commit()
    return call, invitee_ids


def accept_call(call_id: int, user_id: int) -> Call:
    state = call_state.get_call(call_id)
    if not state or user_id not in state["invited"]:
        raise CallServiceError("NOT_INVITED", "Вы не приглашены в этот звонок", 404)
    if user_id in state["joined"]:
        return _get_call(call_id)

    call_state.mark_joined(call_id, user_id)

    call = _get_call(call_id)
    if call.status == "ringing":
        call.status = "active"
    part = _get_participant(call_id, user_id)
    if part and not part.joined_at:
        part.joined_at = _now()
    db.session.commit()
    return call


def decline_call(call_id: int, user_id: int) -> Optional[Call]:
    """Явный отказ. Возвращает обновлённый Call (если ещё существует)."""
    state = call_state.get_call(call_id)
    if not state or user_id not in state["invited"]:
        return None

    call_state.mark_declined(call_id, user_id)

    call = _get_call(call_id)
    part = _get_participant(call_id, user_id)
    if part:
        part.declined = True
        part.left_at = _now()

    # Если в p2p звонке отказался единственный приглашённый — это «не дозвонился».
    if call.kind == "p2p" and call.status == "ringing":
        call.status = "missed"
        call.ended_at = _now()
        db.session.commit()
        call_state.end_call(call_id)
        return call

    if call_state.should_end(call_id):
        _finalize(call)
        call_state.end_call(call_id)
    else:
        db.session.commit()
    return call


def leave_call(call_id: int, user_id: int) -> Optional[Call]:
    state = call_state.get_call(call_id)
    if not state:
        return None
    call_state.remove_user_from_call(call_id, user_id)
    part = _get_participant(call_id, user_id)
    if part and not part.left_at:
        part.left_at = _now()

    call = _get_call(call_id)
    if call_state.should_end(call_id):
        _finalize(call)
        call_state.end_call(call_id)
    else:
        db.session.commit()
    return call


def end_call_by_initiator(call_id: int, user_id: int) -> Optional[Call]:
    """Инициатор может завершить звонок целиком (для всех)."""
    state = call_state.get_call(call_id)
    if not state or state["initiator_id"] != user_id:
        return None
    call = _get_call(call_id)
    for uid in list(state["joined"]):
        part = _get_participant(call_id, uid)
        if part and not part.left_at:
            part.left_at = _now()
    _finalize(call)
    call_state.end_call(call_id)
    return call


def cleanup_user_on_disconnect(user_id: int) -> Optional[tuple[int, list[int]]]:
    """Когда пользователь отвалился (сокет/закрыл вкладку): убираем его из
    активного звонка и возвращаем (call_id, [кому-сообщить]) — список тех,
    кому нужно отправить participant:left."""
    call_id = call_state.get_user_active_call(user_id)
    if call_id is None:
        return None
    notify_targets = [uid for uid in call_state.get_participants(call_id) if uid != user_id]
    call_state.remove_user_from_any_call(user_id)
    # БД: помечаем left_at
    part = _get_participant(call_id, user_id)
    if part and not part.left_at:
        part.left_at = _now()
    # Если в звонке никого не осталось — финализируем
    if call_state.should_end(call_id):
        call = _get_call(call_id)
        if call:
            _finalize(call)
        call_state.end_call(call_id)
    else:
        try:
            db.session.commit()
        except Exception:
            db.session.rollback()
    return call_id, notify_targets


def list_history_for_user(user_id: int, limit: int = 50) -> list[Call]:
    """Звонки, в которых пользователь был участником (любой ролью).
    Сортировка — по убыванию started_at."""
    rows = db.session.execute(
        db.select(Call).join(CallParticipant, CallParticipant.call_id == Call.id)
        .where(CallParticipant.user_id == user_id)
        .order_by(Call.started_at.desc())
        .limit(limit)
    ).scalars().unique().all()
    return rows


def _find_conversation(a: int, b: int) -> Optional[Conversation]:
    lo, hi = (a, b) if a < b else (b, a)
    return db.session.execute(
        db.select(Conversation)
        .where(Conversation.user_a_id == lo, Conversation.user_b_id == hi)
    ).scalar_one_or_none()


def get_system_call_message(message_id: int) -> Optional[Message]:
    """Перечитать системное сообщение о звонке — для эмита message:updated
    в сокет после изменения статуса/длительности звонка."""
    return db.session.get(Message, message_id)


def _get_call(call_id: int) -> Call:
    return db.session.get(Call, call_id)


def _get_participant(call_id: int, user_id: int) -> Optional[CallParticipant]:
    return db.session.execute(
        db.select(CallParticipant).where(
            CallParticipant.call_id == call_id,
            CallParticipant.user_id == user_id,
        )
    ).scalar_one_or_none()


def _finalize(call: Call) -> None:
    if call.status not in ("ended", "missed"):
        call.status = "ended"
    if not call.ended_at:
        call.ended_at = _now()
    try:
        db.session.commit()
    except Exception:
        db.session.rollback()

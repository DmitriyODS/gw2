"""Состояние активных звонков в памяти процесса.

Сам медиа-поток ходит peer-to-peer через WebRTC (с STUN/TURN); сервер только
маршрутизирует сигналинг (offer/answer/ice) между уже знакомыми участниками
и держит список «кто сейчас в звонке N». БД хранит только историю.

Развёртывание — один app-контейнер с eventlet (как и presence), поэтому
in-memory достаточно. Если когда-нибудь будет несколько процессов — выносить
в Redis pubsub.
"""
from typing import Optional


# call_id -> {
#   "initiator_id": int,
#   "kind": "p2p" | "group",
#   "media": "audio" | "video",
#   "invited": set[int]   — кому отправлен invite (включая инициатора)
#   "joined": set[int]    — кто accept'нул и сейчас в звонке
#   "declined": set[int]  — кто явно отклонил
# }
_calls: dict[int, dict] = {}

# user_id -> call_id (активный звонок пользователя; None если свободен).
# Нужен, чтобы при втором invite понять, что человек занят, и чтобы при
# disconnect быстро убрать его из звонка.
_user_call: dict[int, int] = {}


def get_user_active_call(user_id: int) -> Optional[int]:
    return _user_call.get(user_id)


def is_user_busy(user_id: int) -> bool:
    """Пользователь занят, если уже в звонке или ему пришёл активный invite."""
    return user_id in _user_call


def get_call(call_id: int) -> Optional[dict]:
    return _calls.get(call_id)


def get_participants(call_id: int) -> list[int]:
    """Все, кто сейчас в звонке (приняли и не вышли)."""
    state = _calls.get(call_id)
    return list(state["joined"]) if state else []


def get_invited(call_id: int) -> list[int]:
    state = _calls.get(call_id)
    return list(state["invited"]) if state else []


def create_call(call_id: int, initiator_id: int, invitee_ids: list[int],
                kind: str, media: str) -> None:
    """Регистрируем новый звонок. Инициатор сразу считается «joined» (он уже
    в локальной комнате звонка и готов получить streams от accept'нувших).
    Приглашённые висят в invited до accept/decline."""
    invited = {initiator_id, *invitee_ids}
    _calls[call_id] = {
        "initiator_id": initiator_id,
        "kind": kind,
        "media": media,
        "invited": invited,
        "joined": {initiator_id},
        "declined": set(),
    }
    for uid in invited:
        _user_call[uid] = call_id


def mark_joined(call_id: int, user_id: int) -> None:
    state = _calls.get(call_id)
    if not state or user_id not in state["invited"]:
        return
    state["joined"].add(user_id)
    _user_call[user_id] = call_id


def mark_declined(call_id: int, user_id: int) -> None:
    state = _calls.get(call_id)
    if not state:
        return
    state["declined"].add(user_id)
    state["joined"].discard(user_id)
    if _user_call.get(user_id) == call_id:
        _user_call.pop(user_id, None)


def remove_user_from_call(call_id: int, user_id: int) -> None:
    state = _calls.get(call_id)
    if not state:
        return
    state["joined"].discard(user_id)
    if _user_call.get(user_id) == call_id:
        _user_call.pop(user_id, None)


def remove_user_from_any_call(user_id: int) -> Optional[int]:
    """Убрать пользователя из его активного звонка (при disconnect и т. п.).
    Возвращает call_id, если что-то было."""
    call_id = _user_call.pop(user_id, None)
    if call_id is None:
        return None
    state = _calls.get(call_id)
    if state:
        state["joined"].discard(user_id)
    return call_id


def end_call(call_id: int) -> Optional[dict]:
    """Полностью завершить звонок: убрать всех из user→call и снять state.
    Возвращает прежнее состояние (или None)."""
    state = _calls.pop(call_id, None)
    if not state:
        return None
    for uid in state["invited"]:
        if _user_call.get(uid) == call_id:
            _user_call.pop(uid, None)
    return state


def should_end(call_id: int) -> bool:
    """Звонок надо закрыть, когда в нём не осталось активных участников
    или остался только один (одному не с кем разговаривать)."""
    state = _calls.get(call_id)
    if not state:
        return True
    return len(state["joined"]) <= 1 and not _has_pending_invitees(call_id)


def _has_pending_invitees(call_id: int) -> bool:
    state = _calls.get(call_id)
    if not state:
        return False
    pending = state["invited"] - state["joined"] - state["declined"]
    return bool(pending)

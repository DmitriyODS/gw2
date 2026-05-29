"""Юнит-тесты состояния звонков (in-memory) — ядро логики, от которой
зависит «переживание» reload и rejoin. БД/сокеты не нужны: тестируем чистые
функции call_state.

Запуск:  cd back && ./venv/bin/pytest tests/ -q
"""
import pytest

from app.sockets import call_state


@pytest.fixture(autouse=True)
def _clean_state():
    """Сбрасываем глобальное in-memory состояние между тестами."""
    call_state._calls.clear()
    call_state._user_call.clear()
    yield
    call_state._calls.clear()
    call_state._user_call.clear()


def test_create_call_marks_busy_and_initiator_joined():
    call_state.create_call(100, initiator_id=1, invitee_ids=[2], kind="p2p", media="video")

    assert call_state.is_user_busy(1) is True
    assert call_state.is_user_busy(2) is True
    assert call_state.get_user_active_call(1) == 100
    assert call_state.get_user_active_call(2) == 100
    # Инициатор сразу в joined, приглашённый — пока только invited.
    assert call_state.get_participants(100) == [1]
    assert set(call_state.get_invited(100)) == {1, 2}


def test_p2p_should_end_lifecycle():
    call_state.create_call(100, 1, [2], "p2p", "video")
    # Есть pending-приглашённый (2 ещё не принял) — звонок не закрываем.
    assert call_state.should_end(100) is False

    call_state.mark_joined(100, 2)
    assert sorted(call_state.get_participants(100)) == [1, 2]
    assert call_state.should_end(100) is False

    # Один вышел — остался один, новых приглашений нет → пора завершать.
    call_state.remove_user_from_call(100, 2)
    assert call_state.should_end(100) is True


def test_mark_joined_is_idempotent_for_rejoin():
    """rejoin после reload вызывает accept повторно — состояние не должно
    задваиваться, пользователь остаётся ровно одним участником."""
    call_state.create_call(100, 1, [2], "p2p", "video")
    call_state.mark_joined(100, 2)
    call_state.mark_joined(100, 2)  # повторно (rejoin)

    assert sorted(call_state.get_participants(100)) == [1, 2]
    assert call_state.get_user_active_call(2) == 100


def test_decline_removes_from_call():
    call_state.create_call(100, 1, [2], "p2p", "video")
    call_state.mark_declined(100, 2)

    assert call_state.is_user_busy(2) is False
    assert call_state.get_user_active_call(2) is None
    assert 2 not in call_state.get_participants(100)


def test_grace_reload_keeps_user_in_call():
    """Симуляция reload: пользователь временно «исчез», но его НЕ убрали из
    звонка (grace-окно). get_user_active_call по-прежнему отдаёт call_id —
    значит /api/calls/active вернёт звонок и rejoin сработает."""
    call_state.create_call(100, 1, [2], "p2p", "video")
    call_state.mark_joined(100, 2)

    # Во время grace мы НИЧЕГО не удаляем из state — просто проверяем, что
    # пользователь всё ещё «в звонке» и сможет вернуться.
    assert call_state.get_user_active_call(2) == 100
    assert 2 in call_state.get_participants(100)
    # И повторный mark_joined (rejoin) безопасен.
    call_state.mark_joined(100, 2)
    assert sorted(call_state.get_participants(100)) == [1, 2]


def test_disconnect_finalize_removes_and_ends_p2p():
    """А вот если grace истёк и соединения нет — убираем из звонка; в p2p
    после ухода второго участника звонок должен закрыться."""
    call_state.create_call(100, 1, [2], "p2p", "video")
    call_state.mark_joined(100, 2)

    cid = call_state.remove_user_from_any_call(2)
    assert cid == 100
    assert call_state.get_user_active_call(2) is None
    # Остался один — звонок надо завершить.
    assert call_state.should_end(100) is True

    state = call_state.end_call(100)
    assert state is not None
    assert call_state.get_call(100) is None
    assert call_state.get_user_active_call(1) is None


def test_group_call_survives_one_leaving():
    call_state.create_call(200, 1, [2, 3], "group", "video")
    call_state.mark_joined(200, 2)
    call_state.mark_joined(200, 3)
    assert sorted(call_state.get_participants(200)) == [1, 2, 3]

    call_state.remove_user_from_call(200, 3)
    # Ещё двое — звонок продолжается.
    assert call_state.should_end(200) is False
    assert sorted(call_state.get_participants(200)) == [1, 2]


def test_busy_blocks_second_call_invite():
    call_state.create_call(100, 1, [2], "p2p", "video")
    # 2 занят первым звонком — попытка позвать его в другой должна видеть busy.
    assert call_state.is_user_busy(2) is True

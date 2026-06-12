"""Тесты обработчика webhook-событий YouGile.

Без БД и сети — мокаем модель Task, task_repo, comment_repo, db, socketio
и user_repo. Проверяем три ключевых сценария: антицикл, изменение
title/deadline/completed, разрыв связи при task-deleted.
"""
import os
from datetime import datetime, timezone
from types import SimpleNamespace
from unittest.mock import MagicMock, patch

import pytest

os.environ.setdefault("YOUGILE_ENC_KEY",
                      "CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=")

from app.integrations.yougile import task_apply as apply_mod
from app.integrations.yougile.task_service import (
    _dt_to_ms, _sync_hash,
)


def _make_company(**ov):
    base = dict(
        id=1,
        yg_completed_column_id=None,
        yg_first_column_id="col-first",
        yg_board_id="board-1",
    )
    base.update(ov)
    return SimpleNamespace(**base)


def _make_task(**ov):
    base = dict(
        id=10, name="Hello", company_id=1, is_archived=False,
        archived_at=None, deadline=None, author_id=99,
        link_yougile="https://ru.yougile.com/x", yougile_task_id="yg-1",
        yougile_project_id="p", yougile_board_id="board-1",
        yougile_column_id="c1", yougile_synced_at=None, yougile_sync_hash=None,
        company=_make_company(),
    )
    base.update(ov)
    return SimpleNamespace(**base)


# ── маршрутизация ─────────────────────────────────────────────────────────

def test_apply_skipped_when_task_unknown():
    company = _make_company()
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = None
    with patch.object(apply_mod, "Task") as Task_mod:
        Task_mod.query = fake_q
        out = apply_mod.apply_event(company, {"event": "task-updated",
                                              "data": {"id": "yg-x"}})
        assert out["status"] == "skipped"


def test_apply_no_id_skipped():
    company = _make_company()
    out = apply_mod.apply_event(company, {"event": "task-updated"})
    assert out["status"] == "skipped"
    assert out["reason"] == "no-id"


# ── антицикл ──────────────────────────────────────────────────────────────

def test_self_echo_is_skipped():
    task = _make_task(
        name="Hello", deadline=None,
        yougile_sync_hash=_sync_hash(title="Hello", deadline_ms=None,
                                     completed=False),
    )
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "socketio") as sock_mod:
        Task_mod.query = fake_q
        out = apply_mod.apply_event(task.company, {
            "event": "task-updated",
            "data": {"id": "yg-1", "title": "Hello", "completed": False},
        })
        assert out["status"] == "skipped"
        assert out["reason"] == "self-echo"
        repo_mod.update.assert_not_called()
        sock_mod.emit.assert_not_called()


# ── изменения ─────────────────────────────────────────────────────────────

def test_title_change_applies():
    task = _make_task(name="Old")
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "socketio"), \
         patch.object(apply_mod, "db"), \
         patch("app.integrations.yougile.task_dump.enrich_task", return_value={"id": 10, "name": "New"}):
        Task_mod.query = fake_q
        out = apply_mod.apply_event(task.company, {
            "event": "task-updated",
            "data": {"id": "yg-1", "title": "New"},
        })
        assert out["status"] == "applied"
        assert "name" in out["fields"]
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["name"] == "New"
        assert upd_kw["yougile_sync_hash"]


def test_deadline_applies_from_ms():
    task = _make_task(name="X", deadline=None)
    ms = 1717_000_000_000
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "socketio"), \
         patch.object(apply_mod, "db"), \
         patch("app.integrations.yougile.task_dump.enrich_task", return_value={}):
        Task_mod.query = fake_q
        out = apply_mod.apply_event(task.company, {
            "event": "task-updated",
            "data": {"id": "yg-1", "deadline": {"deadline": ms}},
        })
        assert out["status"] == "applied"
        upd_kw = repo_mod.update.call_args.kwargs
        assert isinstance(upd_kw["deadline"], datetime)
        assert _dt_to_ms(upd_kw["deadline"]) == ms


def test_completed_triggers_archive():
    task = _make_task(name="X")
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "socketio"), \
         patch.object(apply_mod, "db"), \
         patch("app.integrations.yougile.task_dump.enrich_task", return_value={}):
        Task_mod.query = fake_q
        repo_mod.has_active_unit.return_value = False
        out = apply_mod.apply_event(task.company, {
            "event": "task-completed",
            "data": {"id": "yg-1", "title": "X", "completed": True},
        })
        assert out["status"] == "applied"
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["is_archived"] is True
        assert isinstance(upd_kw["archived_at"], datetime)


def test_completed_with_active_unit_does_not_archive():
    """Инвариант: задачу с активным юнитом не архивируем даже по completed из YG."""
    task = _make_task(name="X")
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "socketio"), \
         patch.object(apply_mod, "db"), \
         patch("app.integrations.yougile.task_dump.enrich_task", return_value={}):
        Task_mod.query = fake_q
        repo_mod.has_active_unit.return_value = True
        out = apply_mod.apply_event(task.company, {
            "event": "task-completed",
            "data": {"id": "yg-1", "title": "X", "completed": True},
        })
        # Архива нет: либо нет полей (no-changes), либо есть, но без is_archived.
        upd_kw = repo_mod.update.call_args.kwargs if repo_mod.update.called else {}
        assert "is_archived" not in upd_kw


def test_move_to_completed_column_archives():
    company = _make_company(yg_completed_column_id="done-col")
    task = _make_task(company=company, yougile_column_id="other")
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "socketio"), \
         patch.object(apply_mod, "db"), \
         patch("app.integrations.yougile.task_dump.enrich_task", return_value={}):
        Task_mod.query = fake_q
        repo_mod.has_active_unit.return_value = False
        out = apply_mod.apply_event(company, {
            "event": "task-moved",
            "data": {"id": "yg-1", "columnId": "done-col", "completed": False},
        })
        assert out["status"] == "applied"
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["yougile_column_id"] == "done-col"
        assert upd_kw["is_archived"] is True


# ── deleted / restored ───────────────────────────────────────────────────

def test_task_deleted_unlinks_and_comments():
    task = _make_task(yougile_task_id="yg-1", link_yougile="https://yg/x")
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "user_repo") as ur_mod, \
         patch.object(apply_mod, "_post_system_comment") as post_comment_fn, \
         patch.object(apply_mod, "socketio"), \
         patch.object(apply_mod, "db"), \
         patch("app.integrations.yougile.task_dump.enrich_task", return_value={}):
        Task_mod.query = fake_q
        ur_mod.get_by_id.return_value = SimpleNamespace(id=99)
        out = apply_mod.apply_event(task.company, {
            "event": "task-deleted",
            "data": {"id": "yg-1"},
        })
        assert out["status"] == "unlinked"
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["yougile_task_id"] is None
        assert upd_kw["link_yougile"] is None
        post_comment_fn.assert_called_once()


def test_task_restored_unarchives():
    task = _make_task(is_archived=True, archived_at=datetime.now(timezone.utc))
    fake_q = MagicMock()
    fake_q.filter_by.return_value.first.return_value = task
    with patch.object(apply_mod, "Task") as Task_mod, \
         patch.object(apply_mod, "task_repo") as repo_mod, \
         patch.object(apply_mod, "socketio"), \
         patch.object(apply_mod, "db"), \
         patch("app.integrations.yougile.task_dump.enrich_task", return_value={}):
        Task_mod.query = fake_q
        out = apply_mod.apply_event(task.company, {
            "event": "task-restored",
            "data": {"id": "yg-1"},
        })
        assert out["status"] == "restored"
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["is_archived"] is False
        assert upd_kw["archived_at"] is None

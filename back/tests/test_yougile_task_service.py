"""Unit-тесты бизнес-логики импорта/экспорта/отвязки YG-задач.

Без БД и без сети — мокаем `build_client_for_user`, `task_repo`,
`task_service.create_task`, `comment_repo.create`, `db.session.commit`,
а также `UserYougileAccount.query`.
"""
import os
from datetime import datetime, timezone
from types import SimpleNamespace
from unittest.mock import MagicMock, patch

import pytest

os.environ.setdefault("YOUGILE_ENC_KEY",
                      "CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=")

from app.integrations.yougile import task_service as svc
from app.integrations.yougile.task_service import (
    ImportPayload, YougileTaskError, _sync_hash,
)


# ── фикстуры объектов ─────────────────────────────────────────────────────

def _make_company(**overrides):
    base = dict(
        id=1,
        settings={"uses_yougile": True},
        yg_company_id="comp-uuid",
        yg_project_id="proj-uuid",
        yg_board_id="board-uuid",
        yg_first_column_id="col-first-uuid",
    )
    base.update(overrides)
    return SimpleNamespace(**base)


def _make_user(company=None):
    return SimpleNamespace(id=42, company=company or _make_company(),
                           company_id=(company or _make_company()).id)


def _make_task(**overrides):
    base = dict(
        id=101, name="Hello", company_id=1, deadline=None, created_at=None,
        link_yougile=None, yougile_task_id=None, yougile_project_id=None,
        yougile_board_id=None, yougile_column_id=None,
        yougile_synced_at=None, yougile_sync_hash=None,
    )
    base.update(overrides)
    return SimpleNamespace(**base)


# ── чистые функции ────────────────────────────────────────────────────────

def test_sync_hash_stable_and_distinct():
    a = _sync_hash(title="A", deadline_ms=None, completed=False)
    b = _sync_hash(title="A", deadline_ms=None, completed=False)
    c = _sync_hash(title="A", deadline_ms=None, completed=True)
    assert a == b
    assert a != c
    assert len(a) == 40  # sha1 hex


# ── проверки доступа ──────────────────────────────────────────────────────

def test_require_company_disabled():
    company = _make_company(settings={"uses_yougile": False})
    with pytest.raises(YougileTaskError) as ei:
        svc._require_company_enabled(company)
    assert ei.value.code == "COMPANY_DISABLED"


def test_require_company_not_configured():
    company = _make_company(yg_board_id=None)
    with pytest.raises(YougileTaskError) as ei:
        svc._require_company_enabled(company)
    assert ei.value.code == "COMPANY_NOT_CONFIGURED"


def test_require_user_not_connected():
    with patch.object(svc, "build_client_for_user", return_value=None):
        with pytest.raises(YougileTaskError) as ei:
            svc._require_user_connected(_make_user())
        assert ei.value.code == "USER_NOT_CONNECTED"
        assert ei.value.http_status == 412


# ── import_from_url ──────────────────────────────────────────────────────

def test_import_bad_url():
    user = _make_user()
    yg_client = MagicMock()
    with patch.object(svc, "build_client_for_user", return_value=yg_client):
        with pytest.raises(YougileTaskError) as ei:
            svc.import_from_url(user, ImportPayload(url="not-a-url", department_id=5))
        assert ei.value.code == "BAD_URL"


def test_import_foreign_company_blocked():
    user = _make_user()
    yg_client = MagicMock()
    foreign_url = (
        "https://ru.yougile.com/team/11111111-1111-1111-1111-111111111111/"
        "#tasks?task=22222222-2222-2222-2222-222222222222"
    )
    with patch.object(svc, "build_client_for_user", return_value=yg_client):
        with pytest.raises(YougileTaskError) as ei:
            svc.import_from_url(user, ImportPayload(url=foreign_url, department_id=5))
        assert ei.value.code == "FOREIGN_COMPANY"


def test_import_success_writes_link_and_posts_back():
    user = _make_user()
    yg_client = MagicMock()
    yg_task_id = "4f6f0391-0f94-4d30-9b0e-99430a36d4fb"
    url = (
        f"https://ru.yougile.com/team/{user.company.yg_company_id}/"
        f"#tasks?task={yg_task_id}"
    )
    # YG отдаст задачу с дедлайном
    yg_client.get_task.return_value = {
        "id": yg_task_id,
        "title": "Импорт из YG",
        "columnId": "col-yg",
        "deadline": {"deadline": 1717000000000, "startDate": 0, "withTime": False},
        "completed": False,
        "description": "desc",
    }

    created_task = _make_task(id=777, name="Импорт из YG")
    fake_query = MagicMock()
    fake_query.filter_by.return_value.first.return_value = None  # связи нет

    with patch.object(svc, "build_client_for_user", return_value=yg_client), \
         patch.object(svc, "task_service") as ts_mod, \
         patch.object(svc, "task_repo") as repo_mod, \
         patch.object(svc, "comment_repo") as comm_mod, \
         patch.object(svc, "db") as db_mod, \
         patch.object(svc, "Task") as Task_mod:
        Task_mod.query = fake_query
        ts_mod.create_task.return_value = created_task

        out = svc.import_from_url(
            user,
            ImportPayload(url=url, department_id=5, responsible_user_id=None),
            origin="https://gw.example.com",
        )

        assert out is created_task
        # task_service.create_task получил yougile-URL и подтянутый дедлайн.
        kw = ts_mod.create_task.call_args.kwargs
        assert kw["company_id"] == 1
        assert kw["department_id"] == 5
        assert kw["link_yougile"].endswith(f"task={yg_task_id}")
        assert isinstance(kw["deadline"], datetime)

        # Структурные поля обновлены через task_repo.update.
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["yougile_task_id"] == yg_task_id
        assert upd_kw["yougile_project_id"] == "proj-uuid"
        assert upd_kw["yougile_board_id"] == "board-uuid"
        assert upd_kw["yougile_column_id"] == "col-yg"
        assert upd_kw["yougile_sync_hash"]
        assert isinstance(upd_kw["yougile_synced_at"], datetime)

        # В YG-карточке оставили обратную ссылку на GW.
        yg_client.post_chat_message.assert_called_once()
        chat_kw = yg_client.post_chat_message.call_args.args
        assert chat_kw[0] == yg_task_id
        assert "https://gw.example.com/tasks/777" in chat_kw[1]["text"]

        # В GW появился системный комментарий.
        comm_mod.create.assert_called_once()


def test_import_existing_link_returns_existing():
    user = _make_user()
    yg_task_id = "4f6f0391-0f94-4d30-9b0e-99430a36d4fb"
    url = (
        f"https://ru.yougile.com/team/{user.company.yg_company_id}/"
        f"#tasks?task={yg_task_id}"
    )
    existing = _make_task(yougile_task_id=yg_task_id)
    yg_client = MagicMock()
    fake_query = MagicMock()
    fake_query.filter_by.return_value.first.return_value = existing

    with patch.object(svc, "build_client_for_user", return_value=yg_client), \
         patch.object(svc, "Task") as Task_mod, \
         patch.object(svc, "task_service") as ts_mod:
        Task_mod.query = fake_query
        out = svc.import_from_url(user, ImportPayload(url=url, department_id=5))
        assert out is existing
        ts_mod.create_task.assert_not_called()
        yg_client.get_task.assert_not_called()


# ── export_to_yougile ────────────────────────────────────────────────────

def test_export_success_writes_yougile_fields_and_posts_link_back():
    user = _make_user()
    yg_client = MagicMock()
    yg_client.create_task.return_value = {"id": "new-yg-task"}
    task = _make_task(id=55, name="Экспорт", deadline=datetime(2026, 7, 1, tzinfo=timezone.utc))

    fake_acc_query = MagicMock()
    fake_acc_query.filter_by.return_value.first.return_value = SimpleNamespace(yg_user_id="me-yg")

    with patch.object(svc, "build_client_for_user", return_value=yg_client), \
         patch.object(svc, "task_repo") as repo_mod, \
         patch.object(svc, "comment_repo") as comm_mod, \
         patch.object(svc, "db") as db_mod, \
         patch.object(svc, "UserYougileAccount") as Acc_mod:
        repo_mod.get_by_id.return_value = task
        Acc_mod.query = fake_acc_query
        out = svc.export_to_yougile(user, 55, origin="https://gw.example.com")

        assert out is task
        # POST /tasks ушёл в первую колонку с assigned=[me-yg].
        body = yg_client.create_task.call_args.args[0]
        assert body["columnId"] == "col-first-uuid"
        assert body["assigned"] == ["me-yg"]
        assert body["title"] == "Экспорт"
        # deadline → ms
        assert "deadline" in body and isinstance(body["deadline"]["deadline"], int)

        # task обновился: link_yougile + yougile_task_id + sync_hash.
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["yougile_task_id"] == "new-yg-task"
        assert upd_kw["link_yougile"].endswith("task=new-yg-task")
        assert upd_kw["yougile_sync_hash"]

        # В YG отправили линк назад в GW.
        yg_client.post_chat_message.assert_called_once()
        # И системный комментарий в GW.
        comm_mod.create.assert_called_once()


def test_export_already_linked_raises():
    user = _make_user()
    task = _make_task(yougile_task_id="already")
    with patch.object(svc, "build_client_for_user", return_value=MagicMock()), \
         patch.object(svc, "task_repo") as repo_mod:
        repo_mod.get_by_id.return_value = task
        with pytest.raises(YougileTaskError) as ei:
            svc.export_to_yougile(user, task.id)
        assert ei.value.code == "ALREADY_LINKED"


def test_export_task_in_other_company():
    user = _make_user()
    task = _make_task(company_id=999)
    with patch.object(svc, "build_client_for_user", return_value=MagicMock()), \
         patch.object(svc, "task_repo") as repo_mod:
        repo_mod.get_by_id.return_value = task
        with pytest.raises(YougileTaskError) as ei:
            svc.export_to_yougile(user, task.id)
        assert ei.value.code == "NOT_FOUND"


# ── unlink_task ──────────────────────────────────────────────────────────

def test_unlink_clears_fields_and_posts_comment():
    user = _make_user()
    task = _make_task(yougile_task_id="x", link_yougile="https://yg/x")
    with patch.object(svc, "task_repo") as repo_mod, \
         patch.object(svc, "comment_repo") as comm_mod, \
         patch.object(svc, "db"):
        repo_mod.get_by_id.return_value = task
        out = svc.unlink_task(user, task.id)
        assert out is task
        upd_kw = repo_mod.update.call_args.kwargs
        assert upd_kw["yougile_task_id"] is None
        assert upd_kw["link_yougile"] is None
        comm_mod.create.assert_called_once()


def test_unlink_idempotent_no_link():
    user = _make_user()
    task = _make_task(yougile_task_id=None)
    with patch.object(svc, "task_repo") as repo_mod, \
         patch.object(svc, "comment_repo") as comm_mod:
        repo_mod.get_by_id.return_value = task
        out = svc.unlink_task(user, task.id)
        assert out is task
        repo_mod.update.assert_not_called()
        comm_mod.create.assert_not_called()

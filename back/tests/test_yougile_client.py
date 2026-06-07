"""Unit-тесты тонкого HTTP-клиента YouGile.

Без поднятия Flask и БД — мокаем requests на уровне session.request.
"""
import os
import pytest

os.environ.setdefault("YOUGILE_ENC_KEY",
                      "CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=")

from app.integrations.yougile import (
    YougileClient, YougileAuthError, YougileError, YougileRateLimited,
    parse_task_url, encrypt_key, decrypt_key, make_fingerprint,
)


class _FakeResp:
    def __init__(self, status, json_data=None, text=""):
        self.status_code = status
        self._json = json_data
        self.text = text
        self.content = b"x" if json_data is not None or text else b""

    def json(self):
        if self._json is None:
            raise ValueError("no json")
        return self._json


class _FakeSession:
    def __init__(self, responses):
        # responses — список _FakeResp, отдаются по одному.
        self._queue = list(responses)
        self.calls = []
        self.headers = {}

    def request(self, method, url, headers=None, json=None, params=None, timeout=None):
        self.calls.append({"method": method, "url": url, "headers": headers,
                           "json": json, "params": params})
        if not self._queue:
            raise AssertionError("Закончились запасы _FakeResp")
        return self._queue.pop(0)


# ── parser ───────────────────────────────────────────────────────────────

def test_parse_url_hash_query():
    r = parse_task_url(
        "https://ru.yougile.com/team/9347006b-dc75-4550-97d5-3008ba00d4a0/"
        "#tasks?task=4f6f0391-0f94-4d30-9b0e-99430a36d4fb"
    )
    assert r is not None
    assert r.task_id == "4f6f0391-0f94-4d30-9b0e-99430a36d4fb"
    assert r.company_id == "9347006b-dc75-4550-97d5-3008ba00d4a0"


def test_parse_url_task_hash():
    r = parse_task_url(
        "https://yougile.com/board/abc/#task-4F6F0391-0F94-4D30-9B0E-99430A36D4FB"
    )
    assert r is not None and r.task_id == "4f6f0391-0f94-4d30-9b0e-99430a36d4fb"


def test_parse_url_rejects_non_yougile():
    assert parse_task_url("https://example.com/4f6f0391-0f94-4d30-9b0e-99430a36d4fb") is None
    assert parse_task_url("not a url") is None
    assert parse_task_url("") is None


# ── crypto ───────────────────────────────────────────────────────────────

def test_crypto_roundtrip():
    enc = encrypt_key("H6HngIA816fpIhY7tBvWx/it3YbVvEt/33Sk8afA39MCR9a")
    assert isinstance(enc, (bytes, bytearray)) and len(enc) > 0
    assert decrypt_key(enc) == "H6HngIA816fpIhY7tBvWx/it3YbVvEt/33Sk8afA39MCR9a"
    assert decrypt_key(None) is None


def test_fingerprint_last4():
    assert make_fingerprint("abcdef0123") == "0123"
    assert make_fingerprint("") == ""


# ── client ───────────────────────────────────────────────────────────────

def test_client_anonymous_endpoints_dont_send_bearer():
    sess = _FakeSession([_FakeResp(200, json_data={"content": [{"id": "c1", "name": "X"}]})])
    c = YougileClient(session=sess)
    items = c.list_companies("a@b.c", "pw")
    assert items == [{"id": "c1", "name": "X"}]
    assert "Authorization" not in (sess.calls[0]["headers"] or {})


def test_client_bearer_for_authenticated_endpoint():
    sess = _FakeSession([_FakeResp(200, json_data={"id": "u1"})])
    c = YougileClient(key="K", session=sess)
    c.me()
    assert sess.calls[0]["headers"]["Authorization"] == "Bearer K"


def test_client_auth_401_raises():
    sess = _FakeSession([_FakeResp(401)])
    c = YougileClient(key="K", session=sess)
    with pytest.raises(YougileAuthError):
        c.me()


def test_client_429_retries_then_raises():
    sess = _FakeSession([_FakeResp(429), _FakeResp(429), _FakeResp(429)])
    c = YougileClient(key="K", session=sess)
    import time as _t
    orig = _t.sleep
    _t.sleep = lambda s: None  # отключаем backoff в тестах
    try:
        with pytest.raises(YougileRateLimited):
            c.me()
    finally:
        _t.sleep = orig
    assert len(sess.calls) == 3


def test_client_pagination_collects_pages():
    page1 = {"content": [{"id": f"p{i}", "title": str(i)} for i in range(1000)]}
    page2 = {"content": [{"id": f"p{i}", "title": str(i)} for i in range(1000, 1500)]}
    sess = _FakeSession([_FakeResp(200, json_data=page1),
                         _FakeResp(200, json_data=page2)])
    c = YougileClient(key="K", session=sess)
    items = c.list_projects(limit=2000)
    assert len(items) == 1500
    # Второй запрос ушёл с offset=1000.
    assert sess.calls[1]["params"]["offset"] == 1000


def test_create_key_returns_string():
    sess = _FakeSession([_FakeResp(201, json_data={"key": "KKK"})])
    c = YougileClient(session=sess)
    assert c.create_key("a@b.c", "pw", "comp") == "KKK"


def test_create_key_unexpected_payload_raises():
    sess = _FakeSession([_FakeResp(201, json_data={})])
    c = YougileClient(session=sess)
    with pytest.raises(YougileError):
        c.create_key("a@b.c", "pw", "comp")

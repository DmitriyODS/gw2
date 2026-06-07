"""Извлечение YouGile task_id из URL карточки.

YouGile показывает три практических формата ссылок:

  1. Длинный (старый):
     `https://ru.yougile.com/team/<companyUUID>/#tasks?task=<taskUUID>`
  2. Длинный с board (редко):
     `https://yougile.com/team/<companyUUID>/?board=<boardUUID>#task-<taskUUID>`
  3. Короткий (сейчас дефолт в адресной строке и кнопке «Скопировать»):
     `https://yougile.com/team/<shortTeamId>/#<idTaskProject>`
     Где `shortTeamId` — последние 12 hex-символов UUID компании
     (`773aa0c0-0966-4d40-8540-ed7037760782` → `ed7037760782`), а
     `idTaskProject` — человекочитаемый id карточки `OIP1-2454`.

Для (1)/(2) парсер сразу отдаёт `task_id` (UUID). Для (3) UUID в URL нет —
отдаём `short_task_id` и `short_team_id`, а вызывающий код резолвит UUID
через YouGile API (см. `client.find_task_by_short_id`).
"""
from __future__ import annotations

import re
from dataclasses import dataclass
from urllib.parse import urlparse, parse_qs


_UUID_RE = re.compile(
    r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}",
    re.IGNORECASE,
)
_TEAM_UUID_RE = re.compile(r"/team/([0-9a-f-]{36})/?", re.IGNORECASE)
_TEAM_SHORT_RE = re.compile(r"/team/([0-9a-f]{12})/?", re.IGNORECASE)
# `OIP1-2454`, `ABC-1`, `Z9-100500` — буквы (минимум одна) + опц. цифры,
# дефис, цифры. Строго ALPHA-NUM до дефиса (без _), чтобы не словить UUID
# по ошибке.
_SHORT_TASK_RE = re.compile(r"^([A-Za-z][A-Za-z0-9]*-\d+)$")


@dataclass
class ParsedYougileUrl:
    task_id: str | None = None         # UUID карточки (если разобрали сразу)
    company_id: str | None = None      # UUID компании (длинный)
    short_team_id: str | None = None   # 12-hex (короткий формат)
    short_task_id: str | None = None   # `OIP1-2454`


def parse_task_url(url: str) -> ParsedYougileUrl | None:
    """Разобрать ссылку на YG-карточку.

    Возвращает `None`, если в строке ничего YG-подобного. На уровне вызова
    решают: есть UUID → импортируем напрямую; есть только short_task_id →
    идём в API искать UUID.
    """
    if not url or not isinstance(url, str):
        return None

    url = url.strip()
    try:
        parsed = urlparse(url)
    except ValueError:
        return None

    if "yougile" not in (parsed.netloc or "").lower():
        return None

    # 1. UUID карточки явным образом (?task=UUID или #...task=UUID, #task-UUID).
    candidates: list[str] = []
    if parsed.query:
        for v in parse_qs(parsed.query).get("task", []):
            candidates.append(v)
    if parsed.fragment:
        frag = parsed.fragment
        if "?" in frag:
            _, q = frag.split("?", 1)
            for v in parse_qs(q).get("task", []):
                candidates.append(v)
        if "task-" in frag:
            candidates.append(frag.split("task-", 1)[1])

    task_id: str | None = None
    for c in candidates:
        m = _UUID_RE.search(c)
        if m:
            task_id = m.group(0)
            break
    if not task_id:
        # Fallback: первый UUID где-нибудь в URL.
        m = _UUID_RE.search(url)
        if m:
            task_id = m.group(0)

    # 2. companyId из path: либо полный UUID, либо 12-hex короткий.
    company_id: str | None = None
    short_team_id: str | None = None
    path = parsed.path or ""
    m_full = _TEAM_UUID_RE.search(path)
    if m_full:
        company_id = m_full.group(1)
    else:
        m_short = _TEAM_SHORT_RE.search(path)
        if m_short:
            short_team_id = m_short.group(1).lower()

    # 3. Короткий taskId в hash'е (`#OIP1-2454`). Не пытаемся, если уже есть UUID.
    short_task_id: str | None = None
    if task_id is None and parsed.fragment:
        # фрагмент может быть `OIP1-2454`, либо `tasks?...`, либо `task-...`.
        frag = parsed.fragment.split("?", 1)[0]
        m = _SHORT_TASK_RE.match(frag)
        if m:
            short_task_id = m.group(1).upper()

    if not (task_id or short_task_id):
        return None

    return ParsedYougileUrl(
        task_id=task_id.lower() if task_id else None,
        company_id=company_id,
        short_team_id=short_team_id,
        short_task_id=short_task_id,
    )

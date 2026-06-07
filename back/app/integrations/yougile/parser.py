"""Извлечение YouGile task_id из URL карточки.

В UI YouGile открытая карточка даёт несколько форматов ссылок:

  - https://ru.yougile.com/team/<companyId>/#tasks?task=<taskId>
  - https://yougile.com/board/<...>#task-<taskId>
  - https://yougile.com/team/<companyId>/?board=<boardId>#task-<taskId>

Все вариации сводятся к одному правилу: в строке после хеша/в query есть
UUID — это taskId. Дополнительно дёргаем companyId из path'а, если он там
есть — пригодится при показе ошибки «эта карточка из другой компании».
"""
from __future__ import annotations

import re
from dataclasses import dataclass
from urllib.parse import urlparse, parse_qs


_UUID_RE = re.compile(
    r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}",
    re.IGNORECASE,
)
_TEAM_RE = re.compile(r"/team/([0-9a-f-]{36})/?", re.IGNORECASE)


@dataclass
class ParsedYougileUrl:
    task_id: str
    company_id: str | None = None


def parse_task_url(url: str) -> ParsedYougileUrl | None:
    """Вытащить task_id (и опц. companyId) из ссылки на карточку.

    Возвращает None, если разобрать не удалось — вызывающая сторона показывает
    «не похоже на ссылку YouGile». Никаких исключений наружу, чтобы UI мог
    спокойно сообщить пользователю о проблеме.
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

    # Сначала пробуем явное `?task=<uuid>` (как в hash после ?).
    # urlparse не парсит query внутри fragment'а, поэтому делаем это вручную.
    candidates: list[str] = []
    if parsed.query:
        for v in parse_qs(parsed.query).get("task", []):
            candidates.append(v)
    if parsed.fragment:
        # Хеш может быть `tasks?task=<uuid>` или `task-<uuid>`.
        frag = parsed.fragment
        if "?" in frag:
            _, q = frag.split("?", 1)
            for v in parse_qs(q).get("task", []):
                candidates.append(v)
        if "task-" in frag:
            after = frag.split("task-", 1)[1]
            candidates.append(after)

    task_id: str | None = None
    for c in candidates:
        m = _UUID_RE.search(c)
        if m:
            task_id = m.group(0)
            break

    # Fallback: первый UUID в URL вообще.
    if not task_id:
        m = _UUID_RE.search(url)
        if m:
            task_id = m.group(0)

    if not task_id:
        return None

    company_id: str | None = None
    team_m = _TEAM_RE.search(parsed.path or "")
    if team_m:
        company_id = team_m.group(1)

    return ParsedYougileUrl(task_id=task_id.lower(), company_id=company_id)

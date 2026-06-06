"""Эмбеддинги задач: индексация + семантический поиск.

Работаем через raw SQL, потому что pgvector-операции (`<=>`, типы `vector(N)`)
выходят за рамки чистого ORM, а заводить TypeDecorator ради одной таблицы —
оверкилл. Все запросы — параметризованные, никаких склеек строк.

Индексер дёргается ASYNC из task_service после коммита (через
`socketio.start_background_task`, который под капотом eventlet-greenlet). Если
AI выключен / упал — задача остаётся без эмбеддинга, на ближайшем
ре-апдейте перегенерим. Этот компромисс осознанный: блокировать UX задачи
ради ИИ — недопустимо.
"""
from __future__ import annotations

from datetime import datetime, timezone
from typing import Iterable

from flask import current_app
from sqlalchemy import text

from app.extensions import db, socketio
from app.models.company import Company
from app.models.task import Task
from app.services.ai_client import get_ai_client
from app.utils.logger import get_logger


logger = get_logger(__name__)


# Порог сходства cosine. Делаем низким и не отсекающим: пользователь сам
# увидит, что верх выдачи релевантный, а низ — «всё, что есть похожего».
# Совсем нулевые/отрицательные косинусы (≤ 0) отсекаем — это уже точно мусор.
MIN_SEMANTIC_SCORE = 0.0

# Лимит выдачи. Хотим «просмотреть все задачи и найти похожие» — берём с
# запасом: на компании < 1000 задач это и есть «все».
SEMANTIC_LIMIT = 200

EMBED_BATCH_SIZE = 64              # OpenAI поддерживает массивы до 2048


# ───────────────────────── формирование текста для эмбеддинга ─────────────

def _build_text_for_task(task: Task) -> str:
    """Что эмбеддим. Простой формат, без украшений — модель сама разберётся.

    Сюда заведомо НЕ кладём поля, которые часто меняются по UI-флоу (color,
    is_archived, deadline) — иначе пришлось бы переэмбеддить задачи по каждому
    клику. Только то, что определяет смысл задачи.
    """
    parts = [task.name or ""]
    if task.department is not None:
        parts.append(f"Отдел: {task.department.name}")
    if task.responsible is not None:
        parts.append(f"Ответственный: {task.responsible.fio}")
    return "\n".join(p for p in parts if p)


def _embed_text_hash_changed(task: Task, prev_text: str | None) -> bool:
    """Сменился ли текст-для-эмбеддинга. Сравнение по строке: дешевле, чем
    хранить хэш в БД, и достаточно надёжно (тексты короткие)."""
    return _build_text_for_task(task) != (prev_text or "")


# ───────────────────────── upsert одного / батча ──────────────────────────

def _upsert(task_id: int, company_id: int, embedding: list[float], model: str) -> None:
    db.session.execute(text("""
        INSERT INTO task_embeddings (task_id, company_id, embedding, model, updated_at)
        VALUES (:tid, :cid, CAST(:emb AS vector), :model, :now)
        ON CONFLICT (task_id) DO UPDATE
          SET company_id = EXCLUDED.company_id,
              embedding  = EXCLUDED.embedding,
              model      = EXCLUDED.model,
              updated_at = EXCLUDED.updated_at
    """), {
        "tid": task_id,
        "cid": company_id,
        # pgvector принимает строку вида "[0.1, 0.2, ...]"
        "emb": _vec_to_str(embedding),
        "model": model,
        "now": datetime.now(timezone.utc),
    })
    db.session.commit()


def _vec_to_str(v: list[float]) -> str:
    # экономим на парсинге: одна строка вместо list-литерала Python.
    return "[" + ",".join(f"{x:.6f}" for x in v) + "]"


def reindex_task(task_id: int) -> bool:
    """Переиндексирует одну задачу. Возвращает True, если прошло успешно."""
    task = db.session.get(Task, task_id)
    if task is None or task.company_id is None:
        return False
    client = get_ai_client(task.company_id)
    if client is None:
        return False
    text_for_embedding = _build_text_for_task(task)
    if not text_for_embedding.strip():
        return False
    try:
        vec = client.embed(text_for_embedding)[0]
    except Exception as e:
        logger.warning("ai.embed.failed",
                       extra={"task_id": task_id, "err": str(e)})
        return False
    _upsert(task_id, task.company_id, vec, client.model_embedding)
    return True


def reindex_tasks_batch(task_ids: list[int]) -> int:
    """Перегенерация эмбеддингов пачкой. Возвращает число успешных."""
    if not task_ids:
        return 0
    tasks = (Task.query
             .filter(Task.id.in_(task_ids))
             .all())
    # сгруппируем по компании, чтобы каждый клиент дёрнулся 1 раз.
    by_company: dict[int, list[Task]] = {}
    for t in tasks:
        if t.company_id is not None:
            by_company.setdefault(t.company_id, []).append(t)
    ok_total = 0
    for company_id, group in by_company.items():
        client = get_ai_client(company_id)
        if client is None:
            continue
        for chunk_start in range(0, len(group), EMBED_BATCH_SIZE):
            chunk = group[chunk_start:chunk_start + EMBED_BATCH_SIZE]
            texts = [_build_text_for_task(t) for t in chunk]
            try:
                vecs = client.embed(texts)
            except Exception as e:
                logger.warning("ai.embed_batch.failed",
                               extra={"company_id": company_id, "err": str(e)})
                continue
            for t, v in zip(chunk, vecs):
                try:
                    _upsert(t.id, company_id, v, client.model_embedding)
                    ok_total += 1
                except Exception as e:
                    logger.warning("ai.embed_upsert.failed",
                                   extra={"task_id": t.id, "err": str(e)})
                    db.session.rollback()
    return ok_total


# ───────────────────────── async-хук из task_service ──────────────────────

def schedule_reindex(task_id: int) -> None:
    """Поставить переиндексацию в background. Безопасен в любом сервисе:
    тихо игнорирует ошибки и НИКОГДА не валит вызывающий запрос."""
    app = current_app._get_current_object()  # фиксируем контекст для greenlet

    def _job():
        with app.app_context():
            try:
                reindex_task(task_id)
            except Exception as e:
                logger.warning("ai.reindex.async_failed",
                               extra={"task_id": task_id, "err": str(e)})

    try:
        socketio.start_background_task(_job)
    except Exception as e:
        logger.warning("ai.reindex.spawn_failed",
                       extra={"task_id": task_id, "err": str(e)})


# ───────────────────────── семантический поиск ────────────────────────────

def semantic_search(company_id: int, query: str, *,
                    limit: int = SEMANTIC_LIMIT,
                    timeout: float = 4.0) -> list[tuple[int, float]]:
    """Возвращает список (task_id, score) по убыванию релевантности.

    score = 1 - cosine_distance ∈ [-1, 1]. Чем больше — тем релевантнее.
    Фильтр по `model` обязателен: если суперадмин сменил модель в настройках,
    старые эмбеддинги под другую модель не зайдут в выдачу до перегенерации.
    """
    if not query.strip():
        return []
    client = get_ai_client(company_id)
    if client is None:
        return []
    try:
        qvec = client.embed(query, timeout=timeout)[0]
    except Exception as e:
        logger.warning("ai.search.embed_failed",
                       extra={"company_id": company_id, "err": str(e)})
        return []
    rows = db.session.execute(text("""
        SELECT task_id,
               1 - (embedding <=> CAST(:qv AS vector)) AS score
          FROM task_embeddings
         WHERE company_id = :cid
           AND model = :model
         ORDER BY embedding <=> CAST(:qv AS vector)
         LIMIT :lim
    """), {
        "qv": _vec_to_str(qvec),
        "cid": company_id,
        "model": client.model_embedding,
        "lim": limit,
    }).fetchall()
    return [(r.task_id, float(r.score)) for r in rows
            if r.score > MIN_SEMANTIC_SCORE]


def count_embeddings(company_id: int, model: str | None = None) -> int:
    """Сколько задач компании уже проиндексировано. Для UI «нужен бэкфилл?»."""
    sql = "SELECT COUNT(*) AS c FROM task_embeddings WHERE company_id = :cid"
    params: dict = {"cid": company_id}
    if model:
        sql += " AND model = :model"
        params["model"] = model
    row = db.session.execute(text(sql), params).first()
    return int(row.c) if row else 0


# ───────────────────────── бэкфилл ─────────────────────────────────────────

def find_unindexed_task_ids(company_id: int | None = None,
                            model: str | None = None) -> list[int]:
    """Задачи компании, у которых нет эмбеддинга или модель не совпадает.

    Если model=None — берём текущую модель компании из настроек.
    """
    companies: Iterable[Company]
    if company_id is None:
        companies = Company.query.filter_by(ai_enabled=True).all()
    else:
        c = db.session.get(Company, company_id)
        companies = [c] if (c and c.ai_enabled) else []
    result: list[int] = []
    for c in companies:
        target_model = model or c.ai_model_embedding
        rows = db.session.execute(text("""
            SELECT t.id
              FROM tasks t
              LEFT JOIN task_embeddings e ON e.task_id = t.id
             WHERE t.company_id = :cid
               AND (e.task_id IS NULL OR e.model <> :model)
        """), {"cid": c.id, "model": target_model}).fetchall()
        result.extend(int(r.id) for r in rows)
    return result


def run_backfill(company_id: int | None = None) -> dict:
    """Бэкфилл-проход. Возвращает {'total', 'indexed'}."""
    ids = find_unindexed_task_ids(company_id)
    if not ids:
        return {"total": 0, "indexed": 0}
    indexed = 0
    for chunk_start in range(0, len(ids), EMBED_BATCH_SIZE):
        chunk = ids[chunk_start:chunk_start + EMBED_BATCH_SIZE]
        indexed += reindex_tasks_batch(chunk)
    return {"total": len(ids), "indexed": indexed}

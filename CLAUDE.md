# Grove Work v2.0 — Руководство по проекту

## Что это

Grove Work — внутренняя платформа для учёта времени задач и аналитики. Команды с разграниченными ролями ведут задачи и трекают время работы через «юниты» (отрезки времени).

## Стек

| Слой | Технология |
|---|---|
| Бэкенд | Python 3.12 · Flask 3 · SQLAlchemy 2 · Alembic |
| WebSocket | Flask-SocketIO + eventlet + Redis |
| Auth | Flask-JWT-Extended (access 15m / refresh 30d HttpOnly cookie) |
| Валидация | marshmallow |
| БД | PostgreSQL 16 (pgcrypto для паролей) |
| Фронтенд | Vue 3 · Vite · PrimeVue · Pinia · Vue Router 4 |
| Инфра | Docker Compose · Nginx |
| API-документация | flasgger (Swagger UI на /apidocs) |

## Структура директорий

```
back/        — Flask-приложение
front/       — Vue 3 SPA
deploy/      — docker-compose.yml, nginx, init_sql, .env.example
```

## Архитектура бэкенда

```
Routes (Blueprints) → Services (бизнес-логика) → Repositories (SQL) → PostgreSQL
```

**Жёсткое правило:** `request`, `g`, `response` из Flask не проникают глубже Routes.

Папки `back/app/`:
- `models/` — SQLAlchemy ORM-модели (1 файл = 1 таблица)
- `schemas/` — marshmallow-схемы (валидация + сериализация)
- `repositories/` — SQL-запросы через SQLAlchemy. Только I/O
- `services/` — бизнес-логика. Чистые функции, без Flask-контекста
- `api/` — Flask Blueprints, декоратор `@require_permission`
- `sockets/` — Flask-SocketIO события
- `utils/` — permissions, avatar, logger

## Система прав (BIGINT, 64 бита, 8 секций по 8 бит)

| Byte | Section | Биты |
|---|---|---|
| 0 | TASKS | view=0, own_create=1, own_edit=2, own_delete=3, other_create=4, other_edit=5, other_delete=6 |
| 1 | UNITS | аналогично TASKS |
| 2 | USERS | view=0, create=1, edit=2, delete=3 |
| 3 | ROLES | view=0, create=1, edit=2, delete=3, assign=4 |
| 4 | STATS | view=0, view_users=1, export_common=2, export_users=3 |
| 5 | BACKUP | view=0, export=1, import=2 |
| 6 | DEPARTMENTS | view=0, create=1, edit=2, delete=3 |
| 7 | UNIT_TYPES | view=0, create=1, edit=2, delete=3 |

Хелпер: `app/utils/permissions.py` — `has_permission(access, section, bit)`, `@require_permission(section, bit)`

## Ключевые бизнес-правила

- Пользователь с `is_default_pass=TRUE` получает `force_change: true` в JWT — все API кроме `/api/auth/change-default` возвращают 403
- У пользователя единовременно только 1 активный юнит (`datetime_end IS NULL`)
- Нельзя архивировать задачу с активным юнитом
- Единственная всесильная роль (`access == BIGINT_MAX`) защищена от изменения/удаления
- Единственный носитель всесильной роли защищён от скрытия и смены роли
- `avatar_path = NULL` → identicon по `user.id` (GitHub-style 5×5)
- Удаление типа юнита каскадно удаляет все юниты с этим типом

## Auth flow

- `POST /api/auth/login` → access token (Bearer) + refresh token (HttpOnly cookie)
- Access token TTL: 15 минут
- Refresh token TTL: 30 дней, `SameSite=Strict`
- `POST /api/auth/refresh` — обновить access по cookie

## WebSocket

Клиент передаёт access token в query param при handshake. Сервер присоединяет к комнатам `all` и `user_{id}`. Все мутации (задачи, юниты) излучают события в комнату `all`.

## Запуск для разработки

```bash
cd back
pip install -r requirements.txt
flask db upgrade        # применить миграции
flask run --debug
```

## Swagger UI

Доступен на `http://localhost:5000/apidocs` при запущенном бэкенде.

## Логи

JSON-формат в stdout. Docker забирает через `docker logs`.  
`FLASK_DEBUG=1` включает DEBUG-уровень с SQL-запросами.

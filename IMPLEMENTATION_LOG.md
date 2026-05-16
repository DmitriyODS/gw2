# Implementation Log — Grove Work v2.0

> Краткий журнал того, что уже реализовано. Обновляется после каждой сессии.

---

## Сессия 1 — 2026-05-16

### Инфраструктура
- Инициализирован git-репозиторий
- Созданы директории: `back/`, `front/`, `deploy/`
- Написан `CLAUDE.md` с описанием проекта, стека и правил

### Deploy
- `deploy/docker-compose.yml` — сервисы: db (postgres:16), redis:7, app (Flask), nginx
- `deploy/.env.example` — переменные окружения
- `deploy/nginx/nginx.conf` — SPA fallback, проксирование /api/ и /socket.io/, раздача /uploads/
- `deploy/init_sql/01_init.sql` — pgcrypto extension, роль Администратор (access=BIGINT_MAX), пользователь admin/admin

### Бэкенд — полная реализация
**Ядро:**
- `back/requirements.txt` — все зависимости
- `back/Dockerfile` — multi-stage build на python:3.12-slim
- `back/app/config.py` — Config / DevelopmentConfig / ProductionConfig
- `back/app/extensions.py` — db, jwt, socketio, limiter, migrate (глобальные экземпляры)
- `back/app/__init__.py` — Application Factory `create_app()`

**Utils:**
- `back/app/utils/logger.py` — JSONFormatter, `get_logger(name)`
- `back/app/utils/permissions.py` — Section, Bit, `has_permission()`, `@require_permission()`
- `back/app/utils/avatar.py` — генерация identicon (pydenticon, 5×5 GitHub-style), сохранение/удаление аватарки

**Модели (SQLAlchemy 2):**
- `Role` — id, name, access (BIGINT)
- `User` — id, fio, login, hash_password, post, role_id, avatar_path, is_default_pass, is_hidden, created_at
- `Department` — id, name
- `Task` — id, name, author_id, department_id, link_yougile, received_at, deadline, is_archived, archived_at
- `Favorite` — (task_id, user_id) compound PK
- `UnitType` — id, name
- `Unit` — id, name, user_id, unit_type_id, task_id, is_edited, datetime_start, datetime_end

**Схемы (marshmallow):**
- `RoleSchema`, `RoleCreateSchema`, `RoleUpdateSchema`
- `UserSchema`, `UserCreateSchema`, `UserUpdateSchema`, `UserMeUpdateSchema`, `ChangeDefaultSchema`
- `TaskSchema`, `TaskCreateSchema`, `TaskUpdateSchema`
- `UnitSchema`, `UnitCreateSchema`, `UnitUpdateSchema`
- `DepartmentSchema`, `DepartmentCreateSchema`, `UnitTypeSchema`, `UnitTypeCreateSchema`
- `StatsCommonSchema`, `StatsExtendedSchema`, `StatsProfileSchema`

**Репозитории (только SQL, без бизнес-логики):**
- `role_repo` — get_all, get_by_id, create, update, delete, count_almighty
- `user_repo` — get_all, get_by_id, get_by_login, create, update, count_almighty_holders
- `department_repo` — CRUD
- `task_repo` — get_list (с фильтрами/сортировкой/пагинацией), get_by_id, create, update, delete
- `unit_repo` — get_by_task, get_active_for_user, create, update, delete, stop
- `stats_repo` — common metrics, extended metrics, profile metrics

**Сервисы (бизнес-логика):**
- `auth_service` — login, refresh, logout, change_default_credentials
- `user_service` — create, update, hide, update_me, upload_avatar, delete_avatar, assign_role
- `task_service` — create, update, delete, archive, restore, toggle_favorite
- `unit_service` — create, update, delete, stop
- `stats_service` — get_common, get_extended, get_profile, export_common_xlsx, export_extended_xlsx
- `backup_service` — export_zip, import_zip

**API Blueprints (Flask + flasgger Swagger):**
- `POST/POST /api/auth/*` — login, refresh, logout, change-default
- `GET/POST/PATCH/DELETE /api/users/*` — CRUD + me + avatar + identicon
- `GET/POST/PATCH/DELETE /api/roles/*`
- `GET/POST/PATCH/DELETE /api/tasks/*` — с фильтрами, archive, restore, favorite
- `GET/POST/PATCH/DELETE /api/tasks/:id/units` и `/api/units/*`
- `GET/POST/PATCH/DELETE /api/departments/*`
- `GET/POST/PATCH/DELETE /api/unit-types/*`
- `GET /api/stats/*` — common, extended, profile, exports
- `GET/POST /api/backup/*` — export/import ZIP

**WebSocket:**
- `sockets/events.py` — connect/disconnect с JWT-верификацией, join rooms `all` + `user_{id}`
- События: task:created/updated/archived/restored/deleted, unit:started/stopped/updated/deleted/force_stopped

**Alembic:**
- Настроен Flask-Migrate (обёртка над Alembic)
- Начальная миграция для всех таблиц

---

## Что ещё не реализовано

- Фронтенд (Vue 3 + PrimeVue) — следующая сессия

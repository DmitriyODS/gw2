# Grove Work v2.0 — Техническое задание

> **Версия:** 2.0  
> **Статус:** Final Draft  
> **Стек:** Python 3.12 · Flask 3 · PostgreSQL 16 · Vue 3 + PrimeVue · Socket.IO · Docker Compose

---

## Содержание

1. [Общее описание](#1-общее-описание)
2. [Стек технологий](#2-стек-технологий)
3. [Архитектура системы](#3-архитектура-системы)
4. [Docker Compose](#4-docker-compose)
5. [База данных](#5-база-данных)
6. [Система прав доступа](#6-система-прав-доступа)
7. [Бэкенд — структура проекта](#7-бэкенд--структура-проекта)
8. [Авторизация и сессии](#8-авторизация-и-сессии)
9. [REST API](#9-rest-api)
10. [WebSocket события](#10-websocket-события)
11. [Разделы системы — подробно](#11-разделы-системы--подробно)
12. [Фронтенд](#12-фронтенд)
13. [Логирование](#13-логирование)
14. [Безопасность](#14-безопасность)

---

## 1. Общее описание

**Grove Work** — внутренняя платформа для учёта времени выполнения задач, отслеживания количества задач и просмотра аналитики. Система ориентирована на команды с разграниченными ролями доступа.

### Ключевые концепции

| Сущность | Суть |
|---|---|
| **Задача** | Основная единица работы; поступает от отдела-заказчика |
| **Юнит** | Отрезок времени, потраченный конкретным сотрудником на задачу |
| **Роль** | Набор битовых разрешений, определяющий доступные действия |
| **Отдел** | Заказчик задачи (справочник) |
| **Тип юнита** | Категория работы (справочник) |

### Принципы разработки

- **Простота > сложность** — сложные абстракции только там, где они реально нужны
- **Читаемость и масштабируемость** — приоритет первый
- **Производительность** — запросы с индексами, без N+1
- **Предсказуемость** — чёткие контракты на каждом слое; никаких магических side-эффектов
- **Защищённость** — валидация на сервере, проверка прав на каждый endpoint
- **Логирование** — структурированные JSON-логи на всех критичных операциях
- **Общие контракты** — все публичные функции сервисов имеют явные типы и docstring

---

## 2. Стек технологий

### Бэкенд

| Компонент | Технология | Примечание |
|---|---|---|
| Язык | Python 3.12 | |
| Web-фреймворк | Flask 3.x | |
| ORM | SQLAlchemy 2.x | psycopg2 как драйвер |
| Миграции | Alembic | |
| WebSocket | Flask-SocketIO + eventlet | |
| JWT | Flask-JWT-Extended | Access + Refresh tokens |
| Хеширование паролей | pgcrypto | `crypt(password, gen_salt('bf'))` на уровне БД |
| Валидация | marshmallow | Схемы для всех входящих данных |
| Экспорт XLSX | openpyxl | |
| Архивация | zipfile (stdlib) | Резервное копирование |
| Identicon аватарки | py-identicon / pydenticon | GitHub-style, генерация по user.id |

### Фронтенд

| Компонент | Технология | Примечание |
|---|---|---|
| Фреймворк | Vue 3 (Composition API) | |
| Сборщик | Vite | |
| UI-библиотека | PrimeVue (последняя версия) | |
| Стейт | Pinia | |
| Роутинг | Vue Router 4 | |
| Шрифт | Roboto (Google Fonts) | 300, 400, 500, 700 |
| Иконки | Google Material Symbols Outlined | |
| Реальное время | Socket.IO client | |
| Кроппер аватарки | vue-cropper или native Canvas API | Crop + center на клиенте |
| HTTP-клиент | fetch (native) | Обёртка в api/ модуль |
| Темы | localStorage | Только на устройстве пользователя |

### Инфраструктура

| Компонент | Технология |
|---|---|
| Контейнеризация | Docker Compose |
| БД | PostgreSQL 16 |
| Message queue (SocketIO) | Redis 7 |
| Reverse proxy | Nginx |

---

## 3. Архитектура системы

```
┌───────────────────────────────────────────────────────────┐
│                       Docker Compose                       │
│                                                           │
│  ┌──────────┐    ┌──────────────┐    ┌──────────────┐    │
│  │  Nginx   │───▶│  Flask App   │───▶│  PostgreSQL  │    │
│  │ :80      │    │  :5000       │    │  :5432       │    │
│  └──────────┘    └──────┬───────┘    └──────────────┘    │
│       │                 │                                  │
│  /dist (Vue SPA)        ▼                                  │
│  /uploads          ┌──────────────┐                       │
│                    │    Redis     │                        │
│                    │  :6379       │                        │
│                    └──────────────┘                        │
└───────────────────────────────────────────────────────────┘
```

### Слои приложения (бэкенд)

```
HTTP/WS Request
      │
      ▼
┌─────────────┐
│  Nginx      │  ← Статика (Vue SPA), проксирование /api/* и /socket.io/*
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Routes     │  ← Flask Blueprints, декоратор @require_permission
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Services   │  ← Бизнес-логика. Чистые функции. Не знают о Flask/HTTP
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Repositories│  ← SQL через SQLAlchemy. Только I/O. Никакой бизнес-логики
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PostgreSQL │
└─────────────┘
```

**Жёсткое правило:** `request`, `response`, `g` из Flask не проникают глубже Routes. Services тестируются без HTTP-контекста.

---

## 4. Docker Compose

### `docker-compose.yml`

```yaml
version: "3.9"

services:
  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pg_data:/var/lib/postgresql/data
      - ./init_sql:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    volumes:
      - redis_data:/data

  app:
    build: ./backend
    restart: unless-stopped
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started
    environment:
      DATABASE_URL: postgresql://${DB_USER}:${DB_PASSWORD}@db:5432/${DB_NAME}
      REDIS_URL: redis://redis:6379/0
      JWT_SECRET_KEY: ${JWT_SECRET_KEY}
      UPLOAD_FOLDER: /app/uploads
    volumes:
      - uploads:/app/uploads
    expose:
      - "5000"

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    depends_on:
      - app
    ports:
      - "80:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/conf.d/default.conf
      - ./frontend/dist:/usr/share/nginx/html
      - uploads:/app/uploads:ro

volumes:
  pg_data:
  redis_data:
  uploads:
```

### `.env` (пример)

```env
DB_NAME=grovework
DB_USER=grovework
DB_PASSWORD=supersecret
JWT_SECRET_KEY=very-long-random-secret-min-32-chars
```

### `nginx/nginx.conf`

```nginx
server {
    listen 80;

    # Vue SPA
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    # Flask API
    location /api/ {
        proxy_pass http://app:5000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Socket.IO (WebSocket upgrade)
    location /socket.io/ {
        proxy_pass http://app:5000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # Аватарки
    location /uploads/ {
        alias /app/uploads/;
        expires 7d;
        add_header Cache-Control "public";
    }
}
```

---

## 5. База данных

### Расширение pgcrypto

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

---

### Схема таблиц

#### `roles`

```sql
CREATE TABLE roles (
    id      SERIAL PRIMARY KEY,
    name    VARCHAR(100) NOT NULL UNIQUE,
    access  BIGINT NOT NULL DEFAULT 0
    -- BIGINT = 8 байт = 8 секций × 8 бит
);
```

---

#### `users`

```sql
CREATE TABLE users (
    id               SERIAL PRIMARY KEY,
    fio              VARCHAR(255) NOT NULL,
    login            VARCHAR(100) NOT NULL UNIQUE,
    hash_password    TEXT NOT NULL,         -- crypt(password, gen_salt('bf'))
    post             VARCHAR(255),
    role_id          INTEGER NOT NULL REFERENCES roles(id),
    avatar_path      VARCHAR(500),          -- относительный путь в /uploads/avatars/
    is_default_pass  BOOLEAN NOT NULL DEFAULT TRUE,
    is_hidden        BOOLEAN NOT NULL DEFAULT FALSE,  -- soft delete
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_login   ON users(login);
CREATE INDEX idx_users_role    ON users(role_id);
CREATE INDEX idx_users_visible ON users(is_hidden) WHERE is_hidden = FALSE;
```

**Пояснения:**
- `avatar_path` — NULL означает «использовать identicon». Хранится относительный путь (например `avatars/uuid.jpg`), раздаётся через Nginx `/uploads/`.
- `is_hidden = TRUE` — мягкое удаление. Данные пользователя (задачи, юниты) сохраняются, но пользователь не отображается в системе.
- `is_default_pass = TRUE` — блокирует все действия до смены пароля.

---

#### `departments` (Отделы)

```sql
CREATE TABLE departments (
    id    SERIAL PRIMARY KEY,
    name  VARCHAR(255) NOT NULL UNIQUE
);
```

---

#### `tasks` (Задачи)

```sql
CREATE TABLE tasks (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(500) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    author_id     INTEGER NOT NULL REFERENCES users(id),
    link_yougile  TEXT,
    received_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    department_id INTEGER NOT NULL REFERENCES departments(id),
    deadline      TIMESTAMPTZ,
    is_archived   BOOLEAN NOT NULL DEFAULT FALSE,
    archived_at   TIMESTAMPTZ
);

CREATE INDEX idx_tasks_author     ON tasks(author_id);
CREATE INDEX idx_tasks_dept       ON tasks(department_id);
CREATE INDEX idx_tasks_archived   ON tasks(is_archived);
CREATE INDEX idx_tasks_received   ON tasks(received_at);
CREATE INDEX idx_tasks_archived_at ON tasks(archived_at) WHERE is_archived = TRUE;
```

**Пояснения:**
- `archived_at` — устанавливается автоматически при архивировании задачи. При разархивировании — `is_archived = FALSE`, `archived_at = NULL`.
- `author_id` — выставляется при создании по текущему пользователю, изменить нельзя.
- `link_yougile` — необязательное поле, URL.
- `received_at` — по умолчанию = `created_at`, но доступно для ручного изменения.

---

#### `favorites` (Избранное)

```sql
CREATE TABLE favorites (
    task_id  INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, user_id)
);
```

---

#### `unit_types` (Типы юнитов)

```sql
CREATE TABLE unit_types (
    id    SERIAL PRIMARY KEY,
    name  VARCHAR(255) NOT NULL UNIQUE
);
```

> ⚠️ При удалении типа юнита каскадно удаляются все юниты с этим типом.

---

#### `units` (Юниты)

```sql
CREATE TABLE units (
    id             SERIAL PRIMARY KEY,
    name           VARCHAR(500) NOT NULL,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    unit_type_id   INTEGER NOT NULL REFERENCES unit_types(id) ON DELETE CASCADE,
    task_id        INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    is_edited      BOOLEAN NOT NULL DEFAULT FALSE,
    datetime_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    datetime_end   TIMESTAMPTZ,  -- NULL = юнит активен
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_units_user   ON units(user_id);
CREATE INDEX idx_units_task   ON units(task_id);
CREATE INDEX idx_units_active ON units(user_id) WHERE datetime_end IS NULL;
```

**Пояснения:**
- `datetime_end = NULL` — юнит активен.
- `is_edited = TRUE` — юнит был отредактирован вручную. Это меняет его визуальный стиль в UI.
- Цвет левой полоски юнита в UI определяется статусом: активен (зелёный) или завершён (нейтральный), **не типом**.

---

### Init SQL (`init_sql/01_init.sql`)

```sql
-- Расширение
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Роль с полным доступом (все биты = 1)
INSERT INTO roles (name, access)
VALUES ('Администратор', 9223372036854775807);

-- Первый пользователь: логин admin / пароль admin
INSERT INTO users (fio, login, hash_password, role_id, is_default_pass)
VALUES (
    'Администратор',
    'admin',
    crypt('admin', gen_salt('bf')),
    1,
    TRUE
);
```

---

## 6. Система прав доступа

### Структура поля `access` (BIGINT, 64 бита)

Поле разбивается на 8 секций по 1 байту (8 бит). Каждый бит — конкретное разрешение.

| Байт (секция) | Секция | Биты 0–7 |
|---|---|---|
| 0 | **Задачи** | 0: просмотр, 1: создание своих, 2: редактирование своих, 3: удаление своих, 4: создание чужих, 5: редактирование чужих, 6: удаление чужих, 7: — |
| 1 | **Юниты** | 0: просмотр, 1: создание своих, 2: редактирование своих, 3: удаление своих, 4: создание чужих, 5: редактирование чужих, 6: удаление чужих, 7: — |
| 2 | **Пользователи** | 0: просмотр, 1: создание, 2: изменение, 3: удаление, 4–7: — |
| 3 | **Роли** | 0: просмотр, 1: создание, 2: изменение, 3: удаление, 4: назначение, 5–7: — |
| 4 | **Статистика** | 0: просмотр, 1: просмотр стат. пользователей, 2: выгрузка общей, 3: выгрузка пользователей, 4–7: — |
| 5 | **Копирование** | 0: просмотр, 1: выгрузка (backup), 2: загрузка (restore), 3–7: — |
| 6 | **Отделы** | 0: просмотр, 1: создание, 2: изменение, 3: удаление, 4–7: — |
| 7 | **Типы юнитов** | 0: просмотр, 1: создание, 2: изменение, 3: удаление, 4–7: — |

### Python-хелпер

```python
# app/utils/permissions.py

class Section:
    TASKS       = 0
    UNITS       = 1
    USERS       = 2
    ROLES       = 3
    STATS       = 4
    BACKUP      = 5
    DEPARTMENTS = 6
    UNIT_TYPES  = 7

class Bit:
    # Задачи / Юниты
    VIEW         = 0
    OWN_CREATE   = 1
    OWN_EDIT     = 2
    OWN_DELETE   = 3
    OTHER_CREATE = 4
    OTHER_EDIT   = 5
    OTHER_DELETE = 6
    # Пользователи / Роли / Отделы / Типы юнитов
    CREATE       = 1
    EDIT         = 2
    DELETE       = 3
    # Роли
    ASSIGN       = 4
    # Статистика
    VIEW_USERS   = 1
    EXPORT_COMMON = 2
    EXPORT_USERS  = 3
    # Копирование
    EXPORT       = 1
    IMPORT       = 2


def has_permission(access: int, section: int, bit: int) -> bool:
    """Проверить наличие разрешения у пользователя."""
    byte_val = (access >> (section * 8)) & 0xFF
    return bool(byte_val & (1 << bit))


def require_permission(section: int, bit: int):
    """Декоратор для Flask route — проверяет права, возвращает 403 при отсутствии."""
    from functools import wraps
    from flask import abort
    from flask_jwt_extended import get_jwt_identity
    from app.repositories.user_repo import get_user_by_id

    def decorator(fn):
        @wraps(fn)
        def wrapper(*args, **kwargs):
            user_id = get_jwt_identity()
            user = get_user_by_id(user_id)
            if user is None or not has_permission(user.role.access, section, bit):
                abort(403, description="Недостаточно прав")
            return fn(*args, **kwargs)
        return wrapper
    return decorator
```

### Защитные правила на уровне сервиса

Проверяются в `services/`, до записи в БД:

1. **Единственная всесильная роль** — нельзя изменить или удалить роль, если она единственная с полным доступом (`access == BIGINT_MAX`). Возвращать 422 с описанием причины.
2. **Единственный всесильный пользователь** — нельзя изменить роль или скрыть пользователя, если он единственный носитель всесильной роли. Возвращать 422.
3. **Пользователь не может удалить (скрыть) сам себя** и не может изменить свою роль.
4. **Пользователь может редактировать у себя только:** ФИО, аватарку, логин, пароль.

---

## 7. Бэкенд — структура проекта

```
backend/
├── Dockerfile
├── requirements.txt
├── app/
│   ├── __init__.py          # create_app() — Application Factory
│   ├── config.py            # Config, DevelopmentConfig, ProductionConfig
│   ├── extensions.py        # db, jwt, socketio, redis — глобальные экземпляры
│   │
│   ├── models/              # SQLAlchemy-модели (1 файл = 1 таблица)
│   │   ├── __init__.py
│   │   ├── role.py
│   │   ├── user.py
│   │   ├── department.py
│   │   ├── task.py
│   │   ├── favorite.py
│   │   ├── unit_type.py
│   │   └── unit.py
│   │
│   ├── schemas/             # marshmallow — валидация входящих + сериализация исходящих
│   │   ├── role.py
│   │   ├── user.py
│   │   ├── task.py
│   │   ├── unit.py
│   │   └── stats.py
│   │
│   ├── repositories/        # Только SQL-запросы. Никакой бизнес-логики
│   │   ├── role_repo.py
│   │   ├── user_repo.py
│   │   ├── task_repo.py
│   │   ├── unit_repo.py
│   │   ├── stats_repo.py
│   │   └── department_repo.py
│   │
│   ├── services/            # Бизнес-логика. Без Flask-контекста
│   │   ├── auth_service.py
│   │   ├── task_service.py
│   │   ├── unit_service.py
│   │   ├── stats_service.py
│   │   ├── user_service.py
│   │   └── backup_service.py
│   │
│   ├── api/                 # Flask Blueprints
│   │   ├── __init__.py      # register_blueprints()
│   │   ├── auth.py          # /api/auth/*
│   │   ├── tasks.py         # /api/tasks/*
│   │   ├── units.py         # /api/units/*
│   │   ├── users.py         # /api/users/*
│   │   ├── roles.py         # /api/roles/*
│   │   ├── stats.py         # /api/stats/*
│   │   ├── departments.py   # /api/departments/*
│   │   ├── unit_types.py    # /api/unit-types/*
│   │   └── backup.py        # /api/backup/*
│   │
│   ├── sockets/             # Flask-SocketIO обработчики
│   │   ├── __init__.py
│   │   └── events.py        # Все WS события
│   │
│   └── utils/
│       ├── permissions.py   # Section, Bit, has_permission, require_permission
│       ├── avatar.py        # Генерация identicon + сохранение файла аватарки
│       └── logger.py        # Настройка JSON-логгера
│
├── migrations/              # Alembic
│   └── versions/
└── init_sql/
    └── 01_init.sql
```

### Application Factory (`app/__init__.py`)

```python
def create_app(config_name: str = "production") -> Flask:
    app = Flask(__name__)
    app.config.from_object(config[config_name])

    db.init_app(app)
    jwt.init_app(app)
    socketio.init_app(app, message_queue=app.config["REDIS_URL"], cors_allowed_origins="*")

    from app.api import register_blueprints
    register_blueprints(app)

    from app.sockets.events import register_events
    register_events(socketio)

    return app
```

---

## 8. Авторизация и сессии

### JWT-схема

- **Access token** — время жизни: 15 минут. Передаётся в заголовке `Authorization: Bearer <token>`.
- **Refresh token** — время жизни: 30 дней. Хранится в HttpOnly cookie (`SameSite=Strict`).

### Эндпоинты авторизации

```
POST /api/auth/login          # Получить access + refresh
POST /api/auth/refresh        # Обновить access по refresh cookie
POST /api/auth/logout         # Очистить refresh cookie
POST /api/auth/change-default # Сменить логин/пароль при первом входе
```

### Логика первого входа (`is_default_pass = TRUE`)

1. Пользователь вводит `admin / admin` → сервер возвращает `200 OK` + токены, но в payload JWT добавляется флаг `"force_change": true`.
2. Фронт обнаруживает флаг → показывает **неклозабельное** модальное окно смены логина/пароля.
3. Все API-запросы кроме `POST /api/auth/change-default` возвращают `403` с кодом `FORCE_PASSWORD_CHANGE`, пока флаг активен.
4. После успешной смены — `is_default_pass = FALSE`, флаг убирается из JWT при следующем рефреше.

### Смена пароля (валидация)

- Новый логин: уникален, минимум 3 символа.
- Новый пароль: минимум 8 символов.
- Подтверждение пароля: совпадает с новым.

---

## 9. REST API

### Общие правила

- Все эндпоинты под `/api/`.
- Все ответы в JSON.
- Успех: `200`, `201`. Ошибка: `400` (валидация), `401` (не авторизован), `403` (нет прав), `404` (не найден), `409` (конфликт), `422` (бизнес-правило нарушено).
- Структура ошибки: `{ "error": "код", "message": "Описание для пользователя" }`.
- Пагинация через query-параметры `?page=1&per_page=50` где применимо.

---

### 9.1 Авторизация `/api/auth`

| Метод | Путь | Auth | Описание |
|---|---|---|---|
| POST | `/api/auth/login` | — | Вход. Body: `{login, password}` |
| POST | `/api/auth/refresh` | refresh cookie | Обновить access token |
| POST | `/api/auth/logout` | access | Выйти (очистить cookie) |
| POST | `/api/auth/change-default` | access | Сменить логин/пароль при первом входе |

---

### 9.2 Пользователи `/api/users`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/users` | USERS.VIEW | Список видимых пользователей |
| POST | `/api/users` | USERS.CREATE | Создать пользователя |
| GET | `/api/users/:id` | USERS.VIEW | Получить пользователя |
| PATCH | `/api/users/:id` | USERS.EDIT | Редактировать пользователя |
| DELETE | `/api/users/:id` | USERS.DELETE | Скрыть пользователя (soft delete) |
| GET | `/api/users/me` | — | Текущий пользователь |
| PATCH | `/api/users/me` | — | Редактировать свой профиль |
| POST | `/api/users/me/avatar` | — | Загрузить аватарку (multipart/form-data) |
| DELETE | `/api/users/me/avatar` | — | Удалить аватарку (вернуть identicon) |
| PATCH | `/api/users/:id/role` | ROLES.ASSIGN | Назначить роль пользователю |

**Правила:**
- При создании пользователя — пароль по умолчанию `admin`, `is_default_pass = TRUE`.
- Скрытый пользователь (`is_hidden = TRUE`) не возвращается в списках, но его данные (задачи, юниты) сохраняются.
- Текущий пользователь не может скрыть сам себя или изменить свою роль.
- Единственный носитель всесильной роли защищён от скрытия.

**Аватарка:**
- Допустимые форматы: `image/jpeg`, `image/png`.
- Максимальный размер: **2 МБ** (сжатие происходит на **клиенте** до отправки).
- Файл сохраняется в `/uploads/avatars/{uuid}.{ext}`.
- При отсутствии аватарки (`avatar_path = NULL`) — сервер возвращает URL `identicon` (`/api/users/:id/identicon`).

**Identicon:**
- Генерируется детерминированно по `user.id` (GitHub-style: 5×5 симметричная сетка).
- Возвращается как PNG через отдельный эндпоинт `/api/users/:id/identicon`.
- Кешируется на стороне Nginx.

---

### 9.3 Роли `/api/roles`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/roles` | ROLES.VIEW | Список ролей |
| POST | `/api/roles` | ROLES.CREATE | Создать роль |
| PATCH | `/api/roles/:id` | ROLES.EDIT | Изменить роль |
| DELETE | `/api/roles/:id` | ROLES.DELETE | Удалить роль |

**Правила:**
- Нельзя удалить или изменить единственную всесильную роль.
- Нельзя создать роль с `access = 0` (бесполезно, но не запрещено — это бизнес-решение оставить разработчику).

---

### 9.4 Задачи `/api/tasks`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/tasks` | TASKS.VIEW | Список задач с фильтрами |
| POST | `/api/tasks` | TASKS.OWN_CREATE | Создать задачу |
| GET | `/api/tasks/:id` | TASKS.VIEW | Получить задачу |
| PATCH | `/api/tasks/:id` | TASKS.OWN_EDIT / OTHER_EDIT | Редактировать задачу |
| DELETE | `/api/tasks/:id` | TASKS.OWN_DELETE / OTHER_DELETE | Удалить задачу |
| POST | `/api/tasks/:id/archive` | TASKS.OWN_EDIT / OTHER_EDIT | Архивировать задачу |
| POST | `/api/tasks/:id/restore` | TASKS.OWN_EDIT / OTHER_EDIT | Восстановить из архива |
| POST | `/api/tasks/:id/favorite` | — | Добавить/убрать из избранного (toggle) |

**Query-параметры для GET `/api/tasks`:**

| Параметр | Тип | Описание |
|---|---|---|
| `tab` | `active\|favorites\|archive` | Вкладка |
| `search` | string | Поиск по названию (ILIKE) |
| `sort` | `last_activity\|created_at\|deadline` | Сортировка |
| `dept_id` | int | Фильтр по отделу |
| `received_from` | date | Период поступления — начало |
| `received_to` | date | Период поступления — конец |
| `has_units` | `none\|mine` | Фильтр: нет юнитов / я работал |
| `page` | int | Страница |
| `per_page` | int | Записей на странице (default 30) |

**Бизнес-правила:**
- `author_id` выставляется автоматически, изменить нельзя.
- `received_at` по умолчанию = `NOW()`, можно изменить.
- Нельзя архивировать задачу, у которой есть активный юнит.
- При восстановлении из архива: `is_archived = FALSE`, `archived_at = NULL`.
- Права OWN/OTHER проверяются: если `author_id == current_user_id` — нужно OWN, иначе OTHER.

---

### 9.5 Юниты `/api/tasks/:task_id/units` и `/api/units`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/tasks/:task_id/units` | UNITS.VIEW | Юниты задачи |
| POST | `/api/tasks/:task_id/units` | UNITS.OWN_CREATE | Создать юнит |
| GET | `/api/units/active` | — | Активный юнит текущего пользователя |
| PATCH | `/api/units/:id` | UNITS.OWN_EDIT / OTHER_EDIT | Редактировать юнит |
| DELETE | `/api/units/:id` | UNITS.OWN_DELETE / OTHER_DELETE | Удалить юнит |
| POST | `/api/units/:id/stop` | UNITS.OWN_EDIT / OTHER_EDIT | Завершить юнит |

**Бизнес-правила:**
- У пользователя единовременно может быть **только 1 активный юнит** (`datetime_end IS NULL`).
- При попытке создать юнит при наличии активного — `409 Conflict`.
- Юниты нельзя создавать для архивных задач — `422`.
- При редактировании (`PATCH`) — `is_edited = TRUE` автоматически.
- Редактируемые поля: `name`, `unit_type_id`, `datetime_start`, `datetime_end`.
- Остановить юнит другого пользователя можно только при наличии `UNITS.OTHER_EDIT`.
- При остановке: `datetime_end = NOW()`.

---

### 9.6 Отделы `/api/departments`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/departments` | DEPARTMENTS.VIEW | Список |
| POST | `/api/departments` | DEPARTMENTS.CREATE | Создать |
| PATCH | `/api/departments/:id` | DEPARTMENTS.EDIT | Изменить |
| DELETE | `/api/departments/:id` | DEPARTMENTS.DELETE | Удалить |

---

### 9.7 Типы юнитов `/api/unit-types`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/unit-types` | UNIT_TYPES.VIEW | Список |
| POST | `/api/unit-types` | UNIT_TYPES.CREATE | Создать |
| PATCH | `/api/unit-types/:id` | UNIT_TYPES.EDIT | Изменить |
| DELETE | `/api/unit-types/:id` | UNIT_TYPES.DELETE | Удалить (каскадно удаляет все юниты!) |

> ⚠️ При удалении типа юнита — все юниты с этим типом удаляются. Фронт обязан показать явное предупреждение с перечислением последствий.

---

### 9.8 Статистика `/api/stats`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/stats/common` | STATS.VIEW | Общая статистика |
| GET | `/api/stats/extended` | STATS.VIEW | Расширенная статистика |
| GET | `/api/stats/common/export` | STATS.EXPORT_COMMON | Выгрузка общей в XLSX |
| GET | `/api/stats/extended/export` | STATS.EXPORT_USERS | Выгрузка расширенной в XLSX |
| GET | `/api/stats/profile` | — | Личная статистика текущего пользователя |

**Query-параметры:** `?from=YYYY-MM-DD&to=YYYY-MM-DD`

#### Определения метрик (общая статистика)

| Метрика | Формула |
|---|---|
| **Долг** | Незакрытые задачи с `received_at < period_start` |
| **Поступило** | Задачи с `received_at` в диапазоне `[period_start, period_end]` |
| **Закрыто** | Задачи с `archived_at` в диапазоне `[period_start, period_end]` |
| **Осталось** | Незакрытые задачи с `received_at <= period_end` (от начала времён) |

#### Формат ответа `/api/stats/common`

```json
{
  "period": { "from": "2025-05-12", "to": "2026-05-12" },
  "tasks": { "debt": 65, "received": 40, "closed": 25, "remaining": 80 },
  "tasks_by_hours": [
    { "task_id": 1, "name": "...", "total_hours": 12.5 }
  ],
  "tasks_by_employees": [
    { "user_id": 1, "fio": "...", "tasks_count": 5, "total_hours": 40.0 }
  ]
}
```

#### Формат ответа `/api/stats/extended`

```json
{
  "by_unit_types": [
    { "type_id": 1, "name": "Дизайн", "total_hours": 100.0, "tasks_count": 15 }
  ],
  "by_departments": [
    { "dept_id": 1, "name": "...", "tasks_count": 30 }
  ],
  "by_unit_types_per_user": [
    {
      "user_id": 1, "fio": "...",
      "unit_types": [
        { "type_id": 1, "name": "Дизайн", "hours": 20.0, "tasks_count": 5 }
      ]
    }
  ],
  "calendar": [
    { "date": "2026-05-12", "received": 3, "closed": 1, "total_hours": 6.5 }
  ]
}
```

**Примечание по подсчёту:** Если у одного пользователя в одной задаче несколько юнитов одного типа — задача считается одной (GROUP BY `task_id, user_id, unit_type_id` → DISTINCT).

#### Формат ответа `/api/stats/profile`

```json
{
  "period": { "from": "...", "to": "..." },
  "total_hours": 120.5,
  "tasks_count": 18,
  "by_unit_types": [
    { "type_id": 1, "name": "Дизайн", "hours": 60.0, "tasks_count": 10 }
  ]
}
```

---

### 9.9 Резервное копирование `/api/backup`

| Метод | Путь | Права | Описание |
|---|---|---|---|
| GET | `/api/backup/export` | BACKUP.EXPORT | Скачать ZIP-архив |
| POST | `/api/backup/import` | BACKUP.IMPORT | Загрузить ZIP-архив (восстановление) |

**ZIP-архив содержит:**
- `data.json` — полный дамп всех таблиц в JSON (в формате, пригодном для восстановления)
- `avatars/` — файлы аватарок по их именам

> ⚠️ Импорт ZIP — **деструктивная операция**: полная замена данных. На фронте требуется двойное подтверждение с явным предупреждением.

---

## 10. WebSocket события

Используется `Flask-SocketIO` с Redis как message queue.

### Подключение клиента

```javascript
// Передаём access token в query-параметре при handshake
const socket = io('/', { query: { token: accessToken } })
```

При подключении сервер верифицирует токен и присоединяет пользователя к комнатам:
- `all` — общая комната для всех пользователей
- `user_{id}` — персональная комната

### События сервер → клиент

| Событие | Комната | Данные | Описание |
|---|---|---|---|
| `task:created` | `all` | полный объект задачи | Новая задача создана |
| `task:updated` | `all` | `{task_id, ...изменённые поля}` | Задача обновлена |
| `task:archived` | `all` | `{task_id, archived_at}` | Задача архивирована |
| `task:restored` | `all` | `{task_id}` | Задача восстановлена |
| `task:deleted` | `all` | `{task_id}` | Задача удалена |
| `unit:started` | `all` | полный объект юнита | Юнит запущен |
| `unit:stopped` | `all` | `{unit_id, task_id, datetime_end}` | Юнит завершён |
| `unit:updated` | `all` | `{unit_id, task_id, ...изменённые поля}` | Юнит отредактирован |
| `unit:deleted` | `all` | `{unit_id, task_id}` | Юнит удалён |
| `unit:force_stopped` | `user_{id}` | `{unit_id, stopped_by_fio}` | Юнит остановлен другим пользователем |

**Важно:** событие `unit:stopped` / `unit:force_stopped` — сигнал для фронта убрать блокирующее модальное окно.

### Логика активного юнита на фронте

```
1. После логина: GET /api/units/active
2. Если active unit есть → показать блокирующее модальное окно
3. Таймер работает на клиенте: считается разница между NOW() и datetime_start
4. При получении unit:force_stopped для active unit_id → убрать модалку, показать уведомление
5. При нажатии "Завершить" → POST /api/units/:id/stop → убрать модалку
```

**Таймер в модалке** обновляется каждую секунду средствами Vue (setInterval + computed на основе `datetime_start`), без WS-тиков.

---

## 11. Разделы системы — подробно

### 11.1 Экран авторизации

**Маршрут:** `/login`

- Форма: поля "Логин" и "Пароль", кнопка "Войти".
- Ошибка "Неверный логин или пароль" — без конкретики (не говорить, что именно неверно).
- После успешного входа при `force_change: true` в JWT — показывается **неклозабельное** модальное окно смены учётных данных (кнопка × отсутствует, клик вне модалки не закрывает).
- Форма смены: "Новый логин", "Новый пароль", "Подтвердите пароль".
- Валидация на клиенте + на сервере.

---

### 11.2 Раздел "Задачи"

**Маршрут:** `/tasks`

#### Общий layout страницы

```
┌─────────────────────────────────────────────────────────────────────┐
│  [Добавить]    [Поиск по названию задачи...]    Активные Избранное Архив │
├──────────────┬──────────────────────────────────────────────────────┤
│ Сортировки   │                                                       │
│ • Последняя  │   [Карточка] [Карточка] [Карточка]                   │
│   активность │   [Карточка] [Карточка] [Карточка]                   │
│ • Дата       │   [Карточка] [Карточка] [Карточка]                   │
│   создания   │                                                       │
│ • Срок испол │                                                       │
│              │                                                       │
│ Фильтры      │                                                       │
│ • Без фильт. │                                                       │
│ • Не присту. │                                                       │
│ • Уже работал│                                                       │
│              │                                                       │
│ От отдела    │                                                       │
│ [Все отделы] │                                                       │
│              │                                                       │
│ Период поступ│                                                       │
│ • Весь период│                                                       │
│ • Сегодня    │                                                       │
│ • Неделя     │                                                       │
│ • Месяц      │                                                       │
│ • Задать свой│                                                       │
│              │                                                       │
│ Кол-во: N    │                                                       │
└──────────────┴──────────────────────────────────────────────────────┘
```

#### Карточка задачи

- Бейдж (название отдела)
- Иконка избранного (сердечко) — кликабельна, `POST /api/tasks/:id/favorite`
- Название задачи
- "Сделать до: дата дедлайна" (если задан)
- "Поступила: дата поступления"
- Иконка-индикатор наличия юнитов (отображается, если есть хотя бы 1 юнит)
- Цвет карточки: стандартный / в избранном (другой фон) / архивная (приглушённый)

#### Сортировки

| Вариант | Логика |
|---|---|
| Последняя активность | По `MAX(units.datetime_start)` для задачи, затем по `tasks.created_at` |
| Дата создания | По `tasks.created_at DESC` |
| Срок исполнения | По `tasks.deadline ASC NULLS LAST` |

#### Фильтры

| Вариант | Логика |
|---|---|
| Без фильтров | Без дополнительных условий |
| Не приступали | Задачи без юнитов (`NOT EXISTS (SELECT 1 FROM units WHERE task_id = tasks.id)`) |
| Уже работал | Задачи, где текущий пользователь имеет юниты |

#### Кнопка "Добавить"

Видна только пользователям с `TASKS.OWN_CREATE` или `TASKS.OTHER_CREATE`.

#### Модальное окно создания задачи

Поля:
- **Название задачи** — текст, обязательное
- **Ссылка на YouGile** — URL, необязательное
- **Заказчик** — выпадашка из `departments`, обязательное
- **Дата поступления** — datepicker, по умолчанию = сегодня
- **Дедлайн** — datepicker, необязательное

После сохранения: `POST /api/tasks` → WS событие `task:created` → все клиенты обновляют список.

#### Модальное окно просмотра задачи

Открывается кликом на карточку. Разделено на две половины:

**Левая половина — детали задачи:**
- `#ID` задачи
- Кнопки: редактировать (карандаш), удалить (корзина) — отображаются по правам
- Название задачи
- **Заказчик:** название отдела
- **Дата поступления** / **Дата создания**
- **YouGile:** URL с иконкой "копировать" и "открыть в новой вкладке" (отображается только если заполнен)
- **Дедлайн** (если задан)
- **Создатель задачи:** ФИО
- Кнопка "Завершить задачу" (для активных задач) / "Вернуть из архива" (для архивных)

Для архивных задач: кнопки "Начать юнит" не отображаются.

**Правая половина — юниты:**
- Заголовок "Юниты"
- Кнопка "Начать юнит" (видна при наличии прав + задача не архивирована)
- Список юнитов (прокручиваемый):
  - Цветная левая полоска: **зелёная** = активен (`datetime_end IS NULL`), **нейтральная** = завершён
  - Имя юнита + тип юнита + длительность (в правом углу)
  - Стрелка expand для разворачивания
  - В развёрнутом виде: ФИО пользователя, "Начат: дата", "Окончен: дата / В работе"
  - Если `is_edited = TRUE` — другой фон строки (выделен как вручную отредактированный)
  - Кнопки редактировать/удалить — по правам

#### Запуск юнита

1. Клик на "Начать юнит".
2. Модальное: поле "Название юнита" + выпадашка типа + кнопка "Запустить".
3. `POST /api/tasks/:id/units` → WS `unit:started`.
4. Экран блокируется: показывается **неклозабельное** модальное окно активного юнита.

#### Модальное окно активного юнита (блокировка)

Отображается поверх всего, закрыть нельзя.

- "Текущий юнит от [дата], [время]" (розовый заголовок)
- Название юнита (крупный шрифт)
- Задача: пилюля с названием задачи
- "В работе"
- Таймер: `XX мин` / `X ч YY мин` — считается на клиенте от `datetime_start`
- Кнопка "Завершить" → `POST /api/units/:id/stop` → WS `unit:stopped` → модалка закрывается

---

### 11.3 Раздел "Статистика"

**Маршрут:** `/stats`

#### Управление периодом

- Datepicker с диапазоном дат (по умолчанию: весь текущий год)
- Кнопки навигации: `день [+] [–]`, `неделя [+] [–]`, `месяц [+] [–]`

**Логика кнопок:**

| Кнопка | Если текущий режим совпадает | Если режим другой |
|---|---|---|
| `день +` | Сдвинуть период на 1 день вперёд | Переключить режим на день, сохранить start |
| `день –` | Сдвинуть период на 1 день назад | Переключить режим на день, сдвинуть назад |
| `неделя +/–` | Сдвинуть на 7 дней | Переключить на 7-дневный период |
| `месяц +/–` | Перейти к следующему/предыдущему календарному месяцу | Переключить на текущий месяц |

- Неделя = любые 7 дней (не обязательно ПН–ВС).
- Месяц = стандартный календарный месяц.

Переключатель: **Общая** / **Расширенная**.

#### Общая статистика

Виджеты (каждый с кнопкой скачать XLSX):

**Задачи за период:**
```
Долг    Поступило    Закрыто    Осталось
 65       +40          -25         80
```
"Закрыто" отображается отрицательным числом со своим цветом.

**Задачи по часам:**
Таблица: Название задачи | Суммарные часы — сортировка по убыванию часов.

**Отработка задач по сотрудникам:**
Таблица: Сотрудник (ФИО) | Кол-во задач | Суммарные часы.

#### Расширенная статистика

**По типам юнитов:**
Таблица: Тип юнита | Суммарные часы | Кол-во уникальных задач.

**По отделам:**
Таблица: Отдел | Кол-во задач — сортировка по убыванию.

**По типам юнитов для пользователей:**
Таблица: Пользователь | Тип юнита | Часы | Задачи.

**Загруженность по дням:**
Сетка (calendar grid) — ячейки по дням выбранного периода.
В каждой ячейке: `Поступило N / Закрыто N / Часов N`.

---

### 11.4 Раздел "Настройки"

**Маршрут:** `/settings`

Состоит из вкладок. Видимость вкладок определяется правами пользователя.

#### Вкладка: Персонализация (доступна всем)

- Переключатель темы: **Светлая / Тёмная**
- Выбор цветовой схемы: **Классическая / Синяя / Розовая / Красная / Зелёная / Оранжевая / Жёлтая**
- **Конструктор тем:**
  - Список CSS-переменных компонентов с color-picker для каждого
  - Поле "Название темы"
  - Кнопка "Сохранить тему"
  - Список сохранённых пользовательских тем с кнопками "Применить" и "Удалить"
- Экспорт темы в `.json` (скачать)
- Импорт темы из `.json` (применить и добавить в список)

**Хранилище:** все темы и текущие настройки хранятся только в `localStorage` конкретного устройства. Синхронизации между устройствами нет.

#### Вкладка: Пользователи (USERS.VIEW)

- Таблица пользователей с поиском по ФИО
- Кнопка "Создать пользователя" (USERS.CREATE)
- Строка пользователя: аватарка | ФИО | логин | должность | роль | кнопки Edit/Delete
- Форма создания/редактирования: ФИО, логин, должность, роль
- При создании: пароль `admin`, `is_default_pass = TRUE`
- При скрытии: подтверждение + объяснение что данные сохраняются

#### Вкладка: Роли (ROLES.VIEW)

- Список ролей: название | биты прав (summary) | кнопки Edit/Delete
- Форма создания/редактирования роли:
  - Название роли
  - Конструктор прав: таблица с секциями и чекбоксами для каждого бита
- Вложенный список: пользователи с этой ролью (можно переназначить роль — ROLES.ASSIGN)
- Защита: нельзя изменить/удалить единственную всесильную роль

#### Вкладка: Списки (DEPARTMENTS.VIEW или UNIT_TYPES.VIEW)

Две подвкладки: **Отделы** / **Типы юнитов**

Для каждого — CRUD-таблица с inline-редактированием.

При удалении типа юнита — диалог подтверждения с предупреждением:
> «Удаление типа юнита приведёт к удалению всех юнитов с этим типом. Это действие нельзя отменить.»

#### Вкладка: Копирование и восстановление (BACKUP.VIEW)

- Кнопка "Создать резервную копию" → `GET /api/backup/export` → скачать ZIP
- Кнопка "Восстановить из резервной копии" → загрузить ZIP → `POST /api/backup/import`
- При восстановлении: двойное подтверждение с текстом предупреждения о полной замене данных

---

### 11.5 Профиль пользователя

Открывается кликом на аватарку в левом нижнем углу бокового меню.

- Аватарка (или identicon)
- ФИО, должность, роль
- Кнопка "Выйти" → `POST /api/auth/logout` → редирект на `/login`

**Редактирование профиля:**
- ФИО
- Логин (с проверкой уникальности)
- Пароль: "Текущий пароль" + "Новый пароль" + "Подтвердить"
- Аватарка:
  - Кнопка "Загрузить аватарку" → открывает конструктор (кроп + центрирование)
  - Конструктор: перетаскивание зоны обрезки, кнопки подтвердить/отмена
  - Сжатие до ≤ 2 МБ перед отправкой (Canvas API на клиенте)
  - Допустимые форматы: `image/png`, `image/jpeg`
  - Кнопка "Удалить аватарку" (показывается если аватарка загружена) → возврат к identicon

**Личная статистика (период по умолчанию — последние 7 дней):**
- Выбор периода (datepicker)
- Суммарное время
- Кол-во задач
- Разбивка по типам юнитов: тип → часы + кол-во задач

---

## 12. Фронтенд

### 12.1 Структура проекта

```
frontend/
├── index.html
├── vite.config.js
├── package.json
├── src/
│   ├── main.js              # Инициализация Vue, PrimeVue, Router, Pinia
│   ├── App.vue
│   │
│   ├── router/
│   │   └── index.js         # Маршруты + auth guard
│   │
│   ├── stores/              # Pinia stores
│   │   ├── auth.js          # Текущий пользователь, токены, force_change
│   │   ├── tasks.js         # Список задач, фильтры, активная задача
│   │   ├── units.js         # Активный юнит текущего пользователя
│   │   ├── theme.js         # Текущая тема, список тем (localStorage)
│   │   └── notifications.js # Toast-уведомления
│   │
│   ├── api/
│   │   ├── client.js        # fetch-обёртка с JWT refresh-логикой
│   │   ├── auth.js
│   │   ├── tasks.js
│   │   ├── units.js
│   │   ├── users.js
│   │   ├── roles.js
│   │   ├── stats.js
│   │   ├── departments.js
│   │   ├── unitTypes.js
│   │   └── backup.js
│   │
│   ├── socket/
│   │   └── index.js         # Socket.IO клиент + обработчики событий
│   │
│   ├── composables/         # Переиспользуемая логика
│   │   ├── usePermission.js # Проверка прав текущего пользователя
│   │   ├── useStatsPeriod.js# Логика управления периодом статистики
│   │   └── useAvatarCrop.js # Кроппер аватарки
│   │
│   ├── components/
│   │   ├── layout/
│   │   │   ├── AppSidebar.vue
│   │   │   └── ActiveUnitModal.vue   # Блокирующая модалка юнита
│   │   ├── tasks/
│   │   │   ├── TaskCard.vue
│   │   │   ├── TaskModal.vue         # Просмотр задачи + юниты
│   │   │   ├── TaskForm.vue          # Создание/редактирование
│   │   │   ├── TaskFilters.vue       # Левая панель фильтров
│   │   │   └── UnitListItem.vue      # Строка юнита с expand
│   │   ├── units/
│   │   │   ├── StartUnitModal.vue    # Запуск юнита
│   │   │   └── UnitEditModal.vue     # Редактирование юнита
│   │   ├── stats/
│   │   │   ├── StatsPeriodControl.vue
│   │   │   ├── StatsWidget.vue       # Обёртка с кнопкой экспорта
│   │   │   └── CalendarGrid.vue
│   │   ├── settings/
│   │   │   ├── PermissionMatrix.vue  # Конструктор прав роли
│   │   │   ├── ThemeBuilder.vue      # Конструктор тем
│   │   │   └── AvatarCropper.vue     # Кроппер аватарки
│   │   └── common/
│   │       ├── ConfirmDialog.vue
│   │       └── DateRangePicker.vue
│   │
│   └── views/
│       ├── LoginView.vue
│       ├── TasksView.vue
│       ├── StatsView.vue
│       ├── SettingsView.vue
│       └── ProfileView.vue
```

### 12.2 Маршрутизация

```javascript
// src/router/index.js
const routes = [
  { path: '/login',    component: LoginView,    meta: { public: true } },
  { path: '/tasks',    component: TasksView,    meta: { requiresAuth: true } },
  { path: '/stats',    component: StatsView,    meta: { requiresAuth: true } },
  { path: '/settings', component: SettingsView, meta: { requiresAuth: true } },
  { path: '/profile',  component: ProfileView,  meta: { requiresAuth: true } },
  { path: '/',         redirect: '/tasks' },
]

// Navigation guard:
// 1. Если не авторизован → /login
// 2. Если force_change → блокировать всё кроме модалки смены пароля
// 3. Если раздел недоступен по правам → /tasks
```

### 12.3 Боковое меню

```
┌────┐
│ 🦊 │  ← Логотип Grove Work
├────┤
│ ⊞  │  ← Задачи      /tasks
│    │
│ 📊 │  ← Статистика  /stats     (STATS.VIEW)
│    │
│ 🕐 │  ← Юнит        /tasks     (подсвечен если активный юнит)
│    │
│ ⚙  │  ← Настройки  /settings
├────┤
│ 👤 │  ← Аватарка    /profile   (внизу)
└────┘
```

Иконки из Google Material Symbols Outlined.
Активный пункт меню подсвечивается.
Если у пользователя активный юнит — иконка часов имеет визуальный индикатор.

### 12.4 Темы

**Предустановленные темы (8 штук):** Классическая, Тёмная, Синяя, Розовая, Красная, Зелёная, Оранжевая, Жёлтая.

Каждая тема — набор CSS-переменных, которые переопределяют токены PrimeVue через `pt` (PassThrough) и `:root`.

```css
/* Базовые переменные (пример для розовой темы) */
:root[data-theme="pink"] {
  --gw-primary: #e040fb;
  --gw-primary-light: #f3e5f5;
  --gw-accent: #00bfa5;
  --gw-bg: #fce4ec;
  --gw-surface: #ffffff;
  --gw-text: #212121;
  --gw-border: #f8bbd0;
}
```

Шрифт:
```html
<link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&display=swap" rel="stylesheet">
<link href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined" rel="stylesheet">
```

### 12.5 API-клиент (fetch-обёртка)

```javascript
// src/api/client.js

// Автоматический refresh access token при 401
// Добавляет Authorization: Bearer <token> к каждому запросу
// Централизованная обработка ошибок → Pinia notifications store
```

### 12.6 Composable: проверка прав

```javascript
// src/composables/usePermission.js
export function usePermission() {
  const auth = useAuthStore()
  const can = (section, bit) => hasPermission(auth.user.role.access, section, bit)
  return { can }
}

// Использование в компоненте:
const { can } = usePermission()
const canCreateTask = computed(() => can(Section.TASKS, Bit.OWN_CREATE))
```

---

## 13. Логирование

### Уровни

| Уровень | Что логируется |
|---|---|
| `INFO` | Вход/выход, создание/удаление/архивирование задач, старт/стоп юнитов, создание/удаление пользователей |
| `WARNING` | Попытки доступа без прав (403), нарушения бизнес-правил (422) |
| `ERROR` | Необработанные исключения, ошибки БД |
| `DEBUG` | SQL-запросы (только `FLASK_ENV=development`) |

### JSON-формат лога

```json
{
  "ts": "2026-05-12T10:00:00Z",
  "level": "INFO",
  "event": "unit.started",
  "user_id": 3,
  "task_id": 42,
  "unit_id": 100,
  "ip": "192.168.1.5"
}
```

### Реализация

```python
# app/utils/logger.py
import logging, json

class JSONFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "ts":    self.formatTime(record, "%Y-%m-%dT%H:%M:%SZ"),
            "level": record.levelname,
            "event": record.getMessage(),
        }
        if hasattr(record, "extra"):
            payload.update(record.extra)
        return json.dumps(payload, ensure_ascii=False)

def get_logger(name: str) -> logging.Logger:
    logger = logging.getLogger(name)
    if not logger.handlers:
        handler = logging.StreamHandler()
        handler.setFormatter(JSONFormatter())
        logger.addHandler(handler)
    logger.setLevel(logging.DEBUG if os.getenv("FLASK_DEBUG") else logging.INFO)
    return logger
```

Логи пишутся в `stdout`, Docker забирает через `docker logs`.

---

## 14. Безопасность

| Угроза | Защита |
|---|---|
| SQL Injection | SQLAlchemy параметризованные запросы (сырой SQL запрещён) |
| XSS | Vue auto-escaping; CSP-заголовки; Content-Type проверка загружаемых файлов |
| CSRF | JWT в `Authorization` header; refresh в HttpOnly SameSite=Strict cookie |
| Brute force | Flask-Limiter: 5 попыток / минута на `/api/auth/login` |
| Несанкционированный доступ | `@require_permission` на каждом эндпоинте; проверка OWN/OTHER в сервисе |
| Privilege escalation | Запрет смены своей роли; защита единственной всесильной роли |
| Загрузка файлов | Проверка MIME-типа (`python-magic`); файлы вне webroot; UUID в имени; max 2 МБ |
| Разглашение данных | Скрытые пользователи не попадают в ответы API |

### Security-заголовки

```python
# app/__init__.py
@app.after_request
def set_security_headers(response):
    response.headers["X-Content-Type-Options"]  = "nosniff"
    response.headers["X-Frame-Options"]         = "DENY"
    response.headers["Referrer-Policy"]         = "strict-origin-when-cross-origin"
    return response
```

---

*Документ является полной спецификацией для разработки Grove Work v2.0. Вопросов, требующих уточнения, не осталось.*

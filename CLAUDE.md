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
- `api/` — Flask Blueprints, декоратор `@require_role(min_level)`
- `sockets/` — Flask-SocketIO события
- `utils/` — permissions, avatar, logger

## Система прав (4 фиксированные роли)

| Уровень | Роль | Доступ |
|---|---|---|
| 1 | Сотрудник | Задачи: CRUD любых. Юниты: создать/редактировать/остановить/удалить свои. Статистика: просмотр. Отделы/Типы юнитов: просмотр |
| 2 | Менеджер | +управление чужими юнитами, CRUD отделов и типов юнитов, экспорт статистики |
| 3 | Администратор | +управление пользователями (CRUD), назначение ролей ≤ менеджера |
| 4 | Суперадминистратор | +назначение ролей любого уровня, бэкап |

Роли фиксированы в БД (4 строки, уровни 1-4), создавать/удалять нельзя.

Хелпер: `app/utils/permissions.py` — константы `EMPLOYEE=1, MANAGER=2, ADMIN=3, SUPERADMIN=4`, декоратор `@require_role(min_level)`, функция `get_user_level(user)`

Фронт: `composables/usePermission.js` — `const { isAtLeast, myLevel, ROLES } = usePermission()`

## Ключевые бизнес-правила

- Пользователь с `is_default_pass=TRUE` получает `force_change: true` в JWT — все API кроме `/api/auth/change-default` возвращают 403
- У пользователя единовременно только 1 активный юнит (`datetime_end IS NULL`)
- Нельзя архивировать задачу с активным юнитом
- Единственный суперадминистратор защищён от скрытия и смены роли
- Нельзя назначить роль равную или выше собственной (нельзя создать пользователя выше себя)
- Нельзя скрыть пользователя с ролью ≥ своей
- `avatar_path = NULL` → identicon по `user.id` (GitHub-style 5×5)
- Удаление типа юнита каскадно удаляет все юниты с этим типом

## Auth flow

- `POST /api/auth/login` → access token (Bearer) + refresh token (HttpOnly cookie)
- Access token TTL: 15 минут
- Refresh token TTL: 30 дней, `SameSite=Strict`
- `POST /api/auth/refresh` — обновить access по cookie

## WebSocket

Клиент передаёт access token в query param при handshake. Сервер присоединяет к комнатам `all` и `user_{id}`. Все мутации (задачи, юниты) излучают события в комнату `all`.

## Локальная разработка

```bash
./dev.sh             # одна команда: DB+Redis в Docker, Flask :5001, Vite :5173
# или по частям через Make:
make dev-infra       # поднять DB + Redis
make dev-migrate     # flask db upgrade
make dev-back        # Flask :5001 (автоматически поднимает инфру и мигрирует)
make dev-front       # Vite :5173 (второй терминал)
make dev-stop        # остановить контейнеры
```

Flask dev-сервер — порт **5001**. Vite — **5173**. `.flaskenv` содержит локальные настройки.

**Если БД не принимает пароль** (pg_data volume от старого запуска):
```bash
docker exec deploy-db-1 psql -U grovework -d grovework \
  -c "ALTER USER grovework WITH PASSWORD 'grovework_local';"
make dev-migrate
```

## Деплой на сервер

```bash
cp .env.deploy.example .env.deploy   # один раз: заполнить SERVER_HOST, SSH_KEY
make deploy    # git push → SSH на сервер → git pull → docker compose up --build
make logs      # логи app-контейнера
make status    # docker compose ps
make restart   # перезапустить app без пересборки
make shell     # bash внутри контейнера
```

Подробности — в `DEPLOY.md`. GitHub: https://github.com/DmitriyODS/gw2.git

При деплое `entrypoint.sh` автоматически запускает `flask db upgrade`.
Nginx собирает фронт сам через multi-stage `front/Dockerfile`.

## Цветовая система фронтенда

`front/src/assets/tokens.css` — Material You Expressive / M3, три слоя:
1. `--ref-*-h/c` — параметры цвета (hue/chroma), пишет `theme.js`
2. `--_p-*`, `--_s-*` — тональные палитры OKLCH
3. `--color-*` — семантические токены (primary, surface, error, success…)

`[data-dark="true"]` — тёмная тема. `--gw-*` — алиасы для совместимости.

**Цвета-теги задач:** фиксированный набор из 8 пастельных цветов (`red, orange, amber, green, teal, blue, violet, pink`). Токены `--tag-<name>-surface/-border/-accent` в `tokens.css` (адаптированы под светлую/тёмную тему). Цвет **индивидуален для пользователя** — хранится в таблице `user_task_colors (user_id, task_id, color)`. Управление: `PUT /api/tasks/:id/color` с `{color}`. В ответах `_enrich_task` подставляет цвет именно текущего пользователя; в сокет-броадкастах `task:created`/`task:updated` поле `color` вырезается, чтобы чужие клиенты не перезаписали свой цвет. Старый столбец `tasks.color` оставлен как технический архив. Набор продублирован в `front/src/utils/taskColors.js` и `back/app/schemas/task.py` (`TASK_COLORS`).

**Правило:** никаких `#hex` или `rgba()` в компонентах — только `--color-*` / `--tag-*` токены.

## ТВ-режим

Маршрут `/tv` (фронт). Открывается в новой вкладке кнопкой на экране статистики, рендерится без сайдбара/нижней навигации (роут с `meta.fullscreen=true`, App.vue смотрит на `route.meta.fullscreen`). Три слайда — день/неделя/месяц — листаются автоматически каждые 10 секунд (`SLIDE_MS`). Данные тянутся из `/api/stats/common` и `/api/stats/extended`, обновляются раз в минуту (`REFRESH_MS`). Внизу — кнопки prev/pause/next и переключение полноэкранного режима (Fullscreen API).

## Защита от подбора пароля

Хранится в Redis (`gw2:bf:attempts:{login}` и `gw2:bf:locked_until:{login}`). После каждых 5 неудачных подряд попыток ставится блокировка на `10 * 2**(steps-1)` секунд (10с, 20с, 40с…). Удачный логин обнуляет счётчик. Бэк отвечает `429 {retry_after_sec}`, фронт показывает таймер на `LoginView`. Логика — `back/app/services/login_throttle.py`.

## Раздел «Сотрудники» и мессенджер (v2.4.0)

**Каталог сотрудников.** `GET /api/users/directory?q=...&exclude_self=true` — публичный для любого авторизованного, отдаёт `UserDirectorySchema` (без `is_default_pass`/`is_hidden`). `GET /api/users/directory/<id>` — одиночный публичный профиль. Поиск идёт case-insensitive по `fio`+`login`. Фронт: `EmployeesView.vue` (карточки + модалка профиля с кнопкой «Написать», ведущей в `/messenger/<id>`).

**Мессенджер 1:1.** Таблицы `conversations` (user_a_id < user_b_id, уникальная пара), `messages` (text + read_at), `message_attachments` (file_path в `UPLOAD_FOLDER/messages/YYYY/MM/`, message_id nullable до момента отправки). Миграция `d3e4f5a6b7c8`. API в `back/app/api/messenger.py` (`/api/messenger/...`): list/open conversations, list/post messages (курсор по before_id), POST `/uploads` (multipart, 25 МБ макс. — `MESSENGER_ATTACHMENT_MAX`), POST `/read`, GET `/unread`. WebSocket: при отправке `socketio.emit('message:new', ...)` в комнаты `user_{recipient}` и `user_{sender}` (эхо для других вкладок). При пометке прочитанным — `message:read` собеседнику.

**Фронт мессенджера.** `MessengerView.vue` (двухпанельный, на мобильном — единый экран со списком/диалогом), `components/messenger/{ConversationList,MessageBubble,MessageInput,AttachmentView,NewChatDialog}.vue`. Store `stores/messenger.js`: список диалогов, кеш сообщений по convId, общий счётчик непрочитанных, методы `openWith/setActive/send/applyIncomingMessage/applyReadReceipt`. API клиент — `api/messenger.js`. Сокет-handlers подключены в `socket/index.js`. Бейджи непрочитанных — в `AppSidebar.vue` и `AppBottomNav.vue`.

**Уведомления и звук.** `utils/systemNotify.js` — Web Notifications API + Web Audio API (двухтональный «бип», без mp3-файла). Запрос разрешения происходит при первом заходе в `/messenger`. Уведомление не показывается, если страница в фокусе И активен этот диалог. Клик по уведомлению фокусирует окно и эмитит `messenger:open-conversation` (CustomEvent, ловит `MessengerView`).

**Фикс «светлых цветов на кнопках» (v2.4.0).** `theme.js → hexToOklch` теперь возвращает `L`, `applyPaletteKey` пишет `--ref-{name}-l` (с клампом 0.30–0.92) и `--color-on-{name}-user` (белый или тёмный по порогу L≥0.65). В `tokens.css` тон `--_p-40 / --_s-40 / --_t-40` использует `var(--ref-*-l)` вместо фиксированных 0.50; светлая тема `--color-on-{primary,secondary,tertiary}` → `var(--color-on-*-user)`. Дефолты сохранены (0.50/белый) — пресеты без явного L работают как раньше.

## Swagger UI

Доступен на `http://localhost:5001/apidocs` при запущенном dev-сервере.

## Логи

JSON-формат в stdout. Docker забирает через `docker logs`.  
`FLASK_DEBUG=1` включает DEBUG-уровень с SQL-запросами.

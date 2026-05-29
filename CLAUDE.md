# Groove Work v2.0 — Руководство по проекту

## Что это

Groove Work — внутренняя платформа для учёта времени задач и аналитики. Команды с разграниченными ролями ведут задачи и трекают время работы через «юниты» (отрезки времени).

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

Flask dev-сервер — порт **5001** (запускается через `python wsgi.py`, **eventlet**). Vite — **5173**. `.flaskenv` содержит локальные настройки.

**Важно про WebSocket:** dev-команды НЕ используют `flask run` — werkzeug-сервер не поддерживает WS-upgrade, и socket.io падает с `You need to use the eventlet server`. Правильный запуск — `python wsgi.py` (там `eventlet.monkey_patch()` + `socketio.run(app, debug=False)`). Платой за это стало отсутствие auto-reload: после правки бэк-кода процесс нужно перезапустить вручную (Ctrl+C → запустить снова). В `wsgi.py` `debug=False` намеренно — `socketio.run(..., debug=True)` переключает на werkzeug, который опять же ломает WS.

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

## ТВ-режим (v2.6.1 — Live-newsroom)

Маршрут `/tv` (фронт). Открывается в новой вкладке кнопкой на экране статистики, рендерится без сайдбара/нижней навигации (роут с `meta.fullscreen=true`, App.vue смотрит на `route.meta.fullscreen`).

**Архитектура.** Каркас grid 4-rows: `header / progress / canvas / ticker`. Canvas — grid 3-cols: `KPI rail (clamp 220–320px) | stage (1fr) | aside (clamp 240–340px)`. Все размеры через `clamp()` + `vmin`-единицы — гарантия что НИКОГДА не появятся скроллы при любых пропорциях экрана (от 4K до вертикального табло). На portrait-ориентации (`max-aspect-ratio: 1/1`) KPI rail превращается в горизонтальную полосу сверху, aside уезжает вниз.

**Слайды (8 штук, `SLIDE_MS=8000`).** Каждый описывается объектом в массиве `slides[]` с полями: `id, period (day|week|month), kind, icon, periodLabel, heroIcon/heroEyebrow/heroKey/heroFormat/tone, secondaries[], asideTone/asideIcon/asideTitle/asideKind`. Поддерживаемые `kind`: `hero-number` (1 огромное число + 2 secondary), `podium` (топ-3 на ступенях с медалями), `ranking` (top-5 список с барами), `departments` (горизонтальные бары по отделам), `quad` (4 крупные KPI-плитки), `brand` (брендовый слайд с пульсирующим логотипом). Цикл: today-closed → today-podium → today-departments → week-hours → week-ranking → month-quad → month-podium → brand.

**KPI rail.** Всегда видим (слева, 4 чипа): «Поступило / Закрыто / В работе / Часы команды» по периоду текущего слайда. При смене периода числа плавно «доезжают» к новому значению (watch внутри inline-компонента `TvCount`).

**Aside.** Контекстная карточка справа, разная для каждого слайда (`asideKind`): hours-today / hours-period / closed-today / top-dept / sparkline-closed / sparkline-hours / today-snapshot. Спарклайн — SVG-полилиния по `extendedData.calendar` (закрытия по дням или часы по дням).

**Анимации.** (1) Count-up: inline-компонент `TvCount` (`defineComponent` в script setup) — `onMounted` запускает rAF от 0 до value (ease-out-cubic, 900мс), `watch(value)` плавно переезжает на новое значение. Используется и в stage (через `:key="currentSlide.id"` на section перезапускается от 0), и в KPI rail (без key — smooth-transition). (2) Springy bars: CSS `@keyframes tv-bar-fill` `cubic-bezier(0.34, 1.56, 0.64, 1)` от 0 до `var(--bar-width)` (передаётся через `:style="{ '--bar-width': barPercent + '%' }"`). (3) Фоновое сияние: `radial-gradient` в `::before` под `tv-hero-number` (color-mix от текущего tone). (4) Огонёк у топ-1 на подиуме и в ranking: иконка `local_fire_department` (FILL=1), keyframe-pulse 1.4с с поворотом и scale. (5) Подиум: разные `animation-delay` для колонок (1-я в центре, выше; 2-я слева; 3-я справа, ниже) — поднимаются с springy easing. (6) Ranking: каждая строка въезжает слева с `--row-delay = i*80ms`. (7) Quad-плитки: scale-in с шахматной задержкой. (8) Brand-логотип: pulse-анимация с drop-shadow primary-color. (9) Переходы между слайдами: `<transition name="tv-stage">` — `translateX + scale(0.98) + filter: blur(6px)` 550мс.

**Тикер.** Бегущая строка снизу. Items генерируются из `commonByPeriod['day'|'week'|'month']` и `extendedByPeriod['day'|'week']`: «Сегодня закрыто N задач», «Лидер дня — <fio>», «Активный отдел дня — <name>», «За неделю команда отработала <часов>», «Главный тип работ недели — <name>». Дорожка дублируется (`tickerItemsX2`), `keyframe translateX(0 → -50%)` в линейной бесконечной анимации, `tickerDuration = max(20, items.length * 6)` секунд. Mask-image для плавного fade на краях viewport.

**Контролы (auto-hide).** `prev/pause/next/fullscreen` в круглом плавающем «пилюле» (position: fixed, right+bottom). Скрыты по умолчанию (это табло на стене). Появляются при `mousemove`/`touchstart`, прячутся через 2.5 сек (`CONTROLS_HIDE_MS`). Управляется `bumpControls()` + setTimeout.

**Данные.** `commonByPeriod`/`extendedByPeriod` — кеши по периоду. `loadPeriod(period)` грузит обе порции параллельно. При `goTo(idx)`: если period текущего слайда ещё не загружен — грузим, иначе показываем мгновенно. Refresh всех периодов каждые 60 сек (`REFRESH_MS`) silent-режимом.

**Аватары.** `loadUsers()` при mount грузит `/api/users/directory`, заполняет `userMap` (id → user). `avatarOf(uid)` отдаёт `/uploads/${avatar_path}` или `/api/users/${uid}/identicon` если нет фотографии.

**Палитра.** Все цвета — только семантические токены `--color-primary/secondary/tertiary/success/warning/error` + `*-container` + `--color-on-*`. Полная поддержка тёмной темы (наследуется через `data-dark` на `.tv`). Никаких hex/rgba.

## Защита от подбора пароля

Хранится в Redis (`gw2:bf:attempts:{login}` и `gw2:bf:locked_until:{login}`). После каждых 5 неудачных подряд попыток ставится блокировка на `10 * 2**(steps-1)` секунд (10с, 20с, 40с…). Удачный логин обнуляет счётчик. Бэк отвечает `429 {retry_after_sec}`, фронт показывает таймер на `LoginView`. Логика — `back/app/services/login_throttle.py`.

## Раздел «Сотрудники» и мессенджер (v2.4.0)

**Каталог сотрудников.** `GET /api/users/directory?q=...&exclude_self=true` — публичный для любого авторизованного, отдаёт `UserDirectorySchema` (без `is_default_pass`/`is_hidden`). `GET /api/users/directory/<id>` — одиночный публичный профиль. Поиск идёт case-insensitive по `fio`+`login`. Фронт: `EmployeesView.vue` (карточки + модалка профиля с кнопкой «Написать», ведущей в `/messenger/<id>`).

**Мессенджер 1:1.** Таблицы `conversations` (user_a_id < user_b_id, уникальная пара, поля `hidden_for_a/b`, `pinned_at_a/b`), `messages` (text + read_at + `hidden_for_a/b`), `message_attachments` (file_path в `UPLOAD_FOLDER/messages/YYYY/MM/`, message_id nullable до момента отправки). Миграции `d3e4f5a6b7c8` (базовая) и `e4f5a6b7c8d9` (soft-delete + pin). API в `back/app/api/messenger.py` (`/api/messenger/...`): list/open conversations, list/post messages (курсор по before_id), POST `/uploads` (multipart, 25 МБ макс. — `MESSENGER_ATTACHMENT_MAX`), POST `/read`, GET `/unread`, DELETE `/messages/<id>?scope=me|all`, DELETE `/conversations/<id>?scope=me|all`, POST `/conversations/<id>/pin` (toggle). WebSocket: `message:new` в комнаты `user_{recipient}` и `user_{sender}` (эхо для других вкладок), `message:read` собеседнику, `message:deleted` и `conversation:deleted` обеим сторонам (только при scope=all или физическом удалении), `conversation:pin` — эхо в свои вкладки.

**Soft-delete и pin (в repo/service):** Удаление «у себя» проставляет `hidden_for_<side>=true` у сообщения/диалога и каскадно скрывает сообщения на этой стороне (для диалогов). Когда обе стороны скрыли — `messenger_service` физически удаляет запись и файлы вложений. Удаление «для всех» — сразу DELETE + чистка файлов; для сообщения такое доступно только отправителю. При новом сообщении в скрытом диалоге обе `hidden_for_*` сбрасываются — диалог «возвращается» получателю. Pin — `pinned_at_<side>` (timestamp). Сортировка списка: pinned-первыми по `pinned_at` DESC, затем по `last_message_at` DESC; реализована на Python после SQL-сорта (сторонне-зависимая колонка не вырастает в индекс — для парных диалогов это допустимо).

**Фронт мессенджера.** `MessengerView.vue` (двухпанельный, на мобильном — единый экран со списком/диалогом), `components/messenger/{ConversationList,MessageBubble,MessageInput,AttachmentView,NewChatDialog,DeleteScopeDialog}.vue`. Store `stores/messenger.js`: список диалогов (хранится отсортированным через `sortConversations`), кеш сообщений по convId, общий счётчик непрочитанных, методы `openWith/setActive/send/applyIncomingMessage/applyReadReceipt/applyMessageDeleted/applyConversationDeleted/applyPinChange/deleteMessage/deleteConversationAction/togglePinAction`. API клиент — `api/messenger.js`. Сокет-handlers подключены в `socket/index.js`. Бейджи непрочитанных — в `AppSidebar.vue` и `AppBottomNav.vue`. Действия на карточке диалога (pin/delete) показываются на hover; на тач-устройствах — всегда (media `(hover: none)`).

**M3 Expressive: empty-states.** Пустой список диалогов — крупная иконка в `--color-primary-container`, заголовок, мягкая подпись и filled-tonal pill-кнопка с state-layer hover/active (`.btn-filled-tonal` в `ConversationList.vue`). Пустая правая панель (когда чат не выбран) — иконка-круг в `--color-surface-high` с приглушённым primary-цветом, заголовок «Выберите чат», подпись. Кнопок-приглашений «новый чат» в правой панели нет — действие выполняется через FAB в шапке списка слева. `DeleteScopeDialog.vue` — M3-стиль: круглая error-иконка, заголовок, текст, кастомный чекбокс «удалить также у собеседника» (превращается в primary-container при активации), `btn-text` + `btn-filled-error` с pill-формой.

**Уведомления и звук.** `utils/systemNotify.js` — Web Notifications API + Web Audio API (двухтональный «бип», без mp3-файла). Запрос разрешения происходит при первом заходе в `/messenger`. Уведомление не показывается, если страница в фокусе И активен этот диалог. Клик по уведомлению фокусирует окно и эмитит `messenger:open-conversation` (CustomEvent, ловит `MessengerView`).

**Фикс «светлых цветов на кнопках» (v2.4.0).** `theme.js → hexToOklch` теперь возвращает `L`, `applyPaletteKey` пишет `--ref-{name}-l` (с клампом 0.30–0.92) и `--color-on-{name}-user` (белый или тёмный по порогу L≥0.65). В `tokens.css` тон `--_p-40 / --_s-40 / --_t-40` использует `var(--ref-*-l)` вместо фиксированных 0.50; светлая тема `--color-on-{primary,secondary,tertiary}` → `var(--color-on-*-user)`. Дефолты сохранены (0.50/белый) — пресеты без явного L работают как раньше.

## Мессенджер v2.4.1 — ответы, пересылка, мини-чат, уведомления

**Ответы (reply).** Колонка `messages.reply_to_id` (FK на `messages.id`, `ondelete=SET NULL` — при удалении оригинала ответ остаётся, цитата пропадает). Миграция `f5a6b7c8d9e0`. Схема `MessageSchema.reply_to` = вложенный `ReplyPreviewSchema` (id, sender_id, sender_fio, обрезанный text, has_attachments). `MessageCreateSchema.reply_to_id`. Сервис `send_message(reply_to_id=...)` валидирует, что цитируемое сообщение из того же диалога. Фронт: кнопка «reply» на `MessageBubble`, баннер ответа в `MessageInput` (prop `replyTo` + emit `cancel-reply`), цитата над текстом в bubble. `replyTo` хранится локально в `MessengerView`/`MiniMessenger`, не в сторе.

**Пересылка (forward).** Колонка `messages.forwarded_from_user_id` (FK users, SET NULL) — автор оригинала для метки «Переслано от …». Эндпоинт `POST /api/messenger/forward` (`ForwardSchema`: message_id + conversation_ids + user_ids). Сервис `forward_message` копирует текст и **физически** копирует файлы вложений (`_copy_attachment` — новый attachment с новым `file_path`, чтобы удаление одной копии не задевало другую), создаёт сообщения во всех целевых диалогах (для user_ids — создаёт диалог при отсутствии) и шлёт `message:new` обеим сторонам. Фронт: `ForwardDialog.vue` (поиск по каталогу, мультивыбор получателей), кнопка «forward» на `MessageBubble`.

**Мини-мессенджер.** `components/messenger/MiniMessenger.vue` — глобальный плавающий FAB (правый нижний угол, `z-index 10050`, **поверх `ActiveUnitModal` z-9999** — можно отвечать не закрывая активный юнит). Смонтирован в `App.vue` в блоке авторизованного пользователя; скрыт на маршруте `/messenger` (там полноценный вид). Два режима: список диалогов → компактный тред с `MessageInput` (reply поддержан, forward/delete скрыты через props `show-forward`/`show-delete` на `MessageBubble`). Делит `activeConversationId` со стором; при закрытии треда сбрасывает его в null. Ловит `messenger:open-conversation` (из уведомления) и разворачивается.

**Прочтение/бейджи (фикс).** Раньше сообщения, пришедшие в открытый чат, не отмечались прочитанными на сервере → серверный `total_unread` расходился с локальным, бейдж скакал после refetch. Теперь `applyIncomingMessage` при `isViewingActively(convId)` (чат активен И вкладка в фокусе) сразу вызывает `markRead` на сервере; `markRead` всегда бьёт в API (без guard'а `unread_count===0`). Дополнительно прочтение срабатывает: при **возврате фокуса** вкладки (`socket/index.js` → `markActiveReadOnFocus` на `visibilitychange`/`focus`, по `activeConversationId` — общему для основного и мини-чата) и при **отправке** сообщения (`store.send` → `markRead`). Список диалогов грузится глобально в `App.vue` после входа (для бейджа, мини-чата и корректного fio в уведомлении).

**Онлайн-статус и last seen.** Колонка `users.last_seen_at` (миграция `a7b8c9d0e1f2`). Присутствие — in-memory в процессе сервера (`app/sockets/presence.py`: `_counts` user_id→число сокетов, `_sid_user` sid→user_id). На socket connect (`events.py`) `presence.on_connect(sid, user_id)` → при первом соединении broadcast `presence:update {online:true}` в комнату `all`; на disconnect `presence.on_disconnect(sid)` → при обрыве последнего соединения пишет `last_seen_at` в БД и broadcast `{online:false, last_seen_at}`. Эндпоинт `GET /api/messenger/presence` → `{online:[ids]}` (снимок при загрузке/reconnect). `UserDirectorySchema.last_seen_at` отдаёт время для оффлайн. **Развёртывание — один app-контейнер с eventlet, поэтому in-memory ок; при нескольких процессах socketio presence надо вынести в Redis.** Фронт: store `onlineIds`(Set)/`lastSeenById`/`isOnline`/`lastSeenOf`/`applyPresence`/`fetchPresence`; сокет-handler `presence:update`; `fetchPresence` на каждый connect. UI: зелёная точка `.online-dot` на аватаре (ConversationList + шапка MessengerView), статус «в сети»/`formatLastSeen()` (точная дата+время, `utils/presence.js`) в шапке чата.

**Уведомления (Web Notifications + SW).** `public/sw.js` — минимальный service worker ради `ServiceWorkerRegistration.showNotification` (на Android Chrome `new Notification()` запрещён, OS-уведомления только через SW; push-сервера нет — уведомления показываются, пока вкладка жива). `systemNotify.js`: `registerNotifyServiceWorker()` (регистрация + обработка `message` от SW → `messenger:open-conversation`), `installNotifyUnlock()` (по первому клику/нажатию «разогревает» AudioContext и тихо просит разрешение — лечит «звук иногда не играет» и Safari, требующий жест). `showSystemNotification(title, body, {onClick, data})` — сначала через SW, fallback на конструктор; иконка `/logo.svg`. Регистрация/unlock вызываются в `App.vue` после входа.

**Drag-drop / paste файлов.** Drop обрабатывается на **всей области чата**, а не только на поле ввода: обработчики `@drop/@dragover/@dragenter/@dragleave` на `.chat-panel` (`MessengerView`) и на `.mini-thread` (`MiniMessenger`), оверлей на всю зону. `MessageInput` экспонирует `addFiles(files)` (`defineExpose`), который родитель вызывает на drop; в самом `MessageInput` остаётся только `@paste` на textarea (берёт `clipboardData.items` типа file — скриншоты) и кнопка-скрепка. Общий `uploadFiles(files)` переиспользуется всеми тремя путями.

**Мобильная адаптивность.** `.messenger` на ≤768px — `position: fixed; inset:0; z-index:100` (статичный полноэкранный, не «ёрзает» при показе/скрытии адресной строки); нижняя навигация (z-200) поверх, списку диалогов дан `padding-bottom` под неё. Мобильный FAB «новый чат» в `MessengerView` (`Teleport to body`, `.fab`, как на экране задач). Toast: в `App.vue` позиция адаптивная — на мобильном `top-center` (снизу прятала нижняя навигация), на десктопе `bottom-right`; CSS для `.p-toast-top-center` в `main.css`.

## v2.6 — итерации редизайна настроек, ThemeBuilder, Help Center

**Адаптивность настроек (SettingsView).** Раскладка теперь: десктоп — sidebar 340px + pane, оба со своим внутренним scroll, общая шелла `height: 100%; overflow: hidden` (главный main-content не скроллит). Брейкпойнты: ≤1100 sidebar 280px и `nav-desc` скрывается; 769–900 sidebar превращается в icon-rail (88px, только иконки + подсветка активной 3px-ободком); ≤768 drill-down — список секций обычным flow, при выборе `settings-pane.mobile-full` становится `position: fixed; inset: 0; z-index: 90` со sticky-шапкой и safe-area отступом под нижнюю навигацию. На очень узких <380 — `pane-sub` скрыт. `Transition name="pane-swap"` между секциями (translate-X + opacity 0.18s). В шапке секции — `pane-title-icon` (tone-вариант). В sidebar добавлен `data-tutorial="settings-section-{key}"` на каждый пункт и `nav-empty` при пустом поиске.

**Карточки пользователей.** Плашка роли (`.user-card-role`) — `align-self: flex-start; max-width: 100%; overflow: hidden; white-space: nowrap; text-overflow: ellipsis` (раньше `width: max-content` срезал фон у длинных названий). На мобильном — компактнее: padding 12px, avatar 44px, actions в столбик иконок (32×32) справа без border-top, шрифты на 1px меньше.

**Новые встроенные темы.** Каждая тема теперь несёт собственный `neutral` (фоновый тон в той же гамме). Раньше нейтральный был только у `sunset` — поэтому остальные темы выглядели на нейтрально-сером фоне. Сейчас 15 пресетов с единой палитрой акцент+фон: classic/blue/pink/red/green/orange/yellow/violet/lilac/sunset/ocean/mint/coffee/midnight/forest. Primary у classic заменён `#e040fb → #9b4dff` для лучшего контраста на светлом фоне; primary у некоторых тем (green/orange/yellow/pink) тоже сдвинуты к более насыщенным/тёмным для читабельности кнопок.

**ThemeBuilder редизайн.** Полная переработка `front/src/components/settings/ThemeBuilder.vue`:
- Hero-блок с градиентом (primary-container → tertiary-container), внутри mock-превью интерфейса (сайдбар + карточка с pill'ами и тегами — все цвета через токены, обновляются мгновенно при смене темы). Кнопки «Мне повезёт» (gradient primary→tertiary) и «Сбросить» (ghost).
- Сегментированный переключатель «Светлая/Тёмная» в стиле iOS: фон-track, animated indicator с `transform: translateX` (0.3s cubic-bezier) и box-shadow.
- Галерея пресетов: карточки 16:10 aspect-ratio с тремя цветными полосами (primary > secondary > tertiary; flex 1.6/1/0.8), фон карточки = neutral темы, активная — с обводкой primary 4px + checkmark в углу.
- Color-swatches: круглые плашки 52px с inset shadow, edit-icon-pill, скрытый `<input type=color>` поверх (occupied via opacity 0). Live-preview onInput.
- Save-row: pill-input + filled-кнопка с bookmark_add.
- «Мои темы»: tiles с превью + кнопками «Применить» / удалить.
- Импорт/экспорт: tonal-кнопки. Все breakpoints (≤900 / ≤600) — карточки в один столбец, кнопки full-width.

**Help Center.** `front/src/components/settings/HelpCenter.vue` — интерактивная справка по всем разделам. Каталог из 5 групп: «Основная работа», «Общение», «Личное и настройки», «Администрирование» (от admin), «Система» (от superadmin). Каждая статья: title, subtitle, icon+tone, paragraphs, steps (numbered), tips (с иконкой tips_and_updates), route (кнопка «Перейти в раздел») и tourTarget (id шага в туре → «Показать в туре» открывает тур с этого шага). Поиск работает по всему тексту статей. Доступ — секция `help` в Настройках (добавлена в группу «Персонализация»).

**Тур: новые разделы и startAt.** В `useTutorial.js` добавлен `startAtId` ref + `open({ startAt })`. В `AppTutorial.vue` `onMounted` ищет шаг по `startAtId` и стартует с него (используется из Help Center). Новые шаги между `tab-archive` и `stats-nav`: `employees-nav` (целит на `nav-employees`), `messenger-nav` (целит на `nav-messenger`), `calls-info` (без target, описательный). Шаг `settings-theme` теперь целит на `settings-section-theme` (а не на старый `settings-tab-theme`). Добавлен шаг `settings-help`. В сайдбаре и нижней навигации проставлены `data-tutorial="nav-employees"` и `data-tutorial="nav-messenger"`.

## v2.6.2 — кликабельные ссылки в чате + reload-устойчивые звонки

**Кликабельные ссылки.** `front/src/utils/linkify.js` (`linkifyParts(text)`) разбивает текст на сегменты `{type:'text'|'link', value, href}`: ловит `http(s)://…` и `www.…` (последнему подставляет `https://`), откусывает хвостовую пунктуацию `.,;:!?)]}'"»…`. `MessageBubble.vue` рендерит сегменты — ссылка как `<a target="_blank" rel="noopener noreferrer" @click.stop>`. Стиль `.msg-link` — `--color-primary`, на исходящем пузыре `--color-on-primary-container` (никакого hex). Работает и в основном мессенджере, и в мини-чате (общий `MessageBubble`).

**Звонки переживают перезагрузку страницы (grace + rejoin).** Раньше при F5 во время звонка новый сокет успевал подключиться раньше, чем отваливался старый → `_has_visible_connection` = true → звонок НЕ завершался на сервере (висел «идёт»), но клиент после reload был в `idle` и вернуться не мог.
- **Бэк.** `presence.has_any_connection(user_id)` — есть ли хоть одно живое соединение (видимое или нет; для звонков важно именно это). `events.py` disconnect теперь ВСЕГДА вызывает `cleanup_call_on_disconnect` — но та не убирает пользователя сразу: планирует через `socketio.start_background_task` отложенную проверку (`CALL_REJOIN_GRACE_SEC`, дефолт 15 с). Если за окно пользователь вернулся (`has_any_connection`) — остаётся в звонке; иначе `_finalize_disconnect` выводит его и уведомляет остальных. Новое событие `call:rejoin {call_id}` (`call_events.py`): проверяет, что пользователь ещё в `state["invited"]`, идемпотентно `accept_call`, сначала шлёт существующим `call:participant-joined {rejoin:true}` (сброс устаревшего peer), затем вернувшемуся `call:accepted {existing_participants, call}` (он переинициирует offer'ы). `/api/calls/active` (уже был) во время grace отдаёт звонок.
- **Фронт.** `stores/call.js`: `rejoinCall` (state), `checkRejoin()` (дёргается из `App.vue` после connect — GET `/calls/active`, если звонок жив → показываем баннер), `confirmRejoin()` (user-gesture → `rtc.start()` + emit `call:rejoin`), `dismissRejoin()` (emit `call:leave`). `handleParticipantJoined` теперь всегда `removePeer(user_id)` перед плейсхолдером (сброс мёртвого peer при rejoin собеседника; для обычного join — no-op). `handleError` ресетит и на `code==='NOT_IN_CALL'`. Компонент `components/call/ReturnCallBanner.vue` — плавающий баннер сверху «Вернуться / Завершить» (Teleport, токены, safe-area, на узких экранах прячет подписи кнопок). Смонтирован в `App.vue`.

**Баг незавершения p2p при «положил трубку» (найден тестами).** `call_state.remove_user_from_call` / `remove_user_from_any_call` не убирали пользователя из `invited` → ушедший вечно считался «pending invitee» → `should_end` возвращал False → p2p-звонок не завершался, собеседник оставался один на линии. Фикс: при выходе `discard` и из `invited`. (Во время grace-окна НЕ вызывается — там пользователь остаётся в `invited`, поэтому rejoin-gate проходит.)

**Надёжность медиа.** `ParticipantTile.vue`: видео-элемент ВСЕГДА `muted`, звук удалённого участника — на отдельном `<audio v-if="!isLocal" autoplay playsinline>` (привязка `srcObject` в `attach()`). Раньше единственный `<video>` для удалённого был `v-show`-скрыт при выключенной камере собеседника, и аудио было хрупким; теперь звук идёт независимо от видимости видео. `attach()` пере-привязывает оба элемента по `watch([stream, streamTick])`. Мобильная адаптивность `CallView.vue`: шапка учитывает `env(safe-area-inset-top)`, mini-окно на `≤600px` поднято над нижней навигацией (`inset … calc(76px + safe-area-bottom) …`).

**Тесты.** `back/tests/` + `pytest.ini` (`pythonpath=.`). `test_call_state.py` — 8 юнит-тестов состояния (create/join/decline/leave/should_end/busy/grace/rejoin-идемпотентность). `test_call_flow.py` — E2E через `SocketIOTestClient`: start → incoming → accept → accepted/participant-joined → disconnect → `call:rejoin` → accepted с теми же участниками + `participant-joined {rejoin:true}` у инициатора. `conftest.py` поднимает реальный `create_app("production")` поверх dev-БД/Redis (env проставляются в conftest), `app`-фикстура skip'ается если БД недоступна. Запуск: `cd back && ./venv/bin/pytest -q`. **Важно: pytest требует проброшенных портов БД/Redis (`make dev-infra` поднимает с `docker-compose.override.yml`); чистый deploy-стек порты на хост не публикует.**

## v2.6.1 — мобильная уборка (bottom-nav, профиль сотрудника, тур, поле ввода, three-dots в чате)

**AppBottomNav.** На мобильном `min-height: calc(64px + env(safe-area-inset-bottom, 0px))` (вместо фикс `height: 60px`) + `padding-bottom: max(8px, env(safe-area-inset-bottom, 0px))` внутри панели. Иначе safe-area-зона у iPhone «съедала» подписи. `bottom-nav-label` 11px / line-height 1.1.

**EmployeesView.** Диалогу профиля задан `:style="{ width: '420px', maxWidth: 'calc(100vw - 24px)' }"` и `pt.content.style = "overflow-x: hidden; padding: 0"` (содержимое само управляет padding'ом, иначе двойная вложенность съедала ширину на узких экранах). `.employee-profile` сменил `min-width: 320px` на `width: min(420px, calc(100vw - 32px))` + `overflow-x: hidden`; тексты `word-break: break-word; overflow-wrap: anywhere`. На мобильном `profile-actions` — `flex-direction: column` (full-width столбцом), у иконочной кнопки «Аудиозвонок» появляется подпись через `.audio-label` (скрытая на десктопе).

**AppTutorial — мобильный тур.** `updateSpotRect`: (а) ищет ПЕРВЫЙ ВИДИМЫЙ элемент по селектору через `findVisible()` — sidebar и bottom-nav дублируют data-tutorial-якоря, на мобильном sidebar `display: none` отдавал rect 0×0 и подсветка не работала; (б) делает `scrollIntoView({ block: center, behavior: smooth })` если таргет вне viewport (ждёт 220 мс); (в) при нулевом rect возвращает `spotRect = null` (карточка идёт по центру, описательно). `cardStyle` на мобильном смотрит на положение spotlight: если таргет в нижней половине viewport — карточка едет НАВЕРХ (`top: calc(safe-area-top + 12px)`, скругление `0 0 20px 20px`), иначе вниз (как было), `maxHeight: 70dvh`. Так подсказка никогда не закрывает подсвеченную кнопку.

**MessageInput + MessengerView — отступ под полем ввода.** `msg-input` теперь `padding: 10px 14px 12px` (раньше `padding-bottom: calc(16px + env(safe-area-inset-bottom, 0px))`). Safe-area складывалась с `chat-panel.padding-bottom = 60 + safe` → между textarea и нижней навигацией висело ~50 пустых пикселей. Сейчас safe-area отвечает только родитель: `chat-panel` (на мобильном) — `padding-bottom: calc(64px + env(safe-area-inset-bottom, 0px))` (ровно высота bottom-nav с её внутренней safe-зоной). FAB подтянут: `bottom: calc(64px + 16px + env(...))`.

**MessengerView — three-dots-меню действий по чату.** Четыре кнопки в шапке (audio call, video call, pin/unpin, delete) свёрнуты в одну `more_vert`-кнопку. По клику разворачивается `.chat-menu` (position: absolute от `.chat-tools`, `top: calc(100% + 6px); right: 0`, `min-width: 220px`, `box-shadow: var(--shadow-lg)`). Пункты: Аудиозвонок (tone-success), Видеозвонок (tone-success), Закрепить/Открепить (tone-tertiary при active), divider, Удалить чат (.danger + tone-error). Закрывается: клик вне (`mousedown` + `touchstart` listener), при выборе пункта (`onMenuAction`), при смене `activeConversationId` (через watch). Tutorial-якоря `chat-call-audio` / `chat-call-video` сохранены на пунктах меню; `chat-tools` остался на враппере. Transition `chat-menu` — opacity + scale(0.96) + translateY(-4px), origin top right.

## v2.6.0a — фикс звонков и уведомлений о входящих (в той же v2.6.0)

**WebRTC ICE queue (`services/webrtc.js`).** В v2.6.0 `addIceCandidate` вызывался сразу при приходе кандидата — но если он прилетал ДО `setRemoteDescription` (обычное явление: offer/answer + первые ICE-кандидаты идут одной пачкой по сокету), `addIceCandidate` бросал InvalidStateError, кандидаты молча терялись, и P2P не устанавливалось. Теперь каждый peer имеет `pendingIce[]` и флаг `remoteSet`; в `handleRemoteIce` если `remoteSet === false` — кандидат кладётся в очередь, `_flushPendingIce(entry)` дёргается из `handleOffer`/`handleAnswer` после `setRemoteDescription`. Без этого пакета звонки доходили до accept, но дальше плитка собеседника оставалась пустой («pending», без видео и звука).

**Идемпотентность WebRTC.** `start()` теперь возвращает существующий `localStream` если он уже есть (двойной вызов безопасен). `_attachLocalTracks(pc)` идемпотентно добавляет треки в peer'а (через `getSenders()` проверяет, не добавлены ли уже) — нужно, чтобы при позднем появлении localStream (теоретический race) треки доехали до всех уже созданных peer'ов.

**Re-attach видео при ontrack (`ParticipantTile.vue` + `stores/call.js`).** `ontrack` срабатывает отдельно для audio и video треков (они приходят независимо). Старый код эмитил `remote-stream` с тем же объектом `stream` — Vue watch на `props.stream` срабатывал ровно один раз, и второй трек (видео) тихо не подхватывался плиткой. Теперь `WebRTCManager.ontrack` диспатчит `remote-stream` с уникальным `tick: Date.now()`, store кладёт его в `remoteStreams[uid].streamTick`, `ParticipantTile` получает `streamTick` как отдельный prop и watch'ит `[stream, streamTick]` — re-attach гарантирован.

**Уведомления о входящем звонке ВСЕГДА (`socket/index.js`, `utils/systemNotify.js`).** Раньше system notification о входящем показывалось только если `document.visibilityState !== 'visible' || !document.hasFocus()` — т. е. когда вкладка в фокусе, пользователь видел только overlay (а если был в другом окне/вкладке другого приложения — вообще ничего). Звонок — критическое событие, теперь уведомление показывается всегда. Добавлены `showCallNotification(title, body, { callId, onClick })` и `closeCallNotification()` — отдельные функции с тегом `gw2-call`, `requireInteraction: true` (ОС не скрывает через 5с), `data.kind = 'call'`. SW-уведомление закрывается через `swRegistration.getNotifications({ tag })`. `closeCallNotification()` вызывается при `call:accepted`/`call:ended`/`call:error`.

**SW клик по уведомлению о звонке (`public/sw.js`).** `notificationclick` теперь различает `data.kind === 'call'` (focus + postMessage `type: 'focus-call'`) и обычное (`open-conversation`). В `systemNotify.js` SW-message-listener при `type === 'focus-call'` диспатчит window-event `call:focus-overlay`; в `App.vue` подписка на этот event разворачивает мини-режим обратно в полный экран (если был свёрнут).

**Рингтон с gesture-retry (`IncomingCallOverlay.vue`).** AudioContext, созданный до первого user-gesture, оказывается в состоянии `suspended` и звук молчит. Теперь при `startRing()` если контекст застрял в `suspended` — вешается одноразовый слушатель `pointerdown/keydown` на window (capture-phase, `installGestureRetry`) — на первом жесте делает `resume()`. Слушатель снимается в `stopRing` и `onBeforeUnmount`. Без этого пользователи иногда не слышали рингтона до первого клика на странице.

**Auto-decline при «занят» (`stores/call.js`).** `handleIncoming` при `phase !== 'idle'` теперь отправляет `socket.emit('call:decline', ...)` серверу — звонящий получает `call:participant-declined` и/или `call:ended` мгновенно, а не висит в `ringing` до таймаута.

**Toast при ошибках звонка (`stores/call.js`).** `handleError` дополнительно дёргает `useNotificationsStore().warn(text)` — раньше после `reset()` `store.error` терялся (CallView размонтирован, никто не показывает текст). При getUserMedia ошибках в `startCall`/`accept` тоже выводится toast.

**Request notification permission при startCall.** `startCall` дёргает `requestNotificationPermission().catch(() => {})` — клик «позвонить» это надёжный user gesture, и Safari/Firefox разрешают спросить permission именно в этот момент. Раньше запрос шёл только на `onMounted` App.vue (часто игнорировался браузерами без явного жеста).

**Бэк (`sockets/call_events.py`).** Защитные `int()` касты на `user_ids` в `call:start`, `call_id`+`to_user_id` в `webrtc:signal` — раньше при str/None ID проверка `me not in state["invited"]` молча проваливалась без диагностики. Добавлено логирование `call.start`, `call.start_failed`, `webrtc.signal_rejected` для отладки прода.

## v2.6.0 — звонки/видеоконференции и редизайн настроек

**Звонки (WebRTC, P2P + mesh-группы).** Бэк: модели `Call` (id, kind p2p|group, status ringing|active|missed|ended, media audio|video, started_at/ended_at, conversation_id) и `CallParticipant` (call_id, user_id, role initiator|invitee, invited_at/joined_at/left_at, declined). Миграция `b1c2d3e4f5a6`. Сервис `app/services/call_service.py` — start/accept/decline/leave/end_by_initiator/cleanup_user_on_disconnect, валидация занятости (in-memory) и прав. In-memory state в `app/sockets/call_state.py` (`_calls: call_id → {invited, joined, declined, initiator_id, kind, media}`, `_user_call: user_id → call_id`); пока один app-контейнер с eventlet — этого достаточно, при горизонтальном масштабировании выносить в Redis.

**Сигналинг WebRTC через Socket.IO.** `app/sockets/call_events.py`: `call:start` (→ `call:started` инициатору + `call:incoming` приглашённым), `call:accept` (→ `call:accepted{existing_participants, call}` принявшему + `call:participant-joined` существующим), `call:decline` (→ `call:participant-declined` + `call:ended` если p2p отказ), `call:leave` (→ `call:participant-left` + `call:ended` если последний вышел), `call:end` (только инициатор → `call:ended` всем), `webrtc:signal{call_id, to_user_id, kind: offer|answer|ice, payload}` — сервер маршрутизирует сигнал «как есть» между приглашёнными (без парсинга SDP). Дополнительно `call:media-state{audio, video}` — UI-индикация mute/camera без media renegotiation (треки управляются локально через `MediaStreamTrack.enabled`). На disconnect (когда у пользователя нет видимых сокетов) вызывается `cleanup_call_on_disconnect` — пользователь убирается из звонка с уведомлением остальных. REST: `GET /api/calls/history`, `GET /api/calls/active`, `GET /api/calls/ice-servers` (отдаёт STUN Google + опционально coturn с эфемерными credential'ами по HMAC-SHA1 с `TURN_SECRET`).

**Фронт звонков.** `services/webrtc.js` — `WebRTCManager` (EventTarget, не реактивный): локальный `MediaStream`, `Map<userId, {pc, remoteStream}>`, события `local-stream/remote-stream/local-signal/peer-failed`. Логика mesh: новый участник после accept получает `existing_participants` и сам инициирует offer ко всем — у каждой пары один заведомый инициатор (тот, кто пришёл позже), что симметрично решает glare. `stores/call.js` (`useCallStore`) — phase idle|incoming|outgoing|active, `call` (метаданные), `localStream`/`remoteStreams[uid]`, `audioEnabled/videoEnabled`, `media`, `isMinimized`, `error`; экшены `startCall/accept/decline/hangup/toggleMic/toggleCam/handleStarted/handleIncoming/handleAccepted/handleSignal/handleParticipantJoined/handleParticipantLeft/handleEnded`. Сокет-handlers в `socket/index.js` (call:incoming/started/accepted/participant-*/ended, webrtc:signal, call:media-state, call:error). `getSocket()` экспортируется из `socket/index.js` — call-store шлёт сигналы напрямую (без import-cycle через ленивые impl). MediaStream/RTCPeerConnection не реактивны (Vue 3 Proxy ломает) — хранятся в raw-полях стора.

**Компоненты звонка.** `components/call/IncomingCallOverlay.vue` — full-screen модалка с пульсирующим аватаром, рингтоном (Web Audio loop, два тона 520/660 Гц с интервалом 1.7с), кнопками «принять/отклонить»; рингтон в watch(show) start/stop. `CallView.vue` — экран активного звонка: header (статус-точка + название + таймер), сетка плиток (1/2/4/many — auto grid), нижние контролы (mic/cam/hangup, hangup — крупная error-кнопка), сворачивание в плавающее окошко в углу (mini-режим). `ParticipantTile.vue` — `<video autoplay playsinline muted=isLocal>` + placeholder с аватаром при выключенной камере или ещё не подключённом stream'е; локальное видео зеркалится `transform: scaleX(-1)` (как в Zoom). Все три смонтированы в `App.vue` (`<IncomingCallOverlay>` и `<CallView>` в блоке авторизованного пользователя). Кнопки «позвонить» — в шапке чата `MessengerView` (`audio` + `video`) и в карточке профиля `EmployeesView`.

**coturn в docker-compose.** В `deploy/docker-compose.yml` добавлен сервис `coturn` под профилем `calls` (запуск: `docker compose --profile calls up -d`). Network mode host (нужны множественные UDP порты). Эфемерные credential'ы через `--use-auth-secret --static-auth-secret=${TURN_SECRET}` (REST API спецификация TURN); диапазон relay-портов 49152–49200 (умышленно небольшой). На сервере открыть в firewall 3478/UDP+TCP и UDP 49152–49200. В `.env.example` добавлены `TURN_HOST/TURN_PORT/TURN_REALM/TURN_SECRET`. Без TURN звонки идут через STUN-only — работают в одной сети или с дружелюбным NAT, могут не пробиться через мобильный интернет/симметричный NAT.

**Редизайн «Настройки».** `SettingsView.vue` переписан в M3 Expressive с двух-колоночным layout: слева sidebar (340px на десктопе) с поиском по разделам, группами «Персонализация / Администрирование / Система» и иконкой+название+описание для каждого раздела; справа — контент активной секции с шапкой (заголовок+подпись) и pane-body. На мобильном (`<=768px`) — drill-down: список секций на весь экран → тап → отдельный экран секции с кнопкой «назад». Карточки настроек (`.settings-card`) с tone-вариантами иконки (primary/secondary/tertiary/error) через CSS var `--tone-bg/--tone-fg`. Список пользователей — сетка карточек (auto-fill 260px) с аватаром, ФИО, логином, должностью и pill-плашкой роли (цвет по level). Отделы и типы юнитов — список chip-row'ов. Кнопки M3: `.btn-filled` (primary), `.btn-filled-tonal` (secondary container), `.btn-outlined`, `.btn-text` — pill-форма. Поддержка `?section=…` в URL (для глубокой ссылки на раздел из external links). Все цвета — только семантические токены (`--color-*`), никакого хардкода.

**Редизайн раздела «Задачи» (главный экран).** `TasksView.vue` + `TaskCard.vue` + `TaskFilters.vue` + `SortSheet.vue` переписаны в M3 Expressive. Вся функциональность сохранена (поиск с debounce 400мс, табы active/favorites/archive с `data-tutorial="tab-*"`, фильтры по отделу/юнитам/периоду с кастомным диапазоном, сортировки, пагинация, `data-tutorial="task-add-btn"` на кнопке «Добавить», FAB на мобильном). Только семантические токены `--color-*`/`--tag-*`, адаптивно под тёмную/светлую тему.
- **Шапка:** сегментированные pill-табы с иконками (на ≤480px — только иконки), pill-строка поиска с кнопкой-крестиком очистки (`clearSearch`), сегментированный переключатель вида «сетка/список».
- **Переключатель вида (`viewMode`).** grid (сетка `auto-fill minmax(248px,1fr)`) ↔ list (компактные строки), сохраняется в localStorage `gw2_tasks_view`. `TaskCard` принимает prop `view` ('grid'|'list') и рендерит компактную строку в list-режиме (на ≤600px скрывает мета-чипы).
- **Индикатор дедлайна (`deadlineInfo` в TaskCard).** Для не-архивных задач с `deadline`: просрочено (level `overdue` → error-container, иконка warning), сегодня/≤2 дней (level `soon` → warning-container), иначе обычная дата. Чип в `.task-meta`.
- **Быстрый старт юнита с карточки.** Кнопка «В работу» (`.work-btn`) на не-архивных карточках. `TaskCard` смотрит `unitsStore.activeUnit?.task_id === task.id` (computed `isRunningHere`): если юнит запущен здесь — кнопка «Стоп» + карточка подсвечена (`.running`); клик emit `start-unit` (TasksView открывает `StartUnitModal` для этой задачи через `startUnitTaskId`) или `stop-unit` (`unitsStore.stop()`). `cardColorStyle` теперь отдаёт ещё `--card-tag-accent` (для цветной полосы слева `.color-stripe`).
- **Фильтры** — `TaskFilters.vue` в виде M3-чипов (chip-group), шапка с счётчиком в pill, подвал со «Сбросить всё» + (на мобильном) «Показать результаты»; на мобильном — bottom-sheet (как было), сортировки скрыты (они в `SortSheet`). Empty-state с круглой иконкой-контейнером, заголовком, подписью и кнопкой создания (на вкладке active).

## v2.5.0 — присутствие на «Сотрудниках», давние задачи, Safari-фолбэк

**Присутствие через видимость вкладки.** Онлайн-статус теперь драйвится видимостью, а не только наличием сокет-соединения. Клиент шлёт `presence:visibility {visible}` на `visibilitychange`/`focus`/`pagehide` (`socket/index.js`). Сервер (`app/sockets/presence.py`) держит `_sid_user` + `_sid_visible` + множество `_online`; пользователь онлайн, пока есть хотя бы одно соединение с видимой вкладкой (`_has_visible_connection`). Статус и `last_seen_at` меняются только на ПЕРЕХОДЕ (`_set_online`) — это лечит мобильные: при сворачивании/блокировке экрана сокет «замораживается», дисконнект приходит поздно/теряется, а раньше last_seen проставлялся с большим запозданием. Теперь уход в фон сразу даёт точный last_seen. Сокет-хэндлер `presence:visibility` — в `events.py`.

**Онлайн/last seen на вкладке «Сотрудники».** `EmployeesView.vue` переиспользует presence из messenger-store (`isOnline`/`lastSeenOf`/`fetchPresence`): зелёная точка `.online-dot` на аватаре карточки и профиля, статус «в сети»/`formatLastSeen()` под именем. Живые обновления — через глобальный сокет-хэндлер `presence:update`.

**Мини-чат как полноценный.** `MiniMessenger.vue` теперь показывает онлайн-точку в списке и в шапке треда + last seen в шапке (`threadOnline`/`threadLastSeen`). Прочтение работает как в основном: общий `activeConversationId` + `setActive()`/`isViewingActively()`. `fetchPresence()` дёргается при открытии панели.

**Напоминание о давних задачах.** `GET /api/tasks/stale?days=7` (`task_repo.get_stale(threshold)` — активные не-архивные задачи с `received_at` старше порога, сначала самые старые; в ответе `days_pending`). Фронт: composable `useStaleReminder.js` (singleton, раз в день — localStorage `gw2_stale_reminder_shown_date`), компонент `StaleTasksModal.vue` (M3, токены). Монтируется в `App.vue` после входа с задержкой 1.2с и только если не открыты тур/лог версий. Клик по задаче → `/tasks?open=<id>`; `TasksView` в `onMounted` читает `route.query.open` и открывает задачу, затем чистит query.

**Safari-фикс «белый экран».** Вся палитра — на `oklch()`/`color-mix()` (Safari 15.4+/16.2+). На старых iOS (старые iPhone на iOS 15.x) токены становятся невалидными при вычислении → пустой экран. Решение: `@supports not (color: oklch(0 0 0))` в КОНЦЕ `tokens.css` (после всех oklch-правил, чтобы перекрыть их) — плоский hex-фолбэк дефолтной классической темы для светлой и тёмной темы (семантические `--color-*`, `--tag-*`, scrim/shadow). Кастомные акценты на таких устройствах не применяются, но платформа видима и читаема. Доп. усиление: `viewport-fit=cover` в `index.html`, try/catch вокруг `localStorage.setItem` при init `theme.js`.

**Ребрендинг Grove → Groove.** Проект называется **Groove Work** (с двумя «о»). Заменены бренд/тексты/доки/заголовки swagger/alt логотипа/`GroovePreset` в `main.js`. DB-идентификаторы postgres `grovework` (имя БД/пользователь) НЕ трогали — их смена сломала бы существующий деплой.

## Swagger UI

Доступен на `http://localhost:5001/apidocs` при запущенном dev-сервере.

## Логи

JSON-формат в stdout. Docker забирает через `docker logs`.  
`FLASK_DEBUG=1` включает DEBUG-уровень с SQL-запросами.

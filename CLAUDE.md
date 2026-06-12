# Groove Work — Руководство по проекту

Здесь — текущая архитектура и правила. Историю изменений не ведём (для этого есть git и `back/data/changelog.json`): описываем «как есть», а не «как стало».

## Принципы разработки

1. **Простое лучше сложного.** Не плодим абстракции «на будущее». Три одинаковых строки лучше преждевременной фабрики.
2. **Оптимальность и эффективность важнее тормознутости.** Никаких N+1, лишних ререндеров, избыточных re-fetch.
3. **Расширяемость и масштабируемость важнее монолита.** Новый функционал — переиспользуемые блоки (composables, components, services, repositories), не однострочные хаки.
4. **Комментарии — только там, где они реально нужны.** Не описываем «что делает» очевидный код; «почему» — только если есть скрытый инвариант, обход бага или неочевидная мотивация.
5. **Профессиональный, лёгкий и краткий код.** Понятные имена, разумная декомпозиция, без раздувания.
6. **Только архитектура цветов и токенов.** Никаких `#hex` / `rgba()` в компонентах. Только `--color-*`, `--tag-*`, `--shadow-*`, `--radius-*` (см. «Цветовая система»).
7. **Дизайн един и согласован с тем, что уже есть.** Material 3 Expressive — стиль всего приложения. Не отклоняемся от стиля действующих экранов.
8. **При новых/переписываемых разделах — сначала ищем примеры лучших UI/UX подобного функционала в интернете** (ориентир — Material 3 Expressive от Google), и реализуем по образцу, адаптируя под проект.
9. **Дублирующиеся компоненты — выносим в общий.** Если используется в двух+ местах — `components/common/`, `composables/` или `utils/`. Без копипасты.
10. **Общение между микросервисами — ТОЛЬКО через gRPC.** Контракты в `api/proto`, стабы генерит `scripts/gen_proto.sh`. Никаких REST/HTTP-вызовов сервис→сервис. (REST микросервисов — только для клиентского трафика через nginx; Redis-мост `gw2:calls:events` — исторический канал доставки событий в Socket.IO, новые межсервисные вызовы через него не вести.)
11. **Changelog (`back/data/changelog.json`) — НЕ ТРОГАТЬ без явной команды пользователя.** Когда команда дана: никаких технических деталей, технологий и стека — только пользовательские последствия («стало быстрее, надёжнее и безопаснее»), кратко, в шуточном пользовательском тоне.
12. **CLAUDE.md — не помойка.** Здесь только актуальная архитектура и правила; историю версий сюда не писать.

## Что это

Groove Work — внутренняя multi-tenant платформа учёта времени задач, аналитики и коллаборации: компании, внутри которых ведутся задачи/юниты, мессенджер, звонки, геймификация «Мой Groove». Разграничение по ролям.

## Стек

| Слой | Технология |
|---|---|
| Бэкенд (монолит) | Python 3.12 · Flask 3 · SQLAlchemy 2 · Alembic |
| WebSocket | Flask-SocketIO + eventlet + Redis |
| Звонки (сервис) | Go 1.26 · go-kit · Fiber · gRPC · pgx (микросервис `back-go/calls`) |
| Auth (сервис) | Go 1.26 · go-kit · Fiber · pgx (микросервис `back-go/auth`) |
| Auth (токены) | PASETO v4: access — v4.public Ed25519 (15 мин), refresh — v4.local (30 дней, HttpOnly cookie) |
| Звонки (медиа) | LiveKit (SFU, сервис в docker-compose) + livekit-client |
| Валидация | marshmallow (Flask) |
| БД | PostgreSQL 16 (pgcrypto для паролей, pgvector) |
| Фронтенд | Vue 3 · Vite · PrimeVue · Pinia · Vue Router 4 |
| Инфра | Docker Compose · Nginx |
| API-документация | flasgger (Swagger UI на /apidocs — только Flask-эндпоинты) |

## Структура директорий

```
back/        — Flask-приложение
back-go/     — Go-микросервисы: calls (звонки), auth (авторизация и пользователи)
front/       — Vue 3 SPA
deploy/      — docker-compose{,.override,.prod}.yml, nginx, init_sql, .env.example
scripts/     — deploy_server.sh, gen_proto.sh, reset_superadmin_password.sh
```

## Архитектура

Три сервиса над общей PostgreSQL (схему всех таблиц ведёт Alembic на стороне Flask) + Redis + LiveKit:

- **Flask-монолит** (`back/`, :5000 в docker / :5001 в dev) — задачи, юниты, отделы, статистика, мессенджер, компании, Groove, YouGile, бэкап, Socket.IO и фоновые циклы.
- **callsvc** (`back-go/calls`) — вся бизнес-логика звонков + LiveKit (токены/комнаты/вебхуки). Транспорты: gRPC :9090 (ринг-фаза, зовёт Flask) и HTTP :8090 (REST `/api/calls/*`, вебхуки LiveKit, `/healthz`).
- **authsvc** (`back-go/auth`) — авторизация и пользователи. HTTP :8091: `/api/auth/*` (login/refresh/logout/change-default) и `/api/users/*` (CRUD, directory, me, аватарки, identicon, роли, reset-password), `/healthz`. gRPC нет — другим сервисам от него ничего не нужно (проверка токенов локальная).

**Слои Flask:** `Routes (Blueprints) → Services (бизнес-логика) → Repositories (SQL) → PostgreSQL`. Жёсткое правило: `request`, `g`, `response` из Flask не проникают глубже Routes. Папки `back/app/`: models / schemas / repositories / services / api / sockets / utils.

**Слои Go-микросервисов (одинаковые):** `internal/domain` (модели, порты, ошибки `{Code, Message, HTTPStatus}`) → `internal/service` (бизнес-логика на портах, тестируется фейками) → `internal/repository/...` (pgx, raw SQL; redis) → `internal/transport/...` (Fiber / gRPC) + `internal/endpoint` (go-kit обёртки use-case'ов). Формат ошибок REST: `{"error": CODE, "message": ...}` + HTTP-статус. Логи — slog JSON.

**Маршрутизация (nginx и vite-proxy, порядок важен — длинные префиксы раньше `/api/`):** `/api/calls/` → calls:8090; `/api/auth` и `/api/users` (БЕЗ хвостового слэша — `/api/users` без слэша тоже матчится) → auth:8091; остальной `/api/` → Flask; `/socket.io/` и `/uploads/` → Flask/nginx; `/livekit/` → LiveKit. Фронт ходит только на относительные пути.

## Авторизация (PASETO + authsvc)

- **Access-токен** — PASETO v4.public (Ed25519, TTL 15 мин): подписывает только authsvc (`PASETO_PRIVATE_KEY`, hex seed||pub); Flask (`pyseto`, `app/utils/paseto.py`) и callsvc (`aidanwoods.dev/go-paseto`) проверяют подпись по `PASETO_PUBLIC_KEY`. Скомпрометированный сервис-верификатор не может выпустить токен.
- **Refresh-токен** — PASETO v4.local (`PASETO_REFRESH_KEY`, TTL 30 дней), cookie `refresh_token` (HttpOnly, Secure, SameSite=Strict). Читает только authsvc.
- **Клеймы:** sub (id строкой), type=access, exp/iat, force_change, company_id, company_name, company_settings, role_level, is_root_admin.
- **Фронт токен НЕ декодирует** — authsvc дублирует клеймы в телах ответов login/refresh/change-default; стор кладёт их в `claims` (`applySession` в `stores/auth.js`); client.js обновляет их при каждом refresh. 401 → очередь запросов + refresh + повтор.
- **force_change:** пользователь с `is_default_pass=TRUE` получает `force_change: true` — все API (Flask и Go) возвращают 403 `FORCE_PASSWORD_CHANGE`, кроме `/api/auth/change-default` и logout. Дефолтный пароль — `<login>123`.
- **Flask-декораторы:** `utils/permissions.py` — `@require_role(min_level)`, `@require_auth`, `@require_company_scope`; внутри `verify_request_token()` + загрузка пользователя в `g.current_user` + проверка is_hidden и активности компании. id текущего пользователя в роутах — `request_user_id()` из `utils/paseto.py`.
- **Брутфорс-щит** — в authsvc (Redis `gw2:bf:attempts:{login}` / `gw2:bf:locked_until:{login}`): после каждых 5 неудач блокировка 10·2^(n−1) секунд, ответ 429 `{retry_after_sec}`, фронт показывает таймер на LoginView. Redis недоступен → fail-open.
- **Пароли** — pgcrypto в PostgreSQL: `crypt(pw, gen_salt('bf'))`, проверка `crypt(pw, hash) = hash`.
- **Аватарки** — общий uploads-volume (`UPLOAD_FOLDER/avatars/`), наружу отдаёт nginx `/uploads/` (в dev — Flask); `avatar_path = NULL` → identicon `GET /api/users/<id>/identicon` (PNG 8×8 pixel-art от sha256(id), генерит authsvc).
- **Dev-ключи PASETO** захардкожены синхронно в: `dev.sh`, `Makefile`, `back/.flaskenv`, `back/.env.example`, `deploy/docker-compose.override.yml`, `back/tests/conftest.py` (public `15ef4397…3fe1`). Прод-ключи генерирует `deploy_server.sh` (пара Ed25519 — целиком, чтобы публичный соответствовал приватному).
- `front/src/api/auth.js` и `users.js` ведутся вручную (authsvc не публикует Swagger) — `npm run gen:api` их не трогает.

## Система прав (4 фиксированные роли, multi-tenant)

| Уровень | Роль | Особенности |
|---|---|---|
| 1 | Сотрудник (EMPLOYEE) | базовая работа с задачами/юнитами |
| 2 | Менеджер (MANAGER) | +управление чужими юнитами, отделы/типы юнитов, экспорт статистики |
| 3 | Руководитель (DIRECTOR) | +CRUD пользователей, роли ≤ своей |
| 4 | Администратор системы (ADMIN) | `company_id NULL`, работает со всеми компаниями (контекст через `?company_id=`) |

Роли фиксированы в БД, создавать/удалять нельзя. Константы: `app/utils/permissions.py` (Flask) и `domain.Level*` (authsvc); фронт — `composables/usePermission.js` (уровень из `auth.user.role.level`).

**Гарды управления пользователями** (в `back-go/auth/internal/service`): нельзя назначить роль выше своей (равную — можно); нельзя скрыть/разжаловать себя, корневого Администратора (`is_root_admin`), последнего видимого админа; корневого Руководителя компании (`companies.director_id`) скрывает/разжалует только Администратор системы.

**Company-scope:** обычным пользователям компания навязывается (`user.company_id`); Администратор системы передаёт `?company_id=` (фронт подмешивает автоматически — `COMPANY_SCOPED_PREFIXES` в `api/client.js`). Отключённая компания → 403 `COMPANY_DISABLED` на уровне декораторов/middleware.

## Ключевые бизнес-правила

- У пользователя единовременно только 1 активный юнит (`datetime_end IS NULL`); нельзя архивировать задачу с активным юнитом.
- Удаление типа юнита каскадно удаляет все юниты с этим типом.
- Цвет задачи индивидуален для пользователя (`user_task_colors`); в сокет-броадкастах `task:created/updated` поле `color` вырезается, чтобы чужие клиенты не перезаписали свой.
- Собственные действия пользователя на фронте — оптимистичные обновления стора; сокет-события дублируют их для остальных (handlers идемпотентны).

## WebSocket и presence

Клиент передаёт access-токен в auth-payload Socket.IO (или query) при handshake; сервер верифицирует PASETO и присоединяет к комнатам `all` и `user_{id}`. Мутации (задачи, юниты) эмитятся в `all`; company-scoped события несут `company_id` — клиент фильтрует в сторе. Presence — in-memory в процессе Flask (один app-контейнер с eventlet — ок; при нескольких процессах выносить в Redis): онлайн = есть видимая вкладка (`presence:visibility` + heartbeat + sweeper), `last_seen_at` пишется на переходе в офлайн.

## Звонки (callsvc + LiveKit)

- Весь медиа-транспорт — LiveKit (`livekit/livekit-server:v1.9`, сигнальный WS через nginx `/livekit`, медиа-порты 7881/tcp + 7882/udp). Бизнес-логика — callsvc.
- **gRPC-контракт** `calls.v1.CallService` (`back-go/calls/api/proto/calls/v1/calls.proto`): StartCall / InviteToCall / AcceptCall / DeclineCall / LeaveCall / EndCall. Транспорт всегда OK; бизнес-ошибка — поле `error {code, message, http_status}`. Ответы несут готовый снапшот `Call` и списки адресатов — Flask эмитит сокет-события не читая БД. Стабы: `scripts/gen_proto.sh` (Go → `gen/callspb`, Python → `back/app/grpc`), результат коммитится.
- **Flask — тонкий шлюз:** `sockets/call_events.py` → `services/calls_client.py` (ленивый singleton-канал, `CALLS_GRPC_ADDR`, вызовы через `eventlet.tpool`; недоступность → `call:error {code:'CALLS_UNAVAILABLE'}`). Домен мессенджера остаётся во Flask: парный диалог создаётся ДО StartCall, системная плашка `kind='call'` и `message:updated` — `emit_call_system_message_update`.
- **Обратный канал:** callsvc публикует в Redis `gw2:calls:events` (`call_ended`, `call_status_changed`), мост `sockets/call_bridge.py` ретранслирует в Socket.IO — так события вебхуков LiveKit доезжают до клиентов.
- Ринг-state — in-memory в callsvc (`internal/ringstate`), восстанавливается из БД+LiveKit (`ReconcileStartup` и лениво в вебхуках). Лимит 9 участников, гости считаются в нём же. Identity: `u{user_id}` (metadata `{user_id, avatar_path}`), гости `g-{hex}` (`{guest:true}`).
- **Ссылки-приглашения:** `/{origin}/call/<share_code>` — публичный роут; `GET/POST /api/calls/join/<code>` (гость представляется именем, авторизованный входит под собой — optional JWT-заголовок).
- **Фронт:** `services/livekit.js` (`CallRoomManager` поверх livekit-client), `stores/call.js` (фазы idle/incoming/outgoing/active, чат звонка — data-канал topic `chat`, outgoing-таймаут 45с, guard `handleEnded` по `call_id`), `CallView.vue` (mini-режим, перетаскивание, демонстрация экрана, панели участников/чата), `CallAudioSink.vue` — звук всех удалённых.
- Сокет-события клиенту: `call:started {call, livekit:{token,url}}`, `call:incoming`, `call:accepted`, `call:invited`, `call:ended` и т.д.

## Мессенджер

Диалоги 1:1 (`conversations`, уникальная пара user_a<user_b) + pet-чат (`is_pet_chat`, user_b NULL, бот-ответы `sender_id NULL + is_bot`). Сообщения: текст (Markdown-подсветка по выделению), вложения до 25 МБ (`UPLOAD_FOLDER/messages/`), ответы (`reply_to_id`, SET NULL), пересылка (файлы копируются физически — удаление одной копии не задевает другую), закрепление сообщений (общее для обоих) и чатов (личное), soft-delete «у себя/у всех» (обе стороны скрыли → физическое удаление + чистка файлов), прикрепление задач, запись экрана (getDisplayMedia → видео-вложение). REST `/api/messenger/*`, сокеты `message:new/read/deleted/pin`, `conversation:*`, `presence:update`.

Важные инварианты:
- Во всех unread/mark_read запросах фильтр отправителя — `or_(sender_id.is_(None), sender_id != me)`: иначе трёхзначная логика SQL молча теряет бот-сообщения pet-чата.
- Прочтение: открытый+сфокусированный чат отмечает read сразу; плюс на возврат фокуса вкладки и при отправке. `activeConversationId` общий у MessengerView и MiniMessenger.
- MiniMessenger — глобальный FAB поверх всего (z-index 10050, выше ActiveUnitModal) — можно отвечать, не закрывая активный юнит; скрыт на `/messenger`.
- Поле ввода (`MessageInput.vue`): на десктопе Enter отправляет, Shift+Enter — перенос; на тач-устройствах (`(hover: none) and (pointer: coarse)`) Enter — перенос строки, отправка только кнопкой.
- Уведомления: Web Notifications через Service Worker (`public/sw.js`) + Web Audio «бип»; разрешение и разогрев AudioContext — по первому жесту.

## Мой Groove (геймификация)

Роут `/groove`, все таблицы company-scoped. Лента `feed_events` (события пишутся хуками `feed_service.on_*` ПОСЛЕ коммита основных сервисов; обёртка `_safe` гасит ошибки — геймификация не роняет основной флоу). Реакции (фикс. набор в `schemas/groove.py` ≡ `utils/groove.js`), комментарии (1 уровень ответов), кудосы. Питомцы `pets`: грувы за работу с **дневными капами по источникам** в Redis-hash (fail-open), кормление → XP → стадии, эволюция пересчитывает вид по паттерну юнитов, болезнь при простое (XP замораживается, уровень не теряется), характер по ритму работы, личный чат-бот. Рейд недели (`groove_raids`, цель ×1.2 от прошлой недели). Зоопарк, магазин (+сезонные товары), заряды ⚡ (лимит 10/день). AI-фичи (`groove_ai_service`, при `company.ai_enabled`): бот-комментарии, утренний дайджест, реплики кормления; фоновые циклы поднимаются в `create_app`. Wrapped «Моя неделя» (`GET /api/groove/wrapped`) — сторис-карточки. Сокеты в `all` с `company_id` в payload.

## ТВ-режим

Роут `/tv` (`meta.fullscreen`), Live-newsroom: grid `header / progress / canvas / ticker`, canvas = `KPI rail | stage | aside`. Все размеры через `clamp()` + vmin — никаких скроллов на любых пропорциях; portrait-режим перестраивает раскладку. Слайды описаны данными в массиве `slides[]` (kind: hero-number / podium / ranking / departments / quad / brand / groove), count-up анимации (`TvCount`), springy-бары, тикер. Данные кешируются по периодам, refresh раз в 60с. Только семантические токены.

## YouGile-интеграция

Per-company настройка (Руководитель подключает компанию/проект/доску; ключи пользователей и компаний шифруются Fernet — `YOUGILE_ENC_KEY`, `AI_KEY_ENCRYPTION_KEY` в env; ротация ключей = потеря привязок). Импорт карточек по короткой ссылке (поиск включает BFS по подзадачам), экспорт задач, двусторонняя синхра через вебхук `POST /api/yougile/webhook/<companyId>/<secret>` (нужен `YOUGILE_WEBHOOK_PUBLIC_BASE`). Обмен с YG — в фоне, не блокирует API.

## Цветовая система фронтенда

`front/src/assets/tokens.css` — Material You Expressive / M3, слои:
1. `--ref-*-h/c/l` — параметры цвета (пишет `theme.js`)
2. `--_p-*`, `--_s-*`, `--_n-*` — тональные палитры OKLCH (нейтральная гамма — от необязательного `neutral` темы)
3. `--color-*` — семантические токены (primary, surface, error, success…)

`[data-dark="true"]` — тёмная тема. Режим оформления: light | dark | **system** (следует за `prefers-color-scheme`, живое переключение) — `stores/theme.js`, localStorage `gw_theme_mode`. Для старых iOS без oklch — hex-фолбэк через `@supports not (color: oklch(0 0 0))` в конце tokens.css.

**Цвета-теги задач:** 8 пастельных цветов, токены `--tag-<name>-surface/-border/-accent`; набор продублирован в `front/src/utils/taskColors.js` и `back/app/schemas/task.py`.

**Правило:** никаких `#hex`/`rgba()` в компонентах — только токены.

## Локальная разработка

```bash
./dev.sh             # одна команда: инфра в Docker + callsvc + authsvc (Go) + Flask :5001 + Vite :5173
# или по частям:
make dev-infra       # инфра в Docker: DB + Redis + LiveKit (:7880)
make dev-migrate     # flask db upgrade
make dev-calls       # callsvc (go run; gRPC :9090, HTTP :8090)
make dev-auth        # authsvc (go run; HTTP :8091)
make dev-back        # Flask :5001
make dev-front       # Vite :5173
make dev-stop        # остановить dev-контейнеры
make dev-stack       # ВЕСЬ стек в Docker (прод-подобно, фронт :8080)
make gen-proto       # перегенерировать gRPC-стабы после правки calls.proto
```

- **WebSocket:** dev-запуск Flask — ТОЛЬКО `python wsgi.py` (eventlet.monkey_patch + socketio.run, debug=False намеренно). `flask run`/werkzeug не поддерживает WS-upgrade. Auto-reload нет — перезапуск руками.
- **Compose (deploy/):** база `docker-compose.yml` (все сервисы + healthchecks, цепочка depends_on: db/redis → app (миграции в entrypoint) → calls/auth → nginx) + dev-оверлей `override` (порты инфры наружу; app/calls/auth/nginx за `profiles: [full]` — голый `up` поднимает только инфраструктуру) + прод-оверлей `prod` (только парой `-f ... -f ...`: обязательные секреты, TLS/certbot, nginx.prod.conf).
- **Тесты:** Go — `go test ./...` в back-go/{calls,auth} (фейки портов, без БД/Redis/LiveKit). Python — `cd back && ./venv/bin/pytest -q` (нужны проброшенные порты БД/Redis — `make dev-infra`; conftest сам подписывает PASETO dev-ключом, authsvc не нужен; E2E звонков — через in-process fake gRPC + SocketIOTestClient).
- Если БД не принимает пароль (старый pg_data volume): `docker exec deploy-db-1 psql -U grovework -d grovework -c "ALTER USER grovework WITH PASSWORD 'grovework_local';"` затем `make dev-migrate`.

## Деплой

```bash
cp .env.deploy.example .env.deploy   # один раз: SERVER_HOST, SSH_KEY
make push      # собрать (linux/amd64) и запушить образы в Docker Hub; only="app front" — выборочно
make deploy    # make push → git push → SSH → git reset --hard → scripts/deploy_server.sh
make logs s=auth|calls   make status   make restart s=...   make shell s=...
make backup    # pg_dump прод-БД → локально backups/gw2_<дата>.sql.gz (накат на dev-БД: gunzip -c ... | docker exec -i deploy-db-1 psql -U grovework -d grovework)
make reset NEWPASS='...'  # сброс пароля суперадмина (pgcrypto, без утечки в ps)
```

**Сервер образы НЕ собирает.** `scripts/build_push.sh` собирает их локально под `linux/amd64` (Go-стадии — нативный кросс через `$BUILDPLATFORM`, python/node — Rosetta) и пушит в ОДИН репозиторий Docker Hub `osipovskijdima/groove_work` с тегами `app` / `calls` / `auth` / `front` + версионными `<svc>-X.Y.Z` (версия из `front/package.json`). Откат: в `deploy/.env` на сервере `APP_TAG=app-3.6.0` (аналогично `CALLS_TAG`/`AUTH_TAG`/`FRONT_TAG`), затем pull+up. Приватный репозиторий → одноразовый `docker login` локально и на сервере.

`scripts/deploy_server.sh` (идемпотентен): 1) синк `deploy/.env` — недостающие секреты генерирует сам (DB_PASSWORD, SECRET_KEY, Fernet-ключи, LIVEKIT_*, пара PASETO Ed25519 целиком + PASETO_REFRESH_KEY), существующие НЕ перезаписывает, устаревшие (TURN_*, JWT_SECRET_KEY) вычищает, бэкапит .env; 2) ufw: 7881/tcp, 7882/udp; 3) `compose -f docker-compose.yml -f docker-compose.prod.yml pull` + `up -d --no-build --remove-orphans` + prune старых слоёв и build-кэша; 4) `nginx -t` + reload (конфиг bind-mounted); 5) health-чеки: apispec через nginx, callsvc (healthz + gRPC из app), authsvc (healthz + POST /api/auth/login через nginx → ожидается 400), `/livekit/`, TCP 7881. `--env-only` — только синк .env. `entrypoint.sh` app-контейнера сам гонит `flask db upgrade`. Подробности — `DEPLOY.md`. GitHub: https://github.com/DmitriyODS/gw2.git

## Версионирование

Версия = `front/package.json` + Swagger `info.version` (`back/app/__init__.py`) + первая запись `back/data/changelog.json`. Мини-версии за фиксы одного релиза не плодим. Правила changelog — принцип 11.

## Swagger и логи

Swagger UI: `http://localhost:5001/apidocs` (только Flask-эндпоинты; REST Go-сервисов в нём нет). Логи: Flask — JSON в stdout (`FLASK_DEBUG=1` включает DEBUG с SQL), Go — slog JSON; в docker — `docker logs` / `make logs`.

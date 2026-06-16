# Groove Work — Руководство по проекту

Здесь — текущая архитектура и правила. Историю изменений не ведём (для этого есть git и `data/changelog.json`): описываем «как есть», а не «как стало».

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
10. **Общение между микросервисами — ТОЛЬКО через gRPC.** Контракты в `api/proto`, стабы генерит `scripts/gen_proto.sh`. Никаких REST/HTTP-вызовов сервис→сервис. (REST микросервисов — только для клиентского трафика через nginx; Redis-каналы `gw2:<svc>:events` — только доставка сокет-событий клиентам через gatewaysvc, межсервисные вызовы через них не вести.)
11. **Changelog (`data/changelog.json`) — НЕ ТРОГАТЬ без явной команды пользователя.** Когда команда дана: никаких технических деталей, технологий и стека — только пользовательские последствия («стало быстрее, надёжнее и безопаснее»), кратко, в шуточном пользовательском тоне.
12. **CLAUDE.md — не помойка.** Здесь только актуальная архитектура и правила; историю версий сюда не писать.

## Что это

Groove Work — multi-tenant платформа учёта времени задач, аналитики и коллаборации. **Идентичность развязана с компаниями:** пользователь — самостоятельная сущность (регистрируется сам, не привязан ни к чему), может состоять в нескольких компаниях с разной ролью либо ни в одной (тогда ему доступны мессенджер/звонки/профиль/контакты). Внутри компании ведутся задачи/юниты, статистика и геймификация «Мой Groove»; мессенджер и звонки — кросс-компанийные. Любой пользователь создаёт свою компанию и становится её администратором. Платформой (всеми компаниями) управляет единственный супер-админ.

## Стек

| Слой | Технология |
|---|---|
| Бэкенд | Go 1.26 · go-kit · Fiber · gRPC · pgx — микросервисы calls, auth, messenger, ai, groove, tasks, gateway, push, mail (`back-go/<svc>`) |
| Пуш-уведомления | pushsvc (`back-go/push`): FCM HTTP v1 (Android), подписка на `gw2:*:events`, токены устройств в `device_tokens` |
| Почта | mailsvc (`back-go/mail`): SMTP-рассылка брендированных HTML-писем (подтверждение email), gRPC-сервер, stateless (без БД/Redis) |
| WebSocket | gatewaysvc (`back-go/gateway`): свой лёгкий WS-шлюз `/ws` + Redis pub/sub |
| Миграции | goose (`back-go/migrate`, run-once контейнер до старта сервисов) |
| Общий Go-модуль | `back-go/pkg` — pasetoauth, apierror, events, marshform, bootstrap + сгенерированные gRPC-стабы (`pkg/gen/<svc>pb`) |
| Auth (токены) | PASETO v4: access — v4.public Ed25519 (15 мин), refresh — v4.local (30 дней, HttpOnly cookie) |
| Звонки (медиа) | LiveKit (SFU, сервис в docker-compose) + livekit-client |
| Валидация | pkg/marshform (Go, формы и тексты ошибок marshmallow) |
| БД | PostgreSQL 16 (pgcrypto для паролей, pgvector) |
| Фронтенд | Vue 3 · Vite · PrimeVue · Pinia · Vue Router 4 |
| Инфра | Docker Compose · Nginx |

## Структура директорий

```
back-go/     — Go-микросервисы: calls (звонки), auth (авторизация и пользователи),
               messenger (чаты), ai (LLM-шлюз), groove (геймификация),
               tasks (задачи/юниты/статистика/YouGile), gateway (realtime-шлюз),
               push (пуш-уведомления FCM), migrate (goose-миграции) + общий модуль pkg
               (go.work для workspace-режима; replace ../pkg в go.mod каждого сервиса)
front/       — Vue 3 SPA
data/        — changelog.json (отдаётся статикой nginx / vite-плагином)
deploy/      — docker-compose{,.override,.prod}.yml, nginx, init_sql, .env.example
scripts/     — deploy_server.sh, gen_proto.sh, reset_superadmin_password.sh
uploads/     — dev-файлы пользователей (gitignored; в docker — общий volume)
```

## Архитектура

Десять Go-микросервисов (включая шлюз, push и mail) над общей PostgreSQL (схему ВСЕХ таблиц ведёт run-once контейнер `migrate` — goose, baseline + инкременты в `back-go/migrate/migrations`) + Redis + LiveKit. Python/Flask ликвидирован полностью.

- **gatewaysvc** (`back-go/gateway`) — realtime-шлюз (наследник Flask-SocketIO): WebSocket `/ws` (кадры `{"event", "data"}`, первый кадр — `{"event":"auth","data":{"token":<PASETO>}}`, ответ `_connected`/`_error`), комнаты `all`/`user_{id}`, presence в Redis (`gw2:presence:*`: visibility + heartbeat + sweeper, `last_seen_at` → users), ринг-фаза звонков (WS-команды `call:*` → gRPC callsvc) и доставка событий ВСЕХ Redis-каналов `gw2:*:events` клиентам (общий envelope `{event, rooms, payload}`). HTTP :8096: `/ws`, exact REST `GET /api/messenger/presence`, `/healthz`. Свои события публикует в `gw2:gateway:events` — единый путь доставки, готов к нескольким инстансам.
- **callsvc** (`back-go/calls`) — вся бизнес-логика звонков + LiveKit (токены/комнаты/вебхуки) + оркестрация плашки звонка в чате (gRPC msgsvc: EnsureDialog при p2p-старте, CreateCallMessage/GetCallMessage → события `message:new`/`message:updated` через свой Redis-канал). `calls.company_id` опционален (звонки возможны и между людьми без общей компании — проставляется общая компания участников, иначе NULL). gRPC :9090 (ринг-фаза, зовёт gateway) и HTTP :8090 (REST `/api/calls/*`, вебхуки LiveKit, `/healthz`).
- **authsvc** (`back-go/auth`) — идентичность, авторизация (вкл. публичную регистрацию), компании, членство, роли и REST-бэкап. HTTP :8091: `/api/auth/*` (login/register/select-company/switch-company/refresh/logout/change-default), `/api/users/*` (платформенный список — супер-админ; directory — члены активной компании или глобальный поиск `?all=1`; me, аватарки, identicon; CRUD сотрудников активной компании — администратор компании), `/api/companies/*` (создать — любой авторизованный; список всех/toggle-active — супер-админ; карточка/настройки/члены/инвайт — администратор этой компании, проверка `companyAuthority` в сервисе; regex ai-settings — в aisvc), `/api/roles`, `/api/backup/*` (export/import ZIP — супер-админ; ДЕСТРУКТИВНЫЙ импорт TRUNCATE CASCADE), `/healthz`. gRPC нет — проверка токенов у всех локальная.
- **msgsvc** (`back-go/messenger`) — мессенджер. HTTP :8092 (REST `/api/messenger/*`, кроме exact `/api/messenger/presence` — он в gateway) и gRPC :9092 (плашки звонков для callsvc, pet-чат для groovesvc). Сокет-события → Redis `gw2:messenger:events`.
- **aisvc** (`back-go/ai`) — LLM-шлюз: ключи компаний (Fernet, `AI_KEY_ENCRYPTION_KEY`), вызовы ProxyAPI/OpenAI (chat + embeddings), `task_embeddings` (pgvector, семантический поиск задач) + ТВ-факт дня (фоновый goroutine-цикл раз в час по компаниям с `ai_enabled`, кэш Redis `gw2:ai:tv_fact:{cid}`). HTTP :8093 (REST `/api/companies/<id>/ai-settings*` и `/api/ai/tv-fact`) и gRPC :9093 (Status/Chat/Embed/SemanticSearch/ReindexTask — зовут tasksvc и groovesvc; промпты и tools-циклы у вызывающих).
- **tasksvc** (`back-go/tasks`) — ядро платформы: задачи (избранное, личные цвета, комментарии, ответственный/этап), юниты (1 активный на пользователя), типы юнитов, этапы, отделы, статистика (common/extended/profile/user-tasks/responsibles/employees + xlsx-экспорт на excelize) и вся YouGile-интеграция (см. раздел). HTTP :8095 (REST `/api/tasks|units|unit-types|departments|stages|stats|yougile`). Поиск задач: при включённом AI — целиком семантический через gRPC aisvc (SemanticSearch, fail-open в LIKE), реиндекс эмбеддингов — ReindexTask; хуки геймификации — gRPC groovesvc (OnUnitStarted/OnUnitStopped/OnTaskClosed, fire-and-forget). Сокет-события → Redis `gw2:tasks:events`.
- **groovesvc** (`back-go/groove`) — весь «Мой Groove» (см. раздел ниже). HTTP :8094 (REST `/api/groove/*`) и gRPC :9094 (хуки доменных событий: tasksvc — юниты/задачи и YouGile-вебхук, msgsvc — pet-чат). События → Redis `gw2:groove:events`; фоновые циклы заботы, AI и погоды (Open-Meteo) — внутри.
- **pushsvc** (`back-go/push`) — пуш-уведомления (FCM, Android). HTTP :8097 (REST `/api/push/register|unregister` — токены устройств в `device_tokens`, RequireToken; `/healthz`). Подписан на Redis-каналы `gw2:messenger:events` (message:new), `gw2:tasks:events` (task:created → ответственному) и `gw2:gateway:events` (call:incoming → high-priority). Шлёт FCM HTTP v1 (OAuth2 из service-account JSON, env `FIREBASE_CREDENTIALS_JSON`; без ключа отправка отключена) ТОЛЬКО офлайн-получателям (presence `gw2:presence:online`) — приложение на переднем плане получает событие по WS (FCM-first). Мёртвые токены чистятся по ответу FCM. gRPC нет.
- **mailsvc** (`back-go/mail`) — рассылка писем. gRPC :9098 (`mail.v1.MailService.Send` — `{to, to_name, template, params}`: рендерит брендированный HTML-шаблон `internal/service/templates/*.html` и шлёт по SMTP) + HTTP :8098 `/healthz`. Stateless (без БД/Redis). Зовёт его authsvc (письмо подтверждения email). SMTP-параметры — env `SMTP_HOST/PORT/USER/PASSWORD/FROM/FROM_NAME/TLS` (`starttls|tls|none`); в dev письма ловит контейнер `mailpit` (UI :8025). Наружу не торчит (gRPC-only, в nginx не проксируется). Шаблоны писем — исключение из правила токенов: инлайн-CSS с хардкодом цветов бренда (почтовые клиенты не понимают CSS-переменные).

**Слои Go-микросервисов (одинаковые):** `internal/domain` (модели, порты, ошибки) → `internal/service` (бизнес-логика на портах, тестируется фейками) → `internal/repository/...` (pgx, raw SQL; redis) → `internal/transport/...` (Fiber / gRPC) + `internal/endpoint` (go-kit обёртки use-case'ов). Формат ошибок REST: `{"error": CODE, "message": ...}` + HTTP-статус (исторический формат REST звонков — ключ `"code"`). Логи — slog JSON.

**Общий модуль `back-go/pkg`** (подключён `replace ../pkg` в go.mod каждого сервиса; `back-go/go.work` — для workspace-режима локально): `pasetoauth` (Verifier + Fiber-мидлвари RequireAuth/RequireToken/RequireRole/RequireSuperAdmin/OptionalUserID + порт AuthSource; claims — `is_super_admin` + опциональный контекст активной компании, `RequireRole` требует активную компанию, супер-админ в неё не проходит), `apierror` (тип `Error{Code, Message, HTTPStatus, Extra}` + `Respond`; доменные `errors.go` сервисов — алиасы на него), `events` (публикатор `gw2:<svc>:events`), `marshform` (разбор JSON-тел в формах marshmallow — типы значений и тексты ошибок), `bootstrap` (env, slog, pgx/redis, graceful shutdown через `Run(Component...)`), `gen/<svc>pb` — ЕДИНСТВЕННОЕ место сгенерированных Go-стабов всех контрактов (их импортируют и владелец, и клиенты — без перекрёстных module-зависимостей сервисов). В pkg только инфраструктура — домен туда не выносить.

**Маршрутизация (nginx и vite-proxy, порядок важен — длинные префиксы раньше `/api/`):** `/api/calls/` → calls:8090; `/api/auth`, `/api/users`, `/api/roles`, `/api/backup` и `/api/companies` (БЕЗ хвостового слэша) → auth:8091; regex `^/api/companies/\d+/ai-settings` → ai:8093 (regex-location выигрывает у префикса companies; в vite regex-ключ стоит раньше префикса); `/api/ai` → ai:8093; exact `/api/messenger/presence` → gateway:8096, остальной `/api/messenger` → messenger:8092; `/api/groove` → groove:8094; `/api/tasks`, `/api/units`, `/api/unit-types`, `/api/departments`, `/api/stages`, `/api/stats`, `/api/yougile` → tasks:8095 (вебхук `POST /api/yougile/webhook/...` — без авторизации); `/api/push` → push:8097; exact `/api/changelog` — статика nginx (bind-mount `data/changelog.json`; в dev отдаёт vite-плагин serve-changelog); незнакомый `/api/*` — 404 от nginx; `/ws` → gateway:8096 (WebSocket; в dev фронт ходит напрямую на ws://<хост-страницы>:8096/ws — хост из адреса страницы, чтобы работал заход с других устройств сети); `/uploads/` — nginx из volume (в dev — vite-плагин serve-uploads из `uploads/`); `/apps/` — статика nginx (APK мобильного приложения + `version.json` с номером сборки; bind-mount `apps/`, в dev — vite-плагин serve-apps; APK заливает `make deploy-apk`, проверка обновлений и кнопка «Скачать apk» ходят сюда); `/livekit/` → LiveKit. Фронт ходит только на относительные пути.

## Авторизация (PASETO + authsvc)

- **Access-токен** — PASETO v4.public (Ed25519, TTL 15 мин): подписывает только authsvc (`PASETO_PRIVATE_KEY`, hex seed||pub); остальные сервисы (общий `pkg/pasetoauth`) проверяют подпись по `PASETO_PUBLIC_KEY`. Скомпрометированный сервис-верификатор не может выпустить токен.
- **Refresh-токен** — PASETO v4.local (`PASETO_REFRESH_KEY`, TTL 30 дней), cookie `refresh_token` (HttpOnly, Secure, SameSite=Strict). Читает только authsvc.
- **Клеймы:** sub (id строкой), type=access, exp/iat, force_change, is_super_admin и ОПЦИОНАЛЬНЫЙ контекст активной компании (company_id, company_name, company_settings, role_level). «Нет активной компании» (company_id == nil, role_level 0) — НОРМАЛЬНОЕ состояние (мессенджер/профиль/контакты), а НЕ признак админа. Платформенный супер-админ — отдельный флаг `is_super_admin`, не роль.
- **Регистрация публичная с подтверждением email:** `POST /api/auth/register` (`{fio, email, login?, password}`) — самостоятельное создание аккаунта без компании. Логин предлагается транслитом из ФИО (`GET /api/auth/suggest-login?fio=…`: 6 букв фамилии + `.` + буква имени + буква отчества, фронт подставляет в редактируемое поле; пустой логин сервис генерит сам, коллизия → числовой суффикс), пароль фронт генерит на клиенте. Регистрация сессию НЕ выдаёт: создаёт `users.email_verified=FALSE`, пишет код+токен в `email_verifications` и шлёт письмо через mailsvc, ответ `{status:"verification_required", email}`. Подтверждение — `POST /api/auth/verify-email` (`{token}` из ссылки `/{origin}/verify-email?token=…` ИЛИ `{email, code}`): ставит `email_verified=TRUE`, удаляет запись, выдаёт сессию (как login). Переотправка — `POST /api/auth/resend-verification` (троттл 60 с, тихо). Логин неподтверждённого аккаунта → 403 `EMAIL_NOT_VERIFIED` (фронт ведёт на экран кода). Существующие аккаунты и сотрудники, заведённые администратором (force-change вместо верификации), считаются подтверждёнными (`email_verified=TRUE`). Login при 0 компаний не падает (сессия без активной компании); 1 компания — автоактивна; >1 — login-gate (select-company).
- **Сброс пароля по email:** `POST /api/auth/forgot-password {email}` (ответ всегда ok — не раскрываем аккаунт; троттл 60 с) → токен в `password_resets` + письмо `reset_password` через mailsvc со ссылкой `/{origin}/reset-password?token=…`. `POST /api/auth/reset-password {token, new_password}` → меняет пароль, гасит токен, возвращает `{login}` (фронт ведёт на login без автологина, экраны `/forgot-password`, `/reset-password`).
- **Раздел «Компании» (управление):** маршруты фронта `/companies` (список: супер-админ — все компании платформы; обычный — `GET /api/companies/mine`, где он администратор/создатель) и `/companies/:id` (`CompanyManageView` — табы Обзор/Участники/Настройки/Опасная зона). Настройки компании (ИИ, выходные, «Мой Groove», ссылка-приглашение, фичи, YouGile) переехали сюда из `SettingsView` (там остались только платформенно-личные: тема, личный YouGile, справка, о приложении, резервная копия для супер-админа). **Управление участниками/ролями и создание-редактирование сотрудников компании (`POST/PATCH /api/companies/:id/users`, `…/members*`, invite-перевыпуск, удаление компании) — только СОЗДАТЕЛЬ компании (`companies.created_by`) или супер-админ** (`creatorAuthority` в сервисе); не-создатель-администратор видит компанию и правит её настройки, но участниками не управляет (`companyAuthority` — для чтения/настроек). Sidebar показывает пункт «Компании» при `canManageCompanies` (админ ≥1 компании или супер-админ). **Приглашения в компанию — двумя путями:** ссылка-код `/join/:code` (любой → роль Сотрудник) и **email-инвайт с ролью** (`POST /api/companies/:id/invites {email, role_id}` — создатель/супер-админ; токен в `company_invites`, письмо `company_invite` через mailsvc со ссылкой `/{origin}/invite/:token`; получатель: `GET /api/companies/invites/:token` превью + `POST …/accept` → членство с ролью + сессия; токен — capability, как ссылка-код).
- **Фронт токен НЕ декодирует** — authsvc дублирует клеймы в телах ответов register/login/refresh/change-default; стор кладёт их в `claims` (`applySession` в `stores/auth.js`); client.js обновляет их при каждом refresh. 401 → очередь запросов + refresh + повтор.
- **force_change:** пользователь с `is_default_pass=TRUE` получает `force_change: true` — все API возвращают 403 `FORCE_PASSWORD_CHANGE`, кроме `/api/auth/change-default` и logout. Дефолтный пароль — `<login>123`.
- **Брутфорс-щит** — в authsvc (Redis `gw2:bf:attempts:{login}` / `gw2:bf:locked_until:{login}`): после каждых 5 неудач блокировка 10·2^(n−1) секунд, ответ 429 `{retry_after_sec}`, фронт показывает таймер на LoginView. Redis недоступен → fail-open.
- **Пароли** — pgcrypto в PostgreSQL: `crypt(pw, gen_salt('bf'))`, проверка `crypt(pw, hash) = hash`.
- **Аватарки** — общий uploads-volume (`UPLOAD_FOLDER/avatars/`), наружу отдаёт nginx `/uploads/` (в dev — vite-плагин serve-uploads); `avatar_path = NULL` → identicon `GET /api/users/<id>/identicon` (PNG 8×8 pixel-art от sha256(id), генерит authsvc).
- **Dev-ключи PASETO** захардкожены синхронно в: `dev.sh`, `Makefile`, `deploy/docker-compose.override.yml` (public `15ef4397…3fe1`). Прод-ключи генерирует `deploy_server.sh` (пара Ed25519 — целиком, чтобы публичный соответствовал приватному).
- Все api-модули `front/src/api/*.js` ведутся вручную (Go-сервисы Swagger не публикуют; генератор gen-api.mjs удалён вместе с Flask).

## Система прав (идентичность развязана с компаниями)

**Пользователь — самостоятельная сущность** (таблица `users`: только идентичность — login/ФИО/пароль/контакты/аватар, `is_active`, `is_super_admin`). Он НЕ знает про компании. Принадлежность и роль живут ТОЛЬКО в связке `user_companies` `(user_id, company_id, role_id, post)`. Один аккаунт — во многих компаниях с разной ролью; «нет компании» — валидное состояние. Любой пользователь может создать компанию (`POST /api/companies`) и стать её администратором.

**Три роли В КОМПАНИИ** (фиксированы в БД, создавать/удалять нельзя; `domain.Level*`, фронт — `composables/usePermission.js`, уровень из `auth.roleLevel` = роль в активной компании):

| Уровень | Роль | Особенности |
|---|---|---|
| 1 | Сотрудник (EMPLOYEE) | базовая работа с задачами/юнитами |
| 2 | Менеджер (MANAGER) | +управление чужими юнитами, отделы/типы юнитов, экспорт статистики |
| 3 | Администратор (ADMIN) | +управление компанией: настройки, члены (add/role/remove), инвайт, CRUD сотрудников |

**Супер-админ — отдельный класс** (`users.is_super_admin`, единственный, бутстрап — `reset_superadmin_password.sh`/миграция из бывшего `is_root_admin`): видит все компании и всех пользователей, управляет компаниями (`GET /api/companies`, toggle-active, delete), но к компанийному контенту (задачи, грувики, YouGile, статистика) доступа НЕ имеет. Не является ролью и не проходит `RequireRole`.

**Активная компания** сессии (выбранная при login/switch) — в access-токене (`company_id`, `role_level`); переключение — `switch-company`. Хака `?company_id=` для админа БОЛЬШЕ НЕТ.

**Middleware (`pkg/pasetoauth`):** `RequireAuth` (любой залогиненный) → `RequireRole(level)` (нужна активная компания с ролью ≥ level — супер-админ не проходит) / `RequireSuperAdmin` (платформа). Отключённая компания → 403 `COMPANY_DISABLED`.

**Гарды управления** (в `back-go/auth/internal/service`): роль не выше своей (равную — можно); супер-админа изменять нельзя; защита «последнего администратора компании»; управление членами компании — её администратор (роль 3) или супер-админ (`companyAuthority`). Создатель компании автоматически получает членство-администратора.

## Ключевые бизнес-правила

- У пользователя единовременно только 1 активный юнит (`datetime_end IS NULL`); нельзя архивировать задачу с активным юнитом.
- Удаление типа юнита каскадно удаляет все юниты с этим типом.
- Цвет задачи индивидуален для пользователя (`user_task_colors`); в сокет-броадкастах `task:created/updated` поле `color` вырезается, чтобы чужие клиенты не перезаписали свой (в tasksvc — `dto.TaskBroadcast`).
- Собственные действия пользователя на фронте — оптимистичные обновления стора; сокет-события дублируют их для остальных (handlers идемпотентны).

## WebSocket и presence

Realtime — gatewaysvc (`back-go/gateway`), фронт подключается тонкой WS-обёрткой `front/src/socket/gateway.js` (socket.io-подобный API: on/emit/connected/disconnect; реконнект 1с→5с; emit при обрыве буферизуется и уходит после повторной авторизации). Протокол — JSON-кадры `{"event", "data"}`: первый кадр клиента `{"event":"auth","data":{"token":<PASETO>}}` (10с таймаут), сервер отвечает `_connected` (диспатчится слушателям как `connect`) или `_error` + close; дальше события в обе стороны как есть; пинг/понг WS — 25с/60с. Сервер присоединяет к комнатам `all` и `user_{id}`. Мутации приходят из сервисов через Redis-каналы `gw2:<svc>:events` (envelope `{event, rooms, payload}`) и доставляются вербатим; company-scoped события несут `company_id` — клиент фильтрует в сторе. Presence — в Redis (`gw2:presence:beats` ZSET + `gw2:presence:online` SET, мульти-инстанс-ready): онлайн = есть живая видимая вкладка (`presence:visibility` + heartbeat каждые 25с + sweeper раз в 15с, простой >60с — офлайн), `last_seen_at` пишется в users на переходе в офлайн, `presence:update` — только на переходах.

## Звонки (callsvc + LiveKit)

- Весь медиа-транспорт — LiveKit (`livekit/livekit-server:v1.9`, сигнальный WS через nginx `/livekit`, медиа-порты 7881/tcp + 7882/udp). Бизнес-логика — callsvc. NAT/VPN/мобильные сети, где прямой UDP закрыт, пробивает **встроенный в LiveKit TURN-relay** (только прод-оверлей: `turn.enabled`, TLS 5349 + UDP 3478, cert — общий Let's Encrypt-том nginx, домен `LIVEKIT_TURN_DOMAIN`); LiveKit сам анонсирует TURN клиентам с эфемерными кредами — правок в callsvc/фронте/Android не требуется, отдельный диапазон relay-портов не нужен (relay терминируется в SFU). Внешний coturn не используем.
- **gRPC-контракт** `calls.v1.CallService` (`back-go/calls/api/proto/calls/v1/calls.proto`): StartCall / InviteToCall / AcceptCall / DeclineCall / LeaveCall / EndCall. Транспорт всегда OK; бизнес-ошибка — поле `error {code, message, http_status}`. Ответы несут готовый снапшот `Call` и списки адресатов — gateway эмитит сокет-события не читая БД. Стабы: `scripts/gen_proto.sh` (Go → `back-go/pkg/gen/<svc>pb`), результат коммитится.
- **gateway — тонкий шлюз ринг-фазы:** WS-команды `call:*` → `internal/ring` → gRPC callsvc (`CALLS_GRPC_ADDR`); недоступность → `call:error {code:'CALLS_UNAVAILABLE'}`. Оркестрация плашки звонка — в callsvc: при p2p-старте сам создаёт парный диалог (gRPC msgsvc EnsureDialog), плашку `kind='call'` (CreateCallMessage → `message:new`) и обновляет её на каждом изменении статуса (GetCallMessage → `message:updated`) — fire-and-forget горутины, плашка звонок не роняет.
- **Обратный канал:** callsvc публикует `call:ended` / `message:updated` в Redis `gw2:calls:events` в общем envelope — gateway доставляет клиентам; так события вебхуков LiveKit доезжают до клиентов.
- Ринг-state — in-memory в callsvc (`internal/ringstate`), восстанавливается из БД+LiveKit (`ReconcileStartup` и лениво в вебхуках). Лимит 9 участников, гости считаются в нём же. Identity: `u{user_id}` (metadata `{user_id, avatar_path}`), гости `g-{hex}` (`{guest:true}`).
- **Ссылки-приглашения:** `/{origin}/call/<share_code>` — публичный роут; `GET/POST /api/calls/join/<code>` (гость представляется именем, авторизованный входит под собой — optional JWT-заголовок).
- **Фронт:** `services/livekit.js` (`CallRoomManager` поверх livekit-client), `stores/call.js` (фазы idle/incoming/outgoing/active, чат звонка — data-канал topic `chat`, outgoing-таймаут 45с, guard `handleEnded` по `call_id`), `CallView.vue` (mini-режим, перетаскивание, демонстрация экрана, панели участников/чата), `CallAudioSink.vue` — звук всех удалённых.
- Сокет-события клиенту: `call:started {call, livekit:{token,url}}`, `call:incoming`, `call:accepted`, `call:invited`, `call:ended` и т.д.

## Мессенджер

Весь домен — в msgsvc (`back-go/messenger`). Диалоги 1:1 (`conversations`, уникальная пара user_a<user_b; `company_id` НУЛЛАБЛЕ — переписка между людьми без общей компании разрешена) + pet-чат (`is_pet_chat`, user_b NULL, бот-ответы `sender_id NULL + is_bot`, требует активной компании) + dev-чат поддержки (адресаты — супер-админы). Сообщения: текст (Markdown-подсветка по выделению), вложения до 25 МБ (общий uploads-volume), ответы (`reply_to_id`, SET NULL), пересылка (файлы копируются физически), закрепление сообщений (общее) и чатов (личное), soft-delete «у себя/у всех» (обе стороны скрыли → физическое удаление + чистка файлов), прикрепление задач, запись экрана. REST `/api/messenger/*` (msgsvc :8092); сокет-события публикует в Redis `gw2:messenger:events` — gateway доставляет их в WS-комнаты вербатим. gRPC msgsvc: EnsureDialog/CreateCallMessage/GetCallMessage (callsvc, плашки звонков), PostBotMessage/ListRecentMessages (groovesvc, pet-чат). Сообщение хозяина в pet-чате → gRPC-хук msgsvc → groovesvc OnPetMessage (fire-and-forget).

Важные инварианты:
- Во всех unread/mark_read запросах фильтр отправителя — `sender_id IS NULL OR sender_id != me`: иначе теряются бот-сообщения pet-чата.
- Прочтение: открытый+сфокусированный чат отмечает read сразу; плюс на возврат фокуса вкладки и при отправке. `activeConversationId` общий у MessengerView и MiniMessenger.
- MiniMessenger — глобальный FAB поверх всего (z-index 10050, выше ActiveUnitModal) — можно отвечать, не закрывая активный юнит; скрыт на `/messenger`.
- Поле ввода (`MessageInput.vue`): на десктопе Enter отправляет, Shift+Enter — перенос; на тач-устройствах (`(hover: none) and (pointer: coarse)`) Enter — перенос строки, отправка только кнопкой.
- Уведомления: Web Notifications через Service Worker (`public/sw.js`) + Web Audio «бип»; разрешение и разогрев AudioContext — по первому жесту.

## Мой Groove (геймификация)

Весь домен — в groovesvc (`back-go/groove`), роут фронта `/groove`, все таблицы company-scoped. Лента `feed_events`: события пишут gRPC-хуки OnUnitStarted/OnUnitStopped/OnTaskClosed (зовёт tasksvc fire-and-forget ПОСЛЕ коммита, в том числе при закрытии задачи из YouGile-вебхука; ошибки только в лог — геймификация не роняет основной флоу) и OnPetMessage (зовёт msgsvc). Реакции (фикс. набор в `domain/consts.go` ≡ `utils/groove.js`), комментарии (1 уровень ответов), кудосы. Питомцы `pets`: грувы за работу с **дневными капами по источникам** в Redis-hash (fail-open, ключи `gw2:groove:*` сохранены с Flask-времён), кормление → XP → стадии, эволюция пересчитывает вид по паттерну юнитов, болезнь при простое (XP замораживается; простой считается в РАБОЧИХ днях — выходные компании `settings.weekend_days` (0=Пн…6=Вс, дефолт Сб+Вс) задаёт администратор компании, `GET/PUT /api/companies/<id>/weekend-settings` в authsvc; в выходной Грувик не заболевает, а брифинг/дайджест/pet-чат зовут отдыхать — mood `weekend`), характер по ритму работы, личный чат-бот (история и ответы — через gRPC msgsvc, LLM + tools-цикл статистики — через gRPC aisvc). Квест дня (детерминирован по (user_id, дата) — формула совпадает с Python `toordinal`). Рейд недели (`groove_raids`, цель ×1.2 от прошлой недели). Зоопарк, магазин (+сезонные товары), заряды ⚡ (лимит 10/день). AI-фичи (при `company.ai_enabled`): бот-комментарии, утренний дайджест, реплики кормления; фоновые циклы (care + AI) — goroutine'ы groovesvc. Wrapped «Моя неделя» (`GET /api/groove/wrapped`). **Погода Грувика:** пользователь задаёт локацию (таблица `user_locations`; REST `GET/PUT/DELETE /api/groove/location`, поиск города `GET /api/groove/geo` — прокси геокодинга Open-Meteo, чтобы фронт ходил только на наш API; UI — чип в PetCard + LocationDialog: геолокация браузера или поиск города); фоновый цикл `RunWeatherLoop` (30 мин) опрашивает Open-Meteo (бесплатно, без ключа, `internal/weather`), кэширует снимки в Redis `gw2:groove:weather:*`, значимые перемены (дождь/снег/гроза/туман/прояснение/жара ≥30°/мороз ≤−15°) → реплика Грувика в pet-чат (AI или статичные фразы; кулдаун 4 ч, только 8–22 МСК), текущая погода подмешивается в AI-промпты pet-чата и утреннего брифинга. Сокеты → Redis `gw2:groove:events`, в `all` с `company_id` в payload; личные (`pet:update`, `groove:zap`, `groove:stroke`) — в `user_{id}`. `front/src/api/groove.js`, как и все api-модули, ведётся вручную.

## ТВ-режим

Роут `/tv` (`meta.fullscreen`), Live-newsroom: grid `header / progress / canvas / ticker`, canvas = `KPI rail | stage | aside`. Все размеры через `clamp()` + vmin — никаких скроллов на любых пропорциях; portrait-режим перестраивает раскладку. Слайды описаны данными в массиве `slides[]` (kind: hero-number / podium / ranking / departments / quad / brand / groove), count-up анимации (`TvCount`), springy-бары, тикер. Данные кешируются по периодам, refresh раз в 60с. Только семантические токены.

## YouGile-интеграция

Целиком внутри tasksvc (`back-go/tasks`): инфраструктура — `internal/yougile` (HTTP-клиент REST v2 с ретраями 429/5xx, парсер ссылок на карточки, Fernet-крипто), бизнес-логика — `internal/service/yougile_*.go`, REST `/api/yougile/*` (формы ответов байт-в-байт с прежним Flask-блюпринтом, ошибки валидации — `{"error": "VALIDATION", "details": …}`). Per-company настройка (администратор компании подключает компанию/проект/доску; личные ключи пользователей шифруются Fernet — `YOUGILE_ENC_KEY` в env tasksvc, ротация ключа = потеря привязок). Импорт карточек по короткой ссылке (резолв `OIP1-2454` перебором колонок + BFS по подзадачам), экспорт задач, отвязка; системные комментарии в обе стороны. Двусторонняя синхра: исходящий пуш — внутренний вызов tasksvc после update/archive/restore (горутина, best-effort); входящая — публичный вебхук `POST /api/yougile/webhook/<companyId>/<secret>` (нужен `YOUGILE_WEBHOOK_PUBLIC_BASE` в env tasksvc; constant-time проверка секрета, неверный — 404). Антицикл: `yougile_sync_hash` (sha1 от title|deadline_ms|completed) — вебхук игнорит собственное эхо. Инвариант: задача с активным юнитом не архивируется даже по completed из YG. Закрытие из вебхука зовёт groovesvc OnTaskClosed.

## Цветовая система фронтенда

`front/src/assets/tokens.css` — Material You Expressive / M3, слои:
1. `--ref-*-h/c/l` — параметры цвета (пишет `theme.js`)
2. `--_p-*`, `--_s-*`, `--_n-*` — тональные палитры OKLCH (нейтральная гамма — от необязательного `neutral` темы)
3. `--color-*` — семантические токены (primary, surface, error, success…)

`[data-dark="true"]` — тёмная тема. Режим оформления: light | dark | **system** (следует за `prefers-color-scheme`, живое переключение) — `stores/theme.js`, localStorage `gw_theme_mode`. Для старых iOS без oklch — hex-фолбэк через `@supports not (color: oklch(0 0 0))` в конце tokens.css.

**Цвета-теги задач:** 8 пастельных цветов, токены `--tag-<name>-surface/-border/-accent`; набор продублирован в `front/src/utils/taskColors.js` и `domain.TaskColors` tasksvc.

**Правило:** никаких `#hex`/`rgba()` в компонентах — только токены.

## Локальная разработка

```bash
./dev.sh             # одна команда: инфра в Docker (+ mailpit) + миграции + 9 Go-сервисов + Vite :5173
# или по частям:
make dev-infra       # инфра в Docker: DB + Redis + LiveKit (:7880)
make dev-migrate     # goose-миграции (back-go/migrate)
make dev-calls       # callsvc (go run; gRPC :9090, HTTP :8090)
make dev-auth        # authsvc (go run; HTTP :8091)
make dev-messenger   # msgsvc (go run; gRPC :9092, HTTP :8092)
make dev-ai          # aisvc (go run; gRPC :9093, HTTP :8093)
make dev-groove      # groovesvc (go run; gRPC :9094, HTTP :8094)
make dev-tasks       # tasksvc (go run; HTTP :8095)
make dev-gateway     # gatewaysvc (go run; WS /ws + HTTP :8096)
make dev-push        # pushsvc (go run; HTTP :8097; FCM off без ключа)
make dev-mail        # mailsvc (go run; gRPC :9098, HTTP :8098; письма → mailpit :8025)
make dev-front       # Vite :5173
make dev-stop        # остановить dev-контейнеры
make dev-stack       # ВЕСЬ стек в Docker (прод-подобно, фронт :8080)
make gen-proto       # перегенерировать gRPC-стабы после правки *.proto (можно выборочно: scripts/gen_proto.sh groove)
```

- **Compose (deploy/):** база `docker-compose.yml` (все сервисы + healthchecks, цепочка depends_on: db/redis → migrate (run-once) → calls/auth/messenger/ai/gateway/push → groove → tasks → nginx) + dev-оверлей `override` (порты инфры наружу; приложения за `profiles: [full]` — голый `up` поднимает только инфраструктуру) + прод-оверлей `prod` (только парой `-f ... -f ...`: обязательные секреты, TLS/certbot, nginx.prod.conf).
- **Миграции:** goose, `back-go/migrate/migrations` (00001 — baseline всей схемы, снятый с головы прежних Alembic-миграций). Существующая БД с `alembic_version` усыновляется автоматически (baseline помечается применённым без выполнения); свежая БД накатывает baseline целиком. Новые изменения схемы — обычные goose-файлы `0000N_*.sql`.
- **Тесты:** Go — `go test ./...` в back-go/{calls,auth,messenger,ai,groove,tasks,gateway,push} (фейки портов, без БД/LiveKit; в tasks — паритет YouGile sync_hash/parser/apply, в gateway — miniredis + интеграционный WS-тест handshake/доставки, в push — маршрутизация событий в пуши + presence-гейт).
- Если БД не принимает пароль (старый pg_data volume): `docker exec deploy-db-1 psql -U grovework -d grovework -c "ALTER USER grovework WITH PASSWORD 'grovework_local';"` затем `make dev-migrate`.

## Деплой

```bash
cp .env.deploy.example .env.deploy   # один раз: SERVER_HOST, SSH_KEY
make push      # собрать (linux/amd64) и запушить образы в Docker Hub; only="gateway front" — выборочно
make deploy    # make push → git push → SSH → git reset --hard → scripts/deploy_server.sh
make logs s=auth|calls   make status   make restart s=...   make shell s=...
make backup    # pg_dump прод-БД → локально backups/gw2_<дата>.sql.gz (накат на dev-БД: gunzip -c ... | docker exec -i deploy-db-1 psql -U grovework -d grovework)
make reset NEWPASS='...'  # сброс пароля суперадмина (pgcrypto, без утечки в ps)
```

**Сервер образы НЕ собирает.** `scripts/build_push.sh` собирает их локально под `linux/amd64` (Go-стадии — нативный кросс через `$BUILDPLATFORM`, node — Rosetta) и пушит в ОДИН репозиторий Docker Hub `osipovskijdima/groove_work` с тегами `migrate` / `gateway` / `calls` / `auth` / `messenger` / `ai` / `groove` / `tasks` / `push` / `mail` / `front` + версионными `<svc>-X.Y.Z` (версия из `front/package.json`). Go-образы собираются из общего контекста `back-go/` (`-f back-go/<svc>/Dockerfile` — внутрь копируется и модуль pkg). Откат: в `deploy/.env` на сервере `GATEWAY_TAG=gateway-3.7.0` (аналогично `MIGRATE_TAG`/`CALLS_TAG`/`AUTH_TAG`/`MESSENGER_TAG`/`AI_TAG`/`GROOVE_TAG`/`TASKS_TAG`/`PUSH_TAG`/`MAIL_TAG`/`FRONT_TAG`), затем pull+up. Приватный репозиторий → одноразовый `docker login` локально и на сервере.

`scripts/deploy_server.sh` (идемпотентен): 1) синк `deploy/.env` — недостающие секреты генерирует сам (DB_PASSWORD, Fernet-ключи, LIVEKIT_*, пара PASETO Ed25519 целиком + PASETO_REFRESH_KEY), существующие НЕ перезаписывает, устаревшие (TURN_*, JWT_SECRET_KEY, SECRET_KEY, APP_TAG) вычищает, бэкапит .env; 2) ufw: 7881/tcp, 7882/udp (медиа) + 5349/tcp, 3478/udp (TURN) + 2b) UDP-буферы ядра (net.core.rmem_max/wmem_max=16 МБ, persistent в /etc/sysctl.d, для качества WebRTC под нагрузкой; глобальный sysctl, livekit перезапускается при первом подъёме); 3) `compose -f docker-compose.yml -f docker-compose.prod.yml pull` + `up -d --no-build --remove-orphans` + prune старых слоёв и build-кэша (миграции применит run-once контейнер `migrate` до старта сервисов); 4) `nginx -t` + reload (конфиг bind-mounted); 5) health-чеки: фронт через nginx, healthz всех сервисов изнутри контейнеров, маршруты presence/auth/companies/ai/tasks/changelog через nginx, `/livekit/`, TCP 7881 + TURN/TLS 5349. `--env-only` — только синк .env. Подробности — `DEPLOY.md`. GitHub: https://github.com/DmitriyODS/gw2.git

## Версионирование

Версия = `front/package.json` + первая запись `data/changelog.json`. Мини-версии за фиксы одного релиза не плодим. Правила changelog — принцип 11.

## Логи

Все сервисы — slog JSON в stdout; в docker — `docker logs` / `make logs s=<svc>`. Swagger'а нет: все api-модули фронта (`front/src/api/*.js`) ведутся вручную.

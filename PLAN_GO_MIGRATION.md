# План миграции Groove Work на Go (v4.0)

> **Документ — рабочий чек-лист.** По мере выполнения отмечаем пункты `[x]`.
> Источник принципов — соседний проект `~/projects/me_frined2` (Life Flow). Перед стартом каждой фазы перечитываем его `CLAUDE.md` и `auth`-сервис как референс.
>
> **Цель:** полностью переписать бэкенд gw2 с Python (Flask + SQLAlchemy + Flask-SocketIO) на Go (Fiber v3 + pgx v5 + go-kit + gRPC), мигрировать ID на uuidv7, разбить монолит на 6 микросервисов и переписать фронт на TypeScript с типизированным API-клиентом.

---

## Зафиксированные решения (НЕ меняем без явного согласования)

| # | Решение | Значение |
|---|---|---|
| 1 | Стратегия миграции | **Big bang** — фриз новых фич, полный переписывание, разовое переключение |
| 2 | Формат ID | **uuidv7** для всех новых таблиц; data-migration переводит существующие int → uuid |
| 3 | Архитектура | **Настоящие микросервисы** (6 шт.) + **gRPC** между ними (`buf` + protovalidate) |
| 4 | WebSocket | **Чистый WS** (`coder/websocket`), свой минималистичный протокол; фронт переписывает клиент |
| 5 | Фронт | **Инкрементальная миграция** в текущем `front/` (TS strict + openapi-fetch + новый WS-клиент) |
| 6 | Миграции БД | **Все в `deploy/migrations/<service>/`** (по образцу me_frined2), один goose-контейнер накатывает |
| 7 | БД | **Один кластер PostgreSQL 18**, разделение **по схемам на сервис** (`auth.*`, `users.*`, `tasks.*`, …) — компромисс между «одной БД» и изоляцией |
| 8 | AI | **Отдельный сервис `ai`** с pgvector; `tasks` дёргает его через gRPC |

### Предупреждение про п.5 (фронт)

Раз бэк ломается полностью (uuid + новые URL + новый WS-протокол), «инкрементальная миграция фронта» по факту = переделка большинства файлов. В каждой фазе переводим связку «сервис на Go → соответствующие views/api/stores на TS». Если в процессе окажется, что чище сделать `front-v2/`, а старый `front/` удалить — переключимся.

---

## Принципы разработки (из me_frined2 — ОБЯЗАТЕЛЬНО)

### Backend (Go)

1. **Clean Architecture**: `domain → repository → service → endpoint → transport`. Бизнес-логика не знает про Fiber/pgx, репозитории — про HTTP, домен — ни о чём.
2. **Простое лучше сложного.** Никаких преждевременных абстракций. Три одинаковых строки лучше преждевременной фабрики.
3. **Производительность.** Используем БД по максимуму: CTE, `FOR UPDATE SKIP LOCKED`, `RETURNING`, частичные индексы, материализованные представления для тяжёлых отчётов.
4. **Один TxManager** на сервис, репозитории через `Querier`-абстракцию подхватывают `pgx.Tx` из контекста.
5. **pgx v5 native, без ORM.** Никакого GORM. SQL руками через `pgx.CollectExactlyOneRow + RowToStructByName`.
6. **Пароли — `pgcrypto.crypt() + gen_salt('bf', 12)`** на стороне БД. Не хэшируем в Go.
7. **PASETO v4** для access-токенов (15 мин). **Opaque random refresh** (30 дней) с ротацией и reuse-detection. JWT запрещён.
8. **Transactional outbox** для писем и inter-service эвентов. `FOR UPDATE SKIP LOCKED` + экспоненциальный backoff.
9. **slog JSON.** Никакого logrus/zap.
10. **Конфиг через `caarlos0/env/v11`** — struct + env tags + `required`/`envDefault`.
11. **OpenAPI 3.1** — source of truth для REST; proto3 — для gRPC. Фронт типы генерит из них.
12. **Тесты на трёх уровнях:** unit (`*_test.go`), integration (`*_integration_test.go` + `//go:build integration` + testcontainers-go), e2e (`*_e2e_test.go` + `//go:build e2e` + `app.Test()`).
13. **Distroless multi-stage** Docker, pin-версии образов, `:latest` запрещён.
14. **Комментарии — только WHY.** Никогда WHAT.
15. **`make audit`** в каждом сервисе: `tidy/verify/vet/staticcheck/govulncheck` + unit-tests — quality gate для CI.

### Frontend (Vue 3 + TypeScript)

1. **TS strict.** Никакого JS.
2. **PrimeVue 4 styled mode** + кастомный preset на M3-токенах.
3. **M3 Expressive** через `@material/material-color-utilities` (Google). Никаких hex/rgba в шаблонах — только `var(--md-sys-color-*)`.
4. **`openapi-fetch` + `openapi-typescript`** — API-клиент типизирован из OpenAPI каждого сервиса. Никакого axios.
5. **Pinia 3 + persistedstate.** Никакого vuex.
6. **Vitest + Playwright.** Каждая view/composable — со spec'ом. Каждый user-facing flow — Playwright (desktop + Pixel 7).
7. **Без SCSS.** Чистый CSS + custom properties.
8. **Без Tailwind поверх PrimeVue.** PrimeVue в styled mode — единственный источник стилей.

---

## Архитектура целевой системы

```
┌──────────────────────────────────────────────────────────────────────────┐
│  Frontend (Vue 3 + TS + openapi-fetch + WS-client)                       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │  HTTPS + WSS
┌────────────────────────────────────┴─────────────────────────────────────┐
│                               nginx (gateway)                            │
└──┬──────────┬──────────┬──────────┬──────────┬──────────┬────────────────┘
   │/v1/auth  │/v1/users │/v1/tasks │/v1/msgr  │/v1/calls │/v1/ai
   ▼          ▼          ▼          ▼          ▼          ▼
┌──────┐  ┌──────┐  ┌──────┐  ┌──────────┐  ┌──────┐  ┌──────┐
│ auth │  │users │  │tasks │  │ messenger│  │calls │  │  ai  │  Go 1.25 + Fiber v3 +
│:8081 │  │:8082 │  │:8083 │  │  :8084   │  │:8085 │  │:8086 │  go-kit + pgx v5 + grpc
│:9091 │  │:9092 │  │:9093 │  │  :9094   │  │:9095 │  │:9096 │  (REST + gRPC порты)
└───┬──┘  └───┬──┘  └───┬──┘  └────┬─────┘  └──┬───┘  └──┬───┘
    └─────────┴─────────┴────gRPC──┴───────────┴─────────┘
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  PostgreSQL 18 (один кластер; схемы: auth, users, tasks, messenger,      │
│  calls, ai, public — citext, pgcrypto, pgvector, uuidv7)                 │
└──────────────────────────────────────────────────────────────────────────┘
┌────────────────────────┐   ┌──────────────────────────────────┐
│ Redis (presence cache, │   │ MailHog (dev) / SMTP-провайдер   │
│ rate-limit, WS-pubsub) │   │ (prod) для outbox-writer'а       │
└────────────────────────┘   └──────────────────────────────────┘
```

### Сервисная нарезка и ответственности

| Сервис | REST префикс | gRPC порт | Схема БД | Зависит от (gRPC) | LOC оценка |
|---|---|---|---|---|---|
| `auth` | `/v1/auth/*` | :9091 | `auth.*` | — | ~3000 |
| `users` | `/v1/users/*` | :9092 | `users.*` | `auth.VerifySession` | ~3500 |
| `tasks` | `/v1/tasks/*`, `/v1/units/*`, `/v1/unit-types/*`, `/v1/stages/*`, `/v1/stats/*`, `/v1/departments/*` | :9093 | `tasks.*` | `users.GetDirectory`, `ai.IndexTask` | ~6000 |
| `messenger` | `/v1/messenger/*` + WS `/ws/messenger` | :9094 | `messenger.*` | `users.GetProfile`, `users.GetDirectory` | ~3500 |
| `calls` | `/v1/calls/*` + WS `/ws/calls` | :9095 | `calls.*` | `users.GetProfile`, `messenger.SendSystemMessage` | ~3500 |
| `ai` | `/v1/ai/*` | :9096 | `ai.*` (pgvector) | `tasks.GetTaskByID`, `tasks.ListTasksByIDs` | ~2500 |

**Итого ~22K LOC Go против 7.5K Python** — это нормально, в Go всегда больше из-за явных типов, error handling, gRPC-моков.

### Структура `back-go/`

```
back-go/
├── go.work
├── Makefile                          # test/audit/lint/build/run для всех сервисов
├── proto/                            # source of truth для inter-service RPC
│   ├── auth/v1/auth.proto
│   ├── users/v1/users.proto
│   ├── tasks/v1/tasks.proto
│   ├── messenger/v1/messenger.proto
│   ├── calls/v1/calls.proto
│   └── ai/v1/ai.proto
├── buf.yaml, buf.gen.yaml
├── shared/
│   ├── pkg/
│   │   ├── logger/                   # slog JSON
│   │   ├── httperr/                  # маппер domain-error → HTTP
│   │   ├── pgxhelp/                  # TxManager, Querier, retry
│   │   ├── tenancy/                  # middleware: достаёт company_id из PASETO
│   │   ├── paseto/                   # issue/verify access
│   │   ├── uuidv7/                   # генерация на стороне Go
│   │   └── wssub/                    # WebSocket topic-subscriptions
│   └── proto/                        # сгенерированный код из buf
└── services/
    └── <svc>/
        ├── api/openapi.yaml          # REST контракт для фронта
        ├── cmd/<svc>/main.go
        ├── Dockerfile                # multistage distroless
        └── internal/
            ├── bootstrap/            # граф зависимостей (HTTP + gRPC server + gRPC clients)
            ├── config/               # caarlos0/env
            ├── domain/               # сущности + ошибки (sentinel errors)
            ├── endpoint/             # go-kit endpoints + middleware (logging, recovery)
            ├── repository/postgres/  # pgx-репозитории + querier.go + tx.go
            ├── service/              # бизнес-логика
            ├── token/                # PASETO + opaque refresh (только auth)
            ├── outbox/               # transactional outbox worker
            ├── mailer/               # SMTP (только auth и messenger)
            ├── transport/
            │   ├── http/             # Fiber-handlers
            │   ├── grpc/             # gRPC-handlers
            │   └── ws/               # WebSocket (только messenger, calls)
            └── testhelpers/          # testcontainers + миграции
```

---

# Этап 0 — Подготовка (S, ~3 дня)

**Backend**
- [ ] Создать этот файл `PLAN_GO_MIGRATION.md` (✅ создан)
- [ ] Создать ярлык/метку `GW2-V4` в трекере для всех задач, связанных с миграцией
- [ ] Заморозить новые фичи в текущем Python-бэке (только баг-фиксы)
- [ ] Создать папку `back-go/` рядом с `back/`
- [ ] Поднять отдельную dev-БД `gw2_v4_dev` (не трогаем существующую `grovework`)
- [ ] Поставить инструменты: Go 1.25, `buf`, `goose`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protovalidate-go`

**Frontend / DevOps**
- [ ] Создать папку `deploy-v2/` (новый docker-compose со всеми 6 сервисами)
- [ ] Подготовить `back-go/.gitignore`, `Makefile`-заготовку

---

# Этап 1 — Каркас + proto-контракты (M, ~5 дней)

> Все 6 сервисов поднимаются в docker-compose и отвечают `GET /health`. Никакой бизнес-логики, только инфра.

**`back-go/` каркас**
- [ ] `go.work` с `use ./shared/pkg` и всеми `./services/*`
- [ ] `Makefile` верхнего уровня: `build/test/audit/lint/fmt/gen` — таргеты универсальные, рекурсивно по всем сервисам
- [ ] `shared/pkg/logger/` (slog JSON, `New(level string) *slog.Logger`)
- [ ] `shared/pkg/httperr/` (общий маппер domain.Error → HTTP-статус + JSON-DTO)
- [ ] `shared/pkg/pgxhelp/` (`TxManager.Do(ctx, fn)`, `Querier`-интерфейс, `q(ctx, pool)`)
- [ ] `shared/pkg/tenancy/` (Fiber middleware: достаёт `company_id` из PASETO и кладёт в ctx)
- [ ] `shared/pkg/paseto/` (issue/verify v4 local)
- [ ] `shared/pkg/uuidv7/` (генерация на Go)

**Proto-контракты (ВСЕ сразу, как заголовочные RPC)**
- [ ] `proto/auth/v1/auth.proto` — `VerifySession`, `IssueServiceToken` (для serv-to-serv auth)
- [ ] `proto/users/v1/users.proto` — `GetProfile`, `GetDirectory`, `GetEmployees`, `PresenceSnapshot`
- [ ] `proto/tasks/v1/tasks.proto` — `GetTaskByID`, `ListTasksByIDs`, `GetActiveUnit`
- [ ] `proto/messenger/v1/messenger.proto` — `SendSystemMessage`
- [ ] `proto/calls/v1/calls.proto` — `GetActiveCall`
- [ ] `proto/ai/v1/ai.proto` — `IndexTask`, `SearchSimilarTasks`, `ReindexCompany`
- [ ] `buf.yaml`, `buf.gen.yaml`, `make gen/proto`

**Сервисные заготовки**
- [ ] Для каждого из 6 сервисов: `main.go`, пустой `bootstrap`, gRPC-server-заглушка, REST `/v1/health`, `Dockerfile` (distroless)

**Deploy**
- [ ] `deploy-v2/docker-compose.yml`: postgres 18 (init.sql с расширениями `pgcrypto`, `citext`, `pgvector`, схемы), redis, mailhog, swagger-ui, goose-migrate, и 6 сервисов
- [ ] `deploy-v2/Makefile`: `dev/up`, `dev/down`, `dev/fresh`, `dev/migrate`, `dev/logs/<svc>`, `dev/psql`, `dev/gen-key`
- [ ] Smoke: все 6 `/v1/health` отвечают, все gRPC-серверы слушают

---

# Этап 2 — Сервис `auth` (L, ~10 дней)

> Главный референс — `me_frined2/backend/services/auth`. Копируем структуру 1:1, адаптируем под нашу модель идентичности (login + телефон, `is_default_pass`).

**Миграции (`deploy-v2/migrations/auth/`)**
- [ ] `0001_init.sql` — схема `auth`:
  - `auth.credentials` (id uuidv7, login citext UNIQUE, phone citext, password_hash, is_default_pass, failed_attempts, locked_until, created_at, updated_at + trigger touch_updated_at)
  - `auth.refresh_sessions` (id uuidv7, credential_id FK, device_id, device_name, platform CHECK, ip inet, user_agent, token_hash, expires_at, revoked_at, last_seen_at, UNIQUE (credential_id, device_id) WHERE revoked_at IS NULL)
  - `auth.outbox` (id uuidv7, topic text, payload jsonb, status, attempts, available_at, last_error, created_at, sent_at)
  - `auth.verifications` (id, credential_id, kind ENUM('password_reset'), token_hash, expires_at, consumed_at)

**Domain**
- [ ] `domain/types.go` — Credential, RefreshSession, DeviceInfo, OutboxTopic, TokenPair
- [ ] `domain/errors.go` — ErrInvalidCredentials, ErrAccountLocked, ErrSessionNotFound, ErrTokenInvalid, ErrTokenExpired, ErrLoginTaken, ErrPhoneTaken, ErrCredentialNotFound

**Repository (`internal/repository/postgres/`)**
- [ ] `tx.go` + `querier.go` (по образцу me_frined2)
- [ ] `credentials.go` — Create, VerifyPassword (через `crypt()`), GetByID, GetByLogin, UpdatePassword, IncrementFailedAttempts, ResetFailedAttempts
- [ ] `sessions.go` — Upsert, RotateByTokenHash (с reuse-detection), ListActiveForCredential, RevokeByID, RevokeAllForCredential
- [ ] `outbox.go` — Enqueue, FetchBatchForDispatch (FOR UPDATE SKIP LOCKED), MarkSent, MarkFailed
- [ ] `verifications.go` — CreatePasswordReset, ConsumePasswordReset

**Token (`internal/token/`)**
- [ ] PASETO v4 local provider (15 мин TTL), `IssueAccess(claims)` / `Verify(token) → claims`
- [ ] `IssueRefresh() → (plaintext, sha256_hash, expiresAt)`, `HashRefresh(token) → sha256_hash`

**Service (`internal/service/`)**
- [ ] `Login(login, password, device) → TokenPair` (включая throttle: при ErrInvalidCredentials инкремент `failed_attempts`, при превышении — `locked_until`)
- [ ] `Refresh(refresh_token, device_id) → TokenPair` (с ротацией и reuse-detection)
- [ ] `Logout(credential_id, session_id)`, `LogoutAll(credential_id)`
- [ ] `Me(credential_id) → Credential`
- [ ] `ListSessions(credential_id, current_session_id) → []SessionView`
- [ ] `RevokeSession(credential_id, session_id)`
- [ ] `ChangeDefaultPassword(credential_id, new_password)` — для `is_default_pass=true` сценария
- [ ] `RequestPasswordReset(login)` (anti-enumeration — всегда 202), `ConfirmPasswordReset(token, new_password)`

**Endpoint (`internal/endpoint/`)**
- [ ] go-kit обёртки для всех Service-методов
- [ ] `LoggingMiddleware`, `RecoveryMiddleware`

**Transport**
- [ ] `transport/http/` — Fiber-handlers + DTO + AuthMiddleware (verify PASETO + кладёт credential_id/session_id в ctx)
- [ ] `transport/grpc/` — реализация `auth.v1.AuthService.VerifySession`
- [ ] `api/openapi.yaml` — REST-контракт (login, refresh, logout, logout-all, me, sessions, change-default, password-reset/*)

**Outbox**
- [ ] `outbox/worker.go` — тикер 2с, FOR UPDATE SKIP LOCKED, отправка через mailer
- [ ] `mailer/mailer.go` — SMTP (для password-reset писем)

**Тесты**
- [ ] Integration: credentials, sessions, outbox repos (testcontainers)
- [ ] E2E: полный flow login → refresh → logout через `app.Test()`

**Готовность фазы:** `scripts/smoke.sh` логинится тестовым юзером и обновляет токены через nginx.

---

# Этап 3 — Сервис `users` + tenancy (L, ~12 дней)

> Профили, компании, отделы, роли, аватары, identicon, каталог, presence-снапшот.

**Миграции (`deploy-v2/migrations/users/`)**
- [ ] `0001_users.sql`:
  - `users.companies` (id uuidv7, name, is_active, director_id, created_at)
  - `users.roles` (id smallint PK, level, name, display_name) — справочник, сид 4 ролей в этой же миграции
  - `users.profiles` (id uuidv7, auth_id uuid UNIQUE FK на `auth.credentials.id`, company_id FK, role_level smallint FK → roles.level, fio, position, is_hidden, is_root_admin, last_seen_at, avatar_path, created_at, updated_at)
  - `users.departments` (id uuidv7, company_id FK, name, sort_order)
  - Партишн-индекс на active company-членство, индексы для каталога

**Domain / Repository / Service / Endpoint / Transport**
- [ ] CRUD профилей с проверками прав (`role_level` ≥ собственный — запрещено)
- [ ] Identicon: Go-реализация 5×5 GitHub-style (от hash(uuid))
- [ ] Avatar upload: local FS, путь в БД, отдача через nginx `/uploads/`
- [ ] Импорт пользователей (CSV/Excel) — `xuri/excelize/v2`
- [ ] Эндпоинты: `/v1/users/me` (GET/PATCH), `/v1/users/` (CRUD), `/v1/users/<uuid>/identicon`, `/v1/users/directory` (для мессенджера), `/v1/users/employees`, `/v1/users/<uuid>/role` (PATCH), `/v1/users/<uuid>/reset-password`, `/v1/users/<uuid>/avatar` (POST/DELETE), `/v1/users/import` (POST)
- [ ] CRUD компаний и отделов

**Presence**
- [ ] In-memory map + Redis fallback (для горизонтального масштабирования)
- [ ] Heartbeat-обновление `last_seen_at` в БД на onDisconnect
- [ ] gRPC `PresenceSnapshot() → []user_id` для других сервисов

**gRPC**
- [ ] `users.v1.UsersService.GetProfile(auth_id)`, `.GetDirectory(company_id)`, `.GetEmployees(company_id)`, `.PresenceSnapshot(company_id)`

**Тесты**
- [ ] Integration: profiles, companies, departments
- [ ] E2E: CRUD пользователя + смена роли + identicon

---

# Этап 4 — Сервис `tasks` (XL, ~25 дней — самая большая фаза)

> Задачи, юниты, типы юнитов, отделы, стадии, комментарии, цвета задач, избранные, статистика, экспорт. Жёстко применяем TxManager для всех многотабличных операций.

**Миграции (`deploy-v2/migrations/tasks/`)**
- [ ] `0001_tasks.sql`:
  - `tasks.tasks` (id, company_id, department_id, stage_id, responsible_id, title, description, deadline, archived_at, received_at, created_at, updated_at)
  - `tasks.units` (id, task_id FK, user_id, unit_type_id, datetime_start, datetime_end, comment, created_at) + **partial UNIQUE на `(user_id) WHERE datetime_end IS NULL`** (один активный юнит)
  - `tasks.unit_types` (id, company_id, name, color, sort_order)
  - `tasks.stages` (id, company_id, name, color, sort_order)
  - `tasks.comments` (id, task_id FK, author_id, body, created_at, edited_at)
  - `tasks.favorites` (task_id, user_id, PRIMARY KEY (task_id, user_id))
  - `tasks.user_task_colors` (task_id, user_id, color)
  - `tasks.task_responsibles` (исторически — multiple responsible)
  - `tasks.outbox` (для эвентов в ai.IndexTask)
  - CHECK на «не архивировать задачу с активным юнитом» (через триггер)

**Service**
- [ ] CRUD задач (50+ методов суммарно)
- [ ] `StartUnit(user_id, task_id, type_id, comment)` — TX: создаёт юнит + обновляет `tasks.tasks.updated_at` + enqueue `task.unit_started` в outbox (для WS-broadcast)
- [ ] `StopUnit(unit_id, user_id)` — TX: проставляет datetime_end + enqueue эвент
- [ ] `Archive(task_id)` — TX: проставляет archived_at + проверяет, что нет активного юнита
- [ ] `Restore`, `SetStage`, `SetResponsible`, `SetColor` (per-user в `user_task_colors`)
- [ ] Стейтменс через CTE: top-сотрудник за период, отделы по часам, спарклайны календарей — НИКАКИХ N+1
- [ ] Excel-экспорт всех отчётов через `xuri/excelize/v2`
- [ ] `/v1/tasks/stale?days=N` — напоминание о давних задачах

**REST**
- [ ] ~40 endpoint'ов (1:1 с текущим Python — список см. `back/app/api/tasks.py`, `units.py`, `unit_types.py`, `stages.py`, `departments.py`, `stats.py`)

**gRPC**
- [ ] `tasks.v1.TasksService.GetTaskByID`, `.ListTasksByIDs`, `.GetActiveUnit(user_id)`

**Outbox → gRPC ai**
- [ ] После Create/Update/Archive задачи — outbox-event `task.changed` → отдельный воркер дёргает `ai.v1.AIService.IndexTask(task_id, text)` (с retry)

**Тесты**
- [ ] Integration: все репозитории (включая partial UNIQUE на активный юнит)
- [ ] E2E: полный сценарий start_unit → stop_unit → archive → restore через `app.Test()`

---

# Этап 5 — Сервис `messenger` + WS-инфра (XL, ~20 дней)

> 1:1 мессенджер: сообщения, вложения, reply, forward, pin (личный и общий), soft-delete, прочтения, presence. Здесь же — общая инфра для WebSocket, которую переиспользует `calls`.

**Миграции (`deploy-v2/migrations/messenger/`)**
- [ ] `0001_messenger.sql`:
  - `messenger.conversations` (id, user_a_id, user_b_id с user_a_id < user_b_id, hidden_for_a/b, pinned_at_a/b, last_message_at, last_message_id, UNIQUE (user_a_id, user_b_id))
  - `messenger.messages` (id, conversation_id, sender_id, text, kind ENUM('text','call','system'), reply_to_id FK self SET NULL, forwarded_from_user_id, hidden_for_a/b, pinned_at, pinned_by_id, read_at, created_at)
  - `messenger.attachments` (id, message_id FK nullable, file_path, mime, size, original_name)
  - Индексы: `(conversation_id, created_at DESC)`, partial `(conversation_id) WHERE pinned_at IS NOT NULL`

**WS-инфра в `shared/pkg/wssub/`**
- [ ] Хелпер для `coder/websocket`: топик-подписки (`user:<uuid>`, `conv:<uuid>`)
- [ ] Хендшейк: достать PASETO из query / Sec-WebSocket-Protocol → gRPC `auth.VerifySession` → положить identity в ctx
- [ ] Bounded-channel per-client с backpressure (drop старые если queue > N)
- [ ] Heartbeat-пинг каждые 25 сек, dead-conn detection
- [ ] Поддержка распределённой WS через Redis pubsub (на будущее)

**Service**
- [ ] CRUD сообщений: send, edit, soft-delete (per-side), hard-delete (когда обе стороны hid)
- [ ] Reply (валидация: цитируемое из того же диалога)
- [ ] Forward: копирование файлов вложений (новые `attachments`-записи), массовая рассылка
- [ ] Pin: личный (per-side) и общий (`pinned_at`/`pinned_by_id`)
- [ ] Прочтения с broadcast `message:read`
- [ ] Pinned list с курсорной пагинацией
- [ ] Загрузка файлов: 25 МБ лимит, `python-magic`-аналог (`gabriel-vasile/mimetype`)

**REST**
- [ ] ~25 endpoint'ов мессенджера

**WS-события на `/ws/messenger`**
- [ ] `message:new`, `message:read`, `message:deleted`, `message:pin`, `conversation:pin`, `conversation:deleted`, `presence:update`, `presence:visibility`

**gRPC**
- [ ] `messenger.v1.MessengerService.SendSystemMessage(conv_id, kind, payload)` — для calls

**Тесты**
- [ ] Integration: pin (general + per-side), soft-delete cascade
- [ ] E2E: WebSocket-clients (два sender'а в одном диалоге, broadcast прочтения, ретрит после reconnect)

---

# Этап 6 — Сервис `calls` (L, ~15 дней)

> WebRTC signaling, групповые звонки до 9 участников, rejoin grace, TURN credentials.

**Миграции (`deploy-v2/migrations/calls/`)**
- [ ] `0001_calls.sql`:
  - `calls.calls` (id, kind ENUM('p2p','group'), status ENUM('ringing','active','missed','ended'), media ENUM('audio','video'), conversation_id, initiator_id, started_at, ended_at)
  - `calls.participants` (call_id FK, user_id, role ENUM('initiator','invitee'), invited_at, joined_at, left_at, declined boolean)

**In-memory state (`internal/service/state.go`)**
- [ ] Перенос `back/app/sockets/call_state.py` 1:1 на Go: `_calls map[uuid]*CallState`, `_user_call map[uuid]uuid`
- [ ] `AddInvitee`, `ShouldEnd`, `RemoveUserFromCall`, `RemoveUserFromAnyCall` (с обязательным удалением из `invited` — иначе p2p не завершается, баг v2.6.2)
- [ ] Лимит 9 участников (1 инициатор + 8 приглашённых)

**Service**
- [ ] `Start(initiator_id, kind, media, invitees)`, `Accept`, `Decline`, `Leave`, `End` (только инициатор), `Invite` (расширение p2p → group), `Rejoin`
- [ ] Rejoin grace 15 сек через `time.AfterFunc` (заменяет socketio.start_background_task)
- [ ] Финализация зависших звонков при старте сервиса (callback `_finalize_stuck_calls` из текущего gw2)

**WS-события на `/ws/calls`**
- [ ] Incoming: `call:start/invite/accept/decline/leave/end`, `call:rejoin`, `webrtc:signal`, `call:media-state`
- [ ] Outgoing: `call:incoming/started/accepted/participant-joined/participant-left/ended/error`, `call:invited`, `webrtc:signal`

**REST**
- [ ] `/v1/calls/active`, `/v1/calls/history`, `/v1/calls/ice-servers` (HMAC-SHA1 для TURN credentials)

**Системные сообщения в чате**
- [ ] При accept/end: вызов `messenger.SendSystemMessage` через gRPC

**Тесты**
- [ ] Integration: state-машина (8 тестов по образцу `back/tests/test_call_state.py`)
- [ ] E2E: full flow start → accept → rejoin → end через WS-test-client

---

# Этап 7 — Сервис `ai` (M, ~10 дней)

> OpenAI embeddings, поиск похожих задач, TV-факты, support-inbox.

**Миграции (`deploy-v2/migrations/ai/`)**
- [ ] `0001_ai.sql`:
  - Расширение `pgvector` в init.sql postgres
  - `ai.task_embeddings (task_id uuid PK, vector vector(1536), updated_at)`
  - `ai.company_settings (company_id uuid PK, ai_enabled, model, api_key_enc, tv_facts_enabled, support_inbox_enabled)` — `api_key` шифруется AES-GCM мастер-ключом из env
  - `ai.tv_facts (id, company_id, body text, generated_at)`
  - `ai.support_inbox (id, company_id, user_id, body, severity, status, created_at)`
  - `ai.outbox` (для outgoing OpenAI вызовов с retry)

**Service**
- [ ] OpenAI-клиент: `sashabaranov/go-openai`
- [ ] `IndexTask(task_id, text)` — берёт текст задачи через gRPC `tasks.GetTaskByID`, считает embedding, upsert
- [ ] `SearchSimilarTasks(text, top_k)` — embedding запроса + KNN-запрос по pgvector
- [ ] `ReindexCompany(company_id)` — батч `tasks.ListTasksByIDs` + индексация
- [ ] TV-факты: фоновый воркер с `time.Ticker`, считает ближайший слот (как в `tv_facts_service.py`)
- [ ] Шифрование/дешифрование api_key через `crypto/cipher` AES-GCM

**REST**
- [ ] `/v1/ai/companies/<uuid>/settings` (GET/PUT)
- [ ] `/v1/ai/companies/<uuid>/reindex-tasks` (POST)
- [ ] `/v1/ai/companies/<uuid>/settings/test` (POST — проверка api_key)
- [ ] `/v1/ai/tv-fact` (GET)
- [ ] `/v1/ai/support-inbox` (GET)

**gRPC**
- [ ] `ai.v1.AIService.IndexTask`, `.SearchSimilarTasks`, `.ReindexCompany`

**Тесты**
- [ ] Integration: pgvector queries
- [ ] E2E: mock-OpenAI + полный цикл индексации

---

# Этап 8 — Фронт переезжает на новые API (XL, ~20 дней, параллельно с 3-7)

> Идёт **параллельно** с фазами 3-7 по мере появления новых ручек. К концу фазы 7 фронт должен быть готов к фазе 9.

**Подготовка**
- [ ] Включить `strict: true` в `front/tsconfig.json` (создать если нет)
- [ ] Добавить зависимости: `openapi-fetch`, `openapi-typescript`, `@material/material-color-utilities`, `@types/node`, `vue-tsc`
- [ ] Скрипт `front/scripts/gen-api.mjs` — тянет 6 `openapi.yaml` из `back-go/services/*/api/` и генерит `src/api/schema/{auth,users,tasks,messenger,calls,ai}.d.ts`
- [ ] `make gen/api` (или npm-script)

**Новый API-клиент**
- [ ] `src/api/client.ts` — singleton openapi-fetch'ев на каждый сервис
- [ ] Общий `authMiddleware`: подкладывает Bearer, ловит 401, single-flight refresh с очередью, retry оригинального запроса
- [ ] Заменить `src/api/auth.js` → `auth.ts`, `users.js` → `users.ts`, и т.д. (16 файлов)

**Новый WebSocket-клиент**
- [ ] Удалить `socket.io-client`
- [ ] `src/services/ws.ts` — менеджер двух соединений (`/ws/messenger`, `/ws/calls`), topic-subscriptions, heartbeat, auto-reconnect
- [ ] Конверт сообщений: `{event: string, data: object, request_id?: string}`
- [ ] Stores `messenger.ts`, `call.ts` подписываются на нужные топики

**Тема — миграция на M3 через color-utilities**
- [ ] `src/theme/palette.ts` — генератор из seed через `@material/material-color-utilities` (порт `me_frined2/frontend/src/theme/palette.ts`)
- [ ] `src/theme/tokens.ts` — M3 system colors → CSS-переменные `--md-sys-color-*`
- [ ] `src/theme/preset.ts` — PrimeVue preset поверх токенов
- [ ] Конвертация пользовательских тем из localStorage в seed-формат (миграция темы в коде)
- [ ] Удалить старый OKLCH-токенизатор и `front/src/assets/tokens.css`
- [ ] grep по hex/rgba в `.vue`/`.css` — должно быть пусто

**Views — постепенный перевод на TS**
- [ ] `LoginView.vue` → TS + новый `/v1/auth/login`
- [ ] `ProfileView.vue` → TS + новый `/v1/users/me`
- [ ] `EmployeesView.vue` → TS + новый `/v1/users/directory` + `/v1/users/employees`
- [ ] `TasksView.vue` → TS + новый `/v1/tasks/*` (UUID в URL)
- [ ] `StatsView.vue` → TS + новый `/v1/stats/*`
- [ ] `MessengerView.vue` → TS + новый `/v1/messenger/*` + новый WS-клиент
- [ ] `SettingsView.vue` → TS + новый `/v1/users/*`, `/v1/ai/*`, темы
- [ ] `CompaniesView.vue` → TS + новый `/v1/users/companies` (если оставляем под Администратором)
- [ ] `TvView.vue` → TS + `/v1/ai/tv-fact` + `/v1/stats/*`
- [ ] `ListsView.vue` → TS + соответствующие ручки

**Тесты**
- [ ] Vitest spec на каждый store (auth, messenger, call, tasks, theme, notifications, units, companies)
- [ ] Playwright e2e: login, tasks CRUD + start/stop unit, messenger send + read, calls smoke

---

# Этап 9 — Data-migration (M, ~7 дней)

> Решающий момент. Делаем один раз перед prod-переключением.

**Скрипт**
- [ ] `back-go/cmd/data-migrate/main.go` (или Python — что окажется быстрее) — открывает старую и новую БД
- [ ] Для каждой исторической таблицы: генерирует uuid_map (old_int_id → new_uuidv7), сохраняет во вспомогательной таблице `_migration_ids.<table>` в новой БД
- [ ] Заливает данные с трансформацией FK через uuid_map
- [ ] **Порядок строго:** companies → roles → users → departments → unit_types → stages → tasks → units → comments → favorites → user_task_colors → conversations → messages → attachments → calls → participants → embeddings → ai_settings → tv_facts
- [ ] Перенос файлов: avatars и messenger-attachments через cp с переименованием по новой uuid-схеме

**Валидация**
- [ ] Сравнение `count(*)` на каждой таблице old vs new
- [ ] Контрольные хэши: sha256 от текста всех сообщений собранных по conversation_id ORDER BY created_at
- [ ] Все аватары/вложения физически доступны через `nginx /uploads/`

**Прогоны**
- [ ] Минимум 2 прогона на dump production-БД в staging-окружении
- [ ] Финальный прогон за неделю до prod-переключения с timing'ом (важно знать, сколько займёт maintenance-окно)

---

# Этап 10 — Переключение и удаление Python (S, ~3 дня)

**Подготовка**
- [ ] Объявление maintenance-окна (2-4 часа, ночь воскресенья)
- [ ] Чек-лист отката (rollback) на случай катастрофы — `deploy/docker-compose.yml.backup` + dump БД

**Переключение**
- [ ] Бэкап prod gw2-БД (pg_dump + проверка восстановимости в staging)
- [ ] Прогон data-migrate скрипта на prod
- [ ] Подмена `deploy/docker-compose.yml` → `deploy-v2/docker-compose.yml`
- [ ] Рестарт стека
- [ ] Smoke `scripts/smoke.sh` против prod

**Период наблюдения (1 неделя)**
- [ ] Мониторинг логов всех 6 сервисов (slog → JSON в stdout, docker logs)
- [ ] Метрика: error-rate, p95/p99 latency, успешные refresh-ы, активные звонки
- [ ] Любой регресс — фиксим, не откатываем (если только не катастрофа)

**Финальная зачистка**
- [ ] `git rm -rf back/` (Python-бэк удалён)
- [ ] `git rm -rf deploy/` (старый docker-compose удалён) → `deploy-v2/` → `deploy/`
- [ ] `git rm` оставшихся `.js`-файлов на фронте (если есть), проверив через `grep -r "from.*\.js"`
- [ ] Обновить корневой `README.md`, `DEPLOY.md`, `CLAUDE.md` под новый стек

---

## Тесты — конвенция с первого дня

| Уровень | Файл | Зависимости | Команда |
|---|---|---|---|
| Unit (Go) | `*_test.go` | std lib | `make test` |
| Integration (Go) | `*_integration_test.go` + `//go:build integration` | testcontainers-go (Docker) | `make test/integration` |
| E2E (Go) | `*_e2e_test.go` + `//go:build e2e` | in-memory Fiber + grpc-test-client + WS-test-client | `make test/e2e` |
| Smoke | `deploy/scripts/smoke.sh` | живой docker-compose | `make dev/smoke` |
| Frontend unit | `src/**/*.spec.ts` | Vitest + happy-dom | `npm test` |
| Frontend e2e | `e2e/*.spec.ts` | Playwright desktop + Pixel 7 | `npm run test:e2e` |

`make audit` в каждом сервисе блокирует merge: `go mod tidy` + `verify` + `vet` + `staticcheck` + `govulncheck` + unit tests.

---

## Оценка сроков

| Фаза | Описание | Дней (1 человек full-time) |
|---|---|---|
| 0 | Подготовка | 3 |
| 1 | Каркас + proto | 5 |
| 2 | `auth` | 10 |
| 3 | `users` | 12 |
| 4 | `tasks` | 25 |
| 5 | `messenger` + WS-инфра | 20 |
| 6 | `calls` | 15 |
| 7 | `ai` | 10 |
| 8 | Фронт (параллельно с 3-7) | 20 (наложение −10) |
| 9 | Data-migration | 7 |
| 10 | Переключение | 3 |
| **Итого** | | **~110-120 рабочих дней ≈ 5-6 месяцев** |

---

## Риски и митигации

1. **Big bang × WS-протокол × фронт-инкремент.** Старый фронт не будет работать с новым WebSocket. Митигация: переключение в prod откладываем до полной готовности фронта (фаза 8 должна закрыться до фазы 9).
2. **5-6 месяцев фриза фичей.** Если бизнес не готов — пересмотреть стратегию (вернуться к strangler fig). Зафиксировать решение до старта.
3. **gRPC + 6 сервисов в одиночку.** Сложный onboarding. Митигация: на старте поднимаем 2 сервиса (`auth` + `users`), убеждаемся в работоспособности паттернов, дальше тиражируем.
4. **Data-migration ошибки** видны только на staging. Минимум 2 прогона до prod.
5. **OpenAI rate-limits / стоимость.** При полной переиндексации задач — следить за лимитами. Митигация: rate-limit в ai-сервисе через token bucket.

---

## Что НЕ делать (анти-паттерны)

- ❌ ORM в Go (никакого GORM/sqlx/ent).
- ❌ JWT. Только PASETO v4.
- ❌ Хэширование паролей в Go. Только `pgcrypto`.
- ❌ `:latest` Docker-теги в проде.
- ❌ Многословные комментарии. Только WHY.
- ❌ Hex/rgba в Vue-шаблонах и CSS. Только `var(--md-sys-color-*)`.
- ❌ `axios`. Только `openapi-fetch`.
- ❌ `vuex`. Только Pinia.
- ❌ SCSS. Чистый CSS.
- ❌ Tailwind поверх PrimeVue.
- ❌ Чужие схемы БД из сервиса (auth не лезет в users, и т.д.). Только через gRPC.

---

## Полезные ссылки

- `~/projects/me_frined2/CLAUDE.md` — обязательный референс перед каждой фазой
- `~/projects/me_frined2/backend/services/auth/internal/` — копируем структуру 1:1
- `~/projects/me_frined2/deploy/frontend/src/{api,theme,stores}/` — образец фронта
- `~/projects/gw2/PLAN_V3.md` — соседний план (v2 → v3 миграция), стиль документации
- `~/projects/gw2/CLAUDE.md` — нынешний контекст проекта (обновим в фазе 10)

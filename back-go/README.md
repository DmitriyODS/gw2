# back-go

Go-микросервисы платформы Groove Work v4.0. Это **новый бэкенд**, который заменяет Python-реализацию из `back/`. План миграции — корневой `PLAN_GO_MIGRATION.md`.

## Состав

```
back-go/
├── go.work               # workspace со всеми сервисами и shared/
├── Makefile              # test/audit/lint/build/run по всем сервисам
├── proto/                # source of truth для inter-service gRPC
│   ├── auth/v1/
│   ├── users/v1/
│   ├── tasks/v1/
│   ├── messenger/v1/
│   ├── calls/v1/
│   └── ai/v1/
├── buf.yaml, buf.gen.yaml
├── shared/
│   ├── pkg/              # logger, httperr, pgxhelp, tenancy, paseto, uuidv7, wssub
│   └── proto/            # сгенерированный код (buf generate)
└── services/
    ├── auth/             # PASETO + opaque refresh
    ├── users/            # профили, компании, отделы, presence
    ├── tasks/            # задачи, юниты, стадии, статистика
    ├── messenger/        # 1:1 чат + WebSocket
    ├── calls/            # WebRTC signaling + WebSocket
    └── ai/               # OpenAI embeddings, TV-факты, support-inbox
```

Структура каждого сервиса — по образцу `~/projects/me_frined2/backend/services/auth`.

## Стек (фиксирован)

- Go 1.25+, go workspaces
- Fiber v3 (REST), gRPC (inter-service), `coder/websocket` (realtime)
- go-kit endpoints + Clean Architecture
- pgx v5 native (без ORM)
- PostgreSQL 18+ (схема на сервис, общий кластер)
- PASETO v4 (без JWT), opaque refresh с ротацией
- pgcrypto для паролей (на стороне БД)
- goose-миграции в `deploy/migrations/<service>/`
- OpenAPI 3.1 — контракт для фронта
- slog JSON

## Команды

После Этапа 1 заработает:

```sh
make build                # сборка всех сервисов
make test                 # unit
make test/integration     # integration с testcontainers
make test/e2e             # e2e
make test/all             # всё
make audit                # tidy/verify/vet/staticcheck/govulncheck + tests
make gen/proto            # перегенерировать gRPC-код через buf
```

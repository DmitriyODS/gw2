# deploy-v2

Docker-compose стек **нового** Go-бэкенда платформы Groove Work v4.0. Параллельный с текущим `deploy/` (он остаётся работать со старым Python-бэком до Этапа 10 переключения).

## Состав

```
deploy-v2/
├── docker-compose.yml          # dev-стек (postgres + redis + mailhog + 6 сервисов)
├── docker-compose.prod.yml     # prod-overrides (Этап 10)
├── Makefile                    # dev/up, dev/migrate, dev/logs/<svc>, prod/deploy
├── postgres/
│   └── init/                   # init-скрипты при первом старте кластера
│       ├── 00-extensions.sql   # pgcrypto, citext, pgvector
│       └── 01-schemas.sql      # схемы auth, users, tasks, messenger, calls, ai
├── migrations/                 # goose-миграции по сервисам
│   ├── auth/
│   ├── users/
│   ├── tasks/
│   ├── messenger/
│   ├── calls/
│   └── ai/
├── services/                   # Dockerfile'ы (если выносим из back-go/services/<svc>/Dockerfile)
└── scripts/
    ├── smoke.sh                # сквозной API-тест
    ├── gen-paseto-key.sh       # генерация ключа для auth
    └── backup.sh               # pg_dump (Этап 10)
```

## Порты

| Сервис | REST | gRPC |
|---|---|---|
| auth | :8081 | :9091 |
| users | :8082 | :9092 |
| tasks | :8083 | :9093 |
| messenger | :8084 | :9094 |
| calls | :8085 | :9095 |
| ai | :8086 | :9096 |
| Postgres | :5433 | — |
| Redis | :6380 | — |
| MailHog UI | :8026 | (SMTP :1026) |
| Swagger UI | :8090 | — |

Намеренно нестандартные хост-порты (`:5433` вместо `:5432`, `:6380` вместо `:6379`, etc.), чтобы не конфликтовать со старым `deploy/`-стеком на дев-машине.

## Команды (после Этапа 1)

```sh
make dev/up                # поднять весь стек
make dev/down              # остановить
make dev/fresh             # снести тома и пересоздать (теряем dev-данные)
make dev/migrate           # накатить миграции (goose)
make dev/logs/auth         # логи одного сервиса
make dev/psql              # psql внутрь postgres
make dev/gen-key           # сгенерировать PASETO-ключ
make dev/smoke             # сквозной API-тест
```

#!/usr/bin/env bash
# Запускает весь стек для разработки одной командой.
# Ctrl+C останавливает Flask, Vite и контейнеры.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACK="$ROOT/back"
FRONT="$ROOT/front"
DEPLOY="$ROOT/deploy"

BACK_PID=""
FRONT_PID=""

cleanup() {
    printf "\n\033[33mОстанавливаю...\033[0m\n"
    [ -n "$BACK_PID" ]  && kill "$BACK_PID"  2>/dev/null || true
    [ -n "$FRONT_PID" ] && kill "$FRONT_PID" 2>/dev/null || true
    (cd "$DEPLOY" && docker compose stop db redis 2>/dev/null) || true
    printf "\033[32mВсё остановлено.\033[0m\n"
    exit 0
}
trap cleanup INT TERM

# 1. DB + Redis
printf "\033[1m▶ DB + Redis...\033[0m\n"
(cd "$DEPLOY" && docker compose up -d db redis)
printf "\033[32m  PostgreSQL :5432  Redis :6379\033[0m\n\n"

# 2. Миграции
printf "\033[1m▶ Миграции...\033[0m\n"
(cd "$BACK" && . venv/bin/activate && flask db upgrade)
printf "\033[32m  Готово\033[0m\n\n"

# 3. Flask + eventlet (через wsgi.py) — werkzeug-сервер flask run не
#    поддерживает WebSocket, для socket.io WS обязателен eventlet.
printf "\033[1m▶ Flask + eventlet :5001...\033[0m\n"
(cd "$BACK" && . venv/bin/activate && PORT=5001 python wsgi.py 2>&1 \
    | awk '{print "\033[36m[back]\033[0m  " $0; fflush()}') &
BACK_PID=$!

# 3. Vite
printf "\033[1m▶ Vite  :5173...\033[0m\n"
(cd "$FRONT" && npm run dev 2>&1 \
    | awk '{print "\033[35m[front]\033[0m " $0; fflush()}') &
FRONT_PID=$!

printf "\n\033[1mСерверы запущены\033[0m  (Ctrl+C — остановить всё)\n"
printf "  Фронт:   \033[4mhttp://localhost:5173\033[0m\n"
printf "  API:     \033[4mhttp://localhost:5001/api\033[0m\n"
printf "  Swagger: \033[4mhttp://localhost:5001/apidocs\033[0m\n\n"

wait

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

# 2. Flask
printf "\033[1m▶ Flask :5001...\033[0m\n"
(cd "$BACK" && . venv/bin/activate && flask run --debug --port 5001 2>&1 \
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

#!/usr/bin/env bash
# Запускает весь стек для разработки одной командой.
# Ctrl+C корректно останавливает Flask, callsvc, Vite и контейнеры.
set -euo pipefail
# Job control: каждый фоновый процесс получает свою process group,
# и `kill -- -PID` убивает целую группу (вместе с детьми npm/eventlet).
set -m

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACK="$ROOT/back"
FRONT="$ROOT/front"
DEPLOY="$ROOT/deploy"

BACK_PID=""
FRONT_PID=""
CALLS_PID=""

# Глушим INT/TERM на время самой cleanup, чтобы повторный Ctrl+C не
# прерывал её в середине.
cleanup() {
    trap '' INT TERM
    printf "\n\033[33mОстанавливаю...\033[0m\n"

    # Убиваем целые process group (а не только лидера),
    # чтобы дети npm/vite/eventlet/go тоже легли.
    if [ -n "$BACK_PID" ];  then kill -TERM -- "-$BACK_PID"  2>/dev/null || true; fi
    if [ -n "$FRONT_PID" ]; then kill -TERM -- "-$FRONT_PID" 2>/dev/null || true; fi
    if [ -n "$CALLS_PID" ]; then kill -TERM -- "-$CALLS_PID" 2>/dev/null || true; fi

    # Даём ~1 секунду на graceful-shutdown (eventlet, vite, callsvc).
    sleep 1

    # Контрольный выстрел — если что-то всё ещё висит.
    if [ -n "$BACK_PID" ];  then kill -KILL -- "-$BACK_PID"  2>/dev/null || true; fi
    if [ -n "$FRONT_PID" ]; then kill -KILL -- "-$FRONT_PID" 2>/dev/null || true; fi
    if [ -n "$CALLS_PID" ]; then kill -KILL -- "-$CALLS_PID" 2>/dev/null || true; fi

    # Подбираем сирот по имени — защита от случая, когда субшелл уже
    # умер, а его потомки ещё живы. Узко по нашему пути, чужие процессы
    # не трогаем.
    pkill -f "$BACK/.*wsgi\.py"  2>/dev/null || true
    pkill -f "$FRONT/.*vite"     2>/dev/null || true
    # go run собирает бинарь во временный каталог — ловим по имени бинаря.
    pkill -f "exe/callsvc"       2>/dev/null || true

    (cd "$DEPLOY" && docker compose stop 2>/dev/null) || true
    printf "\033[32mВсё остановлено.\033[0m\n"
    exit 0
}
trap cleanup INT TERM

# 1. Инфраструктура (db + redis + livekit). Приложения (app/calls/nginx)
#    в dev-оверлее за профилем "full" и не стартуют — бегут на хосте ниже.
printf "\033[1m▶ DB + Redis + LiveKit...\033[0m\n"
(cd "$DEPLOY" && docker compose up -d)
printf "\033[32m  PostgreSQL :5432  Redis :6379  LiveKit :7880\033[0m\n\n"

# 2. Миграции
printf "\033[1m▶ Миграции...\033[0m\n"
(cd "$BACK" && . venv/bin/activate && flask db upgrade)
printf "\033[32m  Готово\033[0m\n\n"

# 3. Go-микросервис звонков (gRPC :9090 для Flask, HTTP :8090 для REST и
#    вебхуков LiveKit). env синхронизированы с back/.flaskenv и
#    deploy/docker-compose.override.yml.
printf "\033[1m▶ callsvc (Go)  gRPC :9090  HTTP :8090...\033[0m\n"
(
  cd "$ROOT/back-go/calls" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  JWT_SECRET_KEY="dev-jwt-secret-key-min-32-chars-local-xxxx" \
  LIVEKIT_API_KEY="devkey" \
  LIVEKIT_API_SECRET="dev_livekit_secret_min_32_chars_ok" \
  LIVEKIT_URL="http://localhost:7880" \
  LIVEKIT_CLIENT_URL="ws://localhost:7880" \
  exec go run ./cmd/callsvc
) &
CALLS_PID=$!

# 4. Flask + eventlet (через wsgi.py) — werkzeug-сервер flask run не
#    поддерживает WebSocket, для socket.io WS обязателен eventlet.
#    exec заменяет процесс субшелла на python — тогда $! = PID python,
#    и kill его реально касается. Без awk-префикса логи мешаются с фронтом,
#    но это малая цена за корректный shutdown.
printf "\033[1m▶ Flask + eventlet :5001...\033[0m\n"
( cd "$BACK" && . venv/bin/activate && exec python wsgi.py ) &
BACK_PID=$!

# 5. Vite
printf "\033[1m▶ Vite  :5173...\033[0m\n"
( cd "$FRONT" && exec npm run dev ) &
FRONT_PID=$!

printf "\n\033[1mСерверы запущены\033[0m  (Ctrl+C — остановить всё)\n"
printf "  Фронт:   \033[4mhttp://localhost:5173\033[0m\n"
printf "  API:     \033[4mhttp://localhost:5001/api\033[0m\n"
printf "  Звонки:  \033[4mhttp://localhost:8090/api/calls\033[0m (gRPC :9090)\n"
printf "  Swagger: \033[4mhttp://localhost:5001/apidocs\033[0m\n\n"

wait

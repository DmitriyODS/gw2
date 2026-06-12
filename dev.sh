#!/usr/bin/env bash
# Запускает весь стек для разработки одной командой.
# Ctrl+C корректно останавливает Flask, Go-микросервисы, Vite и контейнеры.
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
AUTH_PID=""
MESSENGER_PID=""
AI_PID=""
TASKS_PID=""

# Dev-ключи PASETO (синхронизированы с back/.flaskenv, Makefile и
# back/tests/conftest.py): приватный — только у authsvc, публичный — у
# Flask и callsvc.
PASETO_PRIVATE_KEY_DEV="68eb779b2f672beb8fcd58d72a81ce1565a1417aed3788d1362bf4faaa3f62ac15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1"
PASETO_PUBLIC_KEY_DEV="15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1"
PASETO_REFRESH_KEY_DEV="d525374c4ec7b5e1c5b140fb9c1f4cffd9c3dbf052bb18f2f32bf9f92d9fa05c"

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
    if [ -n "$AUTH_PID" ];  then kill -TERM -- "-$AUTH_PID"  2>/dev/null || true; fi
    if [ -n "$MESSENGER_PID" ]; then kill -TERM -- "-$MESSENGER_PID" 2>/dev/null || true; fi
    if [ -n "$AI_PID" ];    then kill -TERM -- "-$AI_PID"    2>/dev/null || true; fi
    if [ -n "$TASKS_PID" ]; then kill -TERM -- "-$TASKS_PID" 2>/dev/null || true; fi

    # Даём ~1 секунду на graceful-shutdown (eventlet, vite, callsvc).
    sleep 1

    # Контрольный выстрел — если что-то всё ещё висит.
    if [ -n "$BACK_PID" ];  then kill -KILL -- "-$BACK_PID"  2>/dev/null || true; fi
    if [ -n "$FRONT_PID" ]; then kill -KILL -- "-$FRONT_PID" 2>/dev/null || true; fi
    if [ -n "$CALLS_PID" ]; then kill -KILL -- "-$CALLS_PID" 2>/dev/null || true; fi
    if [ -n "$AUTH_PID" ];  then kill -KILL -- "-$AUTH_PID"  2>/dev/null || true; fi
    if [ -n "$MESSENGER_PID" ]; then kill -KILL -- "-$MESSENGER_PID" 2>/dev/null || true; fi
    if [ -n "$AI_PID" ];    then kill -KILL -- "-$AI_PID"    2>/dev/null || true; fi
    if [ -n "$TASKS_PID" ]; then kill -KILL -- "-$TASKS_PID" 2>/dev/null || true; fi

    # Подбираем сирот по имени — защита от случая, когда субшелл уже
    # умер, а его потомки ещё живы. Узко по нашему пути, чужие процессы
    # не трогаем.
    pkill -f "$BACK/.*wsgi\.py"  2>/dev/null || true
    pkill -f "$FRONT/.*vite"     2>/dev/null || true
    # go run собирает бинарь во временный каталог — ловим по имени бинаря.
    pkill -f "exe/callsvc"       2>/dev/null || true
    pkill -f "exe/authsvc"       2>/dev/null || true
    pkill -f "exe/msgsvc"        2>/dev/null || true
    pkill -f "exe/aisvc"         2>/dev/null || true
    pkill -f "exe/groovesvc"     2>/dev/null || true
    pkill -f "exe/tasksvc"       2>/dev/null || true

    (cd "$DEPLOY" && docker compose stop 2>/dev/null) || true
    printf "\033[32mВсё остановлено.\033[0m\n"
    exit 0
}
trap cleanup INT TERM

# 1. Инфраструктура (db + redis + livekit). Приложения (app/calls/auth/nginx)
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
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  LIVEKIT_API_KEY="devkey" \
  LIVEKIT_API_SECRET="dev_livekit_secret_min_32_chars_ok" \
  LIVEKIT_URL="http://localhost:7880" \
  LIVEKIT_CLIENT_URL="ws://localhost:7880" \
  exec go run ./cmd/callsvc
) &
CALLS_PID=$!

# 4. Go-микросервис авторизации (HTTP :8091 — /api/auth/* и /api/users/*,
#    выпускает PASETO-токены). env синхронизированы с back/.flaskenv.
printf "\033[1m▶ authsvc (Go)  HTTP :8091...\033[0m\n"
(
  cd "$ROOT/back-go/auth" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PRIVATE_KEY="$PASETO_PRIVATE_KEY_DEV" \
  PASETO_REFRESH_KEY="$PASETO_REFRESH_KEY_DEV" \
  UPLOAD_FOLDER="$BACK/uploads" \
  exec go run ./cmd/authsvc
) &
AUTH_PID=$!

# 5. Go-микросервис мессенджера (gRPC :9092 для Flask, HTTP :8092 —
#    /api/messenger/* кроме exact presence). env синхронизированы с
#    back/.flaskenv и deploy/docker-compose.override.yml.
printf "\033[1m▶ msgsvc (Go)  gRPC :9092  HTTP :8092...\033[0m\n"
(
  cd "$ROOT/back-go/messenger" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  UPLOAD_FOLDER="$BACK/uploads" \
  HTTP_ADDR=":8092" \
  GRPC_ADDR=":9092" \
  exec go run ./cmd/msgsvc
) &
MESSENGER_PID=$!

# 6. Go-микросервис ИИ (gRPC :9093 для Flask, HTTP :8093 — regex-роут
#    /api/companies/<id>/ai-settings + /api/ai/tv-fact; Redis — кэш
#    ТВ-фактов). env синхронизированы с back/.flaskenv и
#    deploy/docker-compose.override.yml.
printf "\033[1m▶ aisvc (Go)  gRPC :9093  HTTP :8093...\033[0m\n"
(
  cd "$ROOT/back-go/ai" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  AI_KEY_ENCRYPTION_KEY="X3hFOVZ6XbAzlaygv2PfLbnmBIaH373CK8MqrrAhr8k=" \
  HTTP_ADDR=":8093" \
  GRPC_ADDR=":9093" \
  exec go run ./cmd/aisvc
) &
AI_PID=$!

# 7. Go-микросервис «Мой Groove» (gRPC :9094 — хуки tasksvc/msgsvc, HTTP
#    :8094 — /api/groove/*). Зовёт aisvc и msgsvc по gRPC. env
#    синхронизированы с back/.flaskenv и deploy/docker-compose.override.yml.
printf "\033[1m▶ groovesvc (Go)  gRPC :9094  HTTP :8094...\033[0m\n"
(
  cd "$ROOT/back-go/groove" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  AI_GRPC_ADDR="localhost:9093" \
  MESSENGER_GRPC_ADDR="localhost:9092" \
  HTTP_ADDR=":8094" \
  GRPC_ADDR=":9094" \
  exec go run ./cmd/groovesvc
) &
GROOVE_PID=$!

# 8. Go-микросервис задач (HTTP :8095 — /api/tasks|units|unit-types|
#    departments|stages|stats|yougile). Зовёт groovesvc и aisvc по gRPC,
#    события — в Redis gw2:tasks:events. env синхронизированы с
#    deploy/docker-compose.override.yml. Dev-ключ Fernet YouGile — тот же,
#    что был в back/.flaskenv.
printf "\033[1m▶ tasksvc (Go)  HTTP :8095...\033[0m\n"
(
  cd "$ROOT/back-go/tasks" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  GROOVE_GRPC_ADDR="localhost:9094" \
  AI_GRPC_ADDR="localhost:9093" \
  YOUGILE_ENC_KEY="CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=" \
  HTTP_ADDR=":8095" \
  exec go run ./cmd/tasksvc
) &
TASKS_PID=$!

# 8. Flask + eventlet (через wsgi.py) — werkzeug-сервер flask run не
#    поддерживает WebSocket, для socket.io WS обязателен eventlet.
#    exec заменяет процесс субшелла на python — тогда $! = PID python,
#    и kill его реально касается. Без awk-префикса логи мешаются с фронтом,
#    но это малая цена за корректный shutdown.
printf "\033[1m▶ Flask + eventlet :5001...\033[0m\n"
( cd "$BACK" && . venv/bin/activate && exec python wsgi.py ) &
BACK_PID=$!

# 8. Vite
printf "\033[1m▶ Vite  :5173...\033[0m\n"
( cd "$FRONT" && exec npm run dev ) &
FRONT_PID=$!

printf "\n\033[1mСерверы запущены\033[0m  (Ctrl+C — остановить всё)\n"
printf "  Фронт:   \033[4mhttp://localhost:5173\033[0m\n"
printf "  API:     \033[4mhttp://localhost:5001/api\033[0m\n"
printf "  Звонки:  \033[4mhttp://localhost:8090/api/calls\033[0m (gRPC :9090)\n"
printf "  Auth:    \033[4mhttp://localhost:8091/api/auth\033[0m\n"
printf "  Чаты:    \033[4mhttp://localhost:8092/api/messenger\033[0m (gRPC :9092)\n"
printf "  Groove:  \033[4mhttp://localhost:8094/api/groove\033[0m (gRPC :9094)\n"
printf "  Задачи:  \033[4mhttp://localhost:8095/api/tasks\033[0m\n"
printf "  ИИ:      \033[4mhttp://localhost:8093\033[0m (gRPC :9093)\n"
printf "  Swagger: \033[4mhttp://localhost:5001/apidocs\033[0m\n\n"

wait

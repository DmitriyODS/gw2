#!/usr/bin/env bash
# Запускает весь стек для разработки одной командой.
# Ctrl+C корректно останавливает Go-микросервисы, Vite и контейнеры.
set -euo pipefail
# Job control: каждый фоновый процесс получает свою process group,
# и `kill -- -PID` убивает целую группу (вместе с детьми npm/eventlet).
set -m

ROOT="$(cd "$(dirname "$0")" && pwd)"
FRONT="$ROOT/front"
DEPLOY="$ROOT/deploy"
# Файлы пользователей (аватарки, вложения) в dev — каталог в корне репо
# (в gitignore); в docker-стеке это общий uploads-volume.
UPLOADS="$ROOT/uploads"
mkdir -p "$UPLOADS"

# Адреса для доступа с других устройств той же сети (телефон/планшет на том же
# Wi-Fi заходят на http://<IP>:5173). PRIMARY_IP — адрес интерфейса маршрута по
# умолчанию (его же отдаём LiveKit-клиенту), LAN_IPS — все не-петлевые адреса
# для подсказки. Нет сети — откатываемся на localhost (как было).
if [ "$(uname)" = "Darwin" ]; then
    def_if="$(route -n get default 2>/dev/null | awk '/interface:/{print $2}')"
    PRIMARY_IP="$([ -n "$def_if" ] && ipconfig getifaddr "$def_if" 2>/dev/null || true)"
    LAN_IPS="$(ifconfig 2>/dev/null | awk '/inet /{print $2}' | grep -Ev '^(127\.|169\.254\.)' || true)"
else
    PRIMARY_IP="$(ip route get 1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}')"
    LAN_IPS="$(hostname -I 2>/dev/null | tr ' ' '\n' | grep -Ev '^(127\.|169\.254\.)' || true)"
fi
[ -z "${PRIMARY_IP:-}" ] && PRIMARY_IP="$(printf '%s\n' "$LAN_IPS" | head -n1)"
[ -z "${PRIMARY_IP:-}" ] && PRIMARY_IP="localhost"

FRONT_PID=""
CALLS_PID=""
AUTH_PID=""
MESSENGER_PID=""
AI_PID=""
TASKS_PID=""
GATEWAY_PID=""
GROOVE_PID=""
PUSH_PID=""
MAIL_PID=""
REGISTRY_PID=""
CALENDAR_PID=""
DIARY_PID=""

# Все Go-сервисы (имя бинаря в go-build/exe — по нему ловим осиротевшие процессы).
SVCS="callsvc authsvc msgsvc aisvc groovesvc tasksvc gatewaysvc pushsvc mailsvc registrysvc calendarsvc diarysvc"

# Dev-ключи PASETO (синхронизированы с Makefile и
# deploy/docker-compose.override.yml): приватный — только у authsvc,
# публичный — у остальных сервисов.
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
    if [ -n "$FRONT_PID" ]; then kill -TERM -- "-$FRONT_PID" 2>/dev/null || true; fi
    if [ -n "$CALLS_PID" ]; then kill -TERM -- "-$CALLS_PID" 2>/dev/null || true; fi
    if [ -n "$AUTH_PID" ];  then kill -TERM -- "-$AUTH_PID"  2>/dev/null || true; fi
    if [ -n "$MESSENGER_PID" ]; then kill -TERM -- "-$MESSENGER_PID" 2>/dev/null || true; fi
    if [ -n "$AI_PID" ];    then kill -TERM -- "-$AI_PID"    2>/dev/null || true; fi
    if [ -n "$TASKS_PID" ]; then kill -TERM -- "-$TASKS_PID" 2>/dev/null || true; fi
    if [ -n "$GATEWAY_PID" ]; then kill -TERM -- "-$GATEWAY_PID" 2>/dev/null || true; fi
    if [ -n "$GROOVE_PID" ]; then kill -TERM -- "-$GROOVE_PID" 2>/dev/null || true; fi
    if [ -n "$PUSH_PID" ]; then kill -TERM -- "-$PUSH_PID" 2>/dev/null || true; fi
    if [ -n "$MAIL_PID" ]; then kill -TERM -- "-$MAIL_PID" 2>/dev/null || true; fi
    if [ -n "$REGISTRY_PID" ]; then kill -TERM -- "-$REGISTRY_PID" 2>/dev/null || true; fi
    if [ -n "$CALENDAR_PID" ]; then kill -TERM -- "-$CALENDAR_PID" 2>/dev/null || true; fi
    if [ -n "$DIARY_PID" ]; then kill -TERM -- "-$DIARY_PID" 2>/dev/null || true; fi

    # Даём ~1 секунду на graceful-shutdown (vite, Go-сервисы).
    sleep 1

    # Контрольный выстрел — если что-то всё ещё висит.
    if [ -n "$FRONT_PID" ]; then kill -KILL -- "-$FRONT_PID" 2>/dev/null || true; fi
    if [ -n "$CALLS_PID" ]; then kill -KILL -- "-$CALLS_PID" 2>/dev/null || true; fi
    if [ -n "$AUTH_PID" ];  then kill -KILL -- "-$AUTH_PID"  2>/dev/null || true; fi
    if [ -n "$MESSENGER_PID" ]; then kill -KILL -- "-$MESSENGER_PID" 2>/dev/null || true; fi
    if [ -n "$AI_PID" ];    then kill -KILL -- "-$AI_PID"    2>/dev/null || true; fi
    if [ -n "$TASKS_PID" ]; then kill -KILL -- "-$TASKS_PID" 2>/dev/null || true; fi
    if [ -n "$GATEWAY_PID" ]; then kill -KILL -- "-$GATEWAY_PID" 2>/dev/null || true; fi
    if [ -n "$GROOVE_PID" ]; then kill -KILL -- "-$GROOVE_PID" 2>/dev/null || true; fi
    if [ -n "$PUSH_PID" ]; then kill -KILL -- "-$PUSH_PID" 2>/dev/null || true; fi
    if [ -n "$MAIL_PID" ]; then kill -KILL -- "-$MAIL_PID" 2>/dev/null || true; fi
    if [ -n "$REGISTRY_PID" ]; then kill -KILL -- "-$REGISTRY_PID" 2>/dev/null || true; fi
    if [ -n "$CALENDAR_PID" ]; then kill -KILL -- "-$CALENDAR_PID" 2>/dev/null || true; fi
    if [ -n "$DIARY_PID" ]; then kill -KILL -- "-$DIARY_PID" 2>/dev/null || true; fi

    # Подбираем сирот по имени — защита от случая, когда субшелл уже
    # умер, а его потомки ещё живы. Узко по нашему пути, чужие процессы
    # не трогаем. go run собирает бинарь во временный каталог — ловим по имени.
    pkill -f "$FRONT/.*vite" 2>/dev/null || true
    for svc in $SVCS; do pkill -f "exe/$svc" 2>/dev/null || true; done

    (cd "$DEPLOY" && docker compose stop 2>/dev/null) || true
    printf "\033[32mВсё остановлено.\033[0m\n"
    exit 0
}
trap cleanup INT TERM

# 0. Преполёт: убиваем процессы прошлого запуска (по имени, узко по нашему
#    репозиторию). Без этого повторный ./dev.sh натыкается на занятые порты —
#    новый `go run` падает с «address in use», а на порту продолжает отвечать
#    СТАРЫЙ бинарник со старым кодом (правки словно «не применяются»).
preflight() {
    local found=""
    if pkill -f "$FRONT/.*vite" 2>/dev/null; then found=1; fi
    for svc in $SVCS; do
        if pkill -f "exe/$svc" 2>/dev/null; then found=1; fi
    done
    if [ -n "$found" ]; then
        printf "\033[33m▶ Останавливаю процессы прошлого запуска...\033[0m\n"
        sleep 2  # даём портам освободиться до старта свежих сервисов
    fi
}
preflight

# 1. Инфраструктура (db + redis + livekit). Приложения в dev-оверлее за
#    профилем "full" и не стартуют — бегут на хосте ниже.
printf "\033[1m▶ DB + Redis + LiveKit...\033[0m\n"
(cd "$DEPLOY" && docker compose up -d)
printf "\033[32m  PostgreSQL :5432  Redis :6379  LiveKit :7880\033[0m\n\n"

# 2. Миграции (goose, back-go/migrate)
printf "\033[1m▶ Миграции...\033[0m\n"
(cd "$ROOT/back-go/migrate" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  go run ./cmd/migrate)
printf "\033[32m  Готово\033[0m\n\n"

# 3. Go-микросервис звонков (gRPC :9090 для gateway, HTTP :8090 для REST и
#    вебхуков LiveKit; плашки звонков — gRPC msgsvc). env синхронизированы
#    с deploy/docker-compose.override.yml.
printf "\033[1m▶ callsvc (Go)  gRPC :9090  HTTP :8090...\033[0m\n"
(
  cd "$ROOT/back-go/calls" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  LIVEKIT_API_KEY="devkey" \
  LIVEKIT_API_SECRET="dev_livekit_secret_min_32_chars_ok" \
  LIVEKIT_URL="http://localhost:7880" \
  LIVEKIT_CLIENT_URL="ws://${PRIMARY_IP}:7880" \
  MESSENGER_GRPC_ADDR="localhost:9092" \
  exec go run ./cmd/callsvc
) &
CALLS_PID=$!

# 4. Go-микросервис авторизации (HTTP :8091 — /api/auth/* и /api/users/*,
#    выпускает PASETO-токены). env синхронизированы с docker-compose.override.
printf "\033[1m▶ authsvc (Go)  HTTP :8091...\033[0m\n"
(
  cd "$ROOT/back-go/auth" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PRIVATE_KEY="$PASETO_PRIVATE_KEY_DEV" \
  PASETO_REFRESH_KEY="$PASETO_REFRESH_KEY_DEV" \
  UPLOAD_FOLDER="$UPLOADS" \
  MAIL_GRPC_ADDR="localhost:9098" \
  APP_PUBLIC_BASE_URL="http://${PRIMARY_IP}:5173" \
  exec go run ./cmd/authsvc
) &
AUTH_PID=$!

# 5. Go-микросервис мессенджера (gRPC :9092 — плашки звонков callsvc и
#    pet-чат groovesvc, HTTP :8092 — /api/messenger/* кроме exact presence).
printf "\033[1m▶ msgsvc (Go)  gRPC :9092  HTTP :8092...\033[0m\n"
(
  cd "$ROOT/back-go/messenger" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  UPLOAD_FOLDER="$UPLOADS" \
  HTTP_ADDR=":8092" \
  GRPC_ADDR=":9092" \
  exec go run ./cmd/msgsvc
) &
MESSENGER_PID=$!

# 6. Go-микросервис ИИ (gRPC :9093 — tasksvc/groovesvc, HTTP :8093 —
#    regex-роут /api/companies/<id>/ai-settings + /api/ai/tv-fact;
#    Redis — кэш ТВ-фактов).
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
#    :8094 — /api/groove/*). Зовёт aisvc и msgsvc по gRPC.
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
#    что в Makefile/override.
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

# 9. Realtime-шлюз gatewaysvc (HTTP :8096 — WebSocket /ws, exact
#    /api/messenger/presence; presence в Redis; ринг-фаза → gRPC callsvc;
#    доставка событий всех каналов gw2:*:events).
printf "\033[1m▶ gatewaysvc (Go)  HTTP :8096...\033[0m\n"
(
  cd "$ROOT/back-go/gateway" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  CALLS_GRPC_ADDR="localhost:9090" \
  HTTP_ADDR=":8096" \
  exec go run ./cmd/gatewaysvc
) &
GATEWAY_PID=$!

# 10. Go-микросервис пуш-уведомлений pushsvc (HTTP :8097 — регистрация
#     токенов устройств; подписан на gw2:*:events, шлёт FCM офлайн-получателям).
#     Без FIREBASE_CREDENTIALS_JSON отправка отключена — для dev это норма.
printf "\033[1m▶ pushsvc (Go)  HTTP :8097...\033[0m\n"
(
  cd "$ROOT/back-go/push" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  FIREBASE_CREDENTIALS_JSON="${FIREBASE_CREDENTIALS_JSON:-}" \
  HTTP_ADDR=":8097" \
  exec go run ./cmd/pushsvc
) &
PUSH_PID=$!

# 11. Go-микросервис почты mailsvc (gRPC :9098 — Send; HTTP :8098 — /healthz).
#     Письма уходят в mailpit (docker compose up поднимает его) — смотреть на
#     http://localhost:8025. Реальный SMTP в dev не нужен.
printf "\033[1m▶ mailsvc (Go)  gRPC :9098  HTTP :8098...\033[0m\n"
(
  cd "$ROOT/back-go/mail" && \
  SMTP_HOST="localhost" \
  SMTP_PORT="1025" \
  SMTP_TLS="none" \
  SMTP_FROM="noreply@grovework.local" \
  HTTP_ADDR=":8098" \
  GRPC_ADDR=":9098" \
  exec go run ./cmd/mailsvc
) &
MAIL_PID=$!

# 12. Go-микросервис реестров registrysvc (HTTP :8099 — REST /api/registries/*).
#     Загруженные файлы реестров пишутся в общий uploads-том. Межсервисных
#     вызовов нет: проверка токенов локальная (PASETO_PUBLIC_KEY).
printf "\033[1m▶ registrysvc (Go)  HTTP :8099...\033[0m\n"
(
  cd "$ROOT/back-go/registry" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  UPLOAD_FOLDER="$ROOT/uploads" \
  HTTP_ADDR=":8099" \
  exec go run ./cmd/registrysvc
) &
REGISTRY_PID=$!

# 12b. Go-микросервис календарей calendarsvc (HTTP :8100 — REST /api/calendars/*).
#      Загруженные файлы календарей пишутся в общий uploads-том. Межсервисных
#      вызовов нет: проверка токенов локальная (PASETO_PUBLIC_KEY).
printf "\033[1m▶ calendarsvc (Go)  HTTP :8100...\033[0m\n"
(
  cd "$ROOT/back-go/calendar" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  UPLOAD_FOLDER="$ROOT/uploads" \
  HTTP_ADDR=":8100" \
  exec go run ./cmd/calendarsvc
) &
CALENDAR_PID=$!

# 12c. Go-микросервис ежедневников diarysvc (HTTP :8101 — REST /api/diaries/*).
#      Личные заметки-задачи пользователя; файлов нет. Межсервисных вызовов нет:
#      проверка токенов локальная (PASETO_PUBLIC_KEY).
printf "\033[1m▶ diarysvc (Go)  HTTP :8101...\033[0m\n"
(
  cd "$ROOT/back-go/diary" && \
  DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
  REDIS_URL="redis://localhost:6379/0" \
  PASETO_PUBLIC_KEY="$PASETO_PUBLIC_KEY_DEV" \
  HTTP_ADDR=":8101" \
  exec go run ./cmd/diarysvc
) &
DIARY_PID=$!

# 13. Vite (--host 0.0.0.0 — слушаем все интерфейсы, чтобы фронт открывался
#     с других устройств сети по http://<IP>:5173).
printf "\033[1m▶ Vite  :5173...\033[0m\n"
( cd "$FRONT" && exec npm run dev -- --host 0.0.0.0 ) &
FRONT_PID=$!

printf "\n\033[1mСерверы запущены\033[0m  (Ctrl+C — остановить всё)\n"
printf "  Фронт:   \033[4mhttp://localhost:5173\033[0m\n"
if [ "$PRIMARY_IP" != "localhost" ] && [ -n "$LAN_IPS" ]; then
    printf "\n\033[1m  С других устройств в этой сети:\033[0m\n"
    printf '%s\n' "$LAN_IPS" | while IFS= read -r ip; do
        [ -n "$ip" ] && printf "    \033[4mhttp://%s:5173\033[0m\n" "$ip"
    done
    printf "    \033[2m(устройство должно быть в той же сети; firewall macOS может\n"
    printf "     спросить разрешение для node/go при первом подключении)\033[0m\n"
fi
printf "\n  Шлюз:    \033[4mws://localhost:8096/ws\033[0m\n"
printf "  Звонки:  \033[4mhttp://localhost:8090/api/calls\033[0m (gRPC :9090)\n"
printf "  Auth:    \033[4mhttp://localhost:8091/api/auth\033[0m\n"
printf "  Чаты:    \033[4mhttp://localhost:8092/api/messenger\033[0m (gRPC :9092)\n"
printf "  Groove:  \033[4mhttp://localhost:8094/api/groove\033[0m (gRPC :9094)\n"
printf "  Задачи:  \033[4mhttp://localhost:8095/api/tasks\033[0m\n"
printf "  ИИ:      \033[4mhttp://localhost:8093\033[0m (gRPC :9093)\n"
printf "  Пуши:    \033[4mhttp://localhost:8097/api/push\033[0m\n"
printf "  Реестры: \033[4mhttp://localhost:8099/api/registries\033[0m\n"
printf "  Календари: \033[4mhttp://localhost:8100/api/calendars\033[0m\n"
printf "  Ежедневники: \033[4mhttp://localhost:8101/api/diaries\033[0m\n"
printf "  Почта:   \033[4mhttp://localhost:8025\033[0m (mailpit; gRPC :9098)\n\n"

wait

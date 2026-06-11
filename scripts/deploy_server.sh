#!/usr/bin/env bash
# ================================================================
# Серверная часть деплоя Groove Work. Запускается НА сервере из
# `make deploy` (после git fetch + reset --hard). Идемпотентен.
#
# Что делает:
#   1. Сверяет deploy/.env с требуемыми переменными: недостающие
#      секреты генерирует сам (существующие НЕ трогает), устаревшие
#      (TURN_* от выпиленного coturn) вычищает. Перед любой правкой
#      делает бэкап .env рядом.
#   2. Если включён ufw — открывает медиа-порты LiveKit (7881/tcp,
#      7882/udp); 80/443 считаются уже открытыми.
#   3. docker compose up -d --build --remove-orphans (сносит
#      осиротевшие контейнеры вроде старого coturn).
#   4. Перечитывает конфиг nginx (он bind-mounted: git обновляет
#      файл, но без reload контейнер живёт со старым конфигом).
#   5. Health-чеки: API через nginx, микросервис звонков (healthz +
#      gRPC из app), LiveKit через nginx (/livekit), TCP-порт медиа 7881.
# ================================================================
set -euo pipefail
cd "$(cd "$(dirname "$0")/.." && pwd)"

# --env-only: только синхронизировать deploy/.env (без сборки/перезапуска).
# Удобно подготовить сервер заранее или проверить, что сгенерируется.
ENV_ONLY=0
[ "${1:-}" = "--env-only" ] && ENV_ONLY=1

ENV_FILE="deploy/.env"
# Прод-стек = база + оверлей; голый `docker compose` подхватил бы dev-оверлей.
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

log()  { printf '\033[1m▶ %s\033[0m\n' "$*"; }
ok()   { printf '\033[32m✓ %s\033[0m\n' "$*"; }
warn() { printf '\033[33m! %s\033[0m\n' "$*"; }

# ── 1. Синхронизация deploy/.env ─────────────────────────────────
log "Проверяю $ENV_FILE"
if [ ! -f "$ENV_FILE" ]; then
  cp deploy/.env.example "$ENV_FILE"
  warn "$ENV_FILE не было — создан из .env.example"
fi

gen_hex() { openssl rand -hex 32; }
# Fernet-ключ = urlsafe-base64 от 32 байт (cryptography на сервере не нужна).
gen_fernet() { openssl rand 32 | base64 | tr '+/' '-_' | tr -d '\n'; }

ENV_CHANGED=0
backup_once() {
  if [ "$ENV_CHANGED" -eq 0 ]; then
    cp "$ENV_FILE" "$ENV_FILE.bak.$(date +%Y%m%d-%H%M%S)"
    ENV_CHANGED=1
  fi
}

# ensure VAR VALUE — дозаписывает переменную, если её нет или она пустая.
# Существующие непустые значения никогда не перезаписывает (ротация ключей
# шифрования = потеря данных, см. .env.example).
ensure() {
  local var="$1" value="$2"
  if grep -qE "^${var}=..*" "$ENV_FILE"; then
    return 0
  fi
  backup_once
  if grep -qE "^${var}=" "$ENV_FILE"; then
    sed -i "s|^${var}=.*|${var}=${value}|" "$ENV_FILE"
  else
    printf '%s=%s\n' "$var" "$value" >> "$ENV_FILE"
  fi
  ok "сгенерирован ${var}"
}

ensure DB_NAME "grovework"
ensure DB_USER "grovework"
ensure DB_PASSWORD "$(gen_hex)"
ensure JWT_SECRET_KEY "$(gen_hex)"
ensure SECRET_KEY "$(gen_hex)"
ensure AI_KEY_ENCRYPTION_KEY "$(gen_fernet)"
ensure YOUGILE_ENC_KEY "$(gen_fernet)"
ensure LIVEKIT_API_KEY "gw2_$(openssl rand -hex 6)"
ensure LIVEKIT_API_SECRET "$(gen_hex)"

# Устаревшие переменные coturn (заменён LiveKit в v3.5.0).
if grep -qE '^TURN_' "$ENV_FILE"; then
  backup_once
  sed -i '/^TURN_/d' "$ENV_FILE"
  ok "удалены устаревшие TURN_* (coturn заменён LiveKit)"
fi

# Не генерируемое автоматически — только предупреждаем.
if ! grep -qE '^YOUGILE_WEBHOOK_PUBLIC_BASE=..*' "$ENV_FILE"; then
  warn "YOUGILE_WEBHOOK_PUBLIC_BASE пуст — вебхуки YouGile не зарегистрируются"
fi

if [ "$ENV_CHANGED" -eq 1 ]; then
  ok "$ENV_FILE обновлён (бэкап лежит рядом)"
else
  ok "$ENV_FILE в порядке, изменений не потребовалось"
fi

if [ "$ENV_ONLY" -eq 1 ]; then
  ok "Режим --env-only: сборка и перезапуск пропущены"
  exit 0
fi

# ── 2. Firewall (только если ufw реально фильтрует) ──────────────
if command -v ufw >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
  if sudo -n ufw status 2>/dev/null | grep -q 'Status: active'; then
    log "ufw активен — открываю медиа-порты LiveKit"
    sudo -n ufw allow 7881/tcp >/dev/null
    sudo -n ufw allow 7882/udp >/dev/null
    ok "открыты 7881/tcp и 7882/udp"
  else
    ok "ufw неактивен — фильтрации нет, порты доступны"
  fi
else
  warn "ufw недоступен (нет команды или sudo) — проверьте медиа-порты 7881/tcp, 7882/udp вручную"
fi

# ── 3. Сборка и запуск ───────────────────────────────────────────
log "Собираю и поднимаю контейнеры"
cd deploy
$COMPOSE up -d --build --remove-orphans

# ── 4. Перечитать конфиг nginx ───────────────────────────────────
# Конфиг примонтирован файлом: git reset уже обновил его на диске, но
# работающий nginx перечитывает конфигурацию только по сигналу.
log "Перечитываю конфиг nginx"
if $COMPOSE exec -T nginx nginx -t; then
  $COMPOSE exec -T nginx nginx -s reload
  ok "nginx перечитал конфиг"
else
  warn "nginx -t не прошёл — конфиг НЕ перечитан, работает старый"
fi

# ── 5. Health-чеки ───────────────────────────────────────────────
log "Жду готовности API (миграции на старте могут занять время)"
api_code=000
for _ in $(seq 1 30); do
  api_code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 http://localhost/apispec.json || true)
  [ "$api_code" = "200" ] && break
  sleep 2
done
if [ "$api_code" = "200" ]; then
  ok "API отвечает (apispec 200)"
else
  warn "API не ответил за 60с (код $api_code) — смотрите: make logs"
fi

# Микросервис звонков: HTTP-healthz изнутри контейнера (наружу порт не
# торчит, через nginx ходит только /api/calls/*) + досягаемость gRPC из app.
if $COMPOSE exec -T calls wget -qO- --timeout=3 http://127.0.0.1:8090/healthz >/dev/null 2>&1; then
  ok "callsvc отвечает (healthz)"
else
  warn "callsvc не отвечает — звонки не работают: make logs s=calls"
fi
if $COMPOSE exec -T app python -c "import socket; socket.create_connection(('calls', 9090), timeout=3)" >/dev/null 2>&1; then
  ok "gRPC callsvc досягаем из app (calls:9090)"
else
  warn "app не достучался до gRPC calls:9090 — ринг-фаза звонков не работает"
fi

lk_code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/livekit/ || true)
if [ "$lk_code" = "200" ]; then
  ok "LiveKit отвечает через nginx (/livekit)"
else
  warn "LiveKit за nginx не отвечает (код $lk_code) — звонки работать не будут"
fi

if (exec 3<>/dev/tcp/127.0.0.1/7881) 2>/dev/null; then
  exec 3>&- 3<&-
  ok "медиа-порт 7881/tcp слушает (7882/udp проверить извне нельзя — webrtc проверит сам)"
else
  warn "медиа-порт 7881/tcp не слушает — проверьте сервис livekit"
fi

$COMPOSE ps --format 'table {{.Name}}\t{{.Status}}'
ok "Деплой завершён"

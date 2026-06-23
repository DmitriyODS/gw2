#!/usr/bin/env bash
# ================================================================
# Серверная часть деплоя Groove Work. Запускается НА сервере из
# `make deploy` (после git fetch + reset --hard). Идемпотентен.
#
# Что делает:
#   1. Сверяет deploy/.env с требуемыми переменными: недостающие
#      секреты генерирует сам (существующие НЕ трогает), устаревшие
#      (TURN_* от coturn, JWT_SECRET_KEY от flask-jwt, SECRET_KEY от
#      Flask) вычищает. PASETO-ключи генерируются ПАРОЙ (приватный
#      Ed25519 + публичный). Перед любой правкой делает бэкап .env рядом.
#   2. Если включён ufw — открывает порты LiveKit: медиа 7881/tcp,
#      7882/udp и TURN-relay 5349/tcp, 3478/udp; 80/443 считаются
#      уже открытыми. (2b) Поднимает UDP-буферы ядра (net.core.rmem_max/
#      wmem_max=16 МБ) для качества WebRTC под нагрузкой — персистентно.
#   3. docker compose pull + up -d --no-build --remove-orphans.
#      Образы НА СЕРВЕРЕ НЕ СОБИРАЮТСЯ — их пушит локальная машина
#      (`make push` → scripts/build_push.sh) в Docker Hub
#      osipovskijdima/groove_work (теги migrate/gateway/calls/auth/
#      messenger/ai/groove/tasks/front).
#   4. Перечитывает конфиг nginx (он bind-mounted: git обновляет
#      файл, но без reload контейнер живёт со старым конфигом).
#   5. Health-чеки: фронт и маршруты через nginx, healthz всех
#      микросервисов изнутри контейнеров, досягаемость gRPC между
#      сервисами, LiveKit через nginx (/livekit), TCP-порты медиа 7881
#      и TURN/TLS 5349.
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
ensure AI_KEY_ENCRYPTION_KEY "$(gen_fernet)"
ensure YOUGILE_ENC_KEY "$(gen_fernet)"
ensure LIVEKIT_API_KEY "gw2_$(openssl rand -hex 6)"
ensure LIVEKIT_API_SECRET "$(gen_hex)"

# PASETO-ключи (v3.6.0): пара Ed25519 для access-токенов генерируется
# ВМЕСТЕ — публичный обязан соответствовать приватному, поэтому если хотя бы
# одной половины нет, перегенерируем обе. Безопасно: access-токены живут
# 15 минут, сессии восстановятся по refresh-токену (его ключ отдельный
# и без нужды не трогается).
ensure_paseto_pair() {
  if grep -qE '^PASETO_PRIVATE_KEY=..*' "$ENV_FILE" && grep -qE '^PASETO_PUBLIC_KEY=..*' "$ENV_FILE"; then
    return 0
  fi
  backup_once
  local pem seed pub
  pem="$(mktemp)"
  openssl genpkey -algorithm ed25519 -out "$pem"
  # PKCS8 DER: последние 32 байта приватного — seed, публичного — ключ.
  seed=$(openssl pkey -in "$pem" -outform DER | tail -c 32 | od -An -v -tx1 | tr -d ' \n')
  pub=$(openssl pkey -in "$pem" -pubout -outform DER | tail -c 32 | od -An -v -tx1 | tr -d ' \n')
  rm -f "$pem"
  # go-paseto ждёт приватный как seed||public (64 байта hex).
  sed -i '/^PASETO_PRIVATE_KEY=/d;/^PASETO_PUBLIC_KEY=/d' "$ENV_FILE"
  printf 'PASETO_PRIVATE_KEY=%s%s\n' "$seed" "$pub" >> "$ENV_FILE"
  printf 'PASETO_PUBLIC_KEY=%s\n' "$pub" >> "$ENV_FILE"
  ok "сгенерирована пара PASETO_PRIVATE_KEY/PASETO_PUBLIC_KEY"
}
ensure_paseto_pair
ensure PASETO_REFRESH_KEY "$(gen_hex)"

# Устаревшие переменные: coturn (заменён LiveKit в v3.5.0), JWT-секрет
# flask-jwt-extended (заменён PASETO-ключами в v3.6.0) и SECRET_KEY
# Flask-сессий (Flask ликвидирован в фазе 5).
if grep -qE '^TURN_' "$ENV_FILE"; then
  backup_once
  sed -i '/^TURN_/d' "$ENV_FILE"
  ok "удалены устаревшие TURN_* (coturn заменён LiveKit)"
fi
if grep -qE '^JWT_SECRET_KEY=' "$ENV_FILE"; then
  backup_once
  sed -i '/^JWT_SECRET_KEY=/d' "$ENV_FILE"
  ok "удалён устаревший JWT_SECRET_KEY (JWT заменён PASETO)"
fi
if grep -qE '^SECRET_KEY=' "$ENV_FILE"; then
  backup_once
  sed -i '/^SECRET_KEY=/d' "$ENV_FILE"
  ok "удалён устаревший SECRET_KEY (Flask ликвидирован)"
fi
if grep -qE '^APP_TAG=' "$ENV_FILE"; then
  backup_once
  sed -i '/^APP_TAG=/d' "$ENV_FILE"
  ok "удалён устаревший APP_TAG (app-контейнер ликвидирован)"
fi

# Не генерируемое автоматически — только предупреждаем.
if ! grep -qE '^YOUGILE_WEBHOOK_PUBLIC_BASE=..*' "$ENV_FILE"; then
  warn "YOUGILE_WEBHOOK_PUBLIC_BASE пуст — вебхуки YouGile не зарегистрируются"
fi
# SMTP — секреты оператора (не генерируем). Без них прод-оверлей не поднимет
# mailsvc (HOST/FROM обязательны), а регистрация с подтверждением email не пройдёт.
if ! grep -qE '^SMTP_HOST=..*' "$ENV_FILE" || ! grep -qE '^SMTP_FROM=..*' "$ENV_FILE"; then
  warn "SMTP_HOST/SMTP_FROM пусты — заполните в $ENV_FILE, иначе письма подтверждения email не уйдут (mailsvc)"
fi
if ! grep -qE '^APP_PUBLIC_BASE_URL=..*' "$ENV_FILE"; then
  warn "APP_PUBLIC_BASE_URL пуст — ссылки подтверждения email используют домен по умолчанию"
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
    log "ufw активен — открываю медиа- и TURN-порты LiveKit"
    sudo -n ufw allow 7881/tcp >/dev/null
    sudo -n ufw allow 7882/udp >/dev/null
    sudo -n ufw allow 5349/tcp >/dev/null
    sudo -n ufw allow 3478/udp >/dev/null
    ok "открыты 7881/tcp, 7882/udp (медиа) и 5349/tcp, 3478/udp (TURN)"
  else
    ok "ufw неактивен — фильтрации нет, порты доступны"
  fi
else
  warn "ufw недоступен (нет команды или sudo) — проверьте порты 7881/tcp, 7882/udp, 5349/tcp, 3478/udp вручную"
fi

# ── 2b. UDP-буферы ядра для LiveKit (качество WebRTC под нагрузкой) ──
# pion/LiveKit при старте требует net.core.rmem_max ~5 МБ, иначе под нагрузкой
# растут потери аудио/видео-пакетов. Ставим с запасом (16 МБ), персистентно
# (/etc/sysctl.d, применится и после ребута до старта docker) и идемпотентно.
# sysctl глобальный, НЕ namespaced — Docker `sysctls:` его не примет, но
# контейнер и так видит хостовое значение. Буфер берётся при старте процесса,
# поэтому при первом подъёме livekit надо перезапустить (флаг ниже).
LK_BUF_TARGET=16777216
if command -v sysctl >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
  cur_rmem=$(sysctl -n net.core.rmem_max 2>/dev/null || echo 0)
  if [ "${cur_rmem:-0}" -lt "$LK_BUF_TARGET" ]; then
    printf 'net.core.rmem_max=%s\nnet.core.wmem_max=%s\n' "$LK_BUF_TARGET" "$LK_BUF_TARGET" \
      | sudo -n tee /etc/sysctl.d/99-grovework-livekit.conf >/dev/null
    sudo -n sysctl -q -w net.core.rmem_max="$LK_BUF_TARGET" net.core.wmem_max="$LK_BUF_TARGET"
    NEED_LK_RESTART=1
    ok "UDP-буферы подняты до $LK_BUF_TARGET (rmem/wmem)"
  else
    ok "UDP-буферы уже >= $LK_BUF_TARGET"
  fi
else
  warn "нет sudo/sysctl — подними net.core.rmem_max и net.core.wmem_max до $LK_BUF_TARGET вручную"
fi

# ── 3. Образы и запуск ───────────────────────────────────────────
# Сборки на сервере нет: тянем готовые образы из Docker Hub. Если
# репозиторий приватный — на сервере нужен одноразовый `docker login`.
log "Тяну образы из Docker Hub"
cd deploy
if ! $COMPOSE pull --quiet; then
  warn "docker compose pull не прошёл — образы не запушены (make push) или нужен docker login на сервере"
  exit 1
fi
log "Поднимаю контейнеры"
$COMPOSE up -d --no-build --remove-orphans
# UDP-буферы только что подняли (2b) — livekit читает их лишь при старте,
# поэтому перезапускаем его, чтобы предупреждение pion ушло сразу.
if [ "${NEED_LK_RESTART:-0}" = 1 ]; then
  log "Перезапускаю livekit под новые UDP-буферы"
  $COMPOSE restart livekit >/dev/null 2>&1 || true
fi
# Подчищаем: слои, осиротевшие после переезда тегов, и build-кэш
# (он остался от прежней схемы со сборкой на сервере и больше не нужен).
docker image prune -f >/dev/null 2>&1 || true
docker builder prune -af >/dev/null 2>&1 || true

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
# curl -kL: прод-nginx отвечает на http 301-редиректом на https с
# self-issued для localhost сертификатом — идём за редиректом без проверки.
log "Жду готовности фронта (миграции на старте могут занять время)"
front_code=000
for _ in $(seq 1 30); do
  front_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/ || true)
  [ "$front_code" = "200" ] && break
  sleep 2
done
if [ "$front_code" = "200" ]; then
  ok "фронт отвечает через nginx"
else
  warn "фронт не ответил за 60с (код $front_code) — смотрите: make logs"
fi

# Realtime-шлюз: healthz изнутри контейнера + exact-роут presence через
# nginx (без токена ожидаем 401 от gateway, не 404/502).
if $COMPOSE exec -T gateway wget -qO- --timeout=3 http://127.0.0.1:8096/healthz >/dev/null 2>&1; then
  ok "gatewaysvc отвечает (healthz)"
else
  warn "gatewaysvc не отвечает — realtime и звонки не работают: make logs s=gateway"
fi
presence_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/api/messenger/presence || true)
if [ "$presence_code" = "401" ]; then
  ok "маршрут /api/messenger/presence через nginx ведёт в gateway"
else
  warn "маршрут /api/messenger/presence вернул $presence_code (ожидался 401) — проверьте nginx"
fi

# Микросервис звонков: HTTP-healthz изнутри контейнера (наружу порт не
# торчит, через nginx ходит только /api/calls/*) + досягаемость gRPC
# из gateway (ринг-фаза).
if $COMPOSE exec -T calls wget -qO- --timeout=3 http://127.0.0.1:8090/healthz >/dev/null 2>&1; then
  ok "callsvc отвечает (healthz)"
else
  warn "callsvc не отвечает — звонки не работают: make logs s=calls"
fi
if $COMPOSE exec -T gateway sh -c "wget -q --timeout=3 -O /dev/null http://calls:8090/healthz" >/dev/null 2>&1; then
  ok "callsvc досягаем из gateway (calls:8090)"
else
  warn "gateway не достучался до callsvc — ринг-фаза звонков не работает"
fi

# Микросервис авторизации: healthz изнутри контейнера + маршрутизация
# /api/auth/* через nginx (login без тела должен вернуть 400 от authsvc,
# а не 404/502 — значит, префикс уехал в нужный сервис).
if $COMPOSE exec -T auth wget -qO- --timeout=3 http://127.0.0.1:8091/healthz >/dev/null 2>&1; then
  ok "authsvc отвечает (healthz)"
else
  warn "authsvc не отвечает — вход в систему не работает: make logs s=auth"
fi
auth_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 -X POST http://localhost/api/auth/login || true)
if [ "$auth_code" = "400" ]; then
  ok "маршрут /api/auth/ через nginx ведёт в authsvc"
else
  warn "маршрут /api/auth/ вернул $auth_code (ожидался 400) — проверьте nginx"
fi
# Компании уехали в authsvc: без токена ожидаем 401 от него (не 404/502).
companies_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/api/companies || true)
if [ "$companies_code" = "401" ]; then
  ok "маршрут /api/companies через nginx ведёт в authsvc"
else
  warn "маршрут /api/companies вернул $companies_code (ожидался 401) — проверьте nginx"
fi
# Лог изменений — статика nginx (bind-mount changelog.json).
changelog_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/api/changelog || true)
if [ "$changelog_code" = "200" ]; then
  ok "/api/changelog отдаётся статикой"
else
  warn "/api/changelog вернул $changelog_code (ожидался 200) — проверьте bind-mount changelog.json"
fi

# Микросервис мессенджера: HTTP-healthz изнутри контейнера (наружу порт не
# торчит, через nginx ходит только /api/messenger).
if $COMPOSE exec -T messenger wget -qO- --timeout=3 http://127.0.0.1:8092/healthz >/dev/null 2>&1; then
  ok "msgsvc отвечает (healthz)"
else
  warn "msgsvc не отвечает — мессенджер не работает: make logs s=messenger"
fi

# Микросервис ИИ: HTTP-healthz изнутри контейнера (наружу порт не торчит,
# через nginx ходит только regex /api/companies/<id>/ai-settings).
if $COMPOSE exec -T ai wget -qO- --timeout=3 http://127.0.0.1:8093/healthz >/dev/null 2>&1; then
  ok "aisvc отвечает (healthz)"
else
  warn "aisvc не отвечает — ИИ-фичи не работают: make logs s=ai"
fi
# ТВ-факт уехал в aisvc: без токена ожидаем 401 от него (не 404/502).
tvfact_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/api/ai/tv-fact || true)
if [ "$tvfact_code" = "401" ]; then
  ok "маршрут /api/ai через nginx ведёт в aisvc"
else
  warn "маршрут /api/ai/tv-fact вернул $tvfact_code (ожидался 401) — проверьте nginx"
fi
if $COMPOSE exec -T groove wget -qO- --timeout=3 http://127.0.0.1:8094/healthz >/dev/null 2>&1; then
  ok "groovesvc отвечает (healthz)"
else
  warn "groovesvc не отвечает — «Мой Groove» не работает: make logs s=groove"
fi

# Микросервис задач: healthz изнутри контейнера + маршрутизация /api/tasks
# через nginx (без токена ожидаем 401 от tasksvc, не 404/502).
if $COMPOSE exec -T tasks wget -qO- --timeout=3 http://127.0.0.1:8095/healthz >/dev/null 2>&1; then
  ok "tasksvc отвечает (healthz)"
else
  warn "tasksvc не отвечает — задачи и статистика не работают: make logs s=tasks"
fi
tasks_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/api/tasks || true)
if [ "$tasks_code" = "401" ]; then
  ok "маршрут /api/tasks через nginx ведёт в tasksvc"
else
  warn "маршрут /api/tasks вернул $tasks_code (ожидался 401) — проверьте nginx"
fi

# Микросервис пуш-уведомлений: healthz изнутри контейнера + маршрут
# /api/push/register через nginx (без токена ожидаем 401, не 404/502).
if $COMPOSE exec -T push wget -qO- --timeout=3 http://127.0.0.1:8097/healthz >/dev/null 2>&1; then
  ok "pushsvc отвечает (healthz)"
else
  warn "pushsvc не отвечает — пуш-уведомления не работают: make logs s=push"
fi
push_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 -X POST http://localhost/api/push/register || true)
if [ "$push_code" = "401" ]; then
  ok "маршрут /api/push через nginx ведёт в pushsvc"
else
  warn "маршрут /api/push вернул $push_code (ожидался 401) — проверьте nginx"
fi

# Микросервис реестров: healthz изнутри контейнера + маршрут /api/registries
# через nginx (без токена ожидаем 401, не 404/502).
if $COMPOSE exec -T registry wget -qO- --timeout=3 http://127.0.0.1:8099/healthz >/dev/null 2>&1; then
  ok "registrysvc отвечает (healthz)"
else
  warn "registrysvc не отвечает — реестры не работают: make logs s=registry"
fi
registry_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/api/registries || true)
if [ "$registry_code" = "401" ]; then
  ok "маршрут /api/registries через nginx ведёт в registrysvc"
else
  warn "маршрут /api/registries вернул $registry_code (ожидался 401) — проверьте nginx"
fi

# Микросервис календарей: healthz изнутри контейнера + маршрут /api/calendars
# через nginx (без токена ожидаем 401, не 404/502).
if $COMPOSE exec -T calendar wget -qO- --timeout=3 http://127.0.0.1:8100/healthz >/dev/null 2>&1; then
  ok "calendarsvc отвечает (healthz)"
else
  warn "calendarsvc не отвечает — календари не работают: make logs s=calendar"
fi
calendar_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/api/calendars || true)
if [ "$calendar_code" = "401" ]; then
  ok "маршрут /api/calendars через nginx ведёт в calendarsvc"
else
  warn "маршрут /api/calendars вернул $calendar_code (ожидался 401) — проверьте nginx"
fi

# Микросервис почты: healthz изнутри контейнера (наружу не торчит — gRPC-only,
# через nginx не проксируется).
if $COMPOSE exec -T mail wget -qO- --timeout=3 http://127.0.0.1:8098/healthz >/dev/null 2>&1; then
  ok "mailsvc отвечает (healthz)"
else
  warn "mailsvc не отвечает — письма подтверждения email не уйдут: make logs s=mail"
fi

lk_code=$(curl -skL -o /dev/null -w '%{http_code}' --max-time 5 http://localhost/livekit/ || true)
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

if (exec 3<>/dev/tcp/127.0.0.1/5349) 2>/dev/null; then
  exec 3>&- 3<&-
  ok "TURN/TLS-порт 5349/tcp слушает (3478/udp проверит сам webrtc через relay)"
else
  warn "TURN/TLS-порт 5349/tcp не слушает — LiveKit не нашёл cert или turn выключен; звонки с мобильных/VPN-сетей не пробьются"
fi

$COMPOSE ps --format 'table {{.Name}}\t{{.Status}}'
ok "Деплой завершён"

#!/usr/bin/env bash
# Одноразовый перенос файлов из локального uploads-тома в S3-бакет Beget с
# сохранением ключей (registry/, calendar/, messages/, avatars/) и пометкой
# каждого объекта public-read (так nginx отдаёт /uploads/ анонимно).
#
# Запускать на сервере ОДИН раз (идемпотентно — можно повторять для досинка).
# Использует Go-инструмент pkg/cmd/uploadmigrate в одноразовом golang-контейнере
# с примонтированным томом. Креды берёт из deploy/.env.
#
#   bash scripts/migrate_uploads_s3.sh
set -euo pipefail

cd "$(dirname "$0")/.."

ENV_FILE="deploy/.env"
VOLUME="${UPLOADS_VOLUME:-deploy_uploads}" # docker-compose project=deploy → deploy_uploads
GO_IMAGE="${GO_IMAGE:-golang:1.26}"

log()  { printf '\033[1m▶ %s\033[0m\n' "$*"; }
ok()   { printf '\033[32m✓ %s\033[0m\n' "$*"; }
fail() { printf '\033[31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

[ -f "$ENV_FILE" ] || fail "$ENV_FILE не найден"
# shellcheck disable=SC1090
set -a; . "$ENV_FILE"; set +a

: "${S3_ENDPOINT:?S3_ENDPOINT не задан в $ENV_FILE}"
: "${S3_BUCKET:?S3_BUCKET не задан в $ENV_FILE}"
: "${S3_ACCESS_KEY:?S3_ACCESS_KEY не задан в $ENV_FILE}"
: "${S3_SECRET_KEY:?S3_SECRET_KEY не задан в $ENV_FILE}"

docker volume inspect "$VOLUME" >/dev/null 2>&1 || fail "том $VOLUME не найден (задайте UPLOADS_VOLUME=…)"

log "Переношу том $VOLUME → s3://$S3_BUCKET ($S3_ENDPOINT), public-read на объект"
docker run --rm \
  -v "$VOLUME":/data:ro \
  -v "$PWD/back-go/pkg":/src \
  -w /src \
  -e GOWORK=off \
  -e UPLOAD_FOLDER=/data \
  -e S3_ENDPOINT="$S3_ENDPOINT" \
  -e S3_REGION="${S3_REGION:-ru1}" \
  -e S3_BUCKET="$S3_BUCKET" \
  -e S3_ACCESS_KEY="$S3_ACCESS_KEY" \
  -e S3_SECRET_KEY="$S3_SECRET_KEY" \
  "$GO_IMAGE" go run ./cmd/uploadmigrate

ok "Готово. Проверьте объекты: registry/ calendar/ messages/ avatars/"

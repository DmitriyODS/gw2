#!/usr/bin/env bash
# Проверка (и починка) анонимного доступа к объектам бакета.
#
# Файлы платформы раздаёт nginx проксированием на бакет, а анонимное чтение
# держится на ACL public-read КАЖДОГО объекта: ключ Beget не имеет прав на
# политику бакета. Объект, оставшийся приватным, отвечает браузеру 403 — и
# карточка записи показывает пустое место, хотя файл на месте. Тем же 403
# хранилище отвечает и на ЗАПРОС НЕСУЩЕСТВУЮЩЕГО объекта, поэтому причину
# видно только отсюда, с ключом доступа.
#
# Запуск на сервере (креды из deploy/.env):
#
#   bash scripts/s3_public_acl.sh                     # проверить весь бакет
#   bash scripts/s3_public_acl.sh registry/           # только реестры
#   bash scripts/s3_public_acl.sh registry/ --fix     # и проставить public-read
#   KEYS=registry/a.jpg,registry/b.jpg bash scripts/s3_public_acl.sh
set -euo pipefail

cd "$(dirname "$0")/.."

ENV_FILE="deploy/.env"
GO_IMAGE="${GO_IMAGE:-golang:1.26}"
PREFIX=""
FIX=""
for arg in "$@"; do
  case "$arg" in
    --fix) FIX="-fix" ;;
    *)     PREFIX="$arg" ;;
  esac
done

log()  { printf '\033[1m▶ %s\033[0m\n' "$*"; }
fail() { printf '\033[31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

[ -f "$ENV_FILE" ] || fail "$ENV_FILE не найден"
getenv() { grep -E "^$1=" "$ENV_FILE" | head -1 | cut -d= -f2-; }
S3_ENDPOINT="$(getenv S3_ENDPOINT)"
S3_REGION="$(getenv S3_REGION)"
S3_BUCKET="$(getenv S3_BUCKET)"
S3_ACCESS_KEY="$(getenv S3_ACCESS_KEY)"
S3_SECRET_KEY="$(getenv S3_SECRET_KEY)"

: "${S3_ENDPOINT:?S3_ENDPOINT не задан в $ENV_FILE}"
: "${S3_BUCKET:?S3_BUCKET не задан в $ENV_FILE}"
: "${S3_ACCESS_KEY:?S3_ACCESS_KEY не задан в $ENV_FILE}"
: "${S3_SECRET_KEY:?S3_SECRET_KEY не задан в $ENV_FILE}"

log "Бакет $S3_BUCKET ($S3_ENDPOINT), префикс «${PREFIX:-весь бакет}»${FIX:+, с починкой}"
docker run --rm \
  -v "$PWD/back-go/pkg":/src \
  -w /src \
  -e GOWORK=off \
  -e S3_ENDPOINT="$S3_ENDPOINT" \
  -e S3_REGION="${S3_REGION:-ru1}" \
  -e S3_BUCKET="$S3_BUCKET" \
  -e S3_ACCESS_KEY="$S3_ACCESS_KEY" \
  -e S3_SECRET_KEY="$S3_SECRET_KEY" \
  "$GO_IMAGE" go run ./cmd/s3acl -prefix "$PREFIX" -keys "${KEYS:-}" $FIX

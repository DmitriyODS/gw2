#!/usr/bin/env bash
# ================================================================
# Заливка демо-данных в локальную dev-БД (scripts/seed_dev.sql):
# компания «Грув Демо», сотрудники, грувики во всех состояниях,
# портал с ветками комментариев и лайками, задачи с юнитами.
#
# Идемпотентен: прежний посев (компания «Грув Демо» и пользователи
# demo.*) вычищается и создаётся заново. Пароль аккаунтов: demo1234
#
# Использование (см. также `make dev-seed`):
#   scripts/seed_dev.sh
# ================================================================
set -euo pipefail

DB_USER="${DB_USER:-grovework}"
DB_NAME="${DB_NAME:-grovework}"
SQL_FILE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/seed_dev.sql"

CID="$(docker compose -f deploy/docker-compose.yml ps -q db 2>/dev/null || true)"
if [[ -z "$CID" ]]; then
  echo "Контейнер db не запущен. Подними dev-инфру: make dev-infra" >&2
  exit 1
fi

docker exec -i "$CID" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 < "$SQL_FILE"

printf '\n\033[32m✓ Демо-данные залиты.\033[0m Вход: demo.admin … demo.elena / demo1234\n'

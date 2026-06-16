#!/usr/bin/env bash
# ================================================================
# Сброс пароля суперадмина в локальной dev-БД.
#
# Аналог reset_superadmin_password.sh, но без SSH: работает с
# контейнером db из deploy/docker-compose (dev-инфра). Одним SQL-запросом
# перевыставляет hash_password у супер-админа (users.is_super_admin) с
# самым младшим id. Пароль хешируется тем же pgcrypto, что и в приложении:
# crypt(pwd, gen_salt('bf')).
#
# Использование (см. также `make dev-reset NEWPASS=...`):
#   scripts/reset_superadmin_password_dev.sh <new-password>
#
# Пароль не светится в `ps`: уезжает в контейнер БД через stdin во
# временный файл, который psql читает backquote-подстановкой.
# ================================================================
set -euo pipefail

NEW_PASS="${1:-}"

if [[ -z "$NEW_PASS" ]]; then
  echo "Использование: $0 <new-password>" >&2
  exit 2
fi

# Минимальная валидация — синхронизировано с фронтом (>= 8 символов).
if [[ "${#NEW_PASS}" -lt 8 ]]; then
  echo "Пароль должен быть не короче 8 символов." >&2
  exit 2
fi

DB_USER="${DB_USER:-grovework}"
DB_NAME="${DB_NAME:-grovework}"

CID="$(docker compose -f deploy/docker-compose.yml ps -q db 2>/dev/null || true)"
if [[ -z "$CID" ]]; then
  echo "Контейнер db не запущен. Подними dev-инфру: make dev-infra" >&2
  exit 1
fi

# Кладём пароль во временный файл внутри контейнера и трём в trap —
# никаких аргументов с паролем (ни у docker exec, ни у psql).
printf '%s' "$NEW_PASS" | docker exec -i "$CID" sh -c 'cat > /tmp/gw2_newpass && chmod 600 /tmp/gw2_newpass'
trap 'docker exec "$CID" sh -c "rm -f /tmp/gw2_newpass" >/dev/null 2>&1 || true' EXIT

docker exec -i "$CID" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 <<'SQL'
\set newpass `cat /tmp/gw2_newpass`
WITH target AS (
  SELECT id
  FROM users
  WHERE is_super_admin
  ORDER BY id ASC
  LIMIT 1
)
UPDATE users u
SET hash_password   = crypt(:'newpass', gen_salt('bf')),
    is_default_pass = FALSE
FROM target
WHERE u.id = target.id
RETURNING u.id AS user_id, u.login, u.fio;
SQL

echo "✓ Пароль суперадмина в dev-БД сброшен."

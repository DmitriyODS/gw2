#!/usr/bin/env bash
# ================================================================
# Сброс пароля суперадмина на продакшен-сервере.
#
# Запускается на хосте — через SSH открывает psql в контейнере БД и
# одним SQL-запросом перевыставляет hash_password у суперадмина (роль
# уровня 4) с самым младшим id (это «системный» суперадмин, защищён
# на уровне приложения от скрытия/смены роли). Пароль хешируется тем
# же способом, что и в приложении — pgcrypto: crypt(pwd, gen_salt('bf')).
#
# Использование (см. также `make reset NEWPASS=...`):
#   scripts/reset_superadmin_password.sh <user@host> <key> <server-dir> <new-password>
#
# Передача пароля: уезжает на сервер только как переменная окружения
# для запускаемого `bash -s` — не светится в `ps` (cmdline не содержит
# env), env-секция /proc/PID/environ читается только владельцем процесса.
# Это тот же паттерн, по которому на сервере и так передаётся
# DB_PASSWORD → PGPASSWORD.
# ================================================================
set -euo pipefail

SERVER="${1:-}"
SSH_KEY="${2:-}"
SERVER_DIR="${3:-}"
NEW_PASS="${4:-}"

if [[ -z "$SERVER" || -z "$SSH_KEY" || -z "$SERVER_DIR" || -z "$NEW_PASS" ]]; then
  echo "Использование: $0 <user@host> <ssh-key> <server-dir> <new-password>" >&2
  exit 2
fi

# Минимальная валидация — синхронизировано с фронтом (>= 8 символов).
if [[ "${#NEW_PASS}" -lt 8 ]]; then
  echo "Пароль должен быть не короче 8 символов." >&2
  exit 2
fi

# Удалённый bash-скрипт. Пароль читается из переменной GW2_NEW_PASSWORD,
# которую выставим в первой строке heredoc. SERVER_DIR подставляем сейчас.
read -r -d '' REMOTE_SCRIPT <<REMOTE || true
set -euo pipefail
cd "$SERVER_DIR/deploy"

# .env с DB_USER/DB_NAME/DB_PASSWORD лежит рядом с compose-файлом.
set -a; . ./.env; set +a

COMPOSE="docker compose -f docker-compose.prod.yml"
CID="\$(\$COMPOSE ps -q db)"
if [[ -z "\$CID" ]]; then
  echo "Контейнер db не запущен." >&2
  exit 1
fi

# Кладём пароль во временный файл внутри контейнера БД и трём его в trap.
# Никаких аргументов с паролем (ни у docker exec, ни у psql) — поэтому он
# не виден в cmdline ни на хосте, ни в контейнере.
printf '%s' "\$GW2_NEW_PASSWORD" | docker exec -i "\$CID" sh -c 'cat > /tmp/gw2_newpass && chmod 600 /tmp/gw2_newpass'
trap 'docker exec "\$CID" sh -c "rm -f /tmp/gw2_newpass" >/dev/null 2>&1 || true' EXIT

# SQL-скрипт уезжает в psql через stdin — backquote-подстановку файла
# выполняет сам psql, поэтому пароль никогда не попадает в аргументы.
docker exec -i -e PGPASSWORD="\$DB_PASSWORD" "\$CID" \\
  psql -U "\$DB_USER" -d "\$DB_NAME" -v ON_ERROR_STOP=1 <<'SQL'
\set newpass \`cat /tmp/gw2_newpass\`
WITH target AS (
  SELECT u.id
  FROM users u
  JOIN roles r ON r.id = u.role_id
  WHERE r.level = 4
  ORDER BY u.id ASC
  LIMIT 1
)
UPDATE users u
SET hash_password   = crypt(:'newpass', gen_salt('bf')),
    is_default_pass = FALSE
FROM target
WHERE u.id = target.id
RETURNING u.id AS user_id, u.login, u.fio;
SQL
REMOTE

# Пароль выставляем в первой строке тела heredoc — он попадает в bash
# через stdin и не виден в cmdline (`ps auxf`). Одинарные кавычки внутри
# пароля экранируем стандартным паттерном  '  →  '\''.
SAFE_PASS=$(printf '%s' "$NEW_PASS" | sed "s/'/'\\\\''/g")
ssh -i "$SSH_KEY" "$SERVER" "bash -s" <<EOF
GW2_NEW_PASSWORD='$SAFE_PASS'
$REMOTE_SCRIPT
EOF

echo "✓ Пароль суперадмина сброшен."

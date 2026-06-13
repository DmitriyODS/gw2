#!/bin/sh
set -e

# Общий uploads-том Docker монтирует от root, а msgsvc бежит непривилегированным
# — без этого шага os.MkdirAll/WriteFile вложений падают с permission denied
# (HTTP 500 на POST /api/messenger/uploads). Чиним владельца каталога на старте
# (контейнер для этого стартует root'ом) и тут же сбрасываем привилегии на
# msgsvc через su-exec — сам сервис работает не от root.
# Рекурсивно: на уже существующем root-owned томе подкаталоги
# (messages/<год>/<мес>, avatars) могли быть созданы под root до этого фикса —
# без -R запись в них падает permission denied даже после chown самого тома.
DIR="${UPLOAD_FOLDER:-/app/uploads}"
mkdir -p "$DIR"
chown -R msgsvc:msgsvc "$DIR" 2>/dev/null || true

exec su-exec msgsvc "$@"

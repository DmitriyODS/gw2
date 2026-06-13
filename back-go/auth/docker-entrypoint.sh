#!/bin/sh
set -e

# Общий uploads-том Docker монтирует от root, а authsvc бежит непривилегированным
# — без этого шага запись аватарок в UPLOAD_FOLDER/avatars падает с permission
# denied. Чиним владельца каталога на старте (контейнер для этого стартует
# root'ом) и тут же сбрасываем привилегии на authsvc через su-exec — сам сервис
# работает не от root.
DIR="${UPLOAD_FOLDER:-/app/uploads}"
mkdir -p "$DIR"
chown authsvc:authsvc "$DIR" 2>/dev/null || true

exec su-exec authsvc "$@"

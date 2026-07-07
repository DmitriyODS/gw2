#!/bin/sh
set -e

# Общий uploads-том Docker монтирует от root, а portalsvc бежит
# непривилегированным — без этого шага запись вложений постов падает с
# permission denied. Чиним владельца каталога на старте (контейнер для этого
# стартует root'ом) и тут же сбрасываем привилегии на portalsvc через
# su-exec. -R: подкаталоги могли быть созданы под root другими сервисами.
DIR="${UPLOAD_FOLDER:-/app/uploads}"
mkdir -p "$DIR"
chown -R portalsvc:portalsvc "$DIR" 2>/dev/null || true

exec su-exec portalsvc "$@"

#!/bin/sh
set -e

# Общий uploads-том Docker монтирует от root, а formsvc бежит
# непривилегированным — без этого шага запись приложенных к ответам файлов
# падает с permission denied. Чиним владельца каталога на старте (контейнер
# для этого стартует root'ом) и тут же сбрасываем привилегии на formsvc через
# su-exec. -R: подкаталоги могли быть созданы под root другими сервисами.
DIR="${UPLOAD_FOLDER:-/app/uploads}"
mkdir -p "$DIR"
chown -R formsvc:formsvc "$DIR" 2>/dev/null || true

exec su-exec formsvc "$@"

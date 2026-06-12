#!/usr/bin/env bash
# ================================================================
# Сборка и публикация прод-образов в Docker Hub. Запускается ЛОКАЛЬНО
# (`make push`); на сервере ничего не собирается — деплой делает только
# `docker compose pull`.
#
# Все сервисы живут в ОДНОМ репозитории Docker Hub под разными тегами:
#   osipovskijdima/groove_work:app        — Flask-монолит  (back/)
#   osipovskijdima/groove_work:calls      — callsvc, Go    (back-go/calls/)
#   osipovskijdima/groove_work:auth       — authsvc, Go    (back-go/auth/)
#   osipovskijdima/groove_work:messenger  — msgsvc, Go     (back-go/messenger/)
#   osipovskijdima/groove_work:ai         — aisvc, Go      (back-go/ai/)
#   osipovskijdima/groove_work:groove     — groovesvc, Go  (back-go/groove/)
#   osipovskijdima/groove_work:tasks      — tasksvc, Go    (back-go/tasks/)
#   osipovskijdima/groove_work:front      — nginx + SPA    (front/)
#
# Дополнительно каждый образ получает версионный тег `<svc>-X.Y.Z`
# (версия из front/package.json) — для отката: на сервере в deploy/.env
# выставить APP_TAG=app-X.Y.Z (CALLS_TAG / AUTH_TAG / MESSENGER_TAG / GROOVE_TAG /
# AI_TAG / TASKS_TAG / FRONT_TAG — аналогично) и перезапустить деплой.
#
# Требуется один раз: `docker login` под аккаунтом с правом push.
#
# Использование:
#   scripts/build_push.sh              # все образы
#   scripts/build_push.sh app front    # выборочно
# ================================================================
set -euo pipefail
cd "$(cd "$(dirname "$0")/.." && pwd)"

REPO="${DOCKER_REPO:-osipovskijdima/groove_work}"
# Прод — linux/amd64. На Apple Silicon: Go-стадии кросс-компилируют нативно
# (см. $BUILDPLATFORM в Dockerfile), python/node-стадии бегут под Rosetta.
PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"
VERSION="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' front/package.json | head -1)"

log() { printf '\033[1m▶ %s\033[0m\n' "$*"; }
ok()  { printf '\033[32m✓ %s\033[0m\n' "$*"; }

context_of() {
  case "$1" in
    app)   echo back ;;
    front) echo front ;;
    # Go-сервисы собираются из общего контекста back-go/ (модуль pkg
    # подключён через replace ../pkg), Dockerfile — внутри сервиса.
    calls|auth|messenger|ai|groove|tasks) echo back-go ;;
    *) printf 'Неизвестный сервис: %s (ожидается app|calls|auth|messenger|ai|groove|tasks|front)\n' "$1" >&2; return 2 ;;
  esac
}

dockerfile_of() {
  case "$1" in
    calls|auth|messenger|ai|groove|tasks) echo "back-go/$1/Dockerfile" ;;
    *) echo "" ;;
  esac
}

build_push() {
  local tag="$1" ctx="$2" df="$3"
  local args=(-t "$REPO:$tag")
  [ -n "$VERSION" ] && args+=(-t "$REPO:$tag-$VERSION")
  [ -n "$df" ] && args+=(-f "$df")
  log "Собираю и пушу $REPO:$tag ($PLATFORM, контекст $ctx)"
  docker buildx build --platform "$PLATFORM" "${args[@]}" --push "$ctx"
  ok "$REPO:$tag${VERSION:+  (+ $tag-$VERSION)}"
}

SERVICES=("$@")
[ ${#SERVICES[@]} -eq 0 ] && SERVICES=(app calls auth messenger ai groove tasks front)

for svc in "${SERVICES[@]}"; do
  build_push "$svc" "$(context_of "$svc")" "$(dockerfile_of "$svc")"
done
ok "Образы опубликованы в Docker Hub ($REPO)"

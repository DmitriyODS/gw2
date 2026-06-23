#!/usr/bin/env bash
# ================================================================
# Сборка и публикация прод-образов в Docker Hub. Запускается ЛОКАЛЬНО
# (`make push`); на сервере ничего не собирается — деплой делает только
# `docker compose pull`.
#
# Все сервисы живут в ОДНОМ репозитории Docker Hub под разными тегами:
#   osipovskijdima/groove_work:migrate    — миграции, Go    (back-go/migrate/)
#   osipovskijdima/groove_work:gateway    — gatewaysvc, Go  (back-go/gateway/)
#   osipovskijdima/groove_work:calls      — callsvc, Go     (back-go/calls/)
#   osipovskijdima/groove_work:auth       — authsvc, Go     (back-go/auth/)
#   osipovskijdima/groove_work:messenger  — msgsvc, Go      (back-go/messenger/)
#   osipovskijdima/groove_work:ai         — aisvc, Go       (back-go/ai/)
#   osipovskijdima/groove_work:groove     — groovesvc, Go   (back-go/groove/)
#   osipovskijdima/groove_work:tasks      — tasksvc, Go     (back-go/tasks/)
#   osipovskijdima/groove_work:push       — pushsvc, Go     (back-go/push/)
#   osipovskijdima/groove_work:front      — nginx + SPA     (front/)
#
# Дополнительно каждый образ получает версионный тег `<svc>-X.Y.Z`
# (версия из front/package.json) — для отката: на сервере в deploy/.env
# выставить GATEWAY_TAG=gateway-X.Y.Z (MIGRATE_TAG / CALLS_TAG / AUTH_TAG /
# MESSENGER_TAG / GROOVE_TAG / AI_TAG / TASKS_TAG / FRONT_TAG — аналогично)
# и перезапустить деплой.
#
# Требуется один раз: `docker login` под аккаунтом с правом push.
#
# Использование:
#   scripts/build_push.sh                # все образы
#   scripts/build_push.sh gateway front  # выборочно
#   scripts/build_push.sh --changed      # только реально изменившиеся
#                                         (git diff origin/main..рабочее дерево;
#                                          back-go/pkg/* → все Go-сервисы;
#                                          base переопределяется CHANGED_BASE)
# ================================================================
set -euo pipefail
cd "$(cd "$(dirname "$0")/.." && pwd)"

REPO="${DOCKER_REPO:-osipovskijdima/groove_work}"
ALL_SERVICES=(migrate gateway calls auth messenger ai groove tasks push mail registry calendar front)
# Прод — linux/amd64. На Apple Silicon: Go-стадии кросс-компилируют нативно
# (см. $BUILDPLATFORM в Dockerfile), python/node-стадии бегут под Rosetta.
PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"
VERSION="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' front/package.json | head -1)"

log() { printf '\033[1m▶ %s\033[0m\n' "$*"; }
ok()  { printf '\033[32m✓ %s\033[0m\n' "$*"; }

context_of() {
  case "$1" in
    front) echo front ;;
    # Go-сервисы собираются из общего контекста back-go/ (модуль pkg
    # подключён через replace ../pkg), Dockerfile — внутри сервиса.
    migrate|gateway|calls|auth|messenger|ai|groove|tasks|push|mail|registry|calendar) echo back-go ;;
    *) printf 'Неизвестный сервис: %s (ожидается migrate|gateway|calls|auth|messenger|ai|groove|tasks|push|mail|registry|calendar|front)\n' "$1" >&2; return 2 ;;
  esac
}

dockerfile_of() {
  case "$1" in
    migrate|gateway|calls|auth|messenger|ai|groove|tasks|push|mail|registry|calendar) echo "back-go/$1/Dockerfile" ;;
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

# changed_services — какие образы пересобирать по git-диффу рабочего дерева
# против задеплоенного состояния (origin/main). Покрывает и коммиты, и ещё не
# закоммиченные правки. Карта путь→сервис; back-go/pkg/* (общий модуль) тянет
# пересборку всех Go-сервисов. Без bash-4 (на macOS системный bash 3.2).
changed_services() {
  local base="${CHANGED_BASE:-origin/main}"
  if ! git rev-parse --verify -q "$base" >/dev/null 2>&1; then
    printf 'changed: нет %s для сравнения — собираю все образы\n' "$base" >&2
    printf '%s\n' "${ALL_SERVICES[@]}"
    return
  fi
  # git diff не видит НОВЫЕ (untracked) файлы, а docker собирает из рабочего
  # дерева — поэтому добавляем и их (git ls-files --others), иначе свежесозданный
  # файл сервиса не вызвал бы его пересборку.
  local files; files="$( { git diff --name-only "$base" --; git ls-files --others --exclude-standard; } 2>/dev/null || true )"
  local hits="" go=0 front=0 unknown="" f
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    case "$f" in
      front/*) front=1 ;;
      back-go/pkg/*) go=1 ;;
      back-go/migrate/*) hits="$hits migrate" ;;
      back-go/gateway/*) hits="$hits gateway" ;;
      back-go/calls/*) hits="$hits calls" ;;
      back-go/auth/*) hits="$hits auth" ;;
      back-go/messenger/*) hits="$hits messenger" ;;
      back-go/ai/*) hits="$hits ai" ;;
      back-go/groove/*) hits="$hits groove" ;;
      back-go/tasks/*) hits="$hits tasks" ;;
      back-go/push/*) hits="$hits push" ;;
      back-go/mail/*) hits="$hits mail" ;;
      back-go/registry/*) hits="$hits registry" ;;
      back-go/calendar/*) hits="$hits calendar" ;;
      deploy/*|data/*|scripts/*|*.md|.gitignore|.env*) : ;; # bind-mount/серверное — образ не трогаем
      *) unknown="$unknown $f" ;;
    esac
  done <<EOF
$files
EOF
  [ "$go" = 1 ] && hits="$hits migrate gateway calls auth messenger ai groove tasks push mail registry calendar"
  [ "$front" = 1 ] && hits="$hits front"
  [ -n "$unknown" ] && printf 'changed: не отнёс к сервисам (образы не трогаю):%s\n' "$unknown" >&2
  local s
  for s in "${ALL_SERVICES[@]}"; do
    case " $hits " in *" $s "*) echo "$s" ;; esac
  done
}

if [ "${1:-}" = "--changed" ]; then
  SERVICES=()
  while IFS= read -r line; do
    [ -n "$line" ] && SERVICES+=("$line")
  done < <(changed_services)
  if [ ${#SERVICES[@]} -eq 0 ]; then
    ok "Изменившихся сервисов нет — пересобирать нечего"
    exit 0
  fi
  log "Изменились: ${SERVICES[*]}"
else
  SERVICES=("$@")
  [ ${#SERVICES[@]} -eq 0 ] && SERVICES=("${ALL_SERVICES[@]}")
fi

for svc in "${SERVICES[@]}"; do
  build_push "$svc" "$(context_of "$svc")" "$(dockerfile_of "$svc")"
done
ok "Образы опубликованы в Docker Hub ($REPO)"

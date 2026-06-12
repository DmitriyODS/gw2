#!/usr/bin/env bash
# Кодогенерация gRPC-контрактов Go-микросервисов (back-go/<svc>/api/proto).
#   Go → back-go/pkg/gen/<svc>pb (buf + protoc-gen-go{,-grpc}; общий
#        модуль pkg — единственное место Go-стабов, их импортируют и
#        сервис-владелец, и сервисы-клиенты)
# Запускать после любого изменения *.proto; результат коммитится.
# Можно выборочно: scripts/gen_proto.sh calls messenger
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

export PATH="$PATH:$HOME/go/bin"

ALL_SERVICES=(calls messenger ai groove)

if [ "$#" -gt 0 ]; then
  SERVICES=()
  for want in "$@"; do
    for entry in "${ALL_SERVICES[@]}"; do
      [ "$entry" = "$want" ] && SERVICES+=("$entry")
    done
  done
else
  SERVICES=("${ALL_SERVICES[@]}")
fi

for svc in "${SERVICES[@]}"; do
  svc_dir="$ROOT/back-go/$svc"
  echo "▶ [$svc] Go-стабы (buf generate)…"
  (cd "$svc_dir" && buf lint && buf generate)
done

echo "✓ Стабы обновлены: back-go/pkg/gen"

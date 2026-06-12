#!/usr/bin/env bash
# Кодогенерация gRPC-контрактов Go-микросервисов (back-go/<svc>/api/proto).
#   Go     → back-go/pkg/gen/<svc>pb (buf + protoc-gen-go{,-grpc}; общий
#            модуль pkg — единственное место Go-стабов, их импортируют и
#            сервис-владелец, и сервисы-клиенты)
#   Python → back/app/grpc (grpcio-tools из venv бэкенда)
# Запускать после любого изменения *.proto; результат коммитится.
# Можно выборочно: scripts/gen_proto.sh calls messenger
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY_OUT="$ROOT/back/app/grpc"

export PATH="$PATH:$HOME/go/bin"

# <go-каталог>:<имя proto-файла без расширения>
ALL_SERVICES=(
  "calls:calls"
  "messenger:messenger"
  "ai:ai"
  "groove:groove"
)

if [ "$#" -gt 0 ]; then
  SERVICES=()
  for want in "$@"; do
    for entry in "${ALL_SERVICES[@]}"; do
      [ "${entry%%:*}" = "$want" ] && SERVICES+=("$entry")
    done
  done
else
  SERVICES=("${ALL_SERVICES[@]}")
fi

mkdir -p "$PY_OUT"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

for entry in "${SERVICES[@]}"; do
  svc="${entry%%:*}"
  proto="${entry##*:}"
  svc_dir="$ROOT/back-go/$svc"
  proto_file="$svc_dir/api/proto/$proto/v1/$proto.proto"

  echo "▶ [$svc] Go-стабы (buf generate)…"
  (cd "$svc_dir" && buf lint && buf generate)

  echo "▶ [$svc] Python-стабы (grpcio-tools)…"
  # Генерируем из плоской копии: иначе grpcio-tools раскладывает стабы по
  # каталогам <proto>/v1/ с абсолютными импортами, которые не работают внутри
  # пакета app.grpc.
  cp "$proto_file" "$TMP/"

  "$ROOT/back/venv/bin/python" -m grpc_tools.protoc \
      -I "$TMP" \
      --python_out="$PY_OUT" \
      --grpc_python_out="$PY_OUT" \
      "$TMP/$proto.proto"

  # Абсолютный импорт → относительный (стабы живут внутри пакета app.grpc).
  # Через python, а не sed -i: его inplace-синтаксис несовместим между BSD и GNU.
  "$ROOT/back/venv/bin/python" - "$PY_OUT/${proto}_pb2_grpc.py" "$proto" <<'EOF'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
name = sys.argv[2]
p.write_text(p.read_text().replace(
    f"\nimport {name}_pb2 as {name}__pb2\n",
    f"\nfrom . import {name}_pb2 as {name}__pb2\n"))
EOF
done

[ -f "$PY_OUT/__init__.py" ] || cat > "$PY_OUT/__init__.py" <<'EOF'
"""Сгенерированные gRPC-стабы Go-микросервисов.

Не редактировать руками — перегенерация: scripts/gen_proto.sh
(контракты — back-go/<svc>/api/proto/<svc>/v1/<svc>.proto).
"""
EOF

echo "✓ Стабы обновлены: back-go/*/gen, back/app/grpc"

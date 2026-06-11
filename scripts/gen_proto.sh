#!/usr/bin/env bash
# Кодогенерация gRPC-контракта звонков (back-go/calls/api/proto).
#   Go     → back-go/calls/gen/callspb (buf + protoc-gen-go{,-grpc})
#   Python → back/app/grpc (grpcio-tools из venv бэкенда)
# Запускать после любого изменения calls.proto; результат коммитится.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROTO_DIR="$ROOT/back-go/calls/api/proto"
PY_OUT="$ROOT/back/app/grpc"

export PATH="$PATH:$HOME/go/bin"

echo "▶ Go-стабы (buf generate)…"
(cd "$ROOT/back-go/calls" && buf lint && buf generate)

echo "▶ Python-стабы (grpcio-tools)…"
mkdir -p "$PY_OUT"
# Генерируем из плоской копии: иначе grpcio-tools раскладывает стабы по
# каталогам calls/v1/ с абсолютными импортами, которые не работают внутри
# пакета app.grpc.
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cp "$PROTO_DIR/calls/v1/calls.proto" "$TMP/"

"$ROOT/back/venv/bin/python" -m grpc_tools.protoc \
    -I "$TMP" \
    --python_out="$PY_OUT" \
    --grpc_python_out="$PY_OUT" \
    "$TMP/calls.proto"

# Абсолютный импорт → относительный (стабы живут внутри пакета app.grpc).
# Через python, а не sed -i: его inplace-синтаксис несовместим между BSD и GNU.
"$ROOT/back/venv/bin/python" - "$PY_OUT/calls_pb2_grpc.py" <<'EOF'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
p.write_text(p.read_text().replace(
    "\nimport calls_pb2 as calls__pb2\n",
    "\nfrom . import calls_pb2 as calls__pb2\n"))
EOF

[ -f "$PY_OUT/__init__.py" ] || cat > "$PY_OUT/__init__.py" <<'EOF'
"""Сгенерированные gRPC-стабы микросервиса звонков.

Не редактировать руками — перегенерация: scripts/gen_proto.sh
(контракт — back-go/calls/api/proto/calls/v1/calls.proto).
"""
EOF

echo "✓ Стабы обновлены: back-go/calls/gen/callspb, back/app/grpc"

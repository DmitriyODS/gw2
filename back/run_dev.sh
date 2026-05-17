#!/usr/bin/env bash
# Запускает Flask с автоперезапуском при краше
set -e
source "$(dirname "$0")/venv/bin/activate"
echo "Flask dev server starting on port ${PORT:-5001}..."
while true; do
  python wsgi.py 2>&1 | tee /tmp/flask.log
  echo "Flask crashed, restarting in 2s..."
  sleep 2
done

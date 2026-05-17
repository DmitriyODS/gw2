#!/usr/bin/env sh
set -e
flask db upgrade
exec python wsgi.py

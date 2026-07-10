#!/usr/bin/env bash
# Сверяет versionCode внутри APK с current_build в соответствующем version.json.
# Ловит главную причину рассинхрона версии на сервере: APK собран до бампа
# version.json или заливается не тот бинарь (см. make deploy-apk — APK едет на
# сервер только через scp, version.json ещё и через git).
# Использование: check_apk_version.sh [путь/к/app.apk] [путь/к/version.json]
# По умолчанию — новый канал apps/mobile/ (Capacitor-обёртка).
# aapt2 не найден → проверка тихо пропускается (не блокирует деплой).
set -euo pipefail
cd "$(dirname "$0")/.."

APK=${1:-apps/mobile/groovework.apk}
VJSON=${2:-apps/mobile/version.json}

[ -f "$APK" ] || { printf '\033[31m✗ Нет %s\033[0m\n' "$APK" >&2; exit 2; }
[ -f "$VJSON" ] || { printf '\033[31m✗ Нет %s\033[0m\n' "$VJSON" >&2; exit 2; }

want=$(grep -oE '"current_build"[[:space:]]*:[[:space:]]*[0-9]+' "$VJSON" | grep -oE '[0-9]+' | head -1)
[ -n "$want" ] || { printf '\033[31m✗ Не нашёл current_build в %s\033[0m\n' "$VJSON" >&2; exit 2; }

# aapt2: из ANDROID_HOME/ANDROID_SDK_ROOT, либо sdk.dir в local.properties, либо дефолт macOS.
sdk="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}"
if [ -z "$sdk" ]; then
  for props in mobile/android/local.properties GrooveWorkAndroid/local.properties; do
    if [ -f "$props" ]; then
      sdk=$(grep -E '^sdk\.dir=' "$props" | head -1 | cut -d= -f2-)
      [ -n "$sdk" ] && break
    fi
  done
fi
[ -n "$sdk" ] || sdk="$HOME/Library/Android/sdk"
aapt=$(ls "$sdk"/build-tools/*/aapt2 2>/dev/null | sort | tail -1 || true)

if [ -z "$aapt" ]; then
  printf '\033[33m⚠ aapt2 не найден (SDK: %s) — пропускаю проверку versionCode APK\033[0m\n' "$sdk" >&2
  exit 0
fi

got=$("$aapt" dump badging "$APK" | sed -n "s/.*versionCode='\([0-9]*\)'.*/\1/p" | head -1)
if [ -z "$got" ]; then
  printf '\033[33m⚠ Не смог прочитать versionCode из APK — пропускаю проверку\033[0m\n' >&2
  exit 0
fi

if [ "$got" != "$want" ]; then
  printf '\033[31m✗ Рассинхрон версии мобилки:\n    %s current_build = %s\n    %s versionCode = %s\n  APK собран не из текущего version.json. Пересобери: обнови version.json → make apk (или make apk-legacy) → make deploy-apk.\033[0m\n' "$VJSON" "$want" "$APK" "$got" >&2
  exit 1
fi

printf '\033[32m✓ versionCode APK (%s) совпадает с %s (%s)\033[0m\n' "$APK" "$VJSON" "$want"

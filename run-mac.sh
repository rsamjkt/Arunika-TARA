#!/bin/bash
# Launcher APM/T.A.R.A untuk macOS production deployment.
#
# Penempatan: copy file ini ke folder yang sama dengan apm-go.app + config.toml.
# Layout final:
#   APM/
#   ├── run-mac.sh           ← file ini (jalankan: ./run-mac.sh)
#   ├── apm-go.app
#   ├── config.toml
#   └── migrations/
#
# Usage:
#   ./run-mac.sh              # jalankan kiosk
#   ./run-mac.sh --encrypt-config   # encrypt credential di config.toml
#
# Script ini menjamin CWD = direktori run-mac.sh ini, sehingga app
# bisa baca config.toml + migrations/ + bikin data/ + logs/
# di lokasi yang benar tanpa peduli dari mana di-launch.

set -e
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# Cari binary — mendukung "apm-go.app" maupun "apm-go 2.app" (Mac auto-rename)
BIN=""
for app in "apm-go.app" "apm-go 2.app" "apm.app"; do
  if [[ -x "$app/Contents/MacOS/apm-go" ]]; then
    BIN="$app/Contents/MacOS/apm-go"
    break
  elif [[ -x "$app/Contents/MacOS/apm" ]]; then
    BIN="$app/Contents/MacOS/apm"
    break
  fi
done

if [[ -z "$BIN" ]]; then
  echo "ERROR: APM binary tidak ketemu di $SCRIPT_DIR" >&2
  echo "Pastikan apm-go.app atau apm.app ada di folder ini." >&2
  exit 1
fi

if [[ ! -f "config.toml" ]]; then
  echo "ERROR: config.toml tidak ada di $SCRIPT_DIR" >&2
  echo "Copy dari config.example.toml lalu isi sesuai environment." >&2
  exit 1
fi

if [[ ! -d "migrations" ]]; then
  echo "ERROR: folder migrations/ tidak ada di $SCRIPT_DIR" >&2
  exit 1
fi

export APM_CONFIG_PATH="./config.toml"
exec "./$BIN" "$@"

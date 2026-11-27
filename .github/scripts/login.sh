#!/usr/bin/env bash
# Inicia sesión con el admin inicial (login -> enrolamiento TOTP -> verify) e
# imprime SMOKE_TOKEN=<jwt de la cookie sai_staff>.
# Uso: bash .github/scripts/login.sh >> "$GITHUB_ENV"
set -euo pipefail
token=$(go run ./tests/tools/stafflogin -base "http://localhost:${PORT}" \
  -user "$INITIAL_ADMIN_USERNAME" -password "$INITIAL_ADMIN_PASSWORD")
if [ -z "$token" ]; then
  echo "::error::No se obtuvo token del admin inicial" >&2
  exit 1
fi
echo "::add-mask::$token" >&2
echo "SMOKE_TOKEN=$token"

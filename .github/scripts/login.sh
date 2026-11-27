#!/usr/bin/env bash
# Hace login con el admin inicial e imprime SMOKE_TOKEN=<jwt>.
# Uso: bash .github/scripts/login.sh >> "$GITHUB_ENV"
set -euo pipefail
body=$(jq -n --arg u "$INITIAL_ADMIN_USERNAME" --arg p "$INITIAL_ADMIN_PASSWORD" '{usuario: $u, password: $p}')
token=$(curl -fsS -X POST "http://localhost:${PORT}/api/auth/login" \
  -H 'Content-Type: application/json' -d "$body" | jq -r '.token // empty')
if [ -z "$token" ]; then
  echo "::error::No se obtuvo token del admin inicial" >&2
  exit 1
fi
echo "::add-mask::$token" >&2
echo "SMOKE_TOKEN=$token"

#!/usr/bin/env bash
# Imprime secretos aleatorios con el mismo formato que .env (base64 estándar).
# Uso: bash .github/scripts/ci-secrets.sh >> "$GITHUB_ENV"
set -euo pipefail
b64() { openssl rand -base64 "$1" | tr -d '\r\n'; }
echo "JWT_SECRET=$(b64 64)"
echo "MFA_ENCRYPTION_KEY=$(b64 32)"
echo "CURP_HMAC_KEY=$(b64 32)"
echo "KEK_MASTER=$(b64 32)"

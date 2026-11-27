#!/usr/bin/env bash
# Crea esquemas/tablas/vistas y el admin inicial (INITIAL_ADMIN_*), en ese orden.
set -euo pipefail
go run ./cmd/migrate
go run ./cmd/seed

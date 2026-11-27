#!/usr/bin/env bash
# Compila ./cmd, lo deja corriendo en segundo plano y espera a que /health responda.
set -euo pipefail
go build -o bin/api ./cmd
nohup ./bin/api > server.log 2>&1 &
for _ in $(seq 1 30); do
  if curl -fs "http://localhost:${PORT}/health" > /dev/null; then exit 0; fi
  sleep 1
done
echo "::error::El servidor no respondió en /health en 30 s"
cat server.log
exit 1

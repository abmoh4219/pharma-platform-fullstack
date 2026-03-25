#!/usr/bin/env sh
set -eu

echo "[backend-entrypoint] Running backend startup checks..."

if [ "${RUN_BACKEND_TESTS_ON_STARTUP:-1}" = "1" ]; then
  echo "[backend-entrypoint] Executing go test ./..."
  if go test ./...; then
    echo "[backend-entrypoint] Backend unit tests passed."
  else
    echo "[backend-entrypoint] Backend unit tests failed; continuing startup."
  fi
fi

echo "[backend-entrypoint] Starting backend server..."
exec go run ./cmd/server

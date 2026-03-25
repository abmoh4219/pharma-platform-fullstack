#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
API_TEST_DIR="$ROOT_DIR/API_tests"
BASE_URL="${BACKEND_URL:-http://localhost:8080}"

PASS_COUNT=0
FAIL_COUNT=0

run_step() {
  local name="$1"
  shift

  echo ""
  echo "=============================="
  echo "RUNNING: ${name}"
  echo "=============================="

  if "$@"; then
    echo "RESULT: PASS - ${name}"
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    echo "RESULT: FAIL - ${name}"
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi
}

wait_for_backend() {
  local health_url="${BASE_URL}/api/v1/health"
  local retries=60

  echo "Waiting for backend health endpoint: ${health_url}"
  for ((i = 1; i <= retries; i += 1)); do
    if curl -fsS "$health_url" >/dev/null 2>&1; then
      echo "Backend is healthy"
      return 0
    fi
    sleep 2
  done

  echo "Backend did not become healthy in time"
  return 1
}

run_step "Backend Go unit tests" bash -c "cd '$BACKEND_DIR' && go test ./..."
run_step "Frontend dependency install" bash -c "cd '$FRONTEND_DIR' && npm ci --no-audit --no-fund"
run_step "Frontend Vitest unit tests" bash -c "cd '$FRONTEND_DIR' && npm run test"
run_step "Wait for backend API" wait_for_backend
run_step "API login test" bash "$API_TEST_DIR/login_test.sh"
run_step "API recruitment search test" bash "$API_TEST_DIR/recruitment_search_test.sh"
run_step "API case creation test" bash "$API_TEST_DIR/case_creation_test.sh"
run_step "API security negative test" bash "$API_TEST_DIR/security_negative_test.sh"

echo ""
echo "=============================="
echo "TEST SUMMARY"
echo "=============================="
echo "Passed: ${PASS_COUNT}"
echo "Failed: ${FAIL_COUNT}"

if [[ $FAIL_COUNT -gt 0 ]]; then
  exit 1
fi

exit 0

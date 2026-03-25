#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/common.sh"

log "Running login test against ${BASE_URL}"

response=$(curl -sS -X POST "${BASE_URL}/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin123!"}')

assert_contains "$response" '"success":true' 'login endpoint should return success'
assert_contains "$response" '"access_token":"' 'response should include access token'

log "PASS: login_test"

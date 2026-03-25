#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/common.sh"

token=$(login_and_get_token)
stamp="$(date +%s)"
subject="AutoCase-${stamp}"

log "Creating case ${subject}"
create_response=$(curl -sS -X POST "${BASE_URL}/api/v1/cases" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${token}" \
  -d "{\"subject\":\"${subject}\",\"description\":\"Automated integration case ${stamp}\"}")

assert_contains "$create_response" '"success":true' 'case creation should succeed'

if ! printf '%s' "$create_response" | grep -Eq '"case_no":"[0-9]{8}-[A-Z0-9]+-[0-9]{6}"'; then
  printf '[API_TEST][FAIL] case_no format mismatch\nResponse: %s\n' "$create_response" >&2
  exit 1
fi

log "PASS: case_creation_test"

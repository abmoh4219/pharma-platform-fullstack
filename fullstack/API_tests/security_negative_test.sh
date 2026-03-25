#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/common.sh"

log "Validating unauthorized route handling"
unauth_response=$(curl -sS -w "\n%{http_code}" "${BASE_URL}/api/v1/dashboard/summary")
unauth_body=$(printf '%s' "$unauth_response" | sed '$d')
unauth_status=$(printf '%s' "$unauth_response" | tail -n 1)
if [[ "$unauth_status" != "401" ]]; then
  printf '[API_TEST][FAIL] expected 401 for unauthorized request, got %s\nBody: %s\n' "$unauth_status" "$unauth_body" >&2
  exit 1
fi
assert_contains "$unauth_body" '"code":"UNAUTHORIZED"' 'unauthorized response should include UNAUTHORIZED code'

log "Validating malformed bearer token handling"
invalid_token_response=$(curl -sS -w "\n%{http_code}" "${BASE_URL}/api/v1/dashboard/summary" \
  -H 'Authorization: Bearer not-a-valid-token')
invalid_token_body=$(printf '%s' "$invalid_token_response" | sed '$d')
invalid_token_status=$(printf '%s' "$invalid_token_response" | tail -n 1)
if [[ "$invalid_token_status" != "401" ]]; then
  printf '[API_TEST][FAIL] expected 401 for malformed token, got %s\nBody: %s\n' "$invalid_token_status" "$invalid_token_body" >&2
  exit 1
fi
assert_contains "$invalid_token_body" '"code":"INVALID_TOKEN"' 'malformed token should return INVALID_TOKEN code'

log "Validating invalid login payload"
invalid_login=$(curl -sS -w "\n%{http_code}" -X POST "${BASE_URL}/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"short"}')
invalid_login_body=$(printf '%s' "$invalid_login" | sed '$d')
invalid_login_status=$(printf '%s' "$invalid_login" | tail -n 1)
if [[ "$invalid_login_status" != "400" ]]; then
  printf '[API_TEST][FAIL] expected 400 for invalid login payload, got %s\nBody: %s\n' "$invalid_login_status" "$invalid_login_body" >&2
  exit 1
fi
assert_contains "$invalid_login_body" '"success":false' 'invalid login should return error payload'

log "Validating unknown JSON field rejection"
unknown_field_login=$(curl -sS -w "\n%{http_code}" -X POST "${BASE_URL}/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin123!","extra":"x"}')
unknown_field_login_body=$(printf '%s' "$unknown_field_login" | sed '$d')
unknown_field_login_status=$(printf '%s' "$unknown_field_login" | tail -n 1)
if [[ "$unknown_field_login_status" != "400" ]]; then
  printf '[API_TEST][FAIL] expected 400 for unknown login field, got %s\nBody: %s\n' "$unknown_field_login_status" "$unknown_field_login_body" >&2
  exit 1
fi
assert_contains "$unknown_field_login_body" '"code":"INVALID_PAYLOAD"' 'unknown JSON fields should be rejected'

log "Validating missing candidate required fields"
token=$(login_and_get_token)
invalid_candidate=$(curl -sS -w "\n%{http_code}" -X POST "${BASE_URL}/api/v1/recruitment/candidates" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${token}" \
  -d '{"full_name":"","phone":"","id_number":""}')
invalid_candidate_body=$(printf '%s' "$invalid_candidate" | sed '$d')
invalid_candidate_status=$(printf '%s' "$invalid_candidate" | tail -n 1)
if [[ "$invalid_candidate_status" != "400" ]]; then
  printf '[API_TEST][FAIL] expected 400 for missing candidate fields, got %s\nBody: %s\n' "$invalid_candidate_status" "$invalid_candidate_body" >&2
  exit 1
fi
assert_contains "$invalid_candidate_body" '"success":false' 'missing candidate fields should return error payload'

log "Validating unknown route error shape"
unknown_route_response=$(curl -sS -w "\n%{http_code}" "${BASE_URL}/api/v1/does-not-exist")
unknown_route_body=$(printf '%s' "$unknown_route_response" | sed '$d')
unknown_route_status=$(printf '%s' "$unknown_route_response" | tail -n 1)
if [[ "$unknown_route_status" != "404" ]]; then
  printf '[API_TEST][FAIL] expected 404 for unknown route, got %s\nBody: %s\n' "$unknown_route_status" "$unknown_route_body" >&2
  exit 1
fi
assert_contains "$unknown_route_body" '"code":"NOT_FOUND"' 'unknown route should return NOT_FOUND code'

log "PASS: security_negative_test"

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

log "Validating RBAC denial: recruitment user hitting compliance endpoint"
recruiter_token=$(login_and_get_token_for "recruiter_b" "Admin123!")
rbac_response=$(curl -sS -w "\n%{http_code}" "${BASE_URL}/api/v1/compliance/restrictions" \
  -H "Authorization: Bearer ${recruiter_token}")
rbac_body=$(printf '%s' "$rbac_response" | sed '$d')
rbac_status=$(printf '%s' "$rbac_response" | tail -n 1)
if [[ "$rbac_status" != "403" ]]; then
  printf '[API_TEST][FAIL] expected 403 for RBAC violation, got %s\nBody: %s\n' "$rbac_status" "$rbac_body" >&2
  exit 1
fi
assert_contains "$rbac_body" '"code":"FORBIDDEN"' 'RBAC violation should return FORBIDDEN code'

log "Preparing object-level and data-scope isolation fixtures"
stamp="$(date +%s)"
team_b_token=$(login_and_get_token_for "recruiter_b" "Admin123!")
team_d_token=$(login_and_get_token_for "recruiter_d" "Admin123!")

cand_b_resp=$(curl -sS -X POST "${BASE_URL}/api/v1/recruitment/candidates" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${team_b_token}" \
  -d "{\"full_name\":\"ScopeB-${stamp}\",\"phone\":\"+1555${stamp}11\",\"id_number\":\"SCOPE-B-${stamp}\",\"email\":\"scopeb${stamp}@example.test\",\"status\":\"new\"}")
assert_contains "$cand_b_resp" '"success":true' 'fixture candidate for team B should be created'
cand_b_id=$(printf '%s' "$cand_b_resp" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')

cand_d_resp=$(curl -sS -X POST "${BASE_URL}/api/v1/recruitment/candidates" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${team_d_token}" \
  -d "{\"full_name\":\"ScopeD-${stamp}\",\"phone\":\"+1666${stamp}22\",\"id_number\":\"SCOPE-D-${stamp}\",\"email\":\"scoped${stamp}@example.test\",\"status\":\"new\"}")
assert_contains "$cand_d_resp" '"success":true' 'fixture candidate for team D should be created'
cand_d_id=$(printf '%s' "$cand_d_resp" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')

if [[ -z "$cand_b_id" || -z "$cand_d_id" ]]; then
  printf '[API_TEST][FAIL] failed to extract fixture candidate IDs\nB: %s\nD: %s\n' "$cand_b_resp" "$cand_d_resp" >&2
  exit 1
fi

log "Validating object-level authorization: team D user cannot update team B candidate"
obj_response=$(curl -sS -w "\n%{http_code}" -X PUT "${BASE_URL}/api/v1/recruitment/candidates/${cand_b_id}" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${team_d_token}" \
  -d "{\"full_name\":\"IllegalUpdate\",\"phone\":\"+1777${stamp}77\",\"id_number\":\"ILLEGAL-${stamp}\",\"email\":\"illegal${stamp}@example.test\",\"status\":\"new\"}")
obj_body=$(printf '%s' "$obj_response" | sed '$d')
obj_status=$(printf '%s' "$obj_response" | tail -n 1)
if [[ "$obj_status" != "403" ]]; then
  printf '[API_TEST][FAIL] expected 403 for object-level unauthorized update, got %s\nBody: %s\n' "$obj_status" "$obj_body" >&2
  exit 1
fi
assert_contains "$obj_body" '"code":"FORBIDDEN"' 'object-level unauthorized update should return FORBIDDEN code'

log "Validating data-scope isolation between TEAM_B and TEAM_D"
list_b=$(curl -sS "${BASE_URL}/api/v1/recruitment/candidates" -H "Authorization: Bearer ${team_b_token}")
list_d=$(curl -sS "${BASE_URL}/api/v1/recruitment/candidates" -H "Authorization: Bearer ${team_d_token}")

assert_contains "$list_b" '"success":true' 'team B candidate list should load'
assert_contains "$list_d" '"success":true' 'team D candidate list should load'

if ! printf '%s' "$list_b" | grep -q "\"id\":${cand_b_id}"; then
  printf '[API_TEST][FAIL] team B should see its own candidate id=%s\nResponse: %s\n' "$cand_b_id" "$list_b" >&2
  exit 1
fi
if printf '%s' "$list_b" | grep -q "\"id\":${cand_d_id}"; then
  printf '[API_TEST][FAIL] team B must not see team D candidate id=%s\nResponse: %s\n' "$cand_d_id" "$list_b" >&2
  exit 1
fi
if ! printf '%s' "$list_d" | grep -q "\"id\":${cand_d_id}"; then
  printf '[API_TEST][FAIL] team D should see its own candidate id=%s\nResponse: %s\n' "$cand_d_id" "$list_d" >&2
  exit 1
fi
if printf '%s' "$list_d" | grep -q "\"id\":${cand_b_id}"; then
  printf '[API_TEST][FAIL] team D must not see team B candidate id=%s\nResponse: %s\n' "$cand_b_id" "$list_d" >&2
  exit 1
fi

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

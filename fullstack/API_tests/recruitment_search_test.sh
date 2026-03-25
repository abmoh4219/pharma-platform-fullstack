#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/common.sh"

token=$(login_and_get_token)
stamp="$(date +%s)"
name="AutoCandidate${stamp}"
phone="+1555${stamp}"
id_number="AUTO-${stamp}"

log "Creating candidate ${name}"
create_response=$(curl -sS -X POST "${BASE_URL}/api/v1/recruitment/candidates" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${token}" \
  -d "{\"full_name\":\"${name}\",\"phone\":\"${phone}\",\"id_number\":\"${id_number}\",\"email\":\"${name}@example.test\",\"status\":\"new\"}")

assert_contains "$create_response" '"success":true' 'candidate creation should succeed before search'

log "Searching candidate ${name}"
search_response=$(curl -sS "${BASE_URL}/api/v1/recruitment/candidates/search?q=${name}" \
  -H "Authorization: Bearer ${token}")

assert_contains "$search_response" '"success":true' 'search endpoint should return success'
assert_contains "$search_response" '"full_name":"' 'search result should include full_name'
assert_contains "$search_response" '"score":' 'search result should include score'

log "PASS: recruitment_search_test"

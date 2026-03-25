#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-${BACKEND_URL:-http://localhost:8080}}"
LOGIN_PATH="/api/v1/auth/login"

log() {
  printf '[API_TEST] %s\n' "$1"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  if ! printf '%s' "$haystack" | grep -q "$needle"; then
    printf '[API_TEST][FAIL] %s\nResponse: %s\n' "$message" "$haystack" >&2
    exit 1
  fi
}

login_and_get_token() {
  local payload='{"username":"admin","password":"Admin123!"}'
  local response

  response=$(curl -sS -X POST "${BASE_URL}${LOGIN_PATH}" \
    -H 'Content-Type: application/json' \
    -d "$payload")

  assert_contains "$response" '"success":true' 'login should succeed'

  local token
  token=$(printf '%s' "$response" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
  if [[ -z "$token" ]]; then
    printf '[API_TEST][FAIL] unable to extract access token\nResponse: %s\n' "$response" >&2
    exit 1
  fi

  printf '%s' "$token"
}

#!/usr/bin/env bash
# Smoke test for docker API (8080) and host API (8081).
# Reads PUBLIC_API_KEYS and ADMIN_API_KEYS from .env if present.
# Uses the first key from a comma-separated list.
# Set ONLY=docker or ONLY=host to limit which suite runs.

set -euo pipefail

DOCKER_API="${DOCKER_API:-http://localhost:8080}"
HOST_API="${HOST_API:-http://localhost:8081}"
ONLY="${ONLY:-}"

if [ -f ".env" ]; then
  # shellcheck disable=SC1091
  . ./.env
fi

pick_first() { echo "${1:-}" | awk -F',' '{print $1}'; }
mask() { local s="${1:-}"; [ -z "$s" ] && { echo "(empty)"; return; }; printf "%s…\n" "$(printf "%s" "$s" | cut -c1-12)"; }

PUB_KEY="$(pick_first "${PUBLIC_API_KEYS:-pub_example_key}")"
ADM_KEY="$(pick_first "${ADMIN_API_KEYS:-adm_example_key}")"

banner() {
  echo "=== Using keys: PUBLIC='$(mask "$PUB_KEY")' ADMIN='$(mask "$ADM_KEY")'"
  echo "=== Docker API: $DOCKER_API"
  echo "=== Host   API: $HOST_API"
  echo
}

# Single-call curl that returns body + a tagged status code on the last line.
curl_one() {
  local method="$1" url="$2" key="$3" data="${4:-}"
  if [ -n "$data" ]; then
    curl -sS -H "X-API-Key: $key" -H "Content-Type: application/json" \
         -X "$method" "$url" --data "$data" \
         -w "\n__STATUS__:%{http_code}"
  else
    curl -sS -H "X-API-Key: $key" -X "$method" "$url" \
         -w "\n__STATUS__:%{http_code}"
  fi
}

print_resp() {
  # Read a response produced by curl_one on stdin, show code + first payload line
  local resp code body
  resp="$(cat)"
  code="$(printf "%s" "$resp" | awk -F'__STATUS__:' 'END{print $NF}')"
  body="$(printf "%s" "$resp" | sed '$d')"
  echo "HTTP $code"
  printf "%s\n" "$body" | head -n1
  printf "\n"
}

check_health() {
  local base="$1" who="$2"
  echo "---- GET $base/healthz"
  local resp code body
  resp="$(curl_one GET "$base/healthz" "$PUB_KEY" || true)"
  code="$(printf "%s" "$resp" | awk -F'__STATUS__:' 'END{print $NF}')"
  body="$(printf "%s" "$resp" | sed '$d')"
  echo "HTTP $code"
  if [ "$code" = "200" ]; then
    [ "$who" = "host" ] && echo "→ Host HTTP 200" || echo "→ HTTP 200"
    return 0
  fi
  echo "→ $who not healthy/authorized (status $code)"
  return 1
}

run_suite() {
  local base="$1" who="$2"
  local uniq="smoke=$(date +%s%N)"
  local good="https://example.com?${uniq}"
  local bad="not-a-url"

  echo
  echo "---- POST $base/api/targets (unique)"
  curl_one POST "$base/api/targets" "$ADM_KEY" "{\"url\":\"$good\"}" | print_resp

  echo "---- POST $base/api/targets (duplicate should 409)"
  curl_one POST "$base/api/targets" "$ADM_KEY" "{\"url\":\"$good\"}" | print_resp

  echo "---- POST $base/api/targets (invalid should 400)"
  curl_one POST "$base/api/targets" "$ADM_KEY" "{\"url\":\"$bad\"}" | print_resp

  echo "---- GET  $base/api/targets"
  curl_one GET "$base/api/targets" "$PUB_KEY" | print_resp

  echo "---- GET  $base/api/results/latest"
  curl_one GET "$base/api/results/latest" "$PUB_KEY" | print_resp
}

main() {
  banner

  if [ -z "$ONLY" ] || [ "$ONLY" = "docker" ]; then
    if check_health "$DOCKER_API" "docker"; then
      run_suite "$DOCKER_API" "docker"
    else
      echo "Docker API not healthy/authorized; skipping docker tests."
    fi
  fi

  if [ -z "$ONLY" ] || [ "$ONLY" = "host" ]; then
    echo
    if check_health "$HOST_API" "host"; then
      run_suite "$HOST_API" "host"
    else
      echo "Host API ($HOST_API) not running or unauthorized; skipping host tests."
    fi
  fi

  echo "=== Smoke done."
}

main "$@"

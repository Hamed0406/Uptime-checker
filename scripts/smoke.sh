#!/usr/bin/env bash
set -euo pipefail

# Load .env for keys if present
if [ -f .env ]; then
  # shellcheck disable=SC1091
  . ./.env
fi

PUB_KEY="${PUBLIC_API_KEYS:-pub_test}"
ADM_KEY="${ADMIN_API_KEYS:-adm_test}"

DOCKER_URL="${DOCKER_URL:-http://localhost:8080}"
HOST_URL="${HOST_URL:-http://localhost:8081}"

uniqsuf="$(date +%s%N)"
URL_OK="https://example.com?smoke=${uniqsuf}"     # unique each run
URL_BAD="not-a-url"

echo "=== Using keys: PUBLIC='${PUB_KEY:0:3}…' ADMIN='${ADM_KEY:0:3}…'"
echo "=== Docker API: $DOCKER_URL"
echo "=== Host   API: $HOST_URL"
echo

curl_do() {
  local method="$1" url="$2" key="$3" data="${4:-}"
  if [ -n "$data" ]; then
    curl -sS -i -X "$method" -H "X-API-Key: $key" -H 'Content-Type: application/json' -d "$data" "$url"
  else
    curl -sS -i -H "X-API-Key: $key" "$url"
  fi
}

echo "---- GET $DOCKER_URL/healthz"
curl_do GET "$DOCKER_URL/healthz" "$PUB_KEY" | sed -n '1p;$p' || true
echo

echo "---- GET $HOST_URL/healthz"
if curl -fsS -H "X-API-Key: $PUB_KEY" "$HOST_URL/healthz" >/dev/null 2>&1; then
  echo "ok"; echo "→ Host HTTP 200"
else
  echo "Host API ($HOST_URL) not running or unauthorized; skipping host tests."
fi
echo

echo "---- POST $DOCKER_URL/api/targets (unique)"
curl_do POST "$DOCKER_URL/api/targets" "$ADM_KEY" "{\"url\":\"$URL_OK\"}" | sed -n '1p;$p' || true
echo

echo "---- POST $DOCKER_URL/api/targets (duplicate should 409)"
curl_do POST "$DOCKER_URL/api/targets" "$ADM_KEY" "{\"url\":\"$URL_OK\"}" | sed -n '1p;$p' || true
echo

echo "---- POST $DOCKER_URL/api/targets (invalid should 400)"
curl_do POST "$DOCKER_URL/api/targets" "$ADM_KEY" "{\"url\":\"$URL_BAD\"}" | sed -n '1p;$p' || true
echo

echo "---- GET  $DOCKER_URL/api/targets"
curl_do GET "$DOCKER_URL/api/targets" "$PUB_KEY" | sed -n '1p;$p' || true
echo

echo "---- GET  $DOCKER_URL/api/results/latest"
curl_do GET "$DOCKER_URL/api/results/latest" "$PUB_KEY" | sed -n '1p;$p' || true
echo

echo "=== Smoke done."

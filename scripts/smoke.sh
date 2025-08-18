#!/usr/bin/env bash
set -euo pipefail

# -----------------------------
# Config / defaults
# -----------------------------
DOCKER_PORT="${DOCKER_PORT:-8080}"   # container publishes 8080 by default
HOST_PORT="${HOST_PORT:-8081}"       # host run (Makefile) defaults to 8081
D="http://localhost:${DOCKER_PORT}"
H="http://localhost:${HOST_PORT}"

# Load local env if present (for API keys etc.)
if [ -f ".env" ]; then
  # shellcheck disable=SC1091
  source .env
fi

# Take first key if comma-separated lists were provided
PUB="${PUBLIC_API_KEYS:-pub_prod_xxx}"
PUB="${PUB%%,*}"
ADM="${ADMIN_API_KEYS:-adm_prod_xxx}"
ADM="${ADM%%,*}"

echo "=== Using keys: PUBLIC='${PUB:0:4}…' ADMIN='${ADM:0:4}…'"
echo "=== Docker API: ${D}"
echo "=== Host   API: ${H}"

# -----------------------------
# Helpers
# -----------------------------
req() {
  # usage: req GET|POST URL [curl args...]
  local method="$1"; shift
  local url="$1"; shift

  local tmp="/tmp/smoke_body_$$"
  echo
  echo "---- ${method} ${url}"
  local code
  code="$(curl -s -o "${tmp}" -w "%{http_code}" -X "${method}" "${url}" "$@")"
  # Print up to 200 chars of response body (for quick glance)
  if [ -s "${tmp}" ]; then
    head -c 200 "${tmp}" | tr -d '\n'
    [ "$(wc -c < "${tmp}")" -gt 200 ] && printf "…"
    echo
  else
    echo "(no body)"
  fi
  echo "→ HTTP ${code}"
  rm -f "${tmp}"
  return 0
}

is_up() {
  # returns 0 if GET url with header returns 200
  local url="$1"; shift
  local code
  code="$(curl -s -o /dev/null -w "%{http_code}" "$@" "${url}")"
  [ "${code}" = "200" ]
}

# -----------------------------
# 1) Health checks
# -----------------------------
req GET "${D}/healthz" -H "X-API-Key: ${PUB}"

if is_up "${H}/healthz" -H "X-API-Key: ${PUB}"; then
  req GET "${H}/healthz" -H "X-API-Key: ${PUB}"
else
  echo "Host API (${H}) not running or unauthorized; skipping host tests."
fi

# -----------------------------
# 2) ADD endpoint (Docker, admin key)
# -----------------------------
# Create
req POST "${D}/api/targets" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${ADM}" \
  --data '{"url":"https://example.com"}'

# Duplicate (should be 409)
req POST "${D}/api/targets" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${ADM}" \
  --data '{"url":"https://EXAMPLE.com/"}'

# Invalid URL (should be 400)
req POST "${D}/api/targets" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${ADM}" \
  --data '{"url":"ftp://bad"}'

# -----------------------------
# 3) Reads (Docker, public key)
# -----------------------------
req GET "${D}/api/targets"        -H "X-API-Key: ${PUB}"
req GET "${D}/api/results/latest" -H "X-API-Key: ${PUB}"

# -----------------------------
# 4) Optional host ADD/reads (if host API is up)
# -----------------------------
if is_up "${H}/healthz" -H "X-API-Key: ${PUB}"; then
  # Host add
  req POST "${H}/api/targets" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: ${ADM}" \
    --data '{"url":"https://hadeli.com"}'

  # Host list & latest
  req GET "${H}/api/targets"        -H "X-API-Key: ${PUB}"
  req GET "${H}/api/results/latest" -H "X-API-Key: ${PUB}"
fi

# -----------------------------
# 5) (Optional) Rate-limit probe on Docker
#     Set RUN_RATE=1 to enable. Uses xargs for concurrency if available.
# -----------------------------
if [ "${RUN_RATE:-0}" = "1" ]; then
  echo
  echo "== Running burst to probe rate limits (Docker /healthz)…"
  if command -v seq >/dev/null 2>&1 && command -v xargs >/dev/null 2>&1; then
    seq 1 250 | xargs -I{} -P 50 curl -s -o /dev/null -w "%{http_code}\n" \
      -H "X-API-Key: ${PUB}" "${D}/healthz" \
    | sort | uniq -c
  else
    echo "seq/xargs not available; skipping burst test."
  fi
fi

echo
echo "=== Smoke done."

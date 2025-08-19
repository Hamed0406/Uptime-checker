#!/usr/bin/env bash
set -euo pipefail

WEB_BASE="${WEB_BASE:-http://localhost:8082}"
API_BASE="${API_BASE:-http://localhost:8080}"

echo "=== Web smoke against $WEB_BASE (API $API_BASE)"

cfg="$(curl -fsS "$WEB_BASE/config.js")" || { echo "config.js not reachable"; exit 1; }

# crude extraction
adm_key="$(echo "$cfg" | sed -n 's/.*ADMIN_API_KEY":[[:space:]]*"\([^"]*\)".*/\1/p')"
pub_key="$(echo "$cfg" | sed -n 's/.*PUBLIC_API_KEY":[[:space:]]*"\([^"]*\)".*/\1/p')"

[ -n "$adm_key" ] || { echo "✖ ADMIN_API_KEY missing in config.js"; exit 1; }
[ -n "$pub_key" ] || { echo "✖ PUBLIC_API_KEY missing in config.js"; exit 1; }

echo "✔ admin key present in config.js"

unique="https://example.com?smoke=$(date +%s%N)"

# positive: admin can POST
code_adm=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE/api/targets" \
  -H 'Content-Type: application/json' -H "X-API-Key: $adm_key" \
  --data "{\"url\":\"$unique\"}")
if [[ "$code_adm" != "200" && "$code_adm" != "409" ]]; then
  echo "✖ admin POST /api/targets expected 200/409, got $code_adm"
  exit 1
fi
echo "✔ admin POST worked ($code_adm)"

# negative: public must be forbidden
code_pub=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE/api/targets" \
  -H 'Content-Type: application/json' -H "X-API-Key: $pub_key" \
  --data "{\"url\":\"$unique\"}")
if [[ "$code_pub" != "403" ]]; then
  echo "✖ public POST /api/targets expected 403, got $code_pub"
  exit 1
fi
echo "✔ public POST correctly forbidden"

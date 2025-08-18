#!/usr/bin/env bash
set -euo pipefail

if [ -f ".env" ]; then
  # shellcheck disable=SC1091
  source .env
fi

PUB="${PUBLIC_API_KEYS:-pub_prod_xxx}"
ADM="${ADMIN_API_KEYS:-adm_prod_xxx}"

echo "== docker (8080) health"
curl -s -i -H "X-API-Key: $PUB" http://localhost:8080/healthz | head -n1

echo "== docker (8080) list targets"
curl -s -i -H "X-API-Key: $PUB" http://localhost:8080/api/targets | head -n1

echo "== host (8081) health"
curl -s -i -H "X-API-Key: $PUB" http://localhost:8081/healthz | head -n1

echo "== add example.com (docker/admin)"
curl -s -i -X POST http://localhost:8080/api/targets \
  -H 'Content-Type: application/json' -H "X-API-Key: $ADM" \
  --data '{"url":"https://example.com"}' | head -n1 || true

echo "== latest (docker/public)"
curl -s -i -H "X-API-Key: $PUB" http://localhost:8080/api/results/latest | head -n1

#!/usr/bin/env bash
set -euo pipefail

# Use .env values used by docker-compose
if [ -f .env ]; then
  # shellcheck disable=SC1091
  . ./.env
fi

USER="${POSTGRES_USER:-uptime}"
DB="${POSTGRES_DB:-uptime}"

echo "Truncating tables in docker 'db' service database '$DB' as user '$USER'..."
docker compose exec -T db bash -lc \
  'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -c "TRUNCATE results, targets RESTART IDENTITY CASCADE;"'
echo "Done."

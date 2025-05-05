#!/bin/sh

set -e

echo "DEBUG: DB_SOURCE received by start.sh is: [$DB_SOURCE]"
echo "Environment:"
env

echo "run db migration"
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

echo "start the app"
exec "$@"
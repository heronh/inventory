#!/usr/bin/env sh
set -eu

if [ ! -f .env ]; then
  echo ".env file not found"
  exit 1
fi

set -a
. ./.env
set +a

docker build \
  -f docker/postgres/Dockerfile \
  --build-arg POSTGRES_USER_ARG="$DB_USER" \
  --build-arg POSTGRES_PASSWORD_ARG="$DB_PASSWORD" \
  --build-arg POSTGRES_DB_ARG="$DB_NAME" \
  -t inventory-postgres:latest \
  .

echo "PostgreSQL image built: inventory-postgres:latest"
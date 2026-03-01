#!/usr/bin/env sh
set -eu

set -a
. ./.env
set +a

docker compose up -d db
go run ./cmd/server
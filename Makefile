.PHONY: postgres-image up down run start

postgres-image:
	sh ./scripts/build-postgres.sh

up:
	docker compose up --build -d

down:
	docker compose down

run:
	go run ./cmd/server

start:
	sh ./scripts/start.sh
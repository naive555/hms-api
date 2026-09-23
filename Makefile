.PHONY: up down run test

## Start Postgres in the background
up:
	docker compose up -d db

## Stop and remove all compose services
down:
	docker compose down

## Run the Go APP (requires `make up` first and a .env file)
run:
	go run ./cmd/server

## Run tests
test:
	go test ./...

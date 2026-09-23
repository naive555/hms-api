.PHONY: up down run test migrate-up migrate-down migrate-new

-include .env
export DB_URL ?= postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@localhost:5432/$(DATABASE_NAME)?sslmode=disable

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

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

## usage: make migrate-new name=add_xxx
migrate-new:
	migrate create -ext sql -dir migrations -seq $(name)

# Same commands are used locally and in CI.
SHELL := /bin/bash
DEV_DB_URL ?= postgres://mes:mes@127.0.0.1:15432/mes?sslmode=disable

.PHONY: help gen dev-db api web test test-backend test-frontend e2e lint build up down

help:
	@grep -E '^[a-z-]+:' Makefile | cut -d: -f1

gen: ## regenerate code from api/openapi.yaml
	cd backend && go generate ./internal/api/
	cd frontend && npm run gen:api

dev-db: ## start a local PostgreSQL on 127.0.0.1:15432
	docker start gz-mes-dev-db 2>/dev/null || docker run -d --name gz-mes-dev-db \
		-e POSTGRES_USER=mes -e POSTGRES_PASSWORD=mes -e POSTGRES_DB=mes \
		-p 127.0.0.1:15432:5432 postgres:16-alpine

api: ## run the backend on :18000 against the dev DB
	cd backend && DATABASE_URL="$(DEV_DB_URL)" JWT_SECRET=dev-secret-dev-secret-dev-secret-0000 \
		ADMIN_PASSWORD=admin123 HTTP_ADDR=127.0.0.1:18000 go run ./cmd/server

web: ## run the frontend dev server on :15173 (proxies /api to :18000)
	cd frontend && npm run dev

test: test-backend test-frontend

test-backend:
	cd backend && TEST_DATABASE_URL="$(DEV_DB_URL)" go test -race -count=1 ./...

test-frontend:
	cd frontend && npm run typecheck && npm run lint && npm test

e2e: ## end-to-end tests against the demo build
	cd frontend && npx playwright test

lint:
	cd backend && go vet ./... && golangci-lint run
	cd frontend && npm run lint

build:
	cd backend && CGO_ENABLED=0 go build -o bin/server ./cmd/server
	cd frontend && npm run build

up: ## build images and start the full stack on :18080
	cd deploy && docker compose up -d --build

down:
	cd deploy && docker compose down

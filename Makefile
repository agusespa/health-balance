.PHONY: dev run web build docker test lint seed seed-reset backfill e2e

# Go API on :8080 plus the Vite dev server on :5173 (proxies /api to Go).
dev:
	@trap 'kill 0' INT TERM EXIT; \
	go run ./cmd/server & \
	npm --prefix web run dev -- --port 5173

# Build the frontend and run the production server on :8080.
run: web
	go run ./cmd/server

# Build the frontend and copy it where the Go binary embeds it from.
web:
	npm --prefix web ci
	npm --prefix web run build
	find internal/ui/dist -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	cp -R web/build/. internal/ui/dist/

build: web
	go build -o bin/health-balance ./cmd/server

seed:
	go run ./cmd/seed

seed-reset:
	go run ./cmd/seed -reset

# Load history from an Apple Health export: make backfill EXPORT=~/Downloads/apple_health_export/export.xml
backfill:
	go run ./cmd/backfill-health -export $(EXPORT)

docker:
	docker-compose -f docker-compose.dev.yml up --build

test:
	go test -v -race -coverprofile=coverage.out ./...

e2e:
	npm ci
	npm --prefix web ci
	npx playwright install chromium
	npm run test:e2e

lint:
	golangci-lint run ./...

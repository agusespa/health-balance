.PHONY: run docker test lint seed seed-reset

run:
	go run ./cmd/server

seed:
	go run ./cmd/seed/main.go

seed-reset:
	go run ./cmd/seed/main.go -reset

docker:
	docker-compose -f docker-compose.dev.yml up --build

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

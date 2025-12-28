.PHONY: run docker test lint

run:
	go run ./cmd/server/main.go

docker:
	docker-compose -f docker-compose.dev.yml up --build

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

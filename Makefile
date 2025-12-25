.PHONY: run test lint

run:
	go run ./cmd/server/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

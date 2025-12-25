# Variables
BINARY_NAME=health-balance

run-local:
	go run ./cmd/server/main.go

build-local:
	go build -o $(BINARY_NAME) ./cmd/server/main.go

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

#!/bin/sh

set -e

COVERAGE_THRESHOLD=70

echo ">> Running security audit..."
govulncheck ./... > /dev/null 2>&1 || { echo "ERROR: Security audit failed"; exit 1; }

echo ">> Running dependency check..."
go mod verify > /dev/null 2>&1 || { echo "ERROR: Module verification failed"; exit 1; }

echo ">> Running go fmt..."
if [ -n "$(gofmt -l .)" ]; then
    echo "ERROR: Code not formatted. Run 'go fmt ./...'"
    exit 1
fi

echo ">> Running linter..."
golangci-lint run > /dev/null 2>&1 || { echo "ERROR: Lint failed"; exit 1; }

echo ">> Running tests..."
go test ./... > /dev/null 2>&1 || { echo "ERROR: Tests failed"; exit 1; }

echo ">> Running tests with race detection..."
go test -race ./... > /dev/null 2>&1 || { echo "ERROR: Race detection tests failed"; exit 1; }

echo ">> Checking test coverage..."
go test -coverprofile=coverage.out ./... > /dev/null 2>&1
if [ -f coverage.out ]; then
  coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
  
  below_threshold=$(awk -v cov="$coverage" -v threshold="$COVERAGE_THRESHOLD" 'BEGIN {print (cov < threshold)}')
  
  if [ "$below_threshold" -eq 1 ]; then
    echo "  ⚠️ Coverage is ${coverage}% (minimum: ${COVERAGE_THRESHOLD}%)"
  fi
  
  rm coverage.out
fi

echo ">> Running type check and build..."
go build -o /dev/null ./... > /dev/null 2>&1 || { echo "ERROR: Build failed"; exit 1; }

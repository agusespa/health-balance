#!/bin/sh

set -e

COVERAGE_THRESHOLD=70

# Helper function to run commands silently but show output on failure
run_check() {
    title=$1
    shift
    echo ">> $title..."
    if ! output=$("$@" 2>&1); then
        echo "❌ ERROR: $title failed"
        echo "----------------------------------------"
        echo "$output"
        echo "----------------------------------------"
        exit 1
    fi
}

run_check "Security audit" govulncheck ./...
run_check "Dependency check" go mod verify

echo ">> Running go fmt..."
FMT_OUT=$(gofmt -l .)
if [ -n "$FMT_OUT" ]; then
    echo "❌ ERROR: Code not formatted. Run 'go fmt ./...'"
    echo "Files needing format: $FMT_OUT"
    exit 1
fi

run_check "Linter" golangci-lint run
run_check "Tests" go test ./...
run_check "Tests with race detection" go test -race ./...

echo ">> Checking test coverage..."
go test -coverprofile=coverage.out ./... > /dev/null 2>&1
if [ -f coverage.out ]; then
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    # Use bc or awk for float comparison
    is_low=$(awk -v cov="$coverage" -v thr="$COVERAGE_THRESHOLD" 'BEGIN {print (cov < thr ? 1 : 0)}')
    
    if [ "$is_low" -eq 1 ]; then
        echo "⚠️ Warning: Coverage is ${coverage}% (minimum: ${COVERAGE_THRESHOLD}%)"
    else
        echo "✅ Coverage is ${coverage}%"
    fi
    rm coverage.out
fi

run_check "Type check and build" go build -o /dev/null ./...

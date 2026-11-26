#!/bin/sh
# Starts the production server for Playwright with a freshly seeded database.
# Expects web/build to exist; `npm run test:e2e` builds it first.
set -e
cd "$(dirname "$0")/.."

if [ ! -f web/build/index.html ]; then
    echo "web/build is missing: run 'npm --prefix web run build' first" >&2
    exit 1
fi
find internal/ui/dist -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
cp -R web/build/. internal/ui/dist/

mkdir -p test-results
rm -f test-results/e2e.db test-results/e2e.db-wal test-results/e2e.db-shm
go build -o test-results/health-balance ./cmd/server
go run ./cmd/seed -db test-results/e2e.db -days 400
DB_PATH=test-results/e2e.db PORT="${PORT:-8181}" exec ./test-results/health-balance

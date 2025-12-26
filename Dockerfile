FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

# Required for SQLite
RUN apt-get update && apt-get install -y gcc libc6-dev

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o health-balance ./cmd/server

# Use CGO_ENABLED=1 but allow Go to handle the cross-architecture build
RUN CGO_ENABLED=1 GOOS=linux go build -o health-balance ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    sqlite3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/health-balance .
COPY --from=builder /app/web ./web

RUN mkdir -p /app/data

ENV DATABASE_URL=/app/data/health.db

EXPOSE 8080

CMD ["./health-balance"]

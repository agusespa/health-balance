FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

ARG TARGETARCH

RUN apt-get update && apt-get install -y \
    gcc-aarch64-linux-gnu \
    gcc-x86-64-linux-gnu \
    libc6-dev-arm64-cross \
    libc6-dev-amd64-cross

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN if [ "$TARGETARCH" = "arm64" ]; then \
      export CC=aarch64-linux-gnu-gcc; \
    else \
      export CC=x86_64-linux-gnu-gcc; \
    fi && \
    CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH go build -o health-balance ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    sqlite3 \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/health-balance .
COPY --from=builder /app/web ./web

RUN mkdir -p /app/data
ENV DATABASE_URL=/app/data/health.db
EXPOSE 8080

CMD ["./health-balance"]

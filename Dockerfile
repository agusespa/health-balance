# Frontend: platform-independent output, so build it once on the build host.
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci
COPY web/ ./
# The release shown in Settings; .git isn't in the build context, so CI passes the tag.
ARG APP_VERSION=dev
RUN APP_VERSION=$APP_VERSION npm run build

# Go binaries (cgo for SQLite), cross-compiled for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder

ARG TARGETARCH

RUN apt-get update && apt-get install -y \
    gcc-aarch64-linux-gnu \
    gcc-x86-64-linux-gnu \
    libc6-dev-arm64-cross \
    libc6-dev-amd64-cross

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY --from=web /src/web/build ./internal/ui/dist

RUN if [ "$TARGETARCH" = "arm64" ]; then \
      export CC=aarch64-linux-gnu-gcc; \
    else \
      export CC=x86_64-linux-gnu-gcc; \
    fi && \
    CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH go build -o health-balance ./cmd/server && \
    CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH go build -o backfill-health ./cmd/backfill-health

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    sqlite3 \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/health-balance /app/backfill-health ./

RUN mkdir -p /app/data
ENV DB_PATH=/app/data/health.db
EXPOSE 8080

CMD ["./health-balance"]

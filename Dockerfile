FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o health-balance ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    sqlite3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/health-banalce .
COPY --from=builder /app/web ./web

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./health-balance"]

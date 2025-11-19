# Multi-stage Dockerfile for all Go services
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all services
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/ingest ./cmd/ingest
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/bars ./cmd/bars
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/indicator ./cmd/indicator
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/scanner ./cmd/scanner
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/alert ./cmd/alert
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/ws-gateway ./cmd/ws_gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/api ./cmd/api

# Runtime stage - create base image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata netcat-openbsd
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/bin/ingest /app/ingest
COPY --from=builder /build/bin/bars /app/bars
COPY --from=builder /build/bin/indicator /app/indicator
COPY --from=builder /build/bin/scanner /app/scanner
COPY --from=builder /build/bin/alert /app/alert
COPY --from=builder /build/bin/ws-gateway /app/ws-gateway
COPY --from=builder /build/bin/api /app/api

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Default command (can be overridden)
CMD ["./ingest"]


# Stage 1: Build binary
FROM golang:alpine AS builder

WORKDIR /app

# Install certificates and git
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build API binary with production optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/api ./cmd/api

# Stage 2: Minimal non-root runtime image
FROM alpine:latest

# Install runtime dependencies for HTTPS and timezone support
RUN apk add --no-cache ca-certificates tzdata

# Create unprivileged application user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy compiled binary and database migrations
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/migrations /app/migrations

# Assign ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

EXPOSE 8080

CMD ["/app/api"]

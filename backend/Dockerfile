# Development stage
FROM golang:1.23-alpine AS development

WORKDIR /app

# Install development tools
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Default command for development
CMD ["go", "run", "./cmd/api"]

# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Build API
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

# Build Worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker

# Production stage
FROM alpine:3.19 AS production

RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# Copy binaries
COPY --from=builder /api /app/api
COPY --from=builder /worker /app/worker

# Copy migrations
COPY --from=builder /app/migrations /app/migrations

# Set ownership
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["/app/api"]

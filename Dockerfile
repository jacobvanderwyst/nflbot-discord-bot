# NFL Discord Bot - Production Dockerfile
# Multi-stage build for optimal image size and security

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o nfl-bot \
    cmd/nfl-bot/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S nflbot && \
    adduser -u 1001 -S nflbot -G nflbot

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/nfl-bot .

# Copy documentation (optional, for reference)
COPY README.md SLASH_COMMANDS.md ./

# Change ownership to non-root user
RUN chown -R nflbot:nflbot /app

# Switch to non-root user
USER nflbot

# Expose health check endpoint (if implemented)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ps aux | grep '[n]fl-bot' || exit 1

# Run the bot
CMD ["./nfl-bot"]

# Metadata
LABEL org.opencontainers.image.title="NFL Discord Bot"
LABEL org.opencontainers.image.description="Advanced NFL Discord Bot with real-time stats and intelligent caching"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/your-username/nfl-discord-bot"

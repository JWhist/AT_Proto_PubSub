# AT Protocol PubSub Server - Dockerfile
#
# This container supports flexible configuration through multiple methods:
#
# 1. Environment Variables:
#    - CONFIG_FILE: Path to configuration file (default: /app/config-docker.yaml)
#    - SERVER_HOST: Host to bind to (default: 0.0.0.0)
#    - SERVER_PORT: Port to listen on (default: 8080)
#
# 2. Configuration Files:
#    - config-docker.yaml: Default Docker-optimized configuration
#    - config-default.yaml: Original localhost configuration
#    - /app/config/: Directory for mounted custom configurations
#
# 3. Docker Run Examples:
#    # Use default configuration
#    docker run -p 8080:8080 at-proto-pubsub
#    
#    # Use custom configuration file
#    docker run -p 8080:8080 -v /host/config.yaml:/app/config/custom.yaml \
#      -e CONFIG_FILE=/app/config/custom.yaml at-proto-pubsub
#    
#    # Override port
#    docker run -p 9000:9000 -e SERVER_PORT=9000 at-proto-pubsub
#
# 4. Docker Compose Example:
#    services:
#      at-proto-pubsub:
#        image: at-proto-pubsub
#        ports:
#          - "8080:8080"
#        volumes:
#          - ./config/production.yaml:/app/config/production.yaml
#        environment:
#          - CONFIG_FILE=/app/config/production.yaml
#          - SERVER_PORT=8080
#

# Build stage
FROM golang:1.24-alpine AS builder

# Install necessary packages for building
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first to leverage Docker cache for dependencies
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary, GOOS=linux for Linux container
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o at-proto-pubsub ./cmd/atprotopubsub

# Final stage - minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS connections to AT Protocol firehose
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -s /bin/sh atproto

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/at-proto-pubsub .

# Copy configuration files
COPY --from=builder /app/config.yaml ./config-default.yaml
COPY --from=builder /app/config-docker.yaml .

# Create config directory for mounted configurations
RUN mkdir -p /app/config

# Change ownership to non-root user
RUN chown -R atproto:atproto /app

# Switch to non-root user
USER atproto

# Environment variables for configuration
ENV CONFIG_FILE="/app/config-docker.yaml"
ENV SERVER_HOST="0.0.0.0"
ENV SERVER_PORT="8080"

# Expose port (can be overridden by SERVER_PORT env var)
EXPOSE 8080

# Health check to ensure the service is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${SERVER_PORT}/api/status || exit 1

# Run the application with configurable config file
CMD ["sh", "-c", "./at-proto-pubsub -config ${CONFIG_FILE}"]
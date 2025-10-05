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

# Change ownership to non-root user
RUN chown atproto:atproto /app/at-proto-pubsub

# Switch to non-root user
USER atproto

# Expose port 8080 for HTTP/WebSocket connections
EXPOSE 8080

# Health check to ensure the service is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/status || exit 1

# Run the application
CMD ["./at-proto-pubsub"]
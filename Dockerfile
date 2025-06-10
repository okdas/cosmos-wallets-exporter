# Build stage
FROM golang:1.22-alpine AS builder

# Install git for version information
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# Use git describe if available, otherwise use a default version
RUN VERSION=$(git describe --tags 2>/dev/null | sed 's/^v//' || echo "dev") && \
    go build -ldflags "-X main.version=${VERSION}" -o cosmos-wallets-exporter cmd/cosmos-wallets-exporter.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appuser && \
    adduser -u 1001 -S appuser -G appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/cosmos-wallets-exporter .

# Copy example config (optional, for reference)
COPY --from=builder /app/config.example.toml .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the default port
EXPOSE 9550

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9550/metrics || exit 1

# Default command
ENTRYPOINT ["./cosmos-wallets-exporter"] 
# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o simulation_server .

# Runtime stage
FROM alpine:latest

# Install ca-certificates and wget for healthcheck
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/simulation_server .

# Copy scenarios directory
COPY --chown=appuser:appgroup scenarios/ ./scenarios/

# Switch to non-root user
USER appuser

# Expose port (default 3000, can be overridden via -port flag)
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-3000}/ || exit 1

# Run the server
# Default port is 3000, but can be overridden via -port flag or PORT env var
CMD ["./simulation_server", "-scenario", "scenarios/example.yaml", "-port", "3000"]

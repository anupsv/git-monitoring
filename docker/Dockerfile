# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install git and other build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o git-monitor ./cmd/git-monitor/main.go

# Final stage
FROM alpine:3.18

# Install ca-certificates for HTTPS and AWS CLI for S3 access
RUN apk add --no-cache ca-certificates aws-cli && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy the executable from the builder stage
COPY --from=builder /build/git-monitor /usr/local/bin/git-monitor

# Copy the entrypoint script
COPY docker/entrypoint.sh /app/entrypoint.sh

# Create an example config file
COPY config.toml.example /app/config.toml.example

# Set executable permissions
RUN chmod +x /usr/local/bin/git-monitor /app/entrypoint.sh

# Set environment variables
ENV CONFIG_PATH=/config/config.toml

# Create config directory and set ownership
RUN mkdir -p /config && \
    chown -R appuser:appgroup /config /app

# Switch to non-root user
USER appuser

# Set the entrypoint script
ENTRYPOINT ["/app/entrypoint.sh"] 
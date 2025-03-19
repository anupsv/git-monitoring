FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/git-monitor ./cmd/git-monitor

# Create a minimal image
FROM alpine:3.17

# Add labels for GitHub Container Registry
LABEL org.opencontainers.image.source=https://github.com/asv/git-monitoring
LABEL org.opencontainers.image.description="Git repository monitoring tool"
LABEL org.opencontainers.image.licenses=MIT

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/bin/git-monitor /usr/local/bin/git-monitor

# Copy the example config to a reference location
COPY config.toml.example /app/config.toml.example

# Create a directory for user configs
RUN mkdir -p /config

# Set working directory
WORKDIR /app

# Set config path environment variable
ENV CONFIG_PATH="/config/config.toml"

# Copy the entrypoint script
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set the volume mount point for user configs
VOLUME ["/config"]

# Run the entrypoint script
ENTRYPOINT ["/entrypoint.sh"] 
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o mtr-tool

# Final stage
FROM alpine:latest

# Install mtr package
RUN apk add --no-cache mtr sudo

# Copy binary from builder
COPY --from=builder /app/mtr-tool /usr/local/bin/

# Create non-root user
RUN adduser -D appuser && \
    echo "appuser ALL=(ALL) NOPASSWD: /usr/sbin/mtr" >> /etc/sudoers

# Switch to non-root user
USER appuser

# Set MTR path for the application
ENV MTR_PATH=/usr/sbin/mtr

ENTRYPOINT ["mtr-tool"]

# Note: Run this container with --network host to get accurate network metrics:
# docker run --network host mtr-tool [args]

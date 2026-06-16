# Multi-stage build for minimal image size
FROM golang:1.22-alpine AS builder

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o /safeupgrade .

# Final stage - minimal runtime image
FROM alpine:3.19

# Install runtime dependencies for multi-language support
RUN apk add --no-cache \
    git \
    nodejs \
    npm \
    python3 \
    py3-pip \
    ca-certificates \
    tzdata \
    bash \
    curl \
    jq

# Create non-root user
RUN addgroup -g 1000 safeupgrade && \
    adduser -D -u 1000 -G safeupgrade safeupgrade

# Copy binary from builder
COPY --from=builder /safeupgrade /usr/local/bin/safeupgrade

# Copy default config
COPY --from=builder /build/configs /etc/safeupgrade/

# Set working directory
WORKDIR /workspace

# Switch to non-root user
USER safeupgrade

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD safeupgrade --version || exit 1

# Default command
ENTRYPOINT ["safeupgrade"]
CMD ["--help"]

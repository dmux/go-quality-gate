# Build stage
FROM golang:1.24-alpine AS builder

# Install git (needed for version info)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_DATE
ARG GIT_COMMIT

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE} -X main.GitCommit=${GIT_COMMIT} -s -w" \
    -o quality-gate ./cmd/quality-gate

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and git for git operations
RUN apk --no-cache add ca-certificates git

# Create a non-root user
RUN addgroup -g 1001 -S quality && \
    adduser -S -D -H -u 1001 -h /app -s /sbin/nologin -G quality -g quality quality

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/quality-gate /usr/local/bin/quality-gate

# Make sure the binary is executable
RUN chmod +x /usr/local/bin/quality-gate

# Change to non-root user
USER quality

# Set the entrypoint
ENTRYPOINT ["quality-gate"]

# Default command
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="Quality Gate"
LABEL org.opencontainers.image.description="A tool for managing code quality gates in development workflows"
LABEL org.opencontainers.image.source="https://github.com/dmux/go-quality-gate"
LABEL org.opencontainers.image.url="https://github.com/dmux/go-quality-gate"
LABEL org.opencontainers.image.documentation="https://github.com/dmux/go-quality-gate/blob/main/README.md"
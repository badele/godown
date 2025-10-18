# Configuration variables
ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.22
ARG DEFAULT_PORT=8080

# Build stage
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Compile application
RUN CGO_ENABLED=0 GOOS=linux go build -o godown .

# Runtime stage
ARG ALPINE_VERSION
FROM alpine:${ALPINE_VERSION}

WORKDIR /docs

# Copy binary from builder
COPY --from=builder /app/godown /usr/local/bin/godown

# Expose default port
ARG DEFAULT_PORT
EXPOSE ${DEFAULT_PORT}

# Set entrypoint
ENTRYPOINT ["godown"]

# Default arguments (can be overridden)
CMD []

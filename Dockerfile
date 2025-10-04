# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY proto/ ./proto/

# Build static binary
RUN CGO_ENABLED=0 go build -o exporter ./cmd/exporter

# Runtime stage - scratch for minimal image
FROM scratch

# Copy binary from builder
COPY --from=builder /build/exporter /exporter

# Expose metrics port
EXPOSE 9999

# Run exporter
ENTRYPOINT ["/exporter"]

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

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o exporter ./cmd/exporter

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/exporter .

# Expose metrics port
EXPOSE 9999

# Run exporter
ENTRYPOINT ["/app/exporter"]

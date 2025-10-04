# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY proto/ ./proto/

# Cross-compile for target platform (no QEMU emulation needed)
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o exporter ./cmd/exporter

# Runtime stage - scratch for minimal image
FROM scratch

# Copy binary from builder
COPY --from=builder /build/exporter /exporter

# Expose metrics port
EXPOSE 9999

# Run exporter
ENTRYPOINT ["/exporter"]

FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build CSI plugin
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o csi-plugin ./cmd/csi-plugin

# Final stage - minimal runtime
FROM alpine:3.19

# Install mount utilities
RUN apk add --no-cache util-linux e2fsprogs xfsprogs blkid

# Copy binary from builder
COPY --from=builder /build/csi-plugin /csi-plugin

# Make binary executable
RUN chmod +x /csi-plugin

# Set entrypoint
ENTRYPOINT ["/csi-plugin"]

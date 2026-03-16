# ------ Build stage ------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies for libvips
RUN apk add --no-cache \
    gcc \
    musl-dev \
    vips-dev \
    pkgconfig

# Copy go.mod and go.sum first to leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
# CGO_ENABLED=1 required for libvips (cgo binding)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/main.go


# ------ Runtime stage ------
FROM alpine:3.19

WORKDIR /app

# Install libvips runtime dependencies
RUN apk add --no-cache \
    vips \
    ca-certificates \
    tzdata

# Copy the binary from builder stage
COPY --from=builder /app/server .

EXPOSE 8081
CMD ["./server"]
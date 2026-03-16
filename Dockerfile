# ---------- Build stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache \
    build-base \
    pkgconfig \
    meson \
    ninja \
    curl \
    glib-dev \
    expat-dev \
    jpeg-dev \
    libpng-dev \
    libwebp-dev \
    tiff-dev \
    libexif-dev \
    lcms2-dev \
    fftw-dev \
    orc-dev

# Build libvips
RUN curl -L https://github.com/libvips/libvips/releases/download/v8.18.0/vips-8.18.0.tar.xz -o vips.tar.xz \
 && tar -xf vips.tar.xz \
 && cd vips-8.18.0 \
 && meson setup build --prefix=/usr --libdir=lib -Dbuildtype=release \
 && ninja -C build \
 && ninja -C build install \
 && cd /app \
 && rm -rf vips*

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build Go server
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/main.go


# ---------- Runtime stage ----------
FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache \
    glib \
    expat \
    jpeg \
    libpng \
    libwebp \
    tiff \
    libexif \
    lcms2 \
    fftw \
    orc \
    ca-certificates \
    tzdata

# Copy libvips
COPY --from=builder /usr/lib/libvips* /usr/lib/
COPY --from=builder /usr/lib/libvips-cpp* /usr/lib/
COPY --from=builder /usr/lib/vips-modules* /usr/lib/

# Copy binary
COPY --from=builder /app/server .

EXPOSE 8081

CMD ["./server"]
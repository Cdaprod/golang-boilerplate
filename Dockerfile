# Stage 1: Build the Go application
FROM golang:1.19-alpine AS builder

# Install necessary packages
RUN apk add --no-cache git gcc musl-dev

# Set work directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code for the cmd/server application
COPY cmd/server/main.go ./cmd/server/
COPY internal/ ./internal/
COPY web_client/ ./web_client/

# Build the Go app
RUN CGO_ENABLED=1 GOOS=linux go build -o multimedia-sys ./cmd/server/main.go

# Stage 2: Build Nginx with RTMP module
FROM alpine:3.16 AS nginx-builder

# Install dependencies for building Nginx and RTMP module
RUN apk add --no-cache --virtual .build-deps \
    build-base \
    openssl-dev \
    pcre-dev \
    zlib-dev \
    git \
    curl \
    tar

# Clone the nginx-rtmp-module repository
RUN git clone https://github.com/arut/nginx-rtmp-module.git /tmp/nginx-rtmp-module

# Download and extract Nginx source
RUN curl -SL http://nginx.org/download/nginx-1.19.3.tar.gz -o /tmp/nginx-1.19.3.tar.gz \
    && tar -zxvf /tmp/nginx-1.19.3.tar.gz -C /tmp \
    && cd /tmp/nginx-1.19.3 \
    && ./configure --with-http_ssl_module --add-module=/tmp/nginx-rtmp-module \
    && make \
    && make install

# Stage 3: Final image
FROM alpine:3.16

# Install necessary packages: FFmpeg, Nginx, and other tools
RUN apk add --no-cache \
    ffmpeg \
    bash \
    libc6-compat \
    ca-certificates \
    openssl \
    curl \
    tini

# Copy the compiled Nginx from nginx-builder
COPY --from=nginx-builder /usr/local/nginx /usr/local/nginx

# Copy the Go application from builder
COPY --from=builder /app/multimedia-sys /usr/local/bin/multimedia-sys

# Copy Nginx configuration
COPY nginx.conf /usr/local/nginx/conf/nginx.conf

# Create necessary directories for HLS and video storage
RUN mkdir -p /tmp/hls /mnt/nas/videos /run/nginx

# Expose RTMP and HTTP ports
EXPOSE 1935 80 8080

# Use Tini as the entrypoint for better signal handling
ENTRYPOINT ["/sbin/tini", "--"]

# Start Nginx and the Go application using a supervisord-like script
COPY scripts/start.sh /start.sh
RUN chmod +x /start.sh

CMD ["/start.sh"]
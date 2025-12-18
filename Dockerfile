FROM golang:1.23-bullseye AS builder

WORKDIR /build

# Install build dependencies including C compiler for tree-sitter
RUN apt-get update && apt-get install -y git gcc libc6-dev && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -o bot-go ./cmd/main.go

FROM debian:bullseye-slim

# Install minimal runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Language servers will be installed at runtime if needed

# Copy application files
COPY --from=builder /build/bot-go .
COPY --from=builder /build/config ./config
COPY --from=builder /build/scripts ./scripts

# Create directory for logs
RUN mkdir -p /app/logs
RUN mkdir -p /app/data


EXPOSE 8181
EXPOSE 8282
EXPOSE 6334

# Increase file descriptor limits for large repositories
RUN echo "* soft nofile 65536" >> /etc/security/limits.conf && \
    echo "* hard nofile 65536" >> /etc/security/limits.conf

CMD ["./bot-go", "-app=config/app.yaml", "-source=config/source.yaml"]
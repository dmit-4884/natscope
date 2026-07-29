# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG LDFLAGS
ARG COMPILER_FLAGS
ARG APP_VERSION
ARG APP_VERSION_COMMIT
ARG APP_ENV_PREFIX
ARG TARGETOS
ARG TARGETARCH
ARG CGO_ENABLE

# Install build dependencies
RUN apk update && \
    apk add --no-cache git make nodejs npm

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Install the proto codegen toolchain: generated Go/TS code is not tracked in git
# and must be regenerated before building (versions match Makefile install-codegen-tools)
RUN go install github.com/bufbuild/buf/cmd/buf@v1.65.0 && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 && \
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.2 && \
    go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.6.1-0.20240319094008-0393e58bdf10

# Copy the rest of the source code
COPY ./ ./

# Root npm deps provide protoc-gen-es / protoc-gen-connect-es for buf generate
RUN npm ci --no-audit --no-fund

# Generate proto code (Go + TS)
RUN make proto-generate

# Build frontend
RUN cd web && npm install && npm run build

# Build backend
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH \
    LDFLAGS=$LDFLAGS \
    COMPILER_FLAGS=$COMPILER_FLAGS \
    APP_VERSION=${APP_VERSION} \
    APP_VERSION_COMMIT=${APP_VERSION_COMMIT} \
    APP_ENV_PREFIX=${APP_ENV_PREFIX} \
    CGO_ENABLE=${CGO_ENABLE} \
    BIN_OUTPUT_DIR=./bin \
    make build-backend

# Final stage
FROM alpine:3.22

ARG APP_NAME=natscope

# Install only necessary runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget && \
    cp /usr/share/zoneinfo/UTC /etc/localtime && \
    echo "UTC" > /etc/timezone && \
    mkdir -p /opt/bin/

# Copy only the binary
COPY --from=builder /app/bin/${APP_NAME} /opt/bin/
RUN chmod +x /opt/bin/${APP_NAME}

RUN printf '#!/bin/sh\nset -e\nexec /opt/bin/%s "$@"\n' "${APP_NAME}" > /entrypoint.sh && \
    chmod +x /entrypoint.sh

# Run as a non-root user; /data holds the bbolt store and file vault and is the
# volume mount point (STORAGE__LOCAL__DATA_DIR pins it independent of $HOME).
RUN addgroup -S natscope && adduser -S -G natscope -h /home/natscope natscope && \
    mkdir -p /data && chown natscope:natscope /data

# File vault + all-interfaces bind. Fail-closed: won't start without
# WEB_AUTH__USERNAME/PASSWORD or ALLOW_INSECURE=true. -p 127.0.0.1:4280:4280 keeps it local.
ENV GRPC_WEB_ADDRESS=0.0.0.0:4280 \
    ALLOW_REMOTE=true \
    SECRETS__BACKEND=file \
    STORAGE__LOCAL__DATA_DIR=/data

USER natscope
VOLUME ["/data"]
EXPOSE 4280

# Liveness: any HTTP response (including 401 when WebAuth is on) means the
# listener is up; a refused connection prints no HTTP status line.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -S -q -O /dev/null -T 3 "http://127.0.0.1:4280/" 2>&1 | grep -q "HTTP/" || exit 1

ENTRYPOINT ["/entrypoint.sh"]
CMD ["server", "run"]

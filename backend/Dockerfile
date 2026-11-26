# syntax=docker/dockerfile:1

FROM golang:1.27.4-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.24
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S -G app app
WORKDIR /app
COPY --from=builder /out/server /app/server
# Migrations are embedded in the binary; copying them is for debugging only.
COPY migrations /app/migrations
# /data is what both compose files mount: a named volume in development, the
# EBS-backed host path in production. Creating it here means a fresh named
# volume inherits this ownership. Without it Docker creates the mountpoint
# root-owned and the API, which runs as `app` (UID 100), cannot open its
# database ("unable to open database file: out of memory (14)").
RUN mkdir -p /app/data /data && chown -R app:app /app /data
USER app

ENV PORT=8080 \
    DATABASE_PATH=/app/data/promisedland.db \
    APP_ENV=development \
    CORS_ORIGIN=http://localhost:5173
EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O - http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/server"]

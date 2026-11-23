# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S -G app app
WORKDIR /app
COPY --from=builder /out/server /app/server
# Migrations are embedded in the binary; copying them is for debugging only.
COPY migrations /app/migrations
RUN mkdir -p /app/data && chown -R app:app /app
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

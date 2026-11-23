# syntax=docker/dockerfile:1

# Optional single-host edge image. Builds the Vue bundle from source so the
# proxy is self-contained; built with context = repository root.
FROM oven/bun:1-alpine AS build
WORKDIR /app
COPY frontend/ ./
RUN bun install --frozen-lockfile && bun run build

FROM nginx:1.27-alpine
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80

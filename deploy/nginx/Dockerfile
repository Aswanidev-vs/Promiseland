# syntax=docker/dockerfile:1

# Optional single-host edge image. Builds the Vue bundle from source so the
# proxy is self-contained; built with context = repository root.
FROM oven/bun:1-alpine AS build
# `bun run build` runs vue-tsc, whose bin is a Node script that patches
# typescript's compiler host to understand `.vue`. Bun's `node` shim cannot
# apply that patch, so the type check fails with TS2307 for every SFC unless a
# real Node runtime is present. Keep it in the builder stage only.
RUN apk add --no-cache nodejs
WORKDIR /app
COPY frontend/ ./
RUN bun install --frozen-lockfile && bun run build

# Unprivileged nginx: workers run as UID 101. Grant only CAP_NET_BIND_SERVICE
# on the binary so this image can still serve on port 80 like a normal web server.
FROM nginxinc/nginx-unprivileged:1.31.7-alpine
USER root
# The upstream nginx image lags the Alpine point release, so pick up the
# current security patches before installing our own packages.
RUN apk upgrade --no-cache \
 && apk add --no-cache libcap-setcap \
 && setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
 && rm -rf /var/cache/apk/*

# The edge config is real and static: it hardcodes `proxy_pass http://api:8080`,
# so it is copied verbatim with no env substitution.
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

# Unprivileged nginx must be able to read both trees.
RUN chown -R 101:101 /etc/nginx/conf.d /usr/share/nginx/html

USER 101
EXPOSE 80

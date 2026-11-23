# syntax=docker/dockerfile:1.7
FROM oven/bun:1-alpine AS builder
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build

# Unprivileged nginx: workers run as UID 101. Grant only CAP_NET_BIND_SERVICE
# on the binary so this image can still serve on port 80 like a normal web server.
FROM nginxinc/nginx-unprivileged:1.27-alpine
USER root
RUN apk add --no-cache libcap-setcap gettext-envsubst \
 && setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
 && rm -rf /var/cache/apk/*

COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=builder /app/dist/ /usr/share/nginx/html/

# Render the config on every start so docker-compose can override the API host.
# The variable whitelist keeps nginx's own $uri/$host/etc. intact.
RUN cat > /usr/local/bin/entrypoint.sh <<'SH'
#!/bin/sh
set -eu
: "${API_UPSTREAM:=http://api:8080}"
export API_UPSTREAM
envsubst '${API_UPSTREAM}' \
  < /etc/nginx/templates/default.conf.template \
  > /etc/nginx/conf.d/default.conf
exec nginx -g 'daemon off;'
SH
RUN chmod +x /usr/local/bin/entrypoint.sh \
 && chown -R 101:101 /etc/nginx/conf.d /usr/share/nginx/html

USER 101
ENV API_UPSTREAM=http://api:8080
EXPOSE 80
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

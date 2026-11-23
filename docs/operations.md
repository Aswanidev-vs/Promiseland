# PromisedLand — Operations

> Status labels: **Verified in R0** means checked during foundation work; **To test before launch** means the procedure is documented but still needs to be exercised on the target environment (PRD §3.4, §17.3).

## Local development

```bash
cp .env.example .env
docker compose up --build          # api :8080, web :5173
docker compose --profile proxy up --build   # edge on :80
docker compose down                # named volume is retained
docker compose down -v             # DESTROYS the database — use only for a clean reset
```

Without containers: `cd backend && go run ./cmd/server`, and `cd frontend && bun install && bun run dev`.

## Health checks

| Check | Command | Notes |
|---|---|---|
| Liveness | `curl -fsS http://localhost:8080/healthz` | Also the Compose healthcheck (Alpine `wget`) |
| Readiness | `curl -fsS http://localhost:8080/readyz` | DB/migrations ready |
| Container status | `docker compose ps` | Shows health state once probes pass |

Startup should fail clearly when configuration or database access is invalid (PRD §12.2). Status: endpoints per conventions — **To test** against the running backend image.

## Logs and request correlation

- `docker compose logs -f api` / `-f web` / `-f proxy`.
- Backend uses structured `slog` logs and an `X-Request-Id` header for correlation. Logs must not contain passwords, session tokens, private message content or sensitive profile fields (PRD §21).
- Production target: ship container logs to CloudWatch (or a documented equivalent) with basic alarms for EC2 status, disk, memory and application health (PRD §11.4). Status: **To test** on EC2.

## Environment variables

Managed through `.env` (never committed) and injected by Compose. See `.env.example` for the full list and safe defaults.

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | API listen port |
| `DATABASE_PATH` | `/data/promisedland.db` | Fixed in Compose to the mounted volume path |
| `APP_ENV` | `development` | `production` sets Secure cookies; prod override enforces it |
| `CORS_ORIGIN` | `http://localhost:5173` | Must be the public HTTPS origin in production |
| `SESSION_SECRET` | `change-me` | Replace before any non-local use; rotate on suspicion |
| `FEATURE_ADULT_DATING` | `false` | Off by default; R5 gate |
| `API_UPSTREAM` | `http://api:8080` | Frontend nginx runtime upstream |
| `VITE_API_BASE_URL` | *(empty)* | Optional frontend build-time value |

`docker compose config -q` must pass with no `.env` present — **Verified in R0**.

## Backup and restore (SQLite on a volume)

SQLite runs in WAL mode, so a backup is **not** a copy of the `.db` file alone: include the `-wal` and `-shm` sidecars, or use SQLite's online backup API, which produces a single consistent file.

**Backup (target procedure; test before launch):**

1. Confirm the API is up and writes are idle: `curl -fsS localhost:8080/readyz`.
2. Run SQLite's online backup against the mounted data path. On the host with `sqlite3` installed and the EBS path mounted at `/srv/promisedland/data`:
   ```bash
   sqlite3 /srv/promisedland/data/promisedland.db \
     ".backup '/srv/promisedland/backups/promisedland-$(date +%F).db'"
   ```
   Locally the same can be done through a one-off container mounting `promisedland-data`.
3. Store the resulting file off-instance (e.g. encrypted S3 with lifecycle rules) and verify the object exists. Do not keep the only backup on the same EBS volume.
4. Record the backup time, source image tag and migration version in the operations log.

**Restore (target procedure; test before launch):**

1. Stop the API: `docker compose -f docker-compose.yml -f deploy/compose/prod.yml stop api`.
2. Move the current database and its sidecars aside (`promisedland.db`, `promisedland.db-wal`, `promisedland.db-shm`) rather than deleting them.
3. Copy the chosen backup into `/srv/promisedland/data/promisedland.db`. Ensure no stale `-wal`/`-shm` files remain next to it; SQLite will recreate them.
4. Start the API and verify `healthz`/`readyz`, then spot-check critical reads.
5. Only after verification, remove the moved-aside files.

Never restore a backup taken before a migration into a newer schema without a tested migration path; pair the database snapshot with the image tag that produced it.

## Deployment and rollback (target)

- Deploy on EC2 with `docker compose -f docker-compose.yml -f deploy/compose/prod.yml up -d --build` from the repository checkout; environment comes from the instance's `.env` / secrets mechanism.
- Keep the previous known-good image tag available. Rollback means redeploying that tag **and** ensuring the database schema is compatible; if the bad deployment applied a migration, restore the matching backup instead of only swapping images (migrations are forward-only).
- Rotate `SESSION_SECRET` by generating a new value, updating the secret store/`.env`, and recreating the API container; expect active sessions to be invalidated.
- After any deploy or rollback, run the health checks and review logs. Status: **To test** — ECR publishing and scripted deployment are not implemented in R0.

## Verification checklist (R0 vs pre-launch)

| Item | Status |
|---|---|
| Compose config validates with no `.env` | Verified in R0 |
| YAML/security-header files reviewed by inspection | Verified in R0 |
| No secret values committed | Verified in R0 |
| Both images build from a clean checkout | To test before launch |
| `api` + `web` start; frontend reaches backend | To test before launch |
| `healthz` / `readyz` return healthy | To test before launch |
| Recreate API container and confirm data persists | To test before launch |
| EC2 reboot recovery, HTTPS, SG exposure | To test before launch |
| Backup + restore to a clean environment | To test before launch |
| Deployment rollback | To test before launch |
| ECR publishing / scripted deploy / CI deploy job | Deferred (PRD §11.3) |

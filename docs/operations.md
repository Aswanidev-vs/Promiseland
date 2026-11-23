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

## Adult area (R5) — keep disabled, enable only after sign-off

> The adult area is a **disabled-by-default scaffold, not launch-ready**. The definitive gate is `docs/adult-safety.md`; every check below is a hard prerequisite, not a formality. Residual risk is HIGH until the `docs/adult-safety.md` launch checklist is complete.

### Keeping the feature disabled in every deployed environment

- Leave **`FEATURE_ADULT_DATING=false`** and **`ADULT_AGE_ASSURANCE=none`** in the environment file / secret store for **development, staging and production** alike. Compose supplies these as safe defaults (`${FEATURE_ADULT_DATING:-false}`, `${ADULT_AGE_ASSURANCE:-none}`), so a missing value also fails closed.
- Do not add a UI link or nav entry for `/adult`. With the flag off, `/adult/*` endpoints are expected to return `404 not_found` and no navigation entry is rendered.
- `ADULT_AGE_ASSURANCE=development_stub` is valid **only** when `APP_ENV=development` AND `FEATURE_ADULT_DATING=true`. Never set the stub in staging or production; production must reject it. Treat any production occurrence as an incident.
- Because the flag and assurance mode are environment-driven, a relaunch, container recreation or new host inherits the values from that environment — audit the `.env` / secret store on every host and after every deployment.

### What to check before any enablement

1. Every item in `docs/adult-safety.md` §5 is signed off by its owner (product validation, real age assurance, legal/privacy review, safety policy, moderation capacity, appeals, and all product controls).
2. A **real age-assurance provider** replaces `none`/`development_stub`; self-entered birthdays are not accepted (PRD §7.14).
3. `FEATURE_ADULT_DATING` is being turned on intentionally, in a reviewed change, with `APP_ENV` and the active age-assurance mode confirmed for that environment.
4. Producer-side moderation and the high-priority adult-report queue are staffed and tested; appeals are operational.
5. The three adult-safety red lines in `docs/adult-safety.md` §6 are verified in the built API and UI.
6. Rollback (below) has been tested. If any check fails, do not enable.

### Environment variable reference

| Variable | Safe default | Meaning |
|---|---|---|
| `FEATURE_ADULT_DATING` | `false` | Master gate for the `/adult/*` namespace and UI. `false` → endpoints `404`, no nav link. |
| `ADULT_AGE_ASSURANCE` | `none` | Eligibility mode. `none` = **deny-all** (no one eligible); `development_stub` = local-only test double, valid only with `APP_ENV=development` AND `FEATURE_ADULT_DATING=true`; any real provider value must be reviewed and approved before use. |

Both are injected into the `api` service from Compose. See `.env.example` for the canonical comments. Stub or `none` values must never reach a production eligibility decision.

### Monitoring and audit expectations for adult safety reports

- Adult safety reports are written to the existing `reports` table with **`target_type='adult_profile'`** and must enter a **high-priority** moderation path (PRD §7.14; conventions R5 contract). They must not be mixed into the ordinary report queue without priority marking.
- Alerting/ops review: monitor for the presence and volume of `target_type='adult_profile'` rows. Any such row is high priority — review, assign and resolve within the moderation SLA, and record actor, target, reason, timestamp and scope in the moderation audit log (PRD §7.15).
- Logs must never contain age-verification secrets, private contact details, exact/live location, or sensitive eligibility data beyond what audit requires (PRD §21, §12.4). Stub `dev/verify` activity must never be emitted in production.
- If the adult area is disabled, expect **zero** adult-profile activity; any `adult_*` request or report row while disabled is suspicious and warrants investigation.

### Incident rollback — disable the flag

1. Set **`FEATURE_ADULT_DATING=false`** in the affected environment's `.env` / secret store (and confirm `ADULT_AGE_ASSURANCE=none` or an approved value — never a stub in production).
2. Recreate the API container / redeploy so the new environment takes effect: `docker compose -f docker-compose.yml -f deploy/compose/prod.yml up -d --build` (or the equivalent redeploy of the previous known-good image tag, respecting database schema compatibility as in the rollback section above).
3. Verify `healthz`/`readyz`, then confirm `/adult/status` reports disabled and `/adult/*` endpoints return `404`.
4. Preserve reports and moderation audit records for review; do not delete evidence during rollback.
5. Page the safety/ops owners, review logs and any `target_type='adult_profile'` rows, and keep the feature disabled until a documented re-enable decision is made.

Disabling the flag is the immediate containment action. It does not substitute for the long-term fixes in `docs/adult-safety.md`; do not re-enable until those are complete.

# PromisedLand

**Your fandom. Your people. Your place.**

PromisedLand is an anime-first fandom social network in development. The product brings anime fans, manga readers, cosplayers, fan artists, event organizers, and local communities into one place: communities and discussions, creator media, events, and (later) real-time chat. Friend discovery and any adult dating experience are separate, opt-in future phases — dating is not part of the initial release.

> **Naming note:** "PromisedLand" is a working brand name. Trademark, company-name, social-handle, and domain checks have not been cleared.

## Tech stack

| Layer | Choice |
|---|---|
| Frontend | Vue 3, TypeScript, Vite, Vue Router, Pinia, plain CSS |
| Frontend runtime / package manager | **Bun** (no npm/node) |
| Backend | Go 1.27, `net/http` + chi, modular monolith |
| Database | SQLite (single writer, WAL), migrations embedded in the binary |
| Containers | Multi-stage Dockerfiles, Docker Compose |
| Target hosting | AWS EC2 + EBS, ECR, S3, IAM, CloudWatch (single instance first) |
| CI | GitHub Actions |

## Quick start

Prerequisites: Docker with Compose v2, Go 1.27, Bun 1.4+.

```bash
# Configuration (safe defaults are provided)
cp .env.example .env

# Run the whole stack: Go API on :8080, web on :5173
docker compose up --build
```

Health endpoints: `http://localhost:8080/healthz` and `/readyz`. The web app is at `http://localhost:5173`.

For a fully containerized stack, use the edge profile and open `http://localhost` — nginx serves the SPA and proxies `/api/` + `/ws/` to the API:

```bash
docker compose --profile proxy up --build
```

> Integration note: the current frontend image serves static assets only and does not yet use `API_UPSTREAM` to proxy `/api` inside the `web` container. Use `bun run dev` (Vite proxy) for frontend development, or the `proxy` profile when running containers. Wiring the frontend nginx to `API_UPSTREAM` is a frontend-side follow-up.

### Local development without containers

```bash
# API
cd backend && go run ./cmd/server

# Web (separate terminal)
cd frontend && bun install && bun run dev
```

The Vite dev server proxies `/api` and `/ws` to `http://localhost:8080`, so the frontend uses relative paths and cookie-based sessions.

## Repository layout

```text
frontend/            Vue 3 + Vite + Bun application (owned by frontend agents)
backend/             Go modular monolith, migrations, Dockerfile (owned by backend agents)
deploy/
  nginx/             optional edge proxy config + image
  compose/           production Compose override
  aws/               EC2/EBS/SG/IAM/ECR/S3/CloudWatch notes
docs/
  architecture.md    modular monolith topology and constraints
  api.md             v1 endpoint contract and error envelope
  threat-model.md    assets, trust boundaries, mitigations
  decisions/         architecture decision records
  operations.md      local dev, health, backups, rollback
  conventions.md     shared engineering contract
  design-system.md   shared visual contract
.github/workflows/   CI checks (no deploy job yet)
docker-compose.yml   local stack (+ optional proxy profile)
.env.example         configuration template
PRD.md               product requirements
```

## Configuration

Copy `.env.example` to `.env`. Key variables:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | API listen port |
| `DATABASE_PATH` | `/data/promisedland.db` | SQLite path inside the API container |
| `APP_ENV` | `development` | `production` enables Secure cookies |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed browser origin |
| `SESSION_SECRET` | `change-me` | Replace for any non-local use |
| `FEATURE_ADULT_DATING` | `false` | Adults-only module gate (off by default) |
| `API_UPSTREAM` | `http://api:8080` | Frontend nginx → API runtime upstream |
| `VITE_API_BASE_URL` | *(empty)* | Optional frontend build-time base path |

Compose reads `.env` automatically. Never commit real secrets.

## Testing

```bash
cd backend && go test ./... && go vet ./...
cd frontend && bun run test && bun run build
docker compose config -q
```

CI runs the same checks on pushes to `main` and on pull requests. Image publishing and automatic deployment are **not** wired up yet.

## Implementation status

**Release 0 — Technical foundation (in progress):** repository structure and conventions, Vue app shell, Go bootstrap and configuration, SQLite connection and migration runner, Dockerfiles, Docker Compose, health endpoints, structured logs, and CI checks.

All `/api/v1` product endpoints are **planned** for later releases (social foundation in R1, creators/events in R2, chat in R3, discovery in R4, optional adults-only dating in R5). See `docs/api.md` for the contract and status per endpoint. The Docker/AWS setup is the documented target topology; production deployment, backup/restore validation, and ECR publishing remain to be exercised before launch.

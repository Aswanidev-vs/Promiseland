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
  adult-safety.md    R5 gate: disabled adult area, prerequisites, launch checklist
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
| `DEV_EXPOSE_TOKENS` | `false` | Return verification and reset tokens in API responses for local testing. Never enabled in production |
| `FEATURE_ADULT_DATING` | `false` | R5 adults-only module gate (off by default) |
| `ADULT_AGE_ASSURANCE` | `none` | R5 eligibility mode: `none` denies everyone; `development_stub` is dev-only, never production. See `docs/adult-safety.md` |
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

| Release | Status |
|---|---|
| R0 — Technical foundation | **Implemented:** repo structure/conventions, Vue shell, Go bootstrap/config, SQLite + migration runner, Dockerfiles, Compose, health endpoints, structured logs, CI checks |
| R1 — Social foundation | **Implemented:** auth, profiles/interests, communities, posts/comments/voting, feed, search, notifications, reporting/moderation, responsive UI |
| R2 — Creator & events | **Implemented:** creator categories, events + RSVP, team-ups, related endpoints/views |
| R3 — Community communication | **Implemented:** chat servers/channels, conversations, messages, WebSocket hub, presence/typing, chat moderation |
| R4 — Interest discovery | **Implemented:** friend discovery, connections, user-controlled discoverability |
| R5 — Optional adult dating | **Gated scaffold present but disabled and incomplete — NOT launch-ready.** Kept off by `FEATURE_ADULT_DATING=false` and `ADULT_AGE_ASSURANCE=none` (deny-all). Eligibility/age assurance, legal review, safety policy, moderation/appeals and the `/adult/*` implementation are outstanding. See `docs/adult-safety.md` before any enablement. |

R0–R4 modules, migrations and endpoints are present in the codebase; see `docs/api.md` for the endpoint contract and status. The Docker/AWS setup is the documented target topology; production deployment, backup/restore validation and ECR publishing remain to be exercised before launch. Do not treat the R5 scaffold as a completed or shippable feature.

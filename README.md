<p align="center">
  <img src="assets/brand/logo.png" alt="PromisedLand logo" width="96" height="96" />
</p>

<h1 align="center">PromisedLand</h1>

<p align="center"><em>Your fandom. Your people. Your place.</em></p>

<p align="center">
  An anime-first social network for spoiler-aware discussions, creative work and local meetups,
  built around one interest list you set up once.
</p>

> **Naming note:** "PromisedLand" is a working brand name. Trademark, company-name, social-handle
> and domain checks have not been cleared.

---

## Contents

- [What this is](#what-this-is)
- [Status](#status)
- [What you can use today](#what-you-can-use-today)
- [Routes](#routes)
- [Tech stack](#tech-stack)
- [Architecture](#architecture)
- [Repository layout](#repository-layout)
- [Quick start](#quick-start)
- [Health and status](#health-and-status)
- [Configuration](#configuration)
- [Security posture](#security-posture)
- [Testing](#testing)
- [Deployment](#deployment)
- [Documentation](#documentation)
- [Conventions](#conventions)
- [License and naming](#license-and-naming)

## What this is

PromisedLand brings the pieces of anime fandom that usually live in five different places into one
product:

- **Communities and discussions** — series-tagged threads, polls, and spoiler covers that respect
  where you are in a story.
- **A creator feed** — cosplay, fan art and photography, crediting both the title and the person who
  made it.
- **Real-world events** — screenings, convention days and meetups with RSVPs, planned next to the
  community that runs them.
- **Chat** — servers, channels and direct messages over WebSockets (Release 3).

Everything is organised by the interests you pick, so discovery is based on what you follow rather
than on engagement bait. The product is in closed-beta development; see [Status](#status) before
reading any of it as finished.

## Status

| Release | Scope | State |
|---|---|---|
| R0 — Technical foundation | Repo conventions, Vue shell, Go bootstrap and config, SQLite with an embedded migration runner, Dockerfiles, Compose, health endpoints, structured logs, CI checks | **Implemented** |
| R1 — Social foundation | Auth, profiles and interests, communities, posts/comments/voting, feed, search, notifications, reporting and moderation, responsive UI | **Implemented** |
| R2 — Creators and events | Creator categories, events with RSVP, team-ups, related endpoints and views | **Implemented** |
| R3 — Community communication | Chat servers, channels, conversations, messages, WebSocket hub, presence and typing, chat moderation | **Implemented** |
| R4 — Interest discovery | Friend discovery, connections, user-controlled discoverability | **Implemented** |
| R5 — Optional adult dating | Gated scaffold only | **Disabled and not launch-ready.** Off by `FEATURE_ADULT_DATING=false` and `ADULT_AGE_ASSURANCE=none` (deny-all). Eligibility, legal review and safety policy are outstanding — read `docs/adult-safety.md` before touching it |

Production deployment, backup/restore validation and image publishing have **not** been exercised
yet. CI runs checks only; it has no deploy job.

## What you can use today

| Surface | Signed out (guest) | Signed in |
|---|---|---|
| Landing page, legal pages, service status | Read | Read |
| Events list and event pages | Read | Read, RSVP, create |
| Creator directory | Read | Read, apply to team-ups |
| Home feed, communities, posts, profiles, search | Redirected to sign in | Full read/write |
| Chat, notifications, settings, discovery | Redirected to sign in | Full |
| Uploading images | No | JPEG, PNG, GIF or WebP up to 5 MB, 30 per hour |

Guest browsing is deliberate (PRD §6, "visitor"): the **Continue as a guest** action on the landing,
sign-in, register and footer pages opens the public pages with a banner naming what an account adds.
Every write action asks for an account — RSVP is gated with a toast and a redirect back after
sign-in, and the navigation shows only public destinations.

## Routes

| Route | Page | Access |
|---|---|---|
| `/` | Landing: promise, live interest map, experiences, spoiler demo, event pass | Public |
| `/login`, `/register`, `/reset-password`, `/verify-email` | Auth (bare shell, consent gate on sign-up) | Guests |
| `/events`, `/events/:id` | Events list and detail | Public |
| `/creators` | Creator directory | Public |
| `/status` | Service status, reads `/healthz` and `/readyz` | Public |
| `/terms`, `/privacy` | Legal pages (working drafts) | Public |
| `/error/:code` | 403, 404, 429, 500, 503; unknown codes fall back to 404 | Public |
| `/:pathMatch(.*)*` | 404 | Public |
| `/home` | Feed | Account |
| `/communities`, `/communities/:slug` | Communities | Account |
| `/posts/new`, `/posts/:id` | Composer and discussion | Account |
| `/u/:username` | Profile | Account |
| `/search`, `/discover`, `/connections` | Search, discovery, connections | Account |
| `/chat`, `/chat/servers/:id`, `/chat/conversations/:id` | Chat | Account |
| `/notifications`, `/settings`, `/team-ups`, `/moderation` | Account surfaces | Account |
| `/admin` | Staff console: counts, accounts, posts, communities, events | Platform staff |
| `/adult` | Gated R5 scaffold (unlinked; 404s unless enabled) | Account + gates |

## Tech stack

| Layer | Choice |
|---|---|
| Frontend | Vue 3.5, TypeScript, Vite 6, Vue Router 4, Pinia, plain CSS with design tokens |
| Frontend runtime and package manager | **Bun** (1.4.x) — no npm or node |
| Fonts | Self-hosted via `@fontsource-variable`: Plus Jakarta Sans (display and body), JetBrains Mono (figures, coordinates) |
| Backend | Go 1.27, `net/http` + chi 5, modular monolith |
| Realtime | `gorilla/websocket` hub with per-scope subscriptions and presence |
| Database | SQLite through the pure-Go `modernc.org/sqlite` driver; WAL, one writer, migrations embedded in the binary |
| Auth | Cookie sessions (HttpOnly, SameSite=Lax), bcrypt passwords, SHA-256-hashed session and email tokens |
| Containers | Multi-stage Dockerfiles (non-root), Docker Compose, optional edge nginx profile |
| Target hosting | AWS EC2 + EBS, ECR, S3, IAM, CloudWatch (single instance first) |
| CI | GitHub Actions: gofmt, `go test`, `go vet`, `go build`, bun install/build/test, `docker compose config` |

## Architecture

A single Go API process organised as domain modules that never import each other — they register
their routes through `shared.Deps` and exchange data through `shared` DTOs:

`adult · anime · api · auth · chat · communities · config · creators · db · discovery · events ·
feed · media · moderation · notifications · posts · realtime · search · shared · teamups · users`

Public routes are mounted first; a `RequireAuth` sub-router layers authenticated routes on top, and
handlers re-check ownership and membership at the row they act on. SQLite is the single writer
(ADR 0001, ADR 0002). The frontend is a Vue SPA with a bare shell for public pages (landing, auth,
legal, status, errors) and the app shell for everything behind sign-in.

- `docs/architecture.md` — topology, module boundaries, constraints
- `docs/threat-model.md` — assets, trust boundaries, mitigations
- `docs/decisions/` — architecture decision records (modular monolith, single-writer SQLite,
  landing page in Vue)

## Repository layout

```text
assets/
  brand/               source logo and auth illustration (published copies live in
                       frontend/public and docs; see assets/README.md)
backend/               Go modular monolith, migrations, Dockerfile, Makefile
  cmd/server/          API entrypoint
  internal/            domain modules (see Architecture)
  migrations/          embedded SQL, applied in filename order at startup
deploy/
  nginx/               optional single-host edge image and config
  compose/             production Compose override
  aws/                 EC2/EBS/SG/IAM/ECR/S3/CloudWatch notes
docs/                  architecture, API contract, threat model, conventions,
                       design system, operations, adult-safety, ADRs, static site
frontend/              Vue 3 + Vite + Bun application
  public/              served files (logo, auth illustration, theme bootstrap)
  src/                 views, components, stores, api clients, styles (design tokens)
.github/workflows/     CI checks (no deploy job)
docker-compose.yml     local stack, optional proxy profile
.env.example           configuration template
PRD.md                 product requirements
README.md
```

## Quick start

Prerequisites: Docker with Compose v2, Go 1.27, Bun 1.4+.

```bash
# Configuration (safe defaults are provided)
cp .env.example .env

# Whole stack: Go API on :8080 (loopback only), web on :5173
docker compose up --build
```

For a fully containerised stack with the edge proxy, open `http://localhost`:

```bash
docker compose --profile proxy up --build
```

### Local development without containers

```bash
# Terminal 1 — API
cd backend && go run ./cmd/server

# Terminal 2 — web
cd frontend && bun install && bun run dev
```

The Vite dev server proxies `/api`, `/ws`, `/media`, `/healthz` and `/readyz` to
`http://localhost:8080`, so the app uses relative paths and cookie sessions. If the API is not
running, the proxy answers with the API error envelope (503 `api_unavailable`) and logs one
actionable line instead of a stack trace per request.

The frontend uses Bun for everything: `bun install`, `bun run dev`, `bun run build`, `bun run test`.
`bun test` is Bun's own runner and is **not** compatible with this suite — use `bun run test`
(vitest).

## Health and status

| Endpoint | Purpose | Success |
|---|---|---|
| `GET /healthz` | Liveness: the process is up | `200 {"status":"ok"}` |
| `GET /readyz` | Readiness: the database answers a ping | `200 {"status":"ready"}`; `503 database_unavailable` otherwise |
| `GET /status` (web) | Human status page reading both endpoints | Operational, partial, or unreachable |

The status page reports only what the browser can observe: per-endpoint state, latency and HTTP
status, a 15-second refresh that pauses while the tab is hidden, and the checks made since the page
opened. It makes no uptime-history claims.

## Configuration

Copy `.env.example` to `.env`. Key variables:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | API listen port |
| `DATABASE_PATH` | `/data/promisedland.db` (`./data/...` for `go run`) | SQLite file path |
| `APP_ENV` | `development` | `production` enables Secure cookies and refuses dev-only paths |
| `CORS_ORIGIN` | `http://localhost:5173` | The one browser origin allowed with credentials (a wildcard never gets credentials) |
| `SESSION_SECRET` | `change-me` | Replace for any non-local use; the API refuses production with the dev default |
| `MEDIA_PATH` | `./data/media` | Uploaded image directory (never inside SQLite) |
| `PLATFORM_MODERATOR_EMAIL` | *(empty)* | Optional local-testing moderator promotion; leave blank in production |
| `FEATURE_ADULT_DATING` | `false` | R5 gate; keep false |
| `ADULT_AGE_ASSURANCE` | `none` | `none` denies everyone; `development_stub` is a test double, never production |
| `DEV_EXPOSE_TOKENS` | `false` | Return verification and reset tokens in API responses for local testing; forced off in production |
| `API_UPSTREAM` | `http://api:8080` | nginx (web and edge) upstream for `/api`, `/ws`, `/media`, `/healthz`, `/readyz` |
| `VITE_API_BASE_URL` | *(empty)* | Optional build-time base path |

Compose reads `.env` automatically. Never commit real secrets.

## Admin console

A staff console lives at `/api/v1/admin/*` behind an authenticated session plus a
platform staff grant, and at `/admin` in the web app (a "Staff → Admin" entry appears in the
navigation only for granted accounts).

- **Overview** — account, session, content and storage counts, plus the newest accounts, posts and
  open reports, with a link into the moderation queue.
- **Users** — searchable account list with post and session counts, staff badges, and suspend /
  reinstate. Suspending also ends that account's sessions immediately.
- **Posts, communities, events** — searchable, paginated tables with links into the app.

Grant a staff account by setting `PLATFORM_MODERATOR_EMAIL` to an existing account's email and
restarting the API; it inserts the grant at startup. Everyone else receives `403` from every
console route, and `/api/v1/admin/me` tells the client whether to show the entry.

## Security posture

What the code does today, in plain terms:

- **Sessions** are opaque 256-bit tokens in an HttpOnly, SameSite=Lax cookie (Secure in production);
  the database stores only their SHA-256 digest, and a session stops counting when the account is
  deactivated or suspended.
- **Passwords** are bcrypt hashes at the default cost; unknown-email logins do comparable work so the
  response time does not reveal registered addresses.
- **Email and reset tokens** are random, hashed at rest, single-use, and time-limited. They are never
  returned in API responses unless `DEV_EXPOSE_TOKENS` is explicitly enabled, which is forced off in
  production.
- **Rate limits** apply to registration, login, password reset and verification, keyed on the real
  client (forwarded headers are trusted only when the immediate peer is our proxy).
- **CORS** allows exactly one configured origin with credentials; a wildcard origin grants none.
- **Uploads** are checked by extension, declared type, sniffed bytes, decoder, dimensions and size,
  stored under random keys, served with a database-derived type and `nosniff`, and capped per account.
- **Stored URLs** (post links, community images, chat icons) must be absolute http(s).
- **Edge and web containers** send CSP, `nosniff`, frame, referrer and permissions headers, and cap
  request bodies; the API port is bound to loopback in the local stack.
- **Moderation** actions, ownership checks and per-scope WebSocket subscriptions are enforced in
  handlers, not in the client.

This is a beta posture, not a claim of formal assurance. The threat model and known gaps live in
`docs/threat-model.md`; report problems privately to the maintainer rather than in a public issue.

## Testing

```bash
cd backend  && gofmt -l . && go vet ./... && go test ./... && go build ./...
cd frontend && bun run test && bun run build
docker compose config -q
```

Backend tests exercise handlers through `httptest` against a temporary SQLite database, covering
auth, moderation, chat, events, media and the router middleware. Frontend tests cover stores, API
clients, view components and the new public surfaces (landing interest map, auth consent gate, error
pages, status page, guest gating). CI runs these checks on every push to `main` and on pull requests.

## Deployment

The documented target is a single EC2 instance behind nginx, with EBS for the database and media,
ECR for images, S3 for backups and CloudWatch for logs — see `deploy/aws/` and
`docs/operations.md`. Production Compose (`deploy/compose/prod.yml`) requires a real
`SESSION_SECRET`. Nothing is deployed automatically yet, and backup/restore has not been validated.

## Documentation

| File | What it covers |
|---|---|
| `PRD.md` | Product requirements, roles, release scope |
| `docs/architecture.md` | Modular monolith topology and constraints |
| `docs/api.md` | v1 endpoint contract, error envelope, pagination |
| `docs/threat-model.md` | Assets, trust boundaries, mitigations |
| `docs/adult-safety.md` | R5 gate, prerequisites, launch checklist |
| `docs/conventions.md` | Shared engineering contract |
| `docs/design-system.md` | Tokens, type, components, screen notes |
| `docs/operations.md` | Local dev, health, backups, rollback |
| `docs/decisions/` | ADRs: modular monolith, single-writer SQLite, landing page in Vue |
| `assets/README.md` | Where brand art lives and how copies stay in sync |

## Conventions

- **Bun, not npm** for frontend work (`bun install`, `bun run …`, `bun run test`).
- **Design system first:** colours, type, spacing, motion and component recipes are defined in
  `docs/design-system.md` and implemented as tokens in `frontend/src/styles/tokens.css`. No hardcoded
  hex outside the token file.
- **Docs before code** when direction changes; decisions get an ADR in `docs/decisions/`.
- **Guest rule:** public pages stay readable without an account; every write action asks for one.
- **No fabricated proof** in copy or seed data: sample content is labelled as sample.

## License and naming

No licence file has been added yet, so the default applies: all rights reserved until one is chosen.
"PromisedLand" is a working brand name, and trademark, company-name, social-handle and domain checks
are still outstanding.

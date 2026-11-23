# PromisedLand — Technical Architecture

> Derived from PRD §10, §11 and the shared `docs/conventions.md`. This describes the target design for Release 0 and beyond; module internals are owned by the backend agents.

## Approach

A **modular monolith**: one Go API process, with domain modules separated by package boundaries and shared only through `internal/shared` DTOs and dependencies. This is simpler to build, test, deploy and operate than early microservices, and it matches the single-writer SQLite constraint (PRD §10.1, §10.5).

## Component diagram

```mermaid
flowchart TB
    Browser["Browser (Vue 3 SPA)"]

    subgraph Host["Single EC2 host / local Docker host"]
        subgraph Edge["Optional edge (proxy profile)"]
            Proxy["nginx: static SPA + /api/ + /ws/"]
        end
        subgraph App["Docker Compose network: promisedland"]
            Web["web container: nginx serves built SPA"]
            API["api container: Go modular monolith"]
        end
        Volume[("promisedland-data / EBS-backed path\n/data/promisedland.db)]
    end

    S3[("Amazon S3\nuser-uploaded media")]

    Browser -->|HTTPS / HTTP in dev| Edge
    Browser -->|dev: :5173 direct| Web
    Edge -->|/api/ , /ws/| API
    Edge -->|/| Web
    Web -->|/api via API_UPSTREAM| API
    API -->|SQL, single writer| Volume
    API -->|presigned URLs / object keys| S3
```

In local development the edge is omitted: `web` publishes `5173:80`. `API_UPSTREAM=http://api:8080` is supplied to the web container for its runtime nginx to proxy `/api`; the current frontend image only serves static assets, so containerized API access should use the `proxy` profile (or Vite's dev proxy via `bun run dev`). In single-host production the optional `proxy` service publishes port 80 and serves the bundled SPA as well as proxying API and WebSocket traffic.

## Request flow

1. Browser loads the SPA from `web` (dev) or `proxy` (single-host). Client-side routing uses `createWebHistory`; the server falls back to `index.html` for unknown paths.
2. The typed API client prefixes `/api/v1`, sends `credentials: 'include'`, and receives JSON. Auth uses the HTTP-only `pl_session` cookie (`SameSite=Lax`, `Secure` in production) — see `docs/conventions.md`.
3. `/api/` requests reach the Go API. chi routes to the owning module; middleware adds a request ID, recovers panics, enforces auth, and applies CORS/rate limits.
4. Handlers validate input, parse pagination (`?limit=&cursor=`), and call repository code. Ownership/role checks happen on the server for every protected resource.
5. Responses are either the resource directly or the standard error envelope (see `docs/api.md`). Logs are structured and carry the request ID.

## Module boundaries

Backend packages live under `backend/internal/`:

| Module | Owns |
|---|---|
| `api` | Router assembly, middleware, shared HTTP helpers |
| `auth` | Registration, sessions, verification, password reset |
| `users` | Accounts, profiles, interests, follows, blocks |
| `anime` | Title/genre metadata and tagging |
| `communities` | Communities, memberships, roles, rules |
| `posts` / `comments` / `reactions` | Feed, posts, comments, votes/reactions, bookmarks |
| `notifications` | In-app notification records and preferences |
| `moderation` | Reports, moderation actions, audit log |
| `platform` | Configuration and administrative operations |
| `events` | Release 2 events and RSVPs |
| `chat` | Release 3 servers, channels, messages, WebSocket hub |
| `discovery` | Release 4 friend discovery |
| `matching` | Future, adults-only gated module |
| `db` / `config` / `shared` | Infrastructure, configuration, shared DTOs and helpers |

Rules (from conventions): domain packages do not import one another; they communicate through `shared` DTOs. Each package registers its routes via `func Register(r chi.Router, d *shared.Deps)`. All routes are under `/api/v1`, except `/healthz` and `/readyz` at the root.

Frontend boundaries: `src/api` (typed client + domain functions), `src/components/ui`, `src/components/layout`, `src/views` (one route per file), `src/stores` (Pinia), `src/router`, `src/styles`, `src/types`.

## Database constraints

SQLite is used for local development and the first single-instance deployment (PRD §10.5):

- **One writer.** `db.SetMaxOpenConns(1)` and a DSN enabling `foreign_keys(1)`, `journal_mode(WAL)`, `busy_timeout(5000)`.
- **No shared file across writers.** Never mount the same SQLite file into multiple independently writing application instances.
- **Persistent storage.** On EC2 the database lives on an EBS-backed host path; containers can be recreated without data loss. A named Docker volume is used locally.
- **WAL mode** is enabled with an explicit understanding of its `-wal` / `-shm` sidecar files and backup procedure (see `docs/operations.md`).
- **Migrations** are embedded SQL files applied in filename order at startup inside a transaction; schema is added as features land, not all future tables upfront.
- **Monitor** database size, write contention, backup duration and query latency. Plan a PostgreSQL migration before horizontal scaling or high write concurrency.

## Media plan (S3)

User-uploaded images go to Amazon S3, never into SQLite (PRD §10.6):

- SQLite stores media metadata and random object keys; binary files stay in object storage.
- Validate extension, MIME type, file signature, size and image dimensions on the server; reject mismatches.
- Generate random object keys — untrusted filenames must never become storage paths.
- Use presigned upload URLs or a controlled backend upload flow; block public bucket access by default.
- Define retention/deletion for removed content and deleted accounts; add thumbnailing when feeds need it.
- Video uploads are deferred until storage, transcoding, moderation and cost are understood.

## Real-time (later)

Chat arrives in Release 3 (PRD §10.7): an authenticated Go WebSocket endpoint using the same session model, server-side channel authorization, persistent message history, cursor-based recovery, per-user/connection rate limits, and a single-instance hub. Redis/pub-sub or similar is only considered if multiple backend instances become necessary.

## Deployment topology

- **Local:** `api` + `web` containers, named volume, bridge network; `proxy` available behind a Compose profile.
- **Single EC2 (target):** one host runs Docker Compose with all three containers. Only the edge publishes port 80; TLS terminates at nginx/Caddy with a certificate workflow (or at a load balancer if introduced). EBS backs SQLite; S3 backs media; CloudWatch collects logs/metrics; IAM and security groups are least-privilege. See `deploy/aws/README.md`.
- **Scaling boundary:** vertical scaling first. Horizontal API scaling requires migrating off single-file SQLite first (ADR `docs/decisions/0002-sqlite-single-writer.md`).

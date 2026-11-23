# PromisedLand — Engineering Conventions (shared contract)

Every agent MUST follow this file. It exists so parallel work integrates cleanly.

## Repository layout

```text
promisedland/
├── frontend/            # Vue 3 + TS + Vite + Pinia (owned by frontend agents)
│   └── src/{api,components,router,stores,styles,types,views}
├── backend/             # Go modular monolith (owned by backend agents)
│   ├── cmd/server/
│   ├── internal/{api,auth,users,anime,communities,posts,notifications,moderation,events,chat,discovery,matching,platform,db,config,shared}
│   ├── migrations/      # NNN_name.sql, applied in filename order
│   └── Dockerfile
├── deploy/{nginx,compose,scripts,aws}
├── docs/
├── .github/workflows/
├── docker-compose.yml
├── .env.example
└── .gitignore
```

Go module path: `github.com/Aswanidev-vs/promiseland/backend` (in `backend/go.mod`). Import internal packages as `github.com/Aswanidev-vs/promiseland/backend/internal/<pkg>`.

## Backend rules

- Go 1.27. `net/http` + `github.com/go-chi/chi/v5`. SQLite via `modernc.org/sqlite` (pure Go, no cgo). Migrations via `embed` of `migrations/*.sql`.
- Config from env with sane defaults: `PORT` (8080), `DATABASE_PATH` (`./data/promisedland.db`), `APP_ENV`, `CORS_ORIGIN`, `SESSION_SECRET`, `FEATURE_ADULT_DATING=false`.
- Open SQLite DSN with `_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)`. Set `db.SetMaxOpenConns(1)` (single-writer constraint). Run migrations at startup in a transaction; `CREATE TABLE IF NOT EXISTS` style so reruns are safe.
- Router: `internal/api/router.go` builds the chi router and mounts each module. Each domain package exposes `func Register(r chi.Router, d *shared.Deps)`.
- `shared.Deps` carries `*sql.DB`, `*slog.Logger`, `Config`. Domain packages must not import each other; share via `shared` DTOs only.
- All API routes under `/api/v1`. Health: `GET /healthz`, `GET /readyz` at root (no /api prefix).
- JSON success = resource directly. Errors always: `{"error":{"code":"slug_code","message":"human readable","details":{}}}` with proper HTTP status. Use `shared.WriteJSON`, `shared.WriteError`.
- Auth: opaque session token in HTTP-only cookie `pl_session` (`SameSite=Lax`, `Secure` when `APP_ENV=production`, 7-day TTL). Store `sessions(token_hash, user_id, expires_at)`. Passwords: `golang.org/x/crypto/bcrypt`. Never log passwords or tokens.
- Every protected handler calls `shared.UserID(r)` (auth middleware puts id in context). Ownership checks are mandatory.
- Pagination: `?limit=20&cursor=...`; response `{"items":[...],"next_cursor":"..."}`.
- Structured logs with `slog`. Request ID middleware adds `X-Request-Id`.
- Tests: table-driven unit tests for validators/logic and httptest integration tests for each module's routes. Every module ships `*_test.go`.

### Migration numbering (avoid collisions)

- Release 0: `000` (foundation: users, profiles, sessions, reset/verification tokens)
- Release 1: `001`–`010`
- Release 2: `011`–`015`
- Release 3: `016`–`020`
- Release 4: `021`–`023`
- Release 5: `024`–`025`

## Frontend rules

- Package manager and runtime: **Bun 1.4+** (confirmed installed). Use `bun install`, `bun run dev`, `bun run build`, `bun test`. Never use npm/node/npx for this project. Commit `bun.lock` (not package-lock.json).
- Frontend container builds use `oven/bun:1-alpine` (build stage) with `bun install --frozen-lockfile` and `bun run build`; serve static output with nginx/Caddy in the final stage. No Node images.
- Scripts: `dev`, `build` (`vue-tsc --noEmit && vite build`), `test` (`vitest run`), `preview`.
- Vue 3 `<script setup lang="ts">` only. Pinia setup stores. Vue Router 4 with `createWebHistory` and `meta.requiresAuth` guards; augment `RouteMeta` in `src/types/router.d.ts`.
- Typed API client in `src/api/client.ts`: `request<T>(path, init)` sends `credentials: 'include'`, prefixes `/api/v1`, throws `ApiError` with `code`/`status`/`message`. Domain modules `src/api/<domain>.ts` export typed functions. All types in `src/types/index.ts`.
- Vite dev proxy: `'/api': { target: 'http://localhost:8080', changeOrigin: true }` and `'/ws': { target: 'ws://localhost:8080', ws: true }`.
- Styling: plain CSS (no utility framework) with tokens in `src/styles/tokens.css`, base in `src/styles/base.css`, component styles scoped in SFCs. Follow `docs/design-system.md` exactly.
- Components: shared UI in `src/components/ui/` (`BaseButton`, `BaseInput`, `BaseCard`, `EmptyState`, `SkeletonList`, `Avatar`, `TagChip`, `SpoilerShield`, `Toast`). Layout in `src/components/layout/` (`AppShell`, `AppSidebar`, `MobileNav`, `TopBar`).
- Views are route-level components in `src/views/`. One route per file. Names end in `View.vue`.
- Every data view implements loading (skeleton), empty (with action), error (retry), success, and permission-denied states.
- Use `@fontsource/fraunces`, `@fontsource/inter`, `@fontsource/ibm-plex-mono` imports in `main.ts`.
- Accessibility & design quality floor per `docs/design-system.md`.

## API surface (v1) — endpoint names are a contract

Auth: `POST /auth/register|login|logout|verify|password-reset/request|password-reset/confirm`, `GET /auth/me`.
Users: `GET/PATCH /users/me`, `GET /users/:id`, `GET /users/:id/posts`, `GET/PUT /users/me/interests`, `POST/DELETE /users/:id/follow`.
Anime: `GET /anime?q=`, `GET /anime/:id`, `GET /genres`.
Communities: `GET /communities`, `POST /communities`, `GET/PATCH /communities/:slug`, `POST/DELETE /communities/:slug/membership`, `GET /communities/:slug/posts`, `GET/POST /communities/:slug/rules`, `POST /communities/:slug/moderators`.
Posts: `GET /posts?scope=for-you|following&community=&q=`, `POST /posts`, `GET/PATCH/DELETE /posts/:id`, `GET/POST /posts/:id/comments`, `POST/DELETE /posts/:id/reactions`, `POST /posts/:id/bookmark`.
Search: `GET /search?q=&type=users|communities|posts`.
Notifications: `GET /notifications`, `PATCH /notifications/:id/read`.
Reports/Moderation: `POST /reports`, `POST /users/:id/block`, `GET /moderation/reports`, `POST /moderation/reports/:id/resolve`.
Events (R2): `GET/POST /events`, `GET/PATCH/DELETE /events/:id`, `POST/DELETE /events/:id/rsvp`.
Team-ups (R2): `GET/POST /team-ups`, `POST /team-ups/:id/apply`.
Chat (R3): `GET/POST /chat/servers`, `GET/POST /chat/servers/:id/channels`, `GET/POST /chat/channels/:id/messages`, `GET/POST /chat/conversations`, `WS /ws`.
Discovery (R4): `GET /discovery/friends`, `POST /connections`, `PATCH /connections/:id`.
Adult (R5): gated by `FEATURE_ADULT_DATING`; off by default; explicit age gate and separate namespace `/adult/*`.

## Git

One commit per release phase. Do not commit until the orchestrator (main agent) verifies and stages.

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

## Release 1 cross-agent contract (R1)

### Backend JSON shapes (exact keys — every R1 backend agent produces/consumes these)

- `UserSummary`: `{id:int64, username:string, display_name:string, avatar_url:string|null}`
- `User`: `{id, email (self only), username, display_name, avatar_url, banner_url, bio, location, visibility, discoverable, created_at}`
- `UserProfile` extends `User` with `interests: Interest[]`, `anime_interests: AnimeInterest[]`, `follower_count`, `following_count`, `post_count`, `is_following:boolean`, `is_blocked:boolean`
- `Interest`: `{id, name, category}`; `Anime`: `{id, external_id, title, alternate_title, format}`; `AnimeInterest`: `{anime:Anime, interest_type:string, sort_order:int}`
- `Community`: `{id, name, slug, description, icon_url, banner_url, visibility:'public'|'private', member_count, post_count, role:'owner'|'moderator'|'member'|null, is_member:boolean, created_at}`
- `CommunityRule`: `{id, community_id, title, body, sort_order}`
- `Post`: `{id, author:UserSummary, community:Community|null, type:'text'|'link'|'image'|'question'|'poll'|'announcement', title, body, link_url, media:Media[], anime:Anime[], tags:string[], spoiler:boolean, spoiler_note, season, episode, vote_score:int, my_vote:-1|0|1, comment_count:int, bookmarked:boolean, created_at, updated_at, edited_at}`
- `Media`: `{id, url, mime_type, size_bytes, width, height}`
- `Post.poll` (optional): `{question:string, options:[{id:int64,text:string,votes:int}], total_votes:int, my_vote:int64|null}`
- `Comment`: `{id, post_id, author:UserSummary, parent_id:int64|null, body, created_at, edited_at, deleted:boolean}`
- `Notification`: `{id, type, actor:UserSummary|null, target_type, target_id, read_at, created_at, data:object}`
- `Report`: `{id, reporter_id, target_type, target_id, reason, details, status, created_at}`
- `Page<T>`: `{items:T[], next_cursor:string}` (empty string when no more pages)

Voting model: one consistent upvote/downvote per user per post (`my_vote` -1/0/1, `vote_score` aggregated). Do not expose a separate reaction count.

### R1 frontend API module exports (F1 writes, F2 imports — signatures are a contract)

`src/api/auth.ts`: `register(i:{email,password,username,displayName}): Promise<User>`, `login(i:{email,password}): Promise<User>`, `logout(): Promise<void>`, `getMe(): Promise<User>`, `verifyEmail(token:string): Promise<void>`, `requestPasswordReset(email:string): Promise<void>`, `confirmPasswordReset(token:string,password:string): Promise<void>`.

`src/api/users.ts`: `getUser(username:string): Promise<UserProfile>`, `updateMe(patch:Partial<User>): Promise<User>`, `listInterests(): Promise<Interest[]>`, `updateInterests(ids:number[]): Promise<UserProfile>`, `updateAnimeInterests(items:{animeId:number,interestType:string}[]): Promise<UserProfile>`, `followUser(id:number): Promise<void>`, `unfollowUser(id:number): Promise<void>`, `blockUser(id:number): Promise<void>`, `unblockUser(id:number): Promise<void>`, `updateSettings(patch:{visibility?:string,discoverable?:boolean}): Promise<User>`.

`src/api/anime.ts`: `searchAnime(q:string): Promise<Anime[]>`, `getAnime(id:number): Promise<Anime>`, `listGenres(): Promise<Genre[]>`.

`src/api/communities.ts`: `listCommunities(p:{q?,limit?,cursor?}): Promise<Page<Community>>`, `getCommunity(slug:string): Promise<Community>`, `createCommunity(i:{name,slug,description,visibility,rules?}): Promise<Community>`, `updateCommunity(slug:string,patch:Partial<Community>): Promise<Community>`, `joinCommunity(slug:string): Promise<void>`, `leaveCommunity(slug:string): Promise<void>`, `listRules(slug:string): Promise<CommunityRule[]>`, `listMembers(slug:string,p?): Promise<Page<UserSummary>>`, `listCommunityPosts(slug:string,p?): Promise<Page<Post>>`, `addModerator(slug:string,userId:number): Promise<void>`.

`src/api/posts.ts`: `getFeed(p:{scope?:'for-you'|'following',community?,limit?,cursor?}): Promise<Page<Post>>`, `getPost(id:number): Promise<Post>`, `createPost(i:{communitySlug?,type,title,body,linkUrl?,mediaIds?,animeIds?,tags?,spoiler?,spoilerNote?,season?,episode?}): Promise<Post>`, `updatePost(id:number,patch:Partial<Post>): Promise<Post>`, `deletePost(id:number): Promise<void>`, `listComments(id:number,p?): Promise<Page<Comment>>`, `addComment(id:number,body:string,parentId?:number): Promise<Comment>`, `votePost(id:number,vote:-1|0|1): Promise<Post>`, `bookmarkPost(id:number): Promise<void>`, `unbookmarkPost(id:number): Promise<void>`, `uploadMedia(file:File): Promise<Media>`.

`src/api/search.ts`: `search(p:{q:string,type?:'users'|'communities'|'posts',limit?,cursor?}): Promise<{users:UserSummary[],communities:Community[],posts:Post[]}>`.

`src/api/notifications.ts`: `listNotifications(p?): Promise<Page<Notification>>`, `markRead(id:number): Promise<void>`, `markAllRead(): Promise<void>`.

`src/api/moderation.ts`: `report(i:{targetType,targetId,reason,details?}): Promise<void>`, `listReports(p?): Promise<Page<Report>>`, `resolveReport(id:number,i:{action:string,reason:string}): Promise<void>`.

### R1 routes (F1 writes into router/index.ts; F2 creates the matching views)

`/home` HomeView · `/communities` CommunitiesView · `/communities/:slug` CommunityView · `/posts/new` CreatePostView · `/posts/:id` PostView · `/u/:username` ProfileView · `/search` SearchView · `/notifications` NotificationsView · `/settings` SettingsView · `/onboarding` OnboardingView · `/verify-email` VerifyEmailView · `/reset-password` ResetPasswordView · `/moderation` ModerationView. All authenticated except `/verify-email`, `/reset-password`, `/login`, `/register`.

## Git

One commit per release phase. Do not commit until the orchestrator (main agent) verifies and stages.

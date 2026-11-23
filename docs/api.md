# PromisedLand — API Contract (v1)

> Source of truth for endpoint **names**: `docs/conventions.md` (the shared agent contract). PRD §14 contains an earlier illustrative list with different names (e.g. `/verify-email`, `/me`); where they differ, the conventions list wins. Payload fields below are derived from PRD §15 and are illustrative until the OpenAPI specification lands — do not treat them as final schemas.

## General

- **Base path:** all endpoints below are relative to `/api/v1`. Health endpoints are at the root.
- **Transport:** JSON request/response bodies. Authentication uses the HTTP-only `pl_session` cookie sent with `credentials: 'include'`; protected routes are enforced server-side.
- **Request IDs:** middleware emits `X-Request-Id`, echoed in errors for correlation.
- **Pagination:** `?limit=20&cursor=...`; collections return `{"items":[...],"next_cursor":"..."}` with stable ordering.
- **Validation:** field-level errors go in `error.details`. Internal database errors and stack traces are never exposed.
- **Rate limits:** consistent `429` responses on auth, posting, comments, invites, uploads and messaging (PRD §12.3).

### Error envelope (all non-2xx responses)

```json
{
  "error": {
    "code": "slug_code",
    "message": "Human-readable explanation.",
    "details": {}
  }
}
```

Typical statuses: `400` validation, `401` unauthenticated, `403` forbidden / ownership failure, `404` not found, `409` conflict (e.g. duplicate username), `429` rate limited, `500` unexpected. Codes are lowercase snake_case; the set grows with implementation.

### Status legend

| Label | Meaning |
|---|---|
| **R0 — foundation** | Expected in the technical-foundation release (health only) |
| **Implemented (R1)** | Social foundation / first beta |
| **Implemented (R2)** | Creator and events |
| **Implemented (R3)** | Community communication (chat) |
| **Planned (R4)** | Interest-based discovery |
| **Planned (R5)** | Optional, adults-only, feature-gated |

R1–R3 endpoints are implemented; R4–R5 remain planned. See `docs/conventions.md` for the authoritative request/response contracts.

## Health (R0 — foundation)

| Method | Path | Purpose | Status |
|---|---|---|---|
| GET | `/healthz` | Liveness probe used by Compose | R0 — foundation |
| GET | `/readyz` | Readiness probe (DB/migrations ready) | R0 — foundation |

Response: `{"status":"ok"}` (illustrative). These are root paths, not under `/api/v1`.

## Authentication and account (Implemented R1)

| Method | Path | Purpose |
|---|---|---|
| POST | `/auth/register` | Create account with email + password |
| POST | `/auth/login` | Start a session |
| POST | `/auth/logout` | End the session |
| POST | `/auth/verify` | Verify email token |
| POST | `/auth/password-reset/request` | Issue time-limited reset token |
| POST | `/auth/password-reset/confirm` | Complete reset with token |
| GET | `/auth/me` | Current authenticated account |

Illustrative shapes:

- `POST /auth/register` request: `{"email":"...","password":"...","username":"...","displayName":"..."}` → `201` with the user resource plus a development-only verification token. Passwords are hashed (bcrypt) and never returned or logged.
- `POST /auth/login` request: `{"email":"...","password":"..."}` → `200` with the user resource plus `Set-Cookie: pl_session=...; HttpOnly; SameSite=Lax; Secure (prod)`. Invalid credentials must not reveal whether the email is registered (PRD §7.1).
- `POST /auth/verify` request: `{"token":"..."}` → success or `400` with an expired/invalid code.
- `POST /auth/password-reset/request` request: `{"email":"..."}` → generic success regardless of account existence.
- `POST /auth/password-reset/confirm` request: `{"token":"...","password":"..."}`; tokens expire and are single-use.

## Users, profiles and interests (Implemented R1)

| Method | Path | Purpose |
|---|---|---|
| GET | `/users/me` | Read own account/profile |
| PATCH | `/users/me` | Update profile, visibility, preferences |
| GET | `/users/:id` | Read a public profile |
| GET | `/users/:id/posts` | Paginated posts by a user |
| GET | `/users/me/interests` | List anime/interest tags |
| PUT | `/users/me/interests` | Replace ordered interests |
| POST | `/users/:id/follow` | Follow a public profile |
| DELETE | `/users/:id/follow` | Unfollow |
| POST | `/users/:id/block` | Block an account |

No `DELETE /users/:id/block` is defined in the shared contract.

Illustrative shapes:

- Profile resource: `{ "id", "username", "display_name", "bio", "avatar_key", "banner_key", "visibility", "discoverable", "favourite_anime": [], "interests": [], "followers_count", "following_count" }`. Private email/security fields are never included in public responses (PRD §12.4).
- `PATCH /users/me` accepts only public/profile fields and privacy settings; ownership is derived from the session, never from the path.
- `PUT /users/me/interests` body: `{"anime_title_ids":[],"interest_tag_ids":[]}` with `sort_order` preserved.
- Follow/block responses are `204` or a small status object; blocked users cannot initiate new interactions with the blocker (PRD §7.2).

## Anime metadata (Implemented R1 basic)

| Method | Path | Purpose |
|---|---|---|
| GET | `/anime?q=` | Search titles (seeded catalogue / external provider) |
| GET | `/anime/:id` | Title detail |
| GET | `/genres` | Genre list |

Illustrative shapes:

- Anime title: `{ "id", "external_id", "title", "alternate_title", "media_type" }` with optional season/episode references where reliable metadata exists.
- External provider failure must not break core discussions; unknown episode references must not create misleading metadata (PRD §7.3).

## Communities (Implemented R1)

| Method | Path | Purpose |
|---|---|---|
| GET | `/communities` | Paginated community discovery |
| POST | `/communities` | Create a community |
| GET | `/communities/:slug` | Community detail, rules, member count |
| PATCH | `/communities/:slug` | Update settings (owner only) |
| POST | `/communities/:slug/membership` | Join / request to join |
| DELETE | `/communities/:slug/membership` | Leave |
| GET | `/communities/:slug/posts` | Paginated community posts |
| GET | `/communities/:slug/rules` | List rules |
| POST | `/communities/:slug/rules` | Add rule (moderator) |
| POST | `/communities/:slug/moderators` | Assign moderator (owner) |

Illustrative shapes:

- Community: `{ "id", "owner_id", "name", "slug", "description", "visibility", "rules": [], "member_count" }`. `visibility` is `public` or `private`; private communities require invitation/approval.
- Membership: `{ "community_id", "user_id", "role", "status" }`. A client cannot grant itself a role by editing a request (PRD §7.4).

## Posts and comments (Implemented R1)

| Method | Path | Purpose |
|---|---|---|
| GET | `/posts?scope=for-you\|following&community=&q=` | Paginated feed |
| POST | `/posts` | Create post |
| GET | `/posts/:id` | Post detail |
| PATCH | `/posts/:id` | Edit own post |
| DELETE | `/posts/:id` | Delete own post (soft-delete policy) |
| GET | `/posts/:id/comments` | Paginated comment thread |
| POST | `/posts/:id/comments` | Add comment/reply (bounded nesting) |
| POST | `/posts/:id/reactions` | Add reaction/vote |
| DELETE | `/posts/:id/reactions` | Remove reaction/vote |
| POST | `/posts/:id/bookmark` | Bookmark post |

Illustrative shapes:

- Post: `{ "id", "author_id", "community_id"?, "type", "title", "body", "tags": [], "anime_refs": [], "spoiler": bool, "created_at", "updated_at" }`. `spoiler` metadata is returned by the API so the frontend does not decide spoiler status alone (PRD §7.6).
- Feed response: `{"items":[Post],"next_cursor":"..."}` with bounded page size; never load a whole large feed.
- Reactions: one consistent model chosen for v1; duplicate votes from one account are rejected where only one is allowed.
- Bookmarking and removals return `204` or a status object; deleted content follows the documented soft-delete policy.

## Search (Implemented R1)

| Method | Path | Purpose |
|---|---|---|
| GET | `/search?q=&type=users\|communities\|posts` | Cross-entity search |

Illustrative shape: `{"items":[...],"next_cursor":"..."}`; results respect privacy and community permissions and use query limits (PRD §7.16).

## Notifications (Implemented R1)

| Method | Path | Purpose |
|---|---|---|
| GET | `/notifications` | Paginated in-app notifications |
| PATCH | `/notifications/:id/read` | Mark one as read |

Illustrative shape: `{ "id", "user_id", "type", "actor_id", "target_type", "target_id", "read_at" }`. Types include follower, reply, mention, reaction, community result and event update (PRD §7.11). Email/push extensions are later.

## Reports and moderation (Implemented R1 basic)

| Method | Path | Purpose |
|---|---|---|
| POST | `/reports` | Report content/account |
| POST | `/users/:id/block` | Block a user |
| GET | `/moderation/reports` | Moderator/admin queue |
| POST | `/moderation/reports/:id/resolve` | Resolve with action + reason |

Illustrative shapes:

- Report: `{ "id", "reporter_id", "target_type", "target_id", "reason", "details", "status" }`. Reasons: spam, harassment, impersonation, hate/abuse, inappropriate content, copyright, spoiler abuse, unsafe event, other (PRD §7.15).
- Moderation action: `{ "id", "moderator_id", "target_type", "target_id", "action", "reason", "created_at" }`. Every action is logged with actor, target, reason, timestamp and scope.

## Events (Implemented R2)

| Method | Path | Purpose |
|---|---|---|
| GET | `/events` | Discover events (date/type/tag/location filters) |
| POST | `/events` | Create event |
| GET | `/events/:id` | Event detail |
| PATCH | `/events/:id` | Edit event (organizer) |
| DELETE | `/events/:id` | Remove event (organizer) |
| POST | `/events/:id/rsvp` | RSVP |
| DELETE | `/events/:id/rsvp` | Withdraw RSVP |

Illustrative shape: `{ "id", "organizer_id", "title", "description", "event_type", "starts_at", "ends_at", "timezone", "venue_text", "capacity"?, "attendee_count" }`. Times are stored with an explicit timezone strategy; exact addresses and live location are never exposed by default (PRD §7.9). Capacity is enforced transactionally.

## Team-ups (Implemented R2)

| Method | Path | Purpose |
|---|---|---|
| GET | `/team-ups` | List collaboration requests |
| POST | `/team-ups` | Create request |
| POST | `/team-ups/:id/apply` | Request to join |

Illustrative shape: `{ "id", "creator_id", "type", "description", "event_or_anime_ref", "broad_location", "needed" }`. Only the creator accepts/declines; private contact details are not exposed before consent (PRD §7.10).

## Chat (Implemented R3)

| Method | Path | Purpose |
|---|---|---|
| GET | `/chat/servers` | List servers |
| POST | `/chat/servers` | Create server |
| GET | `/chat/servers/:id/channels` | List channels |
| POST | `/chat/servers/:id/channels` | Create channel |
| GET | `/chat/channels/:id/messages` | Paginated message history |
| POST | `/chat/channels/:id/messages` | Send message |
| GET | `/chat/conversations` | List DMs/group conversations |
| POST | `/chat/conversations` | Start conversation |
| WS | `/ws` | Authenticated WebSocket delivery |

Illustrative shape: `{ "id", "channel_id", "author_id", "body", "created_at" }`. Messages are persisted before/as part of delivery; reconnecting clients request messages after the last received ID. WebSocket connections enforce the same authorization as REST (PRD §7.12). The edge proxy already forwards `/ws/`.

## Discovery (Planned R4)

| Method | Path | Purpose |
|---|---|---|
| GET | `/discovery/friends` | Interest-based friend discovery |
| POST | `/connections` | Send connection request |
| PATCH | `/connections/:id` | Accept / ignore / block |

Illustrative shape: a profile summary plus an understandable compatibility explanation (e.g. shared titles), never a "scientifically validated" score. Results respect discoverability, visibility and blocks (PRD §7.13).

## Optional adults-only area (Planned R5 — feature-gated)

No endpoints are active by default. The `matching` module is gated by `FEATURE_ADULT_DATING=false` and uses a separate `/adult/*` namespace with an independent opt-in and age gate (PRD §7.14, conventions). Non-eligible or non-opted-in users must never appear in adult discovery. Nothing here is scheduled for the initial releases.

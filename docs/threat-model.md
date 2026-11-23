# PromisedLand — Threat Model (abridged)

> Scope: the initial web application and single-host deployment. Grounded in PRD §7.1, §7.6, §7.15, §12.3 and §12.4. This is a living document; items are marked by release status rather than claimed as complete.

## Assets

| Asset | Why it matters |
|---|---|
| Account credentials and session tokens | Account takeover, impersonation |
| Email addresses and private profile fields | Privacy; must not appear in public APIs |
| User-generated content (posts, comments, media) | Harassment, copyright, illegal content, spoiler harm |
| Moderation records and audit log | Integrity of enforcement and appeals |
| SQLite database on EBS | Single source of truth; loss = total data loss |
| `SESSION_SECRET`, AWS credentials | Full compromise if leaked |

## Trust boundaries

1. **Browser ↔ edge/API:** untrusted input; TLS required in production; CORS restricts browser origins.
2. **Anonymous ↔ authenticated:** every protected route checks the session server-side, not just via route guards.
3. **Member ↔ moderator/admin:** community roles are scoped; community roles never grant platform-wide permissions (PRD §5).
4. **API ↔ SQLite:** parameterized queries, single writer, transactions for membership/vote/RSVP operations.
5. **API ↔ S3 / external metadata provider:** untrusted or unavailable third parties; core features must degrade gracefully.
6. **Host ↔ internet:** only required ports published; SSH restricted; secrets from environment, not images.

## Top threats and mitigations

| Threat | Mitigation | Status |
|---|---|---|
| **Credential stuffing / brute force** | Rate-limit login and password reset; bcrypt password hashing; generic "invalid credentials" response that does not reveal registered emails; secure HTTP-only `SameSite=Lax` / `Secure` session cookie | Planned R1 (auth module) |
| **IDOR / privilege escalation** | Mandatory server-side ownership and role checks in every protected handler; IDs are immutable; client cannot self-assign roles; centralized helpers in `shared` | Planned R1, enforced per endpoint |
| **XSS via posts, comments, profiles** | Treat all user content as text; no raw HTML injection in templates; sanitize any rich text; escape on output; CSP is a future hardening item | Planned R1; CSP / sanitizer audit future |
| **CSRF on cookie-based sessions** | `SameSite=Lax` and `Secure` cookies; require non-idempotent requests to be same-site; explicit CSRF token/middleware is a future hardening item if cross-site flows are needed | Partial (cookie attributes) R1; token future |
| **Image upload abuse** (malware, oversized files, path traversal) | Validate extension, MIME, file signature, size and dimensions server-side; random object keys in S3; no user filenames in paths; public access blocked by default; upload rate limits | Planned R1 (validation) / R2 (S3 flow) |
| **Spoiler leakage** | API returns spoiler metadata; hidden-by-default rendering; community spoiler rules; report unmarked spoilers. User-supplied tags may be wrong — do not rely on them alone | Planned R1 (manual), R1+ (watch-progress rules) |
| **Moderation bypass / repeat abuse** | Report + block + remove + suspend flows; audit log with actor, target, reason, timestamp, scope; rate limits on posting, comments, invites, messaging; appeal process | Basic controls Planned R1; expanded workflow R2 |
| **Private data exposure** | Separate public/private account fields; no precise location by default; discoverability toggles; parameterized queries; automated authorization tests | Planned R1 |
| **Secret leakage** | `.env` + secrets manager only; `.env` ignored; no secrets baked into images; least-privilege IAM and security groups | R0 configuration in place; production secrets procedure to be tested |

## Explicitly future

- Email verification before sensitive actions (R1), robust age assurance and full safety review for the optional dating module (R5, PRD §7.14) — see the adult-area addendum below and `docs/adult-safety.md`.
- End-to-end encrypted messaging (out of the initial messaging scope, PRD §3.3).
- WAF, automated dependency/image scanning in CI, and a documented incident-response runbook (PRD §11.3, §12.3).
- Abuse-scaling controls for WebSocket chat (per-connection and per-user rate limits) when R3 ships.

## Adult area addendum (R5 — disabled-by-default scaffold)

> **Residual risk: HIGH until all PRD §7.14 prerequisites are met and the `docs/adult-safety.md` launch checklist is signed off.** The scaffold is not launch-ready; the entry below documents the target design and which mitigations actually exist today. Enabling `FEATURE_ADULT_DATING` does not lower this rating.

### Additional assets

| Asset | Why it matters |
|---|---|
| Age-assurance verification results / eligibility state | Determines access to a high-risk area; tampering or leakage enables minor access and privacy harm |
| Adult opt-in profiles, intents, likes and matches | Highly sensitive; reveals romantic/sexual intent and social graph |
| Adult safety reports (`target_type='adult_profile'`) and moderation actions | Evidence for enforcement and appeals; must be prioritised and auditable |
| Linkage between adult and general identities | Cross-linking an adult profile to a general fandom profile is a privacy harm |

### New threats and mitigations

| Threat | Mitigation mapped to the scaffold | Status |
|---|---|---|
| **Age misrepresentation / minor access** | Deny-all `ADULT_AGE_ASSURANCE=none` by default; injectable `AgeAssurance` interface; eligibility decided server-side, never from a self-entered birthday; dev stub restricted to development | **Scaffold implemented, protected by deny-all.** Real provider **not implemented** — see below |
| **Grooming and sexual predation** | Adults-only gate, opt-in profiles, mutual match before messaging, high-priority reporting, safety policy and education (PRD §7.14) | **Not implemented** (policy/staffing pending); scaffold boundary design only |
| **Impersonation** | Verified eligibility + opt-in profile; report reason and high-priority moderation path; no cross-link to general profiles | **Partial** (report path designed); detection and staffing not implemented |
| **Scams** | Safety policy on scams; report + block + high-priority review; no contact details exposed | **Not implemented** (policy/staffing pending) |
| **Unwanted contact / harassment / sexual harassment** | Messaging only after mutual match; immediate unmatch, block and report; separate intent and visibility controls | **Scaffold implemented** (mutual match, unmatch, high-priority report); no moderation capacity yet |
| **Location leakage** | No exact/live location, phone or private contact fields in any adult shape or endpoint | **Scaffold implemented and test-asserted** (no location/contact fields in schema or JSON); re-verify on any future change |
| **Privacy cross-link with main profiles** | Separate namespace (`/adult/*`), separate opt-in profile, non-opted-in users never visible, no inference of intent from ordinary interests | **Scaffold implemented**; gated and unlinked, tests assert non-opted-in targets are hidden |
| **Accidental production enablement** | `FEATURE_ADULT_DATING=false` and `ADULT_AGE_ASSURANCE=none` defaults kept off in `.env.example`, Compose and deployed `.env`; stub unavailable in production | **Partial** (defaults present in config/Compose); deployment guardrails documented in `docs/operations.md` |

### Mitigations explicitly NOT yet implemented

- **Real age assurance** — no real provider is integrated; `none` denies all and `development_stub` is only a local test double. A self-entered birthday must never be accepted as sufficient.
- **Producer-side moderation staffing** — no staffed capacity to triage high-priority adult safety reports within a defined SLA.
- **Appeals** — no appeal process for significant adult-area enforcement actions.
- The `/adult/*` endpoints, `021_adult.sql` migration and unlinked `/adult` UI are implemented but mounted behind a 404 gate; they are unreachable with the default flags and must not be enabled until all prerequisites are met.

Until these are in place — together with the independent validation, legal/privacy review, safety policy and product controls in `docs/adult-safety.md` §3 — **residual risk for the adult area remains HIGH**.

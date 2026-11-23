# ADR 0001 — Modular monolith for the initial backend

- **Status:** Accepted (Release 0)
- **Date:** 2026-11-23
- **Context:** PRD §10.1 calls for one Go API process with domain-separated modules. The project is a learning/portfolio build with a small team and a deliberately constrained Release 1. Microservices would add deployment, networking, observability and data-consistency overhead without product benefit at this scale.

- **Decision:** Build a single Go API (chi router, `net/http`) with domain packages under `backend/internal/` (`auth`, `users`, `anime`, `communities`, `posts`, `comments`, `reactions`, `notifications`, `moderation`, `events`, `chat`, `discovery`, `matching`, `platform`). Modules register routes through `shared.Deps` and exchange data via `shared` DTOs; they do not import each other. One process, one container, one database connection pool. Frontend and backend remain separate images/containers.

- **Consequences:**
  - **Positive:** simpler local setup, CI, deployment and debugging; straightforward transactions across domains; cheap to test with `httptest`.
  - **Negative:** all modules share one process and one database; a slow module affects the whole API. Discipline is required to keep package boundaries from eroding.
  - **Revisit when:** independent scaling, isolation, or team ownership becomes a real requirement. Any move to multiple writing instances also requires leaving single-file SQLite (see ADR 0002).

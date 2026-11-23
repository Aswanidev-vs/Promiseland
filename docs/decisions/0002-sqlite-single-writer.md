# ADR 0002 — SQLite with a single writer

- **Status:** Accepted for the initial single-instance deployment
- **Date:** 2026-11-23
- **Context:** PRD §10.5 and §3.3 specify SQLite for local development and the first EC2 deployment, with no multiple backend instances sharing one file. Release 1 targets a small closed beta (PRD §3.4), where operational simplicity matters more than write concurrency.

- **Decision:** Use SQLite via pure-Go driver with `foreign_keys(1)`, `journal_mode(WAL)` and `busy_timeout(5000)` in the DSN, and `db.SetMaxOpenConns(1)` so there is exactly one writer. The database lives on a persistent EBS-backed host path (named Docker volume locally), never in the disposable container filesystem. Migrations are embedded SQL applied in filename order at startup in a transaction. Database size, write contention, backup duration and query latency are monitored.

- **Consequences:**
  - **Positive:** minimal setup, no separate DB server, cheap local/integration tests, easy backup of a single file set.
  - **Negative:** write throughput is limited; the database cannot be shared across independently writing instances; WAL sidecar files (`-wal`, `-shm`) must be included in backups and handled on restore.
  - **Revisit when:** horizontal scaling or high write concurrency is needed. Migrate to PostgreSQL before adding multiple backend instances; treat that as a deliberate data-migration project, not an incremental config change.
  - **Operational requirement:** test the backup/restore procedure before public launch (PRD §3.4, §11.4); see `docs/operations.md`.

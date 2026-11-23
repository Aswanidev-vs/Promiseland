# PromisedLand — Adult Dating Safety Gate (Release 5)

> **STATUS: DISABLED-BY-DEFAULT SCAFFOLD. NOT LAUNCH-READY.**
>
> This is the definitive gate document for the optional adults-only dating area (PRD §7.14, Release 5 / "R5"). The scaffold exists only to prove the architecture and the safety boundaries. It is **not** a shippable feature. Do **not** enable it in any environment that real users can reach until every prerequisite in §3 is met and every item in the §5 launch checklist is signed off by its owner.

## 1. Scope and framing

PRD §§1, 3.3, 4.5, 7.14 and 9 classify dating as a **separate product gate (P3)** that is explicitly excluded from the MVP and from Releases 0–4. PromisedLand is an anime-first fandom social network, **not** a dating app; the brand promise is "Your fandom. Your people. Your place." Friend discovery (R4) and any adult dating experience (R5) are distinct product decisions and must never be conflated.

This document governs only the R5 adult area. The general platform's moderation, reporting and privacy controls (PRD §7.15, §7.17, §12.4) remain in force everywhere and are a prerequisite, not a substitute, for the additional controls described here.

## 2. Current implementation state (honest status)

| Layer | State in this release |
|---|---|
| `FEATURE_ADULT_DATING` config flag | Present in backend config; parsed and validated (bool), default `false`. |
| `ADULT_AGE_ASSURANCE` / `AgeAssurance` eligibility interface | **Implemented as a scaffold:** an injectable interface with a default `deny-all` implementation and a development-only stub. The stub writes eligibility only via `/adult/dev/verify` and is unavailable in production. This is a test double, **not** real age assurance. |
| `/adult/*` endpoints, `021_adult.sql` migration, `/adult` route/UI | **Implemented but gated and unlinked:** routes exist, the migration ships, and a single unlinked `/adult` view calls status first. With the default flags every endpoint returns 404, so nothing is reachable in a normal deployment. |
| Safety policy, legal/privacy review, age-assurance provider, moderation & appeals capacity | **Not started.** These are organisational controls, not code. |

Because the deployed surface is a scaffold, the residual risk of the adult area is **HIGH** (see `docs/threat-model.md`). Enabling the flag does not by itself make the feature safe.

## 3. PRD §7.14 prerequisites still required before production enablement

All of the following are outstanding. None are satisfied by the scaffold.

1. **Independent product validation.** Confirm through separate research (not inferred from general fandom usage) that an adult discovery/dating experience is wanted, safe and appropriate for this community.
2. **Robust age assurance appropriate to the operating jurisdictions.** A **self-entered birthday is explicitly insufficient** for a high-risk dating feature. A real, jurisdiction-appropriate age-assurance provider must be selected, integrated and reviewed. Only adults who pass it are eligible.
3. **Legal and privacy review.** Cover applicable age, privacy, consumer and online-safety requirements for every jurisdiction in which the feature would be offered, before release. Dating data requires separate consent and access controls (PRD §12.4).
4. **Safety policy.** Published, enforceable policies covering **impersonation, harassment, sexual harassment, scams and unwanted contact**, plus safety education for users.
5. **Moderation and appeals capacity.** Staffed producer-side moderation with the ability to review adult-safety reports at high priority, take enforcement action, and run an appeal process for significant actions (PRD §7.15).
6. **Separate friendship and dating intent.** Independent opt-in and intent settings (`friendship` / `dating` / `both`). Ordinary profile interests or community activity must never be treated as dating intent.
7. **Explicit opt-in visibility.** A separate dating profile is created only by an eligible adult who opts in; no user is enrolled implicitly.
8. **Mutual match before messaging.** A match requires a positive action from **both** users; messaging is unlocked only after a mutual match.
9. **Immediate unmatch, block and report.** Either party can unmatch, block or report at any time, with no friction or cooling-off period.
10. **No location or contact exposure.** No exact or live location, phone number or private contact details are displayed by default anywhere in the adult area.
11. **Separation from general profiles.** Dating profiles are isolated from the profiles of users who have not opted in, and must not leak into general discovery, search or feeds.

Operational acceptance criteria from PRD §7.14: non-eligible users cannot access dating discovery/matching; non-opted-in users are never displayed; a match requires two positive actions; either user can unmatch or block immediately; and a dating safety report enters a prioritised moderation workflow.

## 4. How the gate is enforced (design contract and current state)

The R5 design uses two independent controls. Both must fail closed.

| Control | Required behaviour | Current state |
|---|---|---|
| `FEATURE_ADULT_DATING` | Default `false`. When `false`, every `/adult/*` endpoint returns `404 not_found` and no UI link is rendered. Only a boolean `true` may enable the namespace. | **Present** in backend config and Compose; default `false`. Verify it stays `false` in all deployed environments. |
| `ADULT_AGE_ASSURANCE` | Default `none` = **deny-all**: no user is eligible, regardless of the feature flag. `development_stub` is valid **only when `APP_ENV=development` AND `FEATURE_ADULT_DATING=true`**, and must never be available in production. | **Implemented** in the backend and wired through `.env.example` and Compose; default `none` fails closed. Covered by tests that assert eligibility is denied unless the stub is explicitly enabled in development. |
| `AgeAssurance` interface | Eligibility is decided by an injectable interface whose default implementation denies everyone. Handlers rely on it; eligibility is never inferred from profile fields. | **Implemented**; the deny-all implementation is the default. No test or helper may be shipped that grants eligibility outside development. |
| `POST /adult/dev/verify` | Development-only stub that forces eligibility for local testing. Available **only** when `APP_ENV=development` AND the flag is enabled. Never in production. | **Implemented**; returns 404 outside development. It writes a verification row that has no effect unless `development_stub` is selected. |

**What must be replaced before any real use:** the `none` / `development_stub` age-assurance implementations must be replaced with a **real age-assurance provider** appropriate to the operating jurisdictions (prerequisite §3.2). A stub is a test double, not an eligibility mechanism. Enabling the feature while a stub is active would allow minors and ineligible users into a high-risk area.

Fail-closed expectations: if either control is missing, invalid, or misconfigured, the area must remain inaccessible — never fail open. A self-entered date of birth is not an acceptable input to eligibility.

## 5. Launch checklist (all owners/statuses intentionally unassigned/not started)

| # | Item | Owner | Status |
|---|---|---|---|
| 1 | Independent product validation completed and recorded | Unassigned (TBD) | Not started |
| 2 | Real age-assurance provider selected, jurisdiction-reviewed and integrated; `none`/stub removed from the enabled path | Unassigned (TBD) | Not started |
| 3 | Legal + privacy review signed off (age, privacy, consumer, online safety) | Unassigned (TBD) | Not started |
| 4 | Safety policy published (impersonation, harassment, sexual harassment, scams, unwanted contact) + user safety education | Unassigned (TBD) | Not started |
| 5 | Producer-side moderation staffing and high-priority adult-report workflow operational | Unassigned (TBD) | Not started |
| 6 | Appeals process for significant enforcement actions operational | Unassigned (TBD) | Not started |
| 7 | Separate friendship/dating intent and independent opt-in verified end to end | Unassigned (TBD) | Not started |
| 8 | Mutual-match-before-messaging and immediate unmatch/block/report verified | Unassigned (TBD) | Not started |
| 9 | No exact/live location or contact exposure verified in API and UI | Unassigned (TBD) | Not started |
| 10 | Separation from general profiles/discovery verified (non-opted-in users never shown) | Unassigned (TBD) | Not started |
| 11 | Automated tests for every §7.14 acceptance criterion pass | Unassigned (TBD) | Not started |
| 12 | Monitoring/audit for `target_type='adult_profile'` reports verified (see `docs/operations.md`) | Unassigned (TBD) | Not started |
| 13 | Incident rollback (disable flag) tested | Unassigned (TBD) | Not started |
| 14 | Explicit go/no-go decision recorded by product, legal and safety owners | Unassigned (TBD) | Not started |

No item may be marked complete on the basis of the scaffold existing.

## 6. Red lines — never, under any configuration

1. **Never infer romantic or dating intent from ordinary interests, anime titles, communities, cosplay activity or any other general-platform behaviour.** Dating intent exists only where an eligible adult has explicitly opted in.
2. **Never show non-opted-in users** in adult discovery, matching or any related surface. Absence of an opt-in record means "not eligible to be shown", full stop.
3. **Never expose exact or live location** (or phone numbers / private contact details) in the adult area. Broad, user-supplied region text is the only permissible location-like data and only where separately reviewed.

Violating a red line is grounds to disable the feature immediately, regardless of launch-checklist status.

## 7. References

- PRD §7.14 (Optional adult-only dating and matching), §3.3 (non-goals), §4.5, §5, §7.15, §12.3, §12.4, §20, §21.
- `docs/conventions.md` — "Release 5 cross-agent contract (R5 — GATED, NOT LAUNCH-READY)".
- `docs/threat-model.md` — adult-area threat addendum and residual risk.
- `docs/operations.md` — keeping the feature disabled, pre-enablement checks, monitoring and rollback.

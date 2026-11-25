# ADR 0003 — The promotional landing page lives in the Vue app

- **Status:** Accepted
- **Date:** 2026-11-25
- **Context:** A standalone promotional page existed as plain HTML/CSS/JS under `docs/`, with its own
  palette and font stack, so it drifted from the app's live tokens (`frontend/src/styles/tokens.css`:
  vermilion coral + Plus Jakarta Sans, dual theme). It also duplicated copy and sample content that
  had to be maintained twice. The route `/` already rendered `frontend/src/views/LandingView.vue`,
  but that view sat inside the authenticated app chrome and carried stale references to the previous
  Fraunces/ember direction.
- **Decision:** The promotional landing page is a Vue view at `/`, owned by the frontend app. It
  renders without the app chrome (sidebar, top bar, bottom tabs) and uses only tokens from
  `tokens.css`. `docs/` keeps the documentation site; it is no longer the source of the marketing page.
  `docs/design-system.md` §6 remains the section list of record for the landing, amended by this ADR.
- **Consequences:**
  - **Positive:** one palette, one type stack, one router entry; the promotion page can link into
    real routes and use real components (`BaseButton`, `SpoilerShield`, `WayfinderMark`, `AppIcon`);
    `bun run build` and `bun run test` cover it.
  - **Negative:** the page now depends on the app build, so a marketing-only change ships with the
    app. Static hosting of the old `docs/` page is no longer the delivery route.
  - **Open drift:** `docs/design-system.md` v2 still documents the ember/Fraunces palette that
    `tokens.css` has since replaced. `tokens.css` is the source of truth for palette and type until
    that document is rewritten.

## Landing structure (amends design-system §6)

1. **Masthead** — wordmark, three in-page anchors, theme toggle, sign in, create account.
2. **Hero** — the promise as three designed lines, one ember action, one quiet beta note; the right
   column is the signature **interest map**: sample interests plot as stars, linked when several are
   chosen. This replaces the static wayfinder chart in the hero and is the page's one interactive bet.
3. **Experiences** — Discuss, Share, Meet, Chat as a hairline-divided grid, not cards. Chat is
   labelled as a later release.
4. **Spoilers** — the real `SpoilerShield` component as a live demo, plus the report path.
5. **Events** — an event pass card with the only rich media on the page.
6. **Closing** — wayfinder, wordmark, tagline, one quiet action.
7. **Footer** — wordmark, tagline, public links.

## Decision trace

| Decision | Reason | Alternatives rejected | Trade-off |
|---|---|---|---|
| Interactive interest map in the hero | The product's promise is "your interests graph everything you see"; showing the graph beats describing it, and it is the one invented interaction the page can afford | Static wayfinder emblem (present, memorable, but passive); product screenshot mock-up (invented UI, and a browser-window mock is template chrome) | A hero control asks the visitor for a click before reading the promise; mitigated by the promise column staying the visual first read and the map being optional to touch |
| Coral/Plus Jakarta Sans from `tokens.css`, not the ember/Fraunces spec doc | Existing brand assets and live code outrank a stale document; the logo itself is vermilion | Follow `docs/design-system.md` v2 and diverge from every other screen | The design doc now describes an older palette; recorded as open drift above |
| Bare shell for the landing route | A public page should not show a signed-in sidebar, search bar or app tab bar | Reuse the app shell (duplicates brand chrome, exposes dead app surfaces to guests) | One more prop on `AppShell` and a route-aware branch in `App.vue` |
| Interest map labelled "Sample map, not live data" | No fabricated proof; the demo must not read as real community activity | Fabricated counts ("2,400 fans online") | The demo is quieter than typical marketing pages |
| One ember action per screen (hero only) | Design system rule; the masthead and closing carry secondary actions | Ember in masthead, hero and closing (three competing accents) | Repeat visitors scroll slightly further for a loud CTA |
| Mono reserved for the sample data note, not labels | The design system bans mono as decorative label type | Mono eyebrows above each section | Fewer coded "technical" cues |

# PromisedLand — Design System v2

Contract for every frontend change. PRD §2.5 (brand) and §16 (UX/visual direction) set product
direction; this document sets visual and interaction execution. **Update this file before the code
when direction changes.**

**Precedence:** PRD > this file > component habit. The frontend-design skill's craft floor and
tell-lists apply everywhere this file is silent.

**What this pass exists to fix** (user-validated): the app reads as a generic dark dashboard, is
not warm or cozy enough, is inconsistent and unpolished across screens, has weak brand moments,
and has a flat type hierarchy. Every section below maps to one of those five.

---

## 1. Direction (five lines)

- **Product:** an anime-first fandom social network — spoiler-aware communities, a creator feed
  for cosplay and fan art, real-world events — wrapped around one idea: your anime interests graph
  everything you see. Promise: *Your fandom. Your people. Your place.*
- **Character:** **warm · composed · earnest.** Not childish, not corporate, not anime-cliché.
- **World:** the lantern-lit street outside the convention hall at dusk. Night is deep ink-blue,
  light is lantern amber, printed programmes are warm parchment, the harbour keeps a cool teal.
  Every neutral is chosen as if lit by lantern light — that is where "cozy" comes from.
- **Moments (where richness lives):** (1) the wayfinder constellation — brand mark, hero, empty
  states, profile bearing; (2) event and creator media cards — the only surfaces allowed rich
  color and imagery. Chrome (nav, buttons, inputs, labels) stays quiet.
- **Signature:** *interests plotted as a star chart.* The wayfinder/constellation motif appears in
  the logo, hero, empty states, the profile "bearing" panel, and as a coordinate tag on community
  cards. The wordmark carries Fraunces' WONK axis — a quirk only this brand has.

**Inventiveness dial: medium.** Social product, broad audience, trust matters. Conventional
navigation and interaction; identity carried through type, color, spacing, and one motif. No
scroll-jacking, no ambient loops, no 3D, no glassmorphism.

---

## 2. Foundations

### 2.1 Color — "paper and lantern light on a night street"

The palette is refined, not replaced: surfaces stay cool ink-blue **night**, everything printed on
them (text, hairlines) goes **warm parchment**, and amber does the work. That warm-on-cool split
is the whole cozy/premium strategy.

| Token | Value | Role | Contrast |
|---|---|---|---|
| `--ink-900` | `#090D12` | page background (night) | — |
| `--ink-850` | `#0E141C` | sunken surfaces: inputs, sidebar, wells | — |
| `--ink-800` | `#131B25` | primary surface: cards | — |
| `--ink-700` | `#1B2531` | raised surface, hover, nested wells | — |
| `--ink-600` | `#273345` | strong borders, dividers | — |
| `--text-primary` | `#F2EDE3` | primary text (parchment) | 16.3:1 on page |
| `--text-secondary` | `#B8B0A3` | secondary text (warm taupe) | 8.8:1 on page, 8.1:1 on card |
| `--text-muted` | `#8B857A` | meta, timestamps (warm grey) | 5.1:1 on page, 4.7:1 on card |
| `--ember` | `#F2A65A` | **the lantern** — primary action, active state | 9.4:1 |
| `--ember-strong` | `#F7BC7C` | hover/focus accent | — |
| `--ember-soft` | `color-mix(ember 14%, transparent)` | active nav/tab tints, focus backgrounds | — |
| `--tide` | `#5BC8C0` | links, discovery, "follow/discover" affordances only | 9.7:1 |
| `--paper` | `#EDE6D9` | printed detail: date stamps, badges, wordmark | — |
| `--alert` | `#E2735E` | destructive, moderation, reports | — |
| `--ok` | `#5FB98C` | success, joined, going | — |
| `--scrim` | `rgba(6, 8, 11, 0.72)` | modal/dialog backdrop | — |

Rules:

- **Ember marks the single primary action per screen.** Tide is for links and discovery only.
  Never scatter both as decoration; never ember on two competing buttons in one view.
- **Hairlines** derive from text color: `color-mix(in srgb, var(--text-primary) 10%, transparent)`
  (rgba fallback `rgba(242, 237, 227, 0.10)`), strong at 18%.
- **No hue-to-hue gradients on surfaces, no gradient text, no glows, no blobs, no purple, no
  glassmorphism.** Depth comes from surface steps, hairlines, and one inset lift highlight.
- **No hardcoded hex in any component.** If a color is needed, add a token here first. A `grep`
  for `#[0-9a-fA-F]{3,6}` outside `tokens.css` should return nothing.
- Text contrast ≥ 4.5:1 everywhere, including muted text on cards (values above already verified).

### 2.2 Typography — variable fonts, real hierarchy

**Load variable fonts, not static weights.** Fraunces ships four axes — weight (`wght`),
optical size (`opsz`), softness (`SOFT`), wonk (`WONK`); Inter ships `opsz` (14–32) and `wght`.
Static weight files freeze `opsz`, which is exactly why the hierarchy reads flat today.

- `@fontsource-variable/fraunces` → `--font-display: 'Fraunces Variable', Georgia, serif`
- `@fontsource-variable/inter` → `--font-body: 'Inter Variable', system-ui, sans-serif`
- `@fontsource/ibm-plex-mono` (static, 400/500) → `--font-mono: 'IBM Plex Mono', ui-monospace, monospace`
- Import axis bundles so `opsz` (and Fraunces `SOFT`/`WONK`) are actually present; confirm the
  exact family strings from the installed `index.css` after `bun add`.
- `font-optical-sizing: auto` on all display type; hero display additionally sets
  `font-variation-settings: 'SOFT' 30` for rounder, warmer terminals.

**Roles:**

| Role | Family / weight | Size | Tracking / leading |
|---|---|---|---|
| Hero display (landing) | Fraunces 600, `opsz` auto | `clamp(2.75rem, 5.5vw, 4.25rem)` | `-0.025em` / `1.03` |
| Page title (h1) | Fraunces 560 | `2.5rem` → `3.25rem` | `-0.02em` / `1.1` |
| Section title (h2) | Fraunces 560 | `1.875rem` | `-0.015em` / `1.15` |
| Card/feature title (h3/h4) | Fraunces 560 | `1.375rem` / `1.0625rem` | `-0.01em` / `1.25` |
| Lede | Inter 400 | `1.0625rem` | `1.6` |
| Body | Inter 400 | `0.9375rem` | `1.6` |
| UI label (buttons, tabs, nav) | Inter 500 | `0.9375rem` / `0.8125rem` | `1.2` |
| Meta (timestamps, counts, handles) | IBM Plex Mono 400/500, `tabular-nums` | `0.8125rem` / `0.75rem` | `1.4` |

Rules:

- Headings are **never** bold-by-default: 560 does the work; 600 only for the hero. No 700–900.
- Measure ≤ 70ch. Display line breaks are designed, not accidental (hero = three lines by hand).
- Mono is for figures, timestamps, counts, and coordinate tags — **not** decorative labels.
- **No:** all-caps tracked eyebrows, one word recolored/italicized in a headline, gradient-filled
  text, middle-dot meta strings (`A · B · C`), "WORD — fragment" labels, arrows on every link.
- The wordmark alone may set `font-variation-settings: 'WONK' 1`. Never on running text.

### 2.3 Space, radius, elevation

- **Spacing:** 4px grid (`--space-1…20`). Inside a group < between groups. One-off values
  (`mt-[13px]`, `padding: 13px 17px`) are defects; snap to the grid.
- **Radius:** `--radius-sm 4` / `--radius-md 8` (controls) / `--radius-lg 12` (cards, panels) /
  `--radius-xl 16` (hero media, feature panels only) / `--radius-pill`. Nested rule: inner radius =
  outer − padding between them, floor 4 (card 12 + padding 16 → inner media 8).
- **Elevation:** surface step + hairline first; shadows are quiet and single-direction.
  `--shadow-1` cards, `--shadow-2` dropdowns/sticky bars, `--shadow-3` modals. Raised cards may
  add `--lift-top` (inset 1px parchment highlight at 4.5%) — a bevel, never a glow.
- One radius for the same kind of thing everywhere: buttons `md`, cards `lg`, chips `pill`.

### 2.4 Motion

- One orchestrated page-load reveal: 40ms stagger, 8px rise, opacity, `--dur-reveal 420ms`.
- Everything else responds to a user action only (open, expand, confirm). One easing curve
  `cubic-bezier(.2,.7,.3,1)`.
- Transition **named properties only** (`color`, `background-color`, `border-color`, `opacity`,
  `transform`) — never `transition-all`, never width/height/top/left. No `hover:scale` on cards.
- No looping pulses or glows on anything that is not live. `prefers-reduced-motion` disables all
  of it (already handled in `base.css`; keep it working).

---

## 3. Component recipes

Exact states. If a component deviates, the component is wrong, not the recipe.

**Button** (`BaseButton.vue` — variants `primary | secondary | ghost | danger`, sizes `sm | md | lg`)
- Height: sm 36 / md 44 / lg 48. Radius `md`. Label Inter 500; sm text `--text-sm`, lg `--text-lg`.
- `primary`: ember bg, `--ink-900` text (9.4:1), hover `--ember-strong`, active `translateY(1px)`.
  **One per screen** — everything else is secondary.
- `secondary`: transparent, 1px `--border-strong`, `--text-primary`; hover bg `--ink-700`.
- `ghost`: transparent, `--text-secondary`; hover bg `--ink-800`, text `--text-primary`.
- `danger`: `--alert` bg, `--ink-900` text.
- Focus: `2px solid --ember` outline, offset 2. Disabled: 50% opacity, `not-allowed`.
- Loading: spinner replaces nothing — it sits before the label; width does not jump.

**Input / textarea / select** (`BaseInput.vue`)
- bg `--ink-850`, 1px `--border-hairline`, radius `md`, min-height 44, padding 12/14.
- Label above, Inter 500 `--text-sm`, `--text-secondary`; hint below in `--text-muted`.
- Hover: `--border-strong`. Focus: ember border + `0 0 0 3px var(--ember-soft)` ring.
- Error: `--alert` border + message in `--alert` (never color alone — icon + text).

**Card** (`BaseCard.vue`) — bg `--ink-800`, 1px `--border-hairline`, radius `lg`,
`--shadow-1`, padding `--space-5`. Interactive: hover border `--border-strong` + `--lift-top`;
**no scale, no shadow bloom.** Sunken variant (lists, composer wells): bg `--ink-850`, no shadow.

**Chip / tag** (`TagChip.vue`) — pill, bg `--ink-700`, text `--text-secondary`, mono `--text-xs`,
padding 4/10. Selected/active: `--ember-soft` bg, `--ember` text + 1px ember border. Topic tags are
pills; anime-title tags may use mono (catalogue feel). Max one chip row per card; wrap, never scroll.

**Avatar** — circle; fallback initials in Fraunces 560 on `--ink-700`, `--text-secondary`.
Sizes 28/40/56/96 (profile 96, feed 40, chat 28). No colored rings except verified/organizer
states, which use `--ember` 2px.

**Tabs** (feed tabs, profile tabs) — underline style: label Inter 500, inactive `--text-muted`,
active `--text-primary` with 2px ember underline; tab min-height 44. No pills-in-pills.

**Nav item** (sidebar/bottom bar) — height 40 (sidebar) / 48+ (bottom bar, 44 min touch),
radius `md`, icon 20 + label. Inactive `--text-secondary`; hover bg `--ink-800`;
active: bg `--ink-800`, text `--text-primary`, **2px ember marker** on the leading edge (sidebar)
or ember icon+label (bottom bar). Never two accent colors in one nav.

**Toast** — bottom-right (desktop) / above bottom bar (mobile), bg `--ink-700`, hairline,
radius `md`, `--shadow-2`, icon + one sentence, auto-dismiss 5s, `--ok` for success,
`--alert` for failure.

**Modal / dialog** — `--scrim` backdrop, panel bg `--ink-800`, radius `lg`, `--shadow-3`,
padding `--space-6`, max-width 480 (560 for report/composer). Escape closes, focus trapped,
close button is a ghost icon button with `title`.

**Skeleton** — `--skeleton-bar` on `--ink-700`, breathing opacity (existing primitive). Same
shape as the content it replaces; no centered spinner for full pages.

**Empty state** — WayfinderMark (small, 48–72px) + one sentence telling what will appear here +
one primary action. No emoji, no "Nothing here yet 👋".

**Spoiler shield** (`SpoilerShield.vue`) — content obscured with `backdrop-filter: blur(10px)` +
`--ink-900` 70% wash; label "Spoilers — tap to reveal" in mono `--text-sm`; reveal is an explicit
button; after reveal, a quiet "Hide again" text action stays.

---

## 4. Layout & shell

- **Desktop ≥1024:** sidebar 240 (bg `--ink-850`, right hairline) + main column max 720 +
  right rail 280, gap 24, canvas max 1280, centered. Top bar 64 (page bg, bottom hairline,
  search field max 480 centered).
- **Tablet 768–1023:** sidebar collapses to icon rail 64; rail hides below1280.
- **Mobile <768:** top bar 56 + fixed bottom tab bar (Home, Communities, Events, Notifications,
  Profile), 44px+ targets, page gets `padding-bottom` for it. Never a crowded hamburger mega-menu.
- **Right rail** = quiet discovery: "Trending in your fandom" (mono meta), "Upcoming events",
  "Who to follow". Sections separated by hairlines, not nested cards. No card-in-card.
- **Page rhythm:** vary density — feed is dense and scannable; landing and profile breathe
  (space-16/20 between sections). Section separation by hairline or space, never by a new bg color.

---

## 5. Signature moments (the brand fixes)

1. **Wayfinder constellation** (`WayfinderMark.vue`): logo in sidebar/topbar (24–28px), hero
   (large, with the plotted grid frame), empty states (small), 404. It represents the interest
   graph — a star chart of what you love — so it may also appear in the profile bearing panel.
2. **Profile bearing panel:** a compact star-chart strip under the profile header plotting the
   user's favourite titles as labeled stars connected by hairlines (pure SVG/CSS from data).
   This is the "your graph is real" proof moment on the most personal screen.
3. **Coordinate tag:** community and event cards may carry a mono coordinate-style tag
   (`35.68° N · fan-run` style) — quiet, `--text-muted`, never a middle-dot string.
4. **Event/creator media cards:** poster-like — media top with nested `radius-md`, an ember
   date block (paper-on-ink stamp for all-day/convention dates), Fraunces title, mono meta row
   (time · city in plain spacing, not middle dots). This is where color goes rich.
5. **Wordmark:** Fraunces with `'WONK' 1` — used in sidebar, landing hero, footer. Same
   treatment everywhere; never re-typeset ad hoc.

---

## 6. Screen notes (key screens for this pass)

- **Landing (Vue, route `/`, bare shell — see ADR 0003):** masthead (wordmark, in-page anchors,
  icon theme toggle, sign in, create account), then a hero: the three-line promise and one ember
  CTA left, the **interest map** right — six sample interests plotted as stars that link when
  several are selected, with a chip row and an `aria-live` readout, labelled "Sample map". Then
  the four PRD experiences as a hairline-divided grid (not identical rounded cards), the spoiler
  moment, the events pass card, closing and footer. No fabricated counts, logos or testimonials;
  every sample surface says so in its caption.
- **Home feed:** tabs (Following / For You) with ember underline; composer well sunken at top;
  posts as cards, media with nested radius; action row muted until active (ember like, tide
  nothing). Skeleton rows match card shape.
- **Communities (list):** search + topic chips row; list rows (not a card grid) with icon, name,
  mono member count, one-line description, join action right-aligned. Filter chips use the chip
  recipe; "Joined" state uses `--ok`.
- **Community detail:** banner strip (ink-700 + wayfinder coordinate tag), name in Fraunces,
   rules in a sunken well, posts reuse the feed card exactly.
- **Post detail:** same card, comments nested with a hairline thread line (indent 24/level,
  max depth 3), reply composer sunken. Spoiler blocks per recipe.
- **Profile:** banner +96 avatar overlap, Fraunces display name, mono handle, bio, interest chips,
   bearing panel, tabs (Posts / Media / Saved). Edit action = secondary; follow = primary ember.
- **Events:** poster-style media cards (signature #4), ember date block, RSVP primary; list
  alternates density (next event featured large, rest as rows).
- **Discover/Search:** search field focus state is the hero; results grouped by kind (people,
  communities, posts, events) with mono result counts; tide marks discovery actions.
- **Auth (login/register/reset/verify, bare shell):** own masthead (wordmark link, icon theme
  toggle), then a form column (max 420) and a brand panel (max 400, hidden below 1024) carrying the
  wayfinder chart, the three-line promise and one note. Fields use the sunken input recipe with
  show/hide on password fields; ember primary; secondary actions are quiet text links; errors are
  a bordered alert row with icon + text; the reset flow keeps its real 2-step copy. No illustration
  clutter.
- **Error states (`/error/:code`, plus the catch-all for 404):** one view, copy per code —
  403 forbidden, 404 off the chart, 429 slow down, 500 server error, 503 service unavailable;
  unknown codes fall back to 404. Retry (reload) for 429/500/503, sign-in for 403, and the public
  pages as a way onward. A 503 from the API replaces app screens only; the landing, legal and auth
  pages keep rendering with inline errors.
- **Legal (`/terms`, `/privacy`):** masthead, 70ch prose column, numbered clauses, an explicit
  "working draft, not legally reviewed" line, and a cross-link between the two. Account creation is
  blocked until both boxes are ticked; the links open in a new tab so the form is not lost.
- **Guest browsing (PRD §6 visitor role):** the `pl-guest` flag lets a visitor read public pages —
  events and creators — with a banner naming what an account adds, a trimmed sidebar and tab bar,
  no search, and a sign-in gate on RSVP and every other write action. Guests never see
  account-only destinations as live links.
- **Footer (`SiteFooter`):** brand column, Explore, Account, Legal, and a working small print row.
  Every link resolves to a real route; no `#` placeholders, no social icons for accounts we do not
  run. Reused on the landing, legal and error pages.

---

## 7. Imagery & icons

- User-generated media is the imagery — never stock, never random anime screenshots, never
  AI-generated franchise characters. Where media is absent, build the visual in SVG/CSS
  (wayfinder, grids, stamps) — never grey placeholder boxes, never emoji as icons.
- One icon set, one stroke weight (existing `AppIcon`), 16/20/24 sizes only.
- Media in cards: `object-fit: cover`, explicit aspect-ratio (4:3 feed, 3:4 event posters,
  16:9 banners) to prevent layout shift.

---

## 8. Copy

Sentence case everywhere. Buttons name the action ("Join community", "Publish post", "Save
changes"). One name per action through the flow. Errors state what happened and the fix, no
apology, no exclamation marks. Titles by context ("Home", "Settings"), never greeting headlines.
Placeholder data looks real — plausible titles, names, cities, counts. No rhetorical triplets,
no "seamless/unlock/supercharge", no badges like "New ✨".

---

## 9. Quality floor (every screen)

Responsive 375 → wide, no horizontal scroll. Touch targets ≥ 44px. Contrast ≥ 4.5:1 (3:1 large).
Visible `:focus-visible` ring in ember. Semantic headings, real `<label>`s, alt text where media
carries meaning. Loading / empty / error / success / permission-denied states for every data view
(empty states per recipe). `prefers-reduced-motion` honored. Primary action obvious in the first
screen. No hardcoded hex outside `tokens.css`.

---

## 10. Do / Don't (product-specific tells)

**Do:** warm parchment text on cool ink night; hairline structure; one ember action per screen;
Fraunces at 560 for headings; richness only in media cards and the wayfinder; the 4px grid.

**Don't:** purple; hue-to-hue gradients; gradient text; glow blobs; glassmorphism; emoji icons;
all-caps eyebrows; middle-dot meta strings; arrows on every link; `hover:scale` on cards;
`transition-all`; card-in-card-in-card; two accents fighting in one view; hardcoded hex;
fabricated proof; generic "Welcome back!" headlines; 700–900 heading weights.

---

## Appendix A — token block (source of truth for `tokens.css`)

```css
:root {
  /* Surfaces — night */
  --ink-900: #090d12;
  --ink-850: #0e141c;
  --ink-800: #131b25;
  --ink-700: #1b2531;
  --ink-600: #273345;

  /* Text — printed by lantern light */
  --text-primary: #f2ede3;
  --text-secondary: #b8b0a3;
  --text-muted: #8b857a;

  /* Accents */
  --ember: #f2a65a;
  --ember-strong: #f7bc7c;
  --ember-soft: color-mix(in srgb, var(--ember) 14%, transparent);
  --tide: #5bc8c0;
  --paper: #ede6d9;
  --alert: #e2735e;
  --ok: #5fb98c;
  --scrim: rgba(6, 8, 11, 0.72);

  /* Type — variable families confirmed from node_modules after install */
  --font-display: 'Fraunces Variable', Georgia, serif;
  --font-body: 'Inter Variable', system-ui, -apple-system, sans-serif;
  --font-mono: 'IBM Plex Mono', ui-monospace, monospace;

  --text-xs: 0.75rem;
  --text-sm: 0.8125rem;
  --text-base: 0.9375rem;
  --text-lg: 1.0625rem;
  --text-xl: 1.375rem;
  --text-2xl: 1.875rem;
  --text-3xl: 2.5rem;
  --text-4xl: 3.25rem;
  --text-hero: clamp(2.75rem, 5.5vw, 4.25rem);

  --leading-body: 1.6;
  --leading-heading: 1.15;
  --measure: 70ch;

  /* Spacing — 4px grid */
  --space-1: 4px;  --space-2: 8px;   --space-3: 12px;  --space-4: 16px;
  --space-5: 20px; --space-6: 24px;  --space-8: 32px;  --space-10: 40px;
  --space-12: 48px; --space-16: 64px; --space-20: 80px;

  /* Radii */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;
  --radius-pill: 999px;

  /* Borders & elevation */
  --border-hairline: rgba(242, 237, 227, 0.10);
  --border-strong: rgba(242, 237, 227, 0.18);
  --lift-top: inset 0 1px 0 rgba(242, 237, 227, 0.045);
  --shadow-1: 0 1px 2px rgba(0, 0, 0, 0.40);
  --shadow-2: 0 8px 24px rgba(0, 0, 0, 0.36);
  --shadow-3: 0 16px 48px rgba(0, 0, 0, 0.44);

  /* Motion */
  --ease: cubic-bezier(0.2, 0.7, 0.3, 1);
  --dur-fast: 160ms;
  --dur: 240ms;
  --dur-reveal: 420ms;
  --stagger-step: 40ms;

  /* Layout */
  --canvas-max: 1280px;
  --sidebar-w: 240px;
  --rail-w: 280px;
  --main-max: 720px;
  --layout-gap: 24px;
  --topbar-h: 64px;
  --bottomnav-h: 64px;
}
```

Keep the existing `@supports color-mix` fallback block, rewritten for the warm hairline rgba above.

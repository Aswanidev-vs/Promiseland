# PromisedLand — Design System Contract

Single source of truth for every frontend agent. Derived from PRD §16 and the frontend-design skill. Do not deviate without updating this file.

## Direction (five lines)

- **Product:** An anime-first fandom social network where fans find communities, share creative work, and meet up in real life.
- **Character:** warm, communal, exploratory.
- **World:** a lantern-lit harbour at dusk — the place fans gather after the convention doors close. Neutrals are tinted deep ink-blue; accents are lantern amber and a cool tidal teal.
- **Moments:** the brand mark / wayfinder motif, and event + creator media cards get the richness. Chrome (nav, buttons, inputs, labels) stays quiet.
- **Signature:** a thin "wayfinder" compass/constellation mark used consistently in the logo, empty states, and as a coordinate tag on community cards.

## Inventiveness dial

**Medium** (social product, broad audience, trust matters). Conventional navigation and interaction; distinct identity through type, color, spacing, and one signature motif. No scroll-jacking, no heavy ambient motion, no 3D decoration.

## Color tokens (CSS custom properties, `:root`)

| Token | Value | Role |
|---|---|---|
| `--ink-900` | `#0A0E13` | page background |
| `--ink-850` | `#0F151D` | sunken surfaces, inputs |
| `--ink-800` | `#141C26` | primary surface / cards |
| `--ink-700` | `#1C2734` | raised surface, hover |
| `--ink-600` | `#283546` | borders, dividers |
| `--text-primary` | `#ECF1F5` | primary text |
| `--text-secondary` | `#A8B7C5` | secondary text |
| `--text-muted` | `#6F8090` | meta, timestamps (check contrast) |
| `--ember` | `#F2A65A` | primary action, highlights, active state |
| `--ember-strong` | `#F7BC7C` | hover / focus accent |
| `--tide` | `#5BC8C0` | discovery, links, success-adjacent accents |
| `--alert` | `#E2735E` | reports, moderation, destructive |
| `--ok` | `#5FB98C` | success |

Rules:
- One neutral ramp tinted toward ink-blue. Borders = `color-mix(in srgb, var(--text-primary) 10%, transparent)` (rgba fallback).
- Accent **ember** marks the single primary action per screen. **Tide** only for discovery/links. Do not scatter both.
- No purple, no hue-to-hue gradients on surfaces, no gradient text, no glow blobs, no glassmorphism.
- Text contrast ≥ 4.5:1 (3:1 for large text/UI borders). `--text-muted` on `--ink-900` must be checked before use for small text.

## Typography

Self-host via `@fontsource/*` packages (no CDN dependency):

- `--font-display: 'Fraunces', Georgia, serif` — brand, page titles, hero. Warm, story-driven, literary. Variable weight 500–700, optical size on.
- `--font-body: 'Inter', system-ui, -apple-system, sans-serif` — body, forms, discussion text (deliberate: long threads must stay legible).
- `--font-mono: 'IBM Plex Mono', ui-monospace, monospace` — counts, timestamps, coordinates, code.

Scale (rem, 16px root): `xs .75`, `sm .8125`, `base .9375`, `lg 1.0625`, `xl 1.375`, `2xl 1.875`, `3xl 2.5`, `4xl 3.25`. Line heights: body 1.6, headings 1.15. Measure ≤ 70ch. `font-variant-numeric: tabular-nums` on all metrics.

Avoid: accenting one word in a headline, all-caps eyebrows, arrows on every link, middle-dot meta strings, emoji icons.

## Layout

- Desktop (≥1024px): left sidebar (240px) + main column (max 720px) + right rail (280px), centered gap 24px. Canvas max-width 1280px.
- Mobile (<768px): top bar + bottom tab bar (Home, Communities, Events, Notifications, Profile), 44px+ touch targets.
- Tablet: sidebar collapses to icon rail.
- 4px spacing grid. Nested radii: outer 12px, inner 8px, never below 4px. Consistent card treatment across all modules.

## Motion

- One orchestrated page-load reveal (staggered 40ms, translateY 8px, opacity). Nothing else ambient.
- Respond to user action only (expand, confirm, open). One easing curve: `cubic-bezier(.2,.7,.3,1)`.
- Animate only `opacity` and `transform`. Never `transition-all`. Honor `prefers-reduced-motion: reduce`.

## Quality floor

Semantic HTML, visible `:focus-visible` ring in ember, form labels, alt text, keyboard-navigable, responsive 375px → wide desktop with no horizontal scroll. Loading / empty / error / success / permission-denied states for every data view. Empty states use the wayfinder motif and a useful next action. No fabricated proof (counts, testimonials, logos).

## Copy

Sentence case. Buttons say the action ("Join community", "Publish post", not "Submit"). No greeting headlines; title by context ("Home", "Your communities"). No exclamation marks. Errors say what happened and how to fix it. Placeholder data must look plausible, never `[X]` or `123,456`.

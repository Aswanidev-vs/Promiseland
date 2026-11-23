# PromisedLand — frontend

Vue 3 + TypeScript + Vite + Pinia. Release 0 establishes the application shell,
responsive navigation, design tokens, typed API client, and a polished landing
page. Product copy and visual direction are defined in `PRD.md` and
`docs/design-system.md` at the repository root.

## Requirements

- Bun 1.4+ (do not use npm, node, or npx for this project)

## Commands

```bash
bun install          # generates/uses bun.lock
bun run dev          # Vite dev server, proxies /api and /ws to localhost:8080
bun run build        # vue-tsc --noEmit && vite build → dist/
bun test             # Vitest (jsdom)
bun run preview      # preview the production build
```

## Structure

```text
src/
├── api/client.ts          typed request<T>, ApiError, api facade (/api/v1)
├── stores/auth.ts         session state; /auth/me, 401/404/501 = anonymous
├── stores/ui.ts           toasts, sidebar and mobile-nav state
├── router/index.ts        routes + requiresAuth / guestOnly guard
├── styles/tokens.css      all design tokens (single source of truth)
├── styles/base.css        reset, type scale, focus ring, reveal/skeleton primitives
├── components/ui/         accessible UI kit + AppIcon set + WayfinderMark
├── components/layout/     AppShell, AppSidebar, TopBar, MobileNav
├── types/                 API DTOs and RouteMeta augmentation
└── views/                 LandingView, NotFoundView
```

## Adding routes in later releases

`src/router/index.ts` marks the insertion point for R1/R2/R3 routes. When real
destinations land, replace the disabled (`aria-disabled`, "Soon") nav entries in
`AppSidebar.vue` and `MobileNav.vue` with `<RouterLink>` items, point the landing
`entryTo` at `/home`, and replace the placeholder redirects in the route guard
with `{ name: 'login' }` / `{ name: 'home' }`.

## Design contract

- Dark ink-blue surfaces; ember marks the single primary action; tide marks
  discovery and links. All values live in `src/styles/tokens.css`.
- Fraunces (display), Inter (body), IBM Plex Mono (figures/coordinates), all
  self-hosted through `@fontsource`.
- The wayfinder compass/constellation is the signature motif. It appears in the
  brand mark, the landing hero, empty states, and the 404.
- One orchestrated page-load reveal (40ms stagger, 8px rise, opacity). Motion is
  disabled under `prefers-reduced-motion: reduce`.

## Container

`Dockerfile` is a multi-stage build: Bun compiles the app, then an unprivileged
nginx image serves `dist/` on port 80 with an SPA fallback (`try_files $uri
$uri/ /index.html`). Build and run locally with:

```bash
docker build -t promisedland-frontend:dev frontend
docker run --rm -p 8080:80 promisedland-frontend:dev
```

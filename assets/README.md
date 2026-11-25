# Brand assets

`brand/` is the source art: the logo and the illustration used on the auth screens.
Edit the originals here, then copy them to the two places that publish them.

| File | Source | Published copies |
|---|---|---|
| `brand/logo.png` | edit here | `frontend/public/logo.png` (served as `/logo.png`), `docs/logo.png` (static docs site) |
| `brand/represent.png` | edit here | `frontend/public/represent.png` (served as `/represent.png`), `docs/represent.png` |

The published copies exist because both consumers need the file inside their own
root: Vite serves `frontend/public/`, and the docs site is published from `docs/`.
Keep all three identical when the art changes.

Nothing in the codebase reads from this directory at build or run time, so it is
excluded from the Docker build context.

# HTTP Frontend (SvelteKit + SSG)

SvelteKit frontend for the HTTP APIs defined in `../openapi.yaml`. Uses static prerender (`adapter-static`), Telegram WebApp theme variables (with light/dark fallbacks), and API client codegen. A Telegram WebApp SDK script is injected via `src/app.html`.

## Prerequisites

- Node 20 (use `nvm use 20`); npm is the package manager.
- Backend OpenAPI schema at `../openapi.yaml`.

## Scripts

- `npm run dev` — start dev server.
- `npm run build` / `npm run preview` — build static site and preview.
- `npm run check` — typecheck via `svelte-check`.
- `npm run lint` — `svelte-kit sync` + ESLint (TS + Svelte).
- `npm test` / `npm run test:coverage` — Vitest + Testing Library.
- `npm run gen:api` — regenerate TS client from `../openapi.yaml`.
- `npm run format` — Prettier (with `prettier-plugin-svelte`).

## Env

Copy `.env.example` to `.env` and adjust:

```
VITE_API_BASE_URL=http://localhost:8080
VITE_TG_AUTH=        # optional, falls back to Telegram WebApp initData
VITE_USE_MOCK=false  # set true to force mock data without backend
```

## Structure

- `src/routes/+page.svelte` — search page UI (search bar, cards, load more).
- `src/lib/api` — generated client (do not hand-edit).
- `src/lib/api/client.ts` — OpenAPI base/auth config, avatar URL helper.
- `src/lib/components` — UI pieces (search bar, message card, avatar, skeleton, states).
- `src/lib/theme.ts` — theme detection, avatar palettes (8 light + 8 dark colors).
- `src/lib/mocks` — offline demo data for UI rendering.
- `static/` — place screenshots/assets for QA; served at site root.

## Notes

- The page prerenders by default; data fetching happens client-side.
- Telegram CSS variables are preferred; when absent, the app falls back to built-in light/dark themes and hash-based avatar colors.

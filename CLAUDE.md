# CLAUDE.md

42 AD Bachelor's Tracker: Vue 3 + TS + Vite SPA that maps a 42 student's validated projects to
AD Bachelor's credits (132 total). **Read `docs/ARCHITECTURE.md` first** for the full picture.

## Layout
- `frontend/`: the Vue SPA (all npm commands run from here)
- `worker/` + `wrangler.toml`: legacy Cloudflare Worker (to be removed)
- `docs/`: project notes

## Commands (in `frontend/`)
- `npm install` / `npm run dev`: dev server on :5173 (proxies `/v2` and `/oauth` to api.intra.42.fr)
- `npm run build`: type-check (vue-tsc) + vite build
- `npm run lint`: oxlint + eslint (both with `--fix`)
- Env: copy `frontend/.env.example` → `frontend/.env`

## Key facts
- Curriculum (module → credits → 42 project IDs) lives in `frontend/src/utils/data.ts` **and** is duplicated in
  `worker/index.js`. Keep them in sync until the Worker is removed.
- Credit logic: `frontend/src/composables/useProgress.ts` (each category capped; general is always 0 because there are no IDs).
- All 42 API calls go through `apiFetch` (`frontend/src/services/apiFetch.ts`) with the user's token. Base is `VITE_42_API_BASE`.
- `worker/index.js` (Cloudflare) handles OAuth token proxying, a generic 42 API proxy, and the hourly leaderboard in KV.

## Current direction
Replacing the Cloudflare Worker with a self-hosted **Go + Fiber + SQLite** backend that also
tracks student progress over time. See `docs/ARCHITECTURE.md` §9 for the endpoints that must keep working.

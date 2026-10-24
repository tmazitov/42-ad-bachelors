# CLAUDE.md

42 AD Bachelor's Tracker: Vue 3 + TS + Vite SPA that maps a 42 student's validated projects to
AD Bachelor's credits (132 total). **Read `docs/ARCHITECTURE.md` first** for the full picture.

## Layout
- `frontend/`: the Vue SPA (all npm commands run from here)
- `backend/`: Go + Fiber v3 + SQLite service, DDD layout. See `backend/README.md` for architecture and conventions
- `worker/` + `wrangler.toml`: legacy Cloudflare Worker (to be removed)
- `docs/`: project notes
- `compose.yaml` + `deploy/Caddyfile`: production stack (Caddy serves the SPA + proxies `/api`, HTTPS)

## Commands (in `frontend/`)
- `npm install` / `npm run dev`: dev server on :5173 (proxies `/api` to the backend on :8080; start the backend first)
- `npm run build`: type-check (vue-tsc) + vite build
- `npm run lint`: oxlint + eslint (both with `--fix`)
- No env vars needed for the frontend

## Commands (in `backend/`)
- `make run` (loads `.env`), `make test`, `make vet`, `make build`, `make docker`
- Go 1.26 (auto-downloaded via GOTOOLCHAIN)

## Key facts
- Curriculum (module → credits → 42 project IDs) lives in `frontend/src/utils/data.ts` **and** is duplicated in
  `worker/index.js` and `backend/internal/domain/curriculum/ad_bachelor.go`. Keep all three in sync until
  the frontend reads `GET /api/v1/curriculum` and the Worker is removed.
- Credit logic: `frontend/src/composables/useProgress.ts` (each category capped; general is always 0 because there are no IDs).
- The browser only calls the backend (`frontend/src/services/api.ts`, same origin, httpOnly `session` cookie).
  The backend does OAuth, stores students + validated projects in SQLite and refreshes them from 42 when stale.
- `worker/index.js` (Cloudflare) is the legacy production backend until the VPS deploy.

## Current direction
Deploying to a VPS (frontend + backend on one domain) and retiring the Cloudflare Worker.
See `docs/ARCHITECTURE.md` §9 for what is done and what is left.

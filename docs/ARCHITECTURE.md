# 42 AD Bachelor's Tracker — Architecture & Developer Notes

Working notes for the project as of Oct 2026 (last feature work: May 2026).
Goal of the next phase: replace the Cloudflare Worker with a self-hosted
**Go + Fiber + SQLite** backend that also tracks student progress.

---

## 1. What the app does

A Vue SPA where a 42 student logs in with 42 OAuth and sees how many credits
they have toward the 132-credit **AD Bachelor's** degree. Validated 42 projects are mapped
to bachelor modules (each with a credit value), and the credits are grouped into four
categories:

| Category | Required (cap) | Source of truth |
|---|---|---|
| General Education | 5 | `generalEducationModules`, no 42 project IDs, so it is always 0 |
| Standard | 101 | `standardModules` |
| Stream | 14 | `streamModules` |
| Elective | 12 | `electiveModules` |
| **Total** | **132** | |

Each category's credits are capped at its maximum (`Math.min`).

Features:
- **Home** (`/`): the logged-in user's progress doughnut. Clicking a category opens a table of its modules.
- **Student profile** (`/profile/:login`): any student's progress plus their in-progress projects.
- **Search modal** (header): search students by login (paginated) and 42 projects by name.
- **Leaderboard** (`/leaderboard`): ranking of everyone who has logged into the app. It is built by the Worker.

---

## 2. Repository layout

```
src/
  main.ts                    Vue app + Pinia + Router + PrimeVue (Aura, dark mode via `.dark`)
  App.vue                    Shell: AppHeader + RouterView + footer
  router/index.ts            Routes: /, /auth/callback, /profile/:login, /leaderboard
  services/
    oauth.ts                 Login redirect, code→token exchange, refresh (dev vs proxy mode)
    apiFetch.ts              fetch wrapper: adds Bearer token, retries once after refresh on 401
  stores/
    auth.ts                  Tokens in localStorage, auto-refresh 5 min before expiry, fetchUser(/v2/me)
    theme.ts                 Light/dark toggle persisted in localStorage
    counter.ts               Unused scaffold leftover
  composables/
    useProjects.ts           GET /v2/users/:id/projects_users?filter[status]=finished (1 page, 100 max)
    useProgress.ts           Credit calculation + CREDITS_REQUIRED + isCompleted()
    useStudentSearch.ts      GET /v2/users?search[login]=… (debounced, abortable, X-Total pagination)
    useProjectSearch.ts      GET /v2/cursus/21/projects?search[name]=…
  models/Project.ts          Project(code, name, credit, id: number | number[] | null)
  models/ProjectGroup.ts     ProjectGroup(name, projects[])
  utils/data.ts              ★ The curriculum: module code → credits → 42 project ID(s)
  views/                     HomeView, AuthCallbackView, StudentProfileView, LeaderboardView
  components/
    AppHeader.vue            Logo/home, avatar, leaderboard/search/logout/login/theme buttons
    profile/Profile.vue      Loads projects for a user → ProgressChart
    profile/ProgressChart.vue  Doughnut (chart.js) + per-category cards
    profile/ProjectsModal.vue  Module table per category (done / not done, links to intra)
    search/SearchModal.vue   Student + project search UI
    ProjectLookup.vue        Debug helper that lists cursus 21 projects; commented out in HomeView
worker/index.js              Cloudflare Worker (to be removed), see §4
wrangler.toml                Worker config: KV binding, hourly cron, CAMPUS_ID var
.github/workflows/deploy.yml Build → GitHub Pages; deploy Worker via wrangler-action
public/404.html, index.html  GitHub Pages SPA deep-link redirect trick (sessionStorage 'spa-redirect')
```

Stack: Vue 3.5, TypeScript 6, Vite 8, Pinia 3, vue-router 5, PrimeVue 4 (Aura), Tailwind v4,
Chart.js 4 / vue-chartjs. Node `^20.19 || >=22.12` (see `.nvmrc`). Lint: oxlint + eslint.

---

## 3. Curriculum data (`src/utils/data.ts`)

- `Project.id` can be:
  - `number`: a single 42 project ID.
  - `number[]`: alternatives. Any one of them validated counts, e.g. "Minitalk **or** Pipex", or old and new versions of the same project.
  - `null`: not trackable through the API (GEN seminars, BAI 499 Internship).
- A project counts as completed when its `projects_users` entry has `validated? == true`.
- **ID 1337 (ft_transcendence) appears twice**: as CCP 161 and as BAI 498 (Capstone). Validating it gives 8 credits. This is intentional and mirrored in the Worker.
- 42 cursus ID **21** = 42cursus (used by project search).

⚠️ **The curriculum is duplicated** in `worker/index.js` (`STANDARD` / `STREAM` / `ELECTIVE` arrays).
Any change must be made in both places. In the new backend this should exist **once**: the backend owns it,
and the frontend either fetches it or keeps a generated copy.

---

## 4. Cloudflare Worker (`worker/index.js`): what must be reimplemented

The Worker does three jobs.

### 4.1 OAuth token proxy (keeps `CLIENT_SECRET` off the client)
| Method & path | Body (JSON) | Forwards to 42 as |
|---|---|---|
| `POST /oauth/token` | `{ code, redirect_uri }` | `grant_type=authorization_code` + client_id/secret. **On success it also registers the user** for the leaderboard (calls `/v2/me` and upserts them into the KV `app_users` list) |
| `POST /oauth/refresh` | `{ refresh_token }` | `grant_type=refresh_token` + client_id/secret |

It returns 42's token JSON unchanged: `{ access_token, refresh_token, expires_in, created_at, … }`.
The frontend builds the refresh URL by string-replacing `/oauth/token` → `/oauth/refresh` in `VITE_42_TOKEN_PROXY`.

### 4.2 Generic 42 API proxy
Any other path (`/v2/...`) is forwarded to `https://api.intra.42.fr` **with the client's headers**
(including the user's Bearer token). It re-exposes `X-Total`, `X-Page`, and `X-Per-Page` with CORS.
In production `VITE_42_API_BASE` points at the Worker, so **all** frontend 42 API calls go through it.
This exists only because of CORS. It is an open proxy (any path, any method).

### 4.3 Leaderboard
- KV `app_users`: JSON array of `{ login, display_name, image }`, appended on each login.
- Cron `0 * * * *` (hourly) → `buildLeaderboard`:
  1. Gets an **app token** (`client_credentials`).
  2. For each registered user, in batches of 10 in parallel, fetches every page of `/v2/users/:login/projects_users?filter[status]=finished`.
  3. Computes `standard`, `stream`, and `elective` (each capped); `general` is always 0; `total`; `percentage` rounded to 1 decimal.
  4. Sorts by percentage descending and writes KV `leaderboard` = `{ updated_at, students: [...] }`.
- `GET /leaderboard` returns that JSON, or `{ students: [], updated_at: null }`. **No auth is required.**

Response shape consumed by `LeaderboardView.vue`:
```ts
{ updated_at: string | null,
  students: { login, display_name, image, general, standard, stream, elective, total, percentage }[] }
```

### 4.4 Worker config
- Secrets: `CLIENT_ID`, `CLIENT_SECRET` (set by the deploy workflow from the `VITE_42_CLIENT_*` GitHub secrets).
- `CAMPUS_ID = "43"` (Abu Dhabi) is declared in `wrangler.toml` but **unused** in the code.
- KV namespace binding `LEADERBOARD_KV`.

---

## 5. Auth flow (current)

1. `redirectToLogin()` → `https://api.intra.42.fr/oauth/authorize?client_id&redirect_uri&response_type=code&scope=public projects`
2. 42 redirects to `/auth/callback?code=…` → `AuthCallbackView` → `exchangeCode(code)`
   - **Dev** (no `VITE_42_TOKEN_PROXY`): POSTs directly to `/oauth/token` through the Vite dev proxy, sending the client secret from `.env`.
   - **Prod**: POSTs JSON to the Worker.
3. `auth.setTokens()` stores `42_access_token`, `42_refresh_token`, and `42_expires_at` in **localStorage** and schedules a refresh 5 minutes before expiry.
4. `fetchUser()` calls `/v2/me`. On 401 it refreshes once, then logs out.
5. The frontend uses the **user's own 42 token** for all 42 API calls. The Worker's app token is used only by the leaderboard cron.

---

## 6. Environment variables

See `.env.example` at the repo root.

| Var | Where | Used by | Notes |
|---|---|---|---|
| `VITE_42_CLIENT_ID` | frontend | `oauth.ts` | public, needed for the authorize URL |
| `VITE_42_CLIENT_SECRET` | frontend (**dev only**) | `oauth.ts` | only used when `VITE_42_TOKEN_PROXY` is unset. `VITE_*` values are public, see §8 |
| `VITE_42_REDIRECT_URI` | frontend | `oauth.ts` | must match the 42 app's redirect URI, e.g. `http://localhost:5173/auth/callback` |
| `VITE_42_API_BASE` | frontend | all API calls | empty in dev (Vite proxy handles `/v2` and `/oauth`). In prod it is the Worker URL |
| `VITE_42_TOKEN_PROXY` | frontend | `oauth.ts` | full URL ending in `/oauth/token`. If set, exchange and refresh go through it |
| `VITE_WORKER_URL` | frontend | `LeaderboardView.vue` | optional, falls back to `VITE_42_API_BASE` |
| `GITHUB_ACTIONS`, `GITHUB_REPOSITORY` | build | `vite.config.ts` | sets `base` to `/<repo>/` for GitHub Pages |
| `CLIENT_ID`, `CLIENT_SECRET` | Worker secrets | `worker/index.js` | |
| `CAMPUS_ID` | Worker var | unused | |

GitHub secrets used by CI: `VITE_42_CLIENT_ID`, `VITE_42_CLIENT_SECRET`, `VITE_42_REDIRECT_URI`,
`VITE_42_TOKEN_PROXY` (Worker base URL **without** a path; CI appends `/oauth/token`),
`CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`.

Local dev: `.env` with `VITE_42_CLIENT_ID`, `VITE_42_CLIENT_SECRET`, `VITE_42_REDIRECT_URI`; leave the
API base and proxy empty. The leaderboard **does not work in dev**, because there is no Worker and the
Vite proxy does not forward `/leaderboard`.

---

## 7. Deployment (current)

`.github/workflows/deploy.yml` runs on push to `master`:
- `build`: `npm ci && npm run build` with env from secrets, writes `dist/404.html` for SPA routing, and uploads it to Pages.
- `deploy-worker`: `wrangler deploy` with secrets.
- `deploy`: GitHub Pages.

The frontend is static on GitHub Pages under `/<repo>/`. `.env.production` is committed with placeholder values,
but CI env vars override it.

---

## 8. Known issues and gotchas

1. **Client secret passed to the frontend build.** CI sets `VITE_42_CLIENT_SECRET` during `npm run build`.
   Every `VITE_*` var is inlined into the public bundle wherever it is referenced. Whether the dead dev branch
   in `oauth.ts` is tree-shaken away has **not been verified**. Check with `grep -r <secret> dist/`. Remove that
   env line from CI, and rotate the secret if it leaked.
2. **Projects are not paginated in the frontend.** `useProjects` fetches a single page (`per_page=100`).
   Students with more than 100 finished projects may be undercounted, while the Worker paginates fully.
   As a result, profile and leaderboard numbers can differ.
3. **Percentage mismatch.** The Worker rounds to 1 decimal; the frontend does not round.
4. **The leaderboard only contains users who logged in through the app**, not the whole campus. `CAMPUS_ID` hints at a plan to cover the campus.
5. **KV `app_users` is read-modify-write**, so concurrent logins can lose an entry.
6. **Open proxy**: the Worker forwards any path and method to the 42 API.
7. `GET /leaderboard` is public, and the view only checks for a token client-side.
8. Tokens are kept in localStorage (XSS exposure).
9. Dead code: `stores/counter.ts`, `ProjectLookup.vue` (commented out), and unused `markColor`/`formatDate` and `.project-card` styles in `Profile.vue`.
10. The README's setup section is outdated: it doesn't mention the API base and proxy vars or the leaderboard.
11. 42 API rate limits (default per app: about **2 req/s and 1200 req/hour**). The hourly cron makes ≥1 call per user, so this caps the leaderboard size. The Go backend needs a rate limiter.

---

## 9. Migration notes: Worker → Go/Fiber/SQLite backend

Things the backend must provide so the frontend keeps working, plus the planned additions:

**Must replace**
- `POST /oauth/token` and `POST /oauth/refresh`, with the same request and response shapes, or change `oauth.ts`.
- `GET /leaderboard`, with the same response shape.
- 42 API access for the SPA. Options: (a) keep a narrow `/v2/*` passthrough, possibly restricted to an
  allowlist of the GETs the SPA uses (`/v2/me`, `/v2/users`, `/v2/users/:x`, `/v2/users/:x/projects_users`,
  `/v2/cursus/21/projects`); or (b) move data fetching into backend endpoints that serve from SQLite.
- Hourly refresh job (goroutine with ticker, or cron lib) using a `client_credentials` app token, rate-limited.
- CORS for the frontend origin. Expose `X-Total`, `X-Page`, and `X-Per-Page` if passthrough is kept.

**SQLite (sketch)**
- `users` (id, login, display_name, image, registered_at, last_synced_at)
- `project_users` (user_id, project_id, validated, final_mark, status, marked_at): raw 42 data
- `progress_snapshots` (user_id, general, standard, stream, elective, total, percentage, taken_at): progress history for tracking over time
- Curriculum as Go data or seed tables, a **single source of truth** (see §3).

**Then remove**: `worker/`, `wrangler.toml`, the `deploy-worker` CI job, the Cloudflare secrets, the `.wrangler/` gitignore entry,
and the `.env.production` Worker placeholder. Also update `VITE_42_API_BASE` / `VITE_42_TOKEN_PROXY` to point at the new backend.

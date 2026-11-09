# 42 AD Bachelor's Tracker — Architecture & Developer Notes

Working notes for the project as of Oct 2026 (last feature work: May 2026).

**Current state:** the browser talks only to the self-hosted **Go + Fiber + SQLite** backend (`backend/`) under
`/api/v1`, on the same origin. The backend signs users in with 42 OAuth, stores students and their validated
projects, and refreshes them from 42 when stale. The browser never calls the 42 API and never holds a 42 token.
Sections 4 and 7 describe the legacy Cloudflare Worker and GitHub Pages deploy, kept until production moves to the VPS.

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
frontend/                    Vue SPA, run all npm commands here
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
  public/404.html, index.html  GitHub Pages SPA deep-link redirect trick (sessionStorage 'spa-redirect')
  package.json, vite.config.ts, tsconfig*.json, eslint.config.ts, .oxlintrc.json, .nvmrc
  .env.example, .env.production
worker/index.js                Cloudflare Worker (to be removed), see §4
wrangler.toml                  Worker config: KV binding, hourly cron, CAMPUS_ID var
.github/workflows/deploy.yml   Build frontend/ → GitHub Pages; deploy Worker via wrangler-action
docs/                          These notes
```

Stack: Vue 3.5, TypeScript 6, Vite 8, Pinia 3, vue-router 5, PrimeVue 4 (Aura), Tailwind v4,
Chart.js 4 / vue-chartjs. Node `^20.19 || >=22.12` (see `frontend/.nvmrc`). Lint: oxlint + eslint.

---

## 3. Curriculum data (`frontend/src/utils/data.ts`)

- `Project.id` can be:
  - `number`: a single 42 project ID.
  - `number[]`: alternatives. Any one of them validated counts, e.g. "Minitalk **or** Pipex", or old and new versions of the same project.
  - `null`: not trackable through the API (GEN seminars, BAI 499 Internship).
- A project counts as completed when its `projects_users` entry has `validated? == true`.
- BAI 498 (Senior Capstone Project) is **not tracked** (no project ID). It used to share ID 1337 with CCP 161 (ft_transcendence).
- BAI 499 (Internship) is completed by **Work Experience I** (ID 1638).
- EMP 101 is the `ft_resume` project (ID 2623; it was renamed from `42_collaborative_resume` on intra).
- Each module has a `team` value (`solo` / `group` / `mixed`) taken from the 42 API project sessions (Abu Dhabi campus 43 first,
  then cursus 21). `mixed` means the alternatives or sessions differ. It is shown in the projects table.
- 42 cursus ID **21** = 42cursus (used by project search).

⚠️ **The curriculum is duplicated** in `worker/index.js` (`STANDARD` / `STREAM` / `ELECTIVE` arrays).
Any change must be made in both places. In the new backend this should exist **once**: the backend owns it,
and the frontend either fetches it or keeps a generated copy.

---

### Streams
A student picks **one stream**; only its modules count toward the 14 stream credits, and credits from different
streams do not add up. Until a student picks (`students.stream` is NULL), their **best** stream (most credits;
ties go to the first stream) is counted. The choice is saved with `PUT /api/v1/me/stream` and can be changed.

| Key | Name | Modules |
|---|---|---|
| `gaming-graphics` | Gaming and Graphics | GAM 222, GAM 223, GAM 231, GAM 301, MOB 241, ALG 211, GFX 121 |
| `mobile-web` | Mobile and Web | MOB 121, MOB 122, MOB 211, MOB 241, WEB 101, WEB 202, WEB 231, SEC 211 |
| `kernel-systems` | Kernel and Systems | KER 221, KER 231, SYS 111, SYS 112, SYS 113, SYS 211, SYS 221 |
| `network-security` | Network and Security | NET 111, NET 121, SEC 221, SEC 222, SEC 211 |

MOB 241 and SEC 211 belong to two streams and count in either. The rule lives in `curriculum.Evaluate` (backend)
and is mirrored in `useProgress.ts` / `STREAMS` in `utils/data.ts` (frontend).

### Self-reported modules
Some results are recorded by the Learning Team on ATRIUM (atrium.aevsoft.cc), not on 42: **GEN 101, GEN 201,
GEN 301, BAI 498, BAI 499**, plus **ARI 111** (42 does not mark Python for Data Science validated even when all its
modules are done). ATRIUM has no API, so students tick these themselves in the Status column of their own project
tables (`PUT /api/v1/me/modules {"code","completed"}`, stored in `student_marked_modules`). A ticked module counts
like a validated one everywhere, including the leaderboard; other students see it marked as self-reported. The table
links to the student's ATRIUM study plan so they can check. The list lives in `adBachelorSelfReported` (backend)
and `SELF_REPORTED` in `utils/data.ts` (frontend).

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

## 5. Auth flow and data access

1. "Log In" (shows a "Redirecting to 42…" overlay) → `GET /api/v1/auth/login`: the backend sets a short-lived
   `oauth_state` cookie and redirects to 42's authorize page with `redirect_uri = FT_REDIRECT_URI`
   (default `PUBLIC_URL/auth/callback`; it must be registered on the 42 app).
2. 42 → the SPA page `/auth/callback?code&state` ("Signing you in…" loader), which posts the query to
   `POST /api/v1/auth/callback`. The backend checks the state against the cookie, exchanges the code, reads
   `/v2/me` with the user's token, saves the student as **registered**, syncs their validated projects and
   creates a session: httpOnly `session` cookie (SameSite=Lax, Secure on https, `SESSION_TTL`, default 30 days),
   204. Failures return 400 `{"error": "denied"|"state"|"failed"}` and the SPA shows `/?login_error=…`.
   The 42 user token is discarded after the login.
3. The SPA calls `GET /api/v1/me` on start (router guard, "Loading…" until it answers). `/profile/:login` and
   `/leaderboard` require a session.
4. Reads: `/api/v1/me`, `/api/v1/students/:login` return stored data. If it is older than `STUDENT_MAX_AGE`
   (default 2m) it is refreshed from 42 **before** responding, with the app token (`client_credentials`);
   concurrent requests for one student share one refresh; if 42 fails, stored data is served. Unknown logins are
   fetched and stored but **not registered** (not ranked, searched or background-synced).
5. Search (`/api/v1/students?search=`) only covers registered students, from the DB.
6. Only validated projects are stored, so in-progress projects and location are not shown.
7. `POST /api/v1/auth/logout` deletes the session. Expired sessions are purged by the hourly job.

---

## 6. Environment variables

The frontend needs **none** (see `frontend/.env.example`). Backend: see `backend/.env.example`.

| Var | Default | Notes |
|---|---|---|
| `FT_CLIENT_ID`, `FT_CLIENT_SECRET` | required | 42 OAuth app credentials |
| `PUBLIC_URL` | `http://localhost:5173` | where users open the app; cookies are Secure when https |
| `FT_REDIRECT_URI` | `PUBLIC_URL/auth/callback` | the SPA callback page; must be registered on the 42 app |
| `SESSION_TTL` | `720h` | sign-in lifetime |
| `STUDENT_MAX_AGE` | `2m` | stored data older than this is refreshed before an API response |
| `SYNC_INTERVAL` / `SYNC_MAX_AGE` | `1h` / `24h` | background sync of registered students |
| `FT_RATE_PER_SECOND` / `FT_RATE_PER_HOUR` | `2` / `1200` | 42 API limits shared by all backend calls |
| `HTTP_ADDR`, `DATABASE_PATH`, `LOG_LEVEL` | `:8080`, `./data/app.db`, `info` | |

Local dev: run the backend (`make run` in `backend/`) and `npm run dev` in `frontend/`. Vite proxies `/api`
to `localhost:8080`, so the app and API share the `localhost:5173` origin like in production.

Legacy (Worker / GitHub Pages only): Worker secrets `CLIENT_ID`, `CLIENT_SECRET`; GitHub secrets `VITE_42_*`,
`CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`.

---

## 7. Deployment (current)

`.github/workflows/deploy.yml` runs on push to `master`:
- `build` (working dir `frontend/`): `npm ci && npm run build` with env from secrets, writes `dist/404.html` for SPA routing, and uploads `frontend/dist` to Pages.
- `deploy-worker`: `wrangler deploy` with secrets.
- `deploy`: GitHub Pages.

The frontend is static on GitHub Pages under `/<repo>/`. `frontend/.env.production` is committed with placeholder values,
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

## 9. Migration status: Worker → Go/Fiber/SQLite backend

- ✅ OAuth sign-in with server-side sessions (`/api/v1/auth/*`); no 42 tokens in the browser.
- ✅ Leaderboard (`/api/v1/leaderboard`, now behind the session) and the hourly background sync with rate limits.
- ✅ The SPA reads everything from the backend (`/me`, `/students/:login`, `/students?search=`); no 42 API passthrough.
- ✅ Avatars through `/api/v1/avatar` (the 42 CDN requires auth).
- ✅ Deployment: `compose.yaml` with an nginx entry point (`deploy/nginx/`) routing `/api/*` to the backend and
  everything else to the frontend (its own nginx serving `dist/` with SPA fallback). Frontend and backend are on
  separate internal networks with no published ports; TLS is terminated by the hosting platform (Coolify).
  See the root README.
- ✅ The GitHub Pages/Worker deploy workflow is removed; Coolify deploys from the repository. Still to do outside
  the repo: turn off GitHub Pages (Settings → Pages) and delete the Cloudflare Worker and its KV namespace.
- ⏳ Import the ~60 students registered in the Worker's KV (`app_users`; has logins, not 42 IDs).
- ⏳ Frontend still holds its own copy of the curriculum (`utils/data.ts`) and computes progress for the
  projects table; `/api/v1/curriculum` could replace it.

**Then remove**: `worker/`, `wrangler.toml`, the Pages/Worker CI jobs, the Cloudflare secrets, the `.wrangler/`
gitignore entry, and the GitHub Pages SPA redirect in `frontend/index.html` and `frontend/public/404.html`.
